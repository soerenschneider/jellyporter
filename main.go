package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/soerenschneider/jellyporter/internal"
	"github.com/soerenschneider/jellyporter/internal/config"
	"github.com/soerenschneider/jellyporter/internal/database/sqlite"
	"github.com/soerenschneider/jellyporter/internal/events"
	"github.com/soerenschneider/jellyporter/internal/events/webhook"
	"github.com/soerenschneider/jellyporter/internal/jellyfin"
	"github.com/soerenschneider/jellyporter/internal/metrics"
	"go.uber.org/multierr"

	"github.com/rs/zerolog/log"
)

const (
	waitgroupTimeout  = 30 * time.Second
	envConfigPath     = "JELLYPORTER_CONFIG_PATH"
	defaultConfigPath = "/etc/jellyporter.yaml"
)

var (
	flagConfigPath string
	flagDebug      bool
	flagVersion    bool
	flagDaemonize  bool

	BuildVersion = "dev"
	CommitHash   = "unknown"
	GoVersion    = "unknown"
)

type eventSource interface {
	Listen(ctx context.Context, events chan events.EventSyncRequest, wg *sync.WaitGroup) error
}

func parseFlags() {
	flag.StringVar(&flagConfigPath, "config", "", "Path to YAML config file")
	flag.BoolVar(&flagDaemonize, "daemonize", false, "Daemonize Jellyporter")
	flag.BoolVar(&flagDebug, "debug", false, "Enable debug logging")
	flag.BoolVar(&flagVersion, "version", false, "Print version and exit")

	flag.Parse()
}

func main() {
	parseFlags()

	if flagVersion {
		fmt.Printf("%s %s go%s\n", BuildVersion, CommitHash, GoVersion)
		os.Exit(0)
	}
	metrics.Version.WithLabelValues(BuildVersion, GoVersion).Set(1)
	metrics.Heartbeat.SetToCurrentTime()

	// Load config path from flag or env
	configPath := flagConfigPath
	if configPath == "" {
		configPath = os.Getenv(envConfigPath)
	}
	if configPath == "" {
		configPath = defaultConfigPath
	}

	log.Info().Msgf("Using config file %s", configPath)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("configuration invalid")
	}

	clients := make(map[string]internal.JellyfinClient)
	for name, c := range cfg.Clients {
		clients[name] = jellyfin.NewJellyfinClient(c.Address, c.ApiKey, c.User)
	}

	db, err := sqlite.New(cfg.Database.Path)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not create sqlite db")
	}

	app, err := internal.NewApp(clients, db, cfg)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not build app")
	}

	if !flagDaemonize {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := app.SyncOnce(ctx)
		// shadowing err is intentional here. If writing the metrics fails, we log it and use the err above to signal
		// whether the app ran successfully or not.
		if cfg.MetricsPath != "" {
			if err := metrics.WriteMetrics(cfg.MetricsPath); err != nil {
				log.Error().Err(err).Msg("could not write metrics")
			}
		}

		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	runDaemon(app, cfg)
}

func runDaemon(app *internal.App, cfg *config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}

	webhookRequests := make(chan events.EventSyncRequest, 1)

	eventSources, err := buildEventSources(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("could not build eventsources")
	}

	for _, source := range eventSources {
		go func() {
			if err := source.Listen(ctx, webhookRequests, wg); err != nil {
				log.Error().Err(err).Msg("error listening on event source")
			}
		}()
	}

	go app.Sync(ctx, wg, webhookRequests)
	go func() {
		if cfg.MetricsAddr != "" {
			if err := metrics.StartServer(ctx, cfg.MetricsAddr, wg); err != nil {
				log.Fatal().Err(err).Msg("could not start metrics server")
			}
		}
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	<-sigc
	log.Info().Msg("Received signal to quit")
	cancel()

	// wait on all members of the wg but end forcefully after timeout has passed
	gracefulExitDone := make(chan struct{})

	go func() {
		log.Info().Msgf("Waiting %v for components to shut down gracefully", waitgroupTimeout)
		wg.Wait()
		close(gracefulExitDone)
	}()

	close(webhookRequests)

	select {
	case <-gracefulExitDone:
		log.Info().Msg("All components shut down gracefully within the timeout")
	case <-time.After(waitgroupTimeout):
		log.Error().Msg("Killing process forcefully")
	}
}

func buildEventSources(cfg *config.Config) ([]eventSource, error) {
	if cfg.EventSources == nil {
		return nil, nil
	}

	var errs error
	var eventSources []eventSource
	if cfg.EventSources.WebhookServer != nil {
		var webhookServerOpts []webhook.WebhookServerOpts

		if cfg.EventSources.WebhookServer.Path != "" {
			webhookServerOpts = append(webhookServerOpts, webhook.WithPath(cfg.EventSources.WebhookServer.Path))
		}

		webhookServer, err := webhook.New(cfg.EventSources.WebhookServer.Addr, webhookServerOpts...)
		if err != nil {
			errs = multierr.Append(errs, err)
		} else {
			eventSources = append(eventSources, webhookServer)
		}
	}

	return eventSources, errs
}
