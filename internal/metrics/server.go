package metrics

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"
	"github.com/rs/zerolog/log"
)

func StartServer(ctx context.Context, addr string, wg *sync.WaitGroup) error {
	if wg == nil {
		return errors.New("nil waitgroup passed")
	}

	wg.Add(1)
	defer wg.Done()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       3 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	errChan := make(chan error)
	go func() {
		log.Info().Str("address", addr).Msg("Starting metrics server")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("can not start metrics server: %w", err)
		}
	}()

	heartbeatTimer := time.NewTicker(1 * time.Minute)
	defer heartbeatTimer.Stop()

	for {
		select {
		case <-heartbeatTimer.C:
			Heartbeat.SetToCurrentTime()
		case <-ctx.Done():
			log.Info().Msg("Stopping metrics server")
			return server.Shutdown(ctx)
		case err := <-errChan:
			return err
		}
	}
}

func WriteMetrics(path string) error {
	path, err := url.JoinPath(path, "jellyporter.prom")
	if err != nil {
		return err
	}

	log.Info().Str("path", path).Msg("Dumping metrics")
	metrics, err := dumpMetrics()
	if err != nil {
		return err
	}

	err = os.WriteFile(path, []byte(metrics), 0644) // #nosec: G306
	return err
}

func dumpMetrics() (string, error) {
	var buf = &bytes.Buffer{}
	fmt := expfmt.NewFormat(expfmt.TypeTextPlain)
	enc := expfmt.NewEncoder(buf, fmt)

	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return "", err
	}

	for _, f := range families {
		// Writing these metrics will cause a duplication error with other tools writing the same metrics
		if !strings.HasPrefix(f.GetName(), "go_") {
			if err := enc.Encode(f); err != nil {
				log.Info().Msgf("could not encode metric: %s", err.Error())
			}
		}
	}

	return buf.String(), nil
}
