package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
	"unicode"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/jellyporter/internal/database/sqlite/generated"
	"github.com/soerenschneider/jellyporter/internal/jellyfin"
	"github.com/soerenschneider/jellyporter/internal/metrics"
)

type SQLiteJellyDb struct {
	db        *sql.DB
	generated *generated.Queries
}

func New(dbPath string) (*SQLiteJellyDb, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	gen := generated.New(db)
	ret := &SQLiteJellyDb{
		db:        db,
		generated: gen,
	}

	return ret, ret.Migrate(context.Background())
}

func MustNew(dbPath string) *SQLiteJellyDb {
	db, err := New(dbPath)
	if err != nil {
		log.Fatal().Err(err).Msg("could not create new database")
	}

	return db
}

func (q *SQLiteJellyDb) GetMoviesWithUpdatedUserData(ctx context.Context, server string) ([]ItemWithUpdatedUserData, error) {
	start := time.Now()
	unwatched, err := q.generated.GetMovieWithGreatestWatchedDate(ctx, server)
	if err != nil {
		metrics.DbQueryErrors.WithLabelValues("GetMovieWithGreatestWatchedDate").Inc()
		return nil, err
	}
	metrics.DbQueriesTime.WithLabelValues("GetMovieWithGreatestWatchedDate").Observe(time.Since(start).Seconds())

	ret := make([]ItemWithUpdatedUserData, len(unwatched))
	for idx, movie := range unwatched {
		ret[idx] = ItemWithUpdatedUserData{
			Name:                 movie.Name,
			LocalID:              movie.LocalID,
			WatchedDate:          movie.WatchedDate,
			WatchedProgress:      movie.WatchedProgress,
			WatchedPositionTicks: movie.WatchedPositionTicks,
			IsFavorite:           movie.IsFavorite,
		}
	}

	return ret, nil
}

func (q *SQLiteJellyDb) InsertMovie(ctx context.Context, server string, movie jellyfin.Item) error {
	params := MovieToInsertMovieParam(server, movie)
	return q.generated.InsertMovie(ctx, params)
}

func (q *SQLiteJellyDb) RemoveItemsNotSeenSince(ctx context.Context, server string, itemType jellyfin.ItemType, notSeenSince time.Time) error {
	if notSeenSince.IsZero() {
		return errors.New("notSeenSince must not be zero")
	}

	log.Info().Int64("not_seen_since", notSeenSince.Unix()).Msgf("Deleting %ss not seen since %v", itemType, notSeenSince.Format("2006-01-02 15:04:05"))

	switch itemType {
	case jellyfin.ItemEpisode:
		return q.RemoveEpisodesNotSeenSince(ctx, server, notSeenSince)
	case jellyfin.ItemMovie:
		return q.RemoveMoviesNotSeenSince(ctx, server, notSeenSince)
	default:
		return fmt.Errorf("unknown itemtype: %v", itemType)
	}
}

func (q *SQLiteJellyDb) RemoveMoviesNotSeenSince(ctx context.Context, server string, since time.Time) error {
	start := time.Now()

	if err := q.generated.RemoveMoviesNotSeenSince(ctx, generated.RemoveMoviesNotSeenSinceParams{
		Server: server,
		Since:  since.Unix(),
	}); err != nil {
		metrics.DbQueryErrors.WithLabelValues("RemoveMoviesNotSeenSince").Inc()
		return err
	}

	metrics.DbQueriesTime.WithLabelValues("RemoveMoviesNotSeenSince").Observe(time.Since(start).Seconds())
	return nil
}

func (q *SQLiteJellyDb) RemoveEpisodesNotSeenSince(ctx context.Context, server string, since time.Time) error {
	start := time.Now()

	if err := q.generated.RemoveEpisodesNotSeenSince(ctx, generated.RemoveEpisodesNotSeenSinceParams{
		Server: server,
		Since:  since.Unix(),
	}); err != nil {
		metrics.DbQueryErrors.WithLabelValues("RemoveEpisodesNotSeenSince").Inc()
		return err
	}

	metrics.DbQueriesTime.WithLabelValues("RemoveEpisodesNotSeenSince").Observe(time.Since(start).Seconds())
	return nil
}

func (q *SQLiteJellyDb) InsertMovies(ctx context.Context, server string, movies []jellyfin.Item) error {
	start := time.Now()
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		metrics.DbQueryErrors.WithLabelValues("InsertMovies").Inc()
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	queries := q.generated.WithTx(tx)
	for _, movie := range movies {
		params := MovieToInsertMovieParam(server, movie)
		if err := queries.InsertMovie(ctx, params); err != nil {
			metrics.DbQueryErrors.WithLabelValues("InsertMovies").Inc()
			return err
		}
	}

	err = tx.Commit()
	metrics.DbQueriesTime.WithLabelValues("InsertMovies").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.DbQueryErrors.WithLabelValues("InsertMovies").Inc()
	}
	return err
}

