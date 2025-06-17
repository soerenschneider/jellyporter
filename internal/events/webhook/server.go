package webhook

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/jellyporter/internal/events"
	"go.uber.org/multierr"
)

const defaultPath = "/webhook"

type WebhookServer struct {
	address string

	// optional
	path     string
	certFile string
	keyFile  string
}

type WebhookServerOpts func(*WebhookServer) error

func New(address string, opts ...WebhookServerOpts) (*WebhookServer, error) {
	if len(address) == 0 {
		return nil, errors.New("empty address provided")
	}

	w := &WebhookServer{
		address: address,
		path:    defaultPath,
	}

	var errs error
	for _, opt := range opts {
		if err := opt(w); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return w, errs
}

func (w *WebhookServer) IsTLSConfigured() bool {
	return len(w.certFile) > 0 && len(w.keyFile) > 0
}

func (w *WebhookServer) Listen(ctx context.Context, eventChan chan events.EventSyncRequest, wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()

	// isShuttingDown prevents writing to a isShuttingDown channel when the server is shutting down but still accepted a request
	isShuttingDown := atomic.Bool{}
	mux := http.NewServeMux()

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		syncRequest := events.EventSyncRequest{
			Source:   "webhook",
			Metadata: getIP(r),
		}

		if isShuttingDown.Load() {
			http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
			return
		}
		eventChan <- syncRequest
		w.WriteHeader(http.StatusOK)
	}
	mux.HandleFunc(w.path, handler)

	server := http.Server{
		Addr:              w.address,
		Handler:           mux,
		ReadTimeout:       3 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	errChan := make(chan error)
	go func() {
		if w.IsTLSConfigured() {
			if err := server.ListenAndServeTLS(w.certFile, w.keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errChan <- fmt.Errorf("can not start webhook_server server: %w", err)
			}
		} else {
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errChan <- fmt.Errorf("can not start webhook_server server: %w", err)
			}
		}
	}()

	select {
	case <-ctx.Done():
		isShuttingDown.Store(true)

		log.Info().Msg("Stopping webhook_server server")
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-errChan:
		return err
	}
}

func getIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	xrip := r.Header.Get("X-Real-IP")
	if xrip != "" {
		return xrip
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
