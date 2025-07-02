package internal

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/jellyporter/internal/config"
	"github.com/soerenschneider/jellyporter/internal/database/sqlite"
	"github.com/soerenschneider/jellyporter/internal/events"
	"github.com/soerenschneider/jellyporter/internal/jellyfin"
	"github.com/soerenschneider/jellyporter/internal/metrics"
	"go.uber.org/multierr"
)

const defaultCooldownDuration = 30 * time.Second

type JellyfinClient interface {
	GetUserId(ctx context.Context) (string, error)
	GetItems(ctx context.Context, userID string, opts jellyfin.ItemQueryOpts) (*jellyfin.ItemsResponse, error)
	UpdateUserData(ctx context.Context, userID, itemID string, data jellyfin.UserDataUpdate) error
}

type LibraryDb interface {
	InsertChangelog(ctx context.Context, server string, change sqlite.ChangelogData) error
	InsertItems(ctx context.Context, server string, itemType jellyfin.ItemType, episodes []jellyfin.Item) error

	GetMoviesWithUpdatedUserData(ctx context.Context, server string) ([]sqlite.ItemWithUpdatedUserData, error)
	GetEpisodesWithUpdatedUserData(ctx context.Context, server string) ([]sqlite.ItemWithUpdatedUserData, error)
	RemoveItemsNotSeenSince(ctx context.Context, server string, itemType jellyfin.ItemType, since time.Time) error

	UpsertState(ctx context.Context, server string, itemType jellyfin.ItemType, ts time.Time) error
	GetState(ctx context.Context, server string, itemType jellyfin.ItemType) (time.Time, error)
}

type App struct {
	clients map[string]JellyfinClient
	db      LibraryDb

	mutex sync.Mutex

	// cooldown is a cooldown phase for when receiving a burst of requests from the webhook
	cooldown      atomic.Bool
	cooldownTimer time.Duration

	// counter tracks invocations to control fetching deltas or full data from Jellyfin
	counter                 atomic.Int32
	syncIntervalMinutes     int32
	fullSyncIntervalMinutes int32
}

func NewApp(clients map[string]JellyfinClient, db LibraryDb, cfg *config.Config) (*App, error) {
	if len(clients) == 0 {
		return nil, errors.New("empty client map provided")
	}

	if db == nil {
		return nil, errors.New("nil implementation supplied")
	}

	if cfg == nil {
		return nil, errors.New("nil config passed")
	}

	app := &App{
		clients: clients,
		db:      db,

		cooldownTimer:           defaultCooldownDuration,
		syncIntervalMinutes:     int32(cfg.SyncIntervalMinutes),     //nolint G115
		fullSyncIntervalMinutes: int32(cfg.FullSyncIntervalMinutes), //nolint G115
	}

	return app, nil
}