func (q *SQLiteJellyDb) GetEpisodesWithUpdatedUserData(ctx context.Context, server string) ([]ItemWithUpdatedUserData, error) {
	start := time.Now()
	unwatched, err := q.generated.GetEpisodeWithGreatestWatchedDate(ctx, server)
	if err != nil {
		metrics.DbQueryErrors.WithLabelValues("GetEpisodesWithUpdatedUserData").Inc()
		return nil, err
	}
	metrics.DbQueriesTime.WithLabelValues("GetEpisodeWithGreatestWatchedDate").Observe(time.Since(start).Seconds())

	ret := make([]ItemWithUpdatedUserData, len(unwatched))
	for idx, episode := range unwatched {
		ret[idx] = ItemWithUpdatedUserData{
			LocalID:              episode.LocalID,
			Name:                 episode.Name,
			SeriesName:           episode.SeriesName,
			WatchedDate:          episode.WatchedDate,
			WatchedProgress:      episode.WatchedProgress,
			WatchedPositionTicks: episode.WatchedPositionTicks,
			IsFavorite:           episode.IsFavorite,
		}
	}

	return ret, nil
}

func (q *SQLiteJellyDb) InsertItems(ctx context.Context, server string, itemType jellyfin.ItemType, items []jellyfin.Item) error {
	switch itemType {
	case jellyfin.ItemEpisode:
		return q.InsertEpisodes(ctx, server, items)
	case jellyfin.ItemMovie:
		return q.InsertMovies(ctx, server, items)
	default:
		return fmt.Errorf("unknown type: %s", itemType)
	}
}

func (q *SQLiteJellyDb) InsertEpisodes(ctx context.Context, server string, episodes []jellyfin.Item) error {
	start := time.Now()
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		metrics.DbQueryErrors.WithLabelValues("InsertEpisodes").Inc()
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	queries := q.generated.WithTx(tx)
	for _, episode := range episodes {
		params := EpisodeToInsertEpisodeParam(server, episode)
		if err := queries.InsertEpisode(ctx, params); err != nil {
			metrics.DbQueryErrors.WithLabelValues("InsertEpisodes").Inc()
			return err
		}
	}

	err = tx.Commit()
	metrics.DbQueriesTime.WithLabelValues("InsertEpisodes").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.DbQueryErrors.WithLabelValues("InsertEpisodes").Inc()
		return err
	}
	return nil
}

func (q *SQLiteJellyDb) InsertEpisode(ctx context.Context, server string, episode jellyfin.Item) error {
	params := EpisodeToInsertEpisodeParam(server, episode)
	return q.generated.InsertEpisode(ctx, params)
}

func (q *SQLiteJellyDb) InsertChangelog(ctx context.Context, server string, change ChangelogData) error {
	start := time.Now()

	params := generated.InsertChangelogParams{
		Server:                  server,
		LocalID:                 change.LocalID,
		Date:                    start.Unix(),
		NewWatchedDate:          change.NewWatchedDate,
		NewWatchedProgress:      change.NewWatchedProgress,
		NewWatchedPositionTicks: change.NewWatchedPositionTicks,
		NewIsFavorite:           change.NewIsFavorite,
	}

	if err := q.generated.InsertChangelog(ctx, params); err != nil {
		metrics.DbQueryErrors.WithLabelValues("InsertChangelog").Inc()
		return err
	}

	metrics.DbQueriesTime.WithLabelValues("InsertChangelog").Observe(time.Since(start).Seconds())
	return nil
}

func (q *SQLiteJellyDb) UpsertState(ctx context.Context, server string, itemType jellyfin.ItemType, ts time.Time) error {
	args := generated.UpsertStateParams{
		Server:   server,
		Type:     string(itemType),
		LastSync: ts.Unix(),
	}

	return q.generated.UpsertState(ctx, args)
}

func (q *SQLiteJellyDb) GetState(ctx context.Context, server string, itemType jellyfin.ItemType) (time.Time, error) {
	arg := generated.GetLastCheckParams{
		Server: server,
		Type:   string(itemType),
	}
	lastSync, err := q.generated.GetLastCheck(ctx, arg)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(lastSync, 0), nil
}

type ChangelogData struct {
	LocalID                 string
	NewWatchedDate          int64
	NewWatchedProgress      float64
	NewWatchedPositionTicks int64
	NewIsFavorite           bool
}

type ItemWithUpdatedUserData struct {
	LocalID              string
	Name                 string
	SeriesName           string
	WatchedDate          int64
	WatchedProgress      float64
	WatchedPositionTicks int64
	IsFavorite           bool
}

