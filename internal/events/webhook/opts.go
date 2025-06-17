package webhook

import (
	"errors"
	"fmt"
)

func WithPath(path string) func(w *WebhookServer) error {
	return func(w *WebhookServer) error {
		if len(path) <= 1 {
			return fmt.Errorf("invalid path: %s", path)
		}

		w.path = path
		return nil
	}
}

func WithTLS(certFile, keyFile string) func(w *WebhookServer) error {
	return func(w *WebhookServer) error {
		if len(certFile) == 0 {
			return errors.New("empty certfile")
		}

		if len(keyFile) == 0 {
			return errors.New("empty keyfile")
		}

		w.certFile = certFile
		w.keyFile = keyFile
		return nil
	}
}