func (a *App) Sync(ctx context.Context, wg *sync.WaitGroup, hook chan events.EventSyncRequest) {
	if wg == nil {
		log.Fatal().Msg("nil wg passed")
	}

	wg.Add(1)
	defer wg.Done()

	ticker := time.NewTicker(time.Duration(a.syncIntervalMinutes) * time.Minute)
	_ = a.SyncOnce(ctx)

	for {
		select {
		case event := <-hook:
			metrics.EventSourceRequestsTotal.WithLabelValues(event.Source).Inc()
			if a.cooldown.CompareAndSwap(false, true) {
				metrics.EventSourceCooldownPhases.Inc()
				log.Info().Str("source", event.Source).Str("metadata", event.Metadata).Msg("Received external request to sync data")

				cooldownCtx, cancel := context.WithTimeout(ctx, a.cooldownTimer)
				go func() {
					defer cancel()
					<-cooldownCtx.Done()
					a.cooldown.Store(false)
				}()

				select {
				case event.Response <- nil:
					// nop
				case <-time.After(1 * time.Second):
					log.Warn().Msg("hanging goroutine")
				}
				_ = a.SyncOnce(ctx)
			} else {
				metrics.EventSourceErrorsTotal.WithLabelValues(event.Source).Inc()
				log.Debug().Str("source", event.Source).Str("metadata", event.Metadata).Msgf("Not initiating sync due to having received too many requests in the last %v", a.cooldownTimer)
				event.Response <- errors.New("too many requests")
			}
		case <-ticker.C:
			_ = a.SyncOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (a *App) SyncOnce(ctx context.Context) error {
	defer func() {
		a.counter.Add(1)
		a.mutex.Unlock()
	}()
	// Prevent multiple goroutines running this code simultaneously
	a.mutex.Lock()

	start := time.Now()
	var errs error
	if err := a.syncMoviesWatchedState(ctx); err != nil {
		errs = multierr.Append(errs, err)
		log.Error().Err(err).Dur("duration", time.Since(start)).Msgf("Experienced errors while syncing 'watched' data for movies between %d servers", len(a.clients))
	}

	if err := a.syncEpisodesWatchedState(ctx); err != nil {
		errs = multierr.Append(errs, err)
		log.Error().Err(err).Dur("duration", time.Since(start)).Msgf("Experienced errors while syncing 'watched' data for episodes between %d servers", len(a.clients))
	}

	log.Info().Dur("duration", time.Since(start)).Msgf("Finished syncing data between %d servers", len(a.clients))
	return errs
}

func (a *App) syncMoviesWatchedState(ctx context.Context) error {
	err := a.fetchUpdatesFromJellyfin(ctx, jellyfin.ItemMovie)
	if err != nil {
		return err
	}

	return a.synchronizeUpdatedUserData(ctx, jellyfin.ItemMovie)
}

func (a *App) syncEpisodesWatchedState(ctx context.Context) error {
	err := a.fetchUpdatesFromJellyfin(ctx, jellyfin.ItemEpisode)
	if err != nil {
		return err
	}

	return a.synchronizeUpdatedUserData(ctx, jellyfin.ItemEpisode)
}

func (a *App) fetchUpdatesFromJellyfin(ctx context.Context, itemType jellyfin.ItemType) error {
	start := time.Now()
	var mutex sync.Mutex
	var errs error
	var wg sync.WaitGroup
	log.Info().Str("type", string(itemType)).Msg("Fetching data from Jellyfin")
	for server, client := range a.clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := a.fetchUpdateFromJellyfin(ctx, itemType, server, client); err != nil {
				mutex.Lock()
				errs = multierr.Append(errs, err)
				mutex.Unlock()
			}
		}()
	}
	wg.Wait()
	log.Info().Dur("duration", time.Since(start)).Str("type", string(itemType)).Msgf("Finished fetching items from %d servers", len(a.clients))
	return errs
}

func (a *App) fetchUpdateFromJellyfin(ctx context.Context, itemType jellyfin.ItemType, server string, client JellyfinClient) error {
	start := time.Now()

	userId, err := client.GetUserId(ctx)
	if err != nil {
		return err
	}

	lastSeenUserDataUpdate, err := a.db.GetState(ctx, server, itemType)
	if err != nil {
		log.Error().Err(err).Str("server", server).Str("type", string(itemType)).Msg("could not get state from DB")
	}
	opts := a.getQueryOpts(lastSeenUserDataUpdate, server, itemType)
	items, err := client.GetItems(ctx, userId, opts)
	if err != nil {
		return err
	}

	if !opts.IsDelta() {
		// Only set metric when fetching the full list of items
		metrics.TotalItems.WithLabelValues(server, strings.ToLower(string(itemType))).Set(float64(len(items.Items)))
		metrics.TotalItemsTimestamp.WithLabelValues(server, strings.ToLower(string(itemType))).SetToCurrentTime()
	}
	log.Info().Str("server", server).Str("type", string(itemType)).Msgf("Fetched %d items from server", len(items.Items))
	if err = a.db.InsertItems(ctx, server, itemType, items.Items); err != nil {
		return err
	}

	return a.db.RemoveItemsNotSeenSince(ctx, server, itemType, start)
}

func (a *App) synchronizeUpdatedUserData(ctx context.Context, itemType jellyfin.ItemType) error {
	var mutex sync.Mutex
	var errs error
	var wg sync.WaitGroup

	wg.Add(len(a.clients))
	for server, client := range a.clients {
		go func() {
			defer wg.Done()
			if err := a.synchronizeSingleUpdatedUserData(ctx, itemType, server, client); err != nil {
				mutex.Lock()
				errs = multierr.Append(errs, err)
				mutex.Unlock()
			}
		}()
	}

	wg.Wait()
	return errs
}

func (a *App) synchronizeSingleUpdatedUserData(ctx context.Context, itemType jellyfin.ItemType, server string, client JellyfinClient) error {
	var updated []sqlite.ItemWithUpdatedUserData
	var err error

	switch itemType {
	case jellyfin.ItemMovie:
		updated, err = a.db.GetMoviesWithUpdatedUserData(ctx, server)
	case jellyfin.ItemEpisode:
		updated, err = a.db.GetEpisodesWithUpdatedUserData(ctx, server)
	default:
		return fmt.Errorf("invalid type: %s", itemType)
	}
	if err != nil {
		return err
	}

	metrics.ItemsUpdatedUserData.WithLabelValues(server, strings.ToLower(string(itemType))).Set(float64(len(updated)))
	if len(updated) == 0 {
		if err := a.db.UpsertState(ctx, server, itemType, time.Now()); err != nil {
			log.Warn().Str("server", server).Err(err).Msg("could not upsert timestamp")
		} else {
			log.Info().Str("server", server).Time("ts", time.Now()).Int("updated", len(updated)).Str("type", string(itemType)).Msg("Upsert state")
		}
		return nil
	}

	log.Info().Str("server", server).Int("updated", len(updated)).Str("server", server).Str("type", string(itemType)).Msg("Found items with updated UserData")

	userId, err := client.GetUserId(ctx)
	if err != nil {
		return err
	}

	var lowestTimestamp int64 = math.MaxInt64
	var encounteredErrorsWhileUpdatingUserData bool
	var errs error
	for _, item := range updated {
		if item.WatchedDate < lowestTimestamp {
			lowestTimestamp = item.WatchedDate
		}

		if err := client.UpdateUserData(ctx, userId, item.LocalID, item.AsUserData()); err != nil {
			encounteredErrorsWhileUpdatingUserData = true
			errs = multierr.Append(errs, err)
			log.Error().Err(err).Str("id", item.LocalID).Str("name", item.Name).Str("server", server).Str("type", string(itemType)).Msg("Could not update UserData for item")
		} else {
			log.Info().Str("id", item.LocalID).Str("name", item.Name).Time("ts", time.Unix(item.WatchedDate, 0)).Str("server", server).Str("type", string(itemType)).Msg("Updated UserData for item")
			err := a.db.InsertChangelog(ctx, server, getChangelogData(item))
			if err != nil {
				log.Error().Str("server", server).Err(err).Msg("Could not insert changelog")
			}
		}
	}

	if !encounteredErrorsWhileUpdatingUserData {
		timestamp := time.Unix(lowestTimestamp-1, 0)
		log.Info().Str("server", server).Time("ts", timestamp).Int("updated", len(updated)).Str("type", string(itemType)).Msg("Upsert state")
		if err := a.db.UpsertState(ctx, server, itemType, timestamp); err != nil {
			log.Error().Str("server", server).Err(err).Str("type", string(itemType)).Msg("could not upsert timestamp")
		}
	}

	return errs
}

func (a *App) getQueryOpts(lastCheck time.Time, server string, itemType jellyfin.ItemType) jellyfin.ItemQueryOpts {
	cnt := a.counter.Load()
	if lastCheck.IsZero() || cnt%(a.fullSyncIntervalMinutes/a.syncIntervalMinutes) == 0 {
		log.Info().Str("server", server).Str("type", string(itemType)).Msg("Requesting full list of items")
		// querying for full list
		return jellyfin.ItemQueryOpts{
			Limit:      500,
			Since:      nil,
			StartIndex: 0,
			Type:       itemType,
		}
	}

	// querying for deltas only
	log.Info().Str("server", server).Time("since", lastCheck).Msg("Not requesting full list of movies, only deltas since last check")
	return jellyfin.ItemQueryOpts{
		Limit:      25,
		Since:      &lastCheck,
		StartIndex: 0,
		SortBy:     jellyfin.SortFieldDatePlayed,
		SortOrder:  jellyfin.SortOrderDescending,
		Type:       itemType,
	}
}

func getChangelogData(item sqlite.ItemWithUpdatedUserData) sqlite.ChangelogData {
	return sqlite.ChangelogData{
		LocalID:                 item.LocalID,
		NewWatchedDate:          item.WatchedDate,
		NewWatchedProgress:      item.WatchedProgress,
		NewWatchedPositionTicks: item.WatchedPositionTicks,
		NewIsFavorite:           item.IsFavorite,
	}
}