func (m *ItemWithUpdatedUserData) AsUserData() jellyfin.UserDataUpdate {
	return jellyfin.UserDataUpdate{
		PlaybackPositionTicks: &m.WatchedPositionTicks,
		PlayedPercentage:      &m.WatchedProgress,
		LastPlayedDate:        time.Unix(m.WatchedDate, 0),
		Played:                true,
		IsFavorite:            &m.IsFavorite,
	}
}

func SanitizeAndParseInt64(input string) int64 {
	var filtered []rune
	for _, r := range input {
		if unicode.IsDigit(r) || r == '-' { // Include digits and negative sign
			filtered = append(filtered, r)
		}
	}

	cleaned := string(filtered)
	if cleaned == "" || cleaned == "-" {
		return 0
	}

	result, err := strconv.ParseInt(cleaned, 10, 64)
	if err != nil {
		return 0
	}

	return result
}

func EpisodeToInsertEpisodeParam(server string, episode jellyfin.Item) generated.InsertEpisodeParams {
	imdbId := SanitizeAndParseInt64(episode.ProviderIDs.IMDB)
	tmdbId := SanitizeAndParseInt64(episode.ProviderIDs.TMDB)
	tvdbId := SanitizeAndParseInt64(episode.ProviderIDs.TVDB)

	var watchedDate int64 = 0
	if !episode.UserData.LastPlayedDate.IsZero() {
		watchedDate = episode.UserData.LastPlayedDate.Unix()
	}
	return generated.InsertEpisodeParams{
		Server:     server,
		Name:       episode.Name,
		LocalID:    episode.ID,
		SeriesName: episode.SeriesName,
		SeasonName: episode.SeasonName,
		ImdbID: sql.NullInt64{
			Int64: imdbId,
			Valid: imdbId != 0,
		},
		TmdbID: sql.NullInt64{
			Int64: tmdbId,
			Valid: tmdbId != 0,
		},
		TvdbID: sql.NullInt64{
			Int64: tvdbId,
			Valid: tvdbId != 0,
		},
		WatchedDate:          watchedDate,
		WatchedPositionTicks: episode.UserData.PlaybackPositionTicks,
		WatchedProgress:      episode.UserData.PlayedPercentage,
		Runtime:              episode.Runtime,
		IsFavorite:           episode.UserData.IsFavorite,
	}
}
func MovieToInsertMovieParam(server string, movie jellyfin.Item) generated.InsertMovieParams {
	imdbId := SanitizeAndParseInt64(movie.ProviderIDs.IMDB)
	tmdbId := SanitizeAndParseInt64(movie.ProviderIDs.TMDB)
	var watchedDate int64 = 0
	if !movie.UserData.LastPlayedDate.IsZero() {
		watchedDate = movie.UserData.LastPlayedDate.Unix()
	}
	return generated.InsertMovieParams{
		Server:  server,
		Name:    movie.Name,
		LocalID: movie.ID,
		ImdbID: sql.NullInt64{
			Int64: imdbId,
			Valid: imdbId != 0,
		},
		TmdbID: sql.NullInt64{
			Int64: tmdbId,
			Valid: tmdbId != 0,
		},
		WatchedDate:          watchedDate,
		WatchedPositionTicks: movie.UserData.PlaybackPositionTicks,
		WatchedProgress:      movie.UserData.PlayedPercentage,
		Runtime:              movie.Runtime,
		IsFavorite:           movie.UserData.IsFavorite,
	}
}

func (db *SQLiteJellyDb) Migrate(ctx context.Context) error {
	if schemaVersionReadError != nil {
		return schemaVersionReadError
	}

	var currentVersion int
	_ = db.db.QueryRowContext(ctx, `SELECT version FROM schema_version`).Scan(&currentVersion)

	log.Info().Msgf("Current DB schema at version %d, latest schema version is %d", currentVersion, schemaVersion)
	if currentVersion >= schemaVersion {
		return nil
	}

	migrations, err := GetMigrations()
	if err != nil {
		return err
	}

	for version := currentVersion; version < schemaVersion; version++ {
		newVersion := version + 1

		tx, err := db.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("can not start transaction %w", err)
		}

		sql := migrations[version]
		_, err = tx.ExecContext(ctx, string(sql))
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("[Migration v%d] %v", newVersion, err)
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM schema_version`); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("[Migration v%d] %v", newVersion, err)
		}

		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_version (version) VALUES ($1)`, newVersion); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("[Migration v%d] %v", newVersion, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("[Migration v%d] %v", newVersion, err)
		}
		log.Info().Msgf("Successfully migrated DB to version %d", newVersion)
	}

	return nil
}
