package viewer

import (
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/coreofscience/go-bibx/models"
)

//go:embed templates/index.html
var indexTemplate []byte

// New creates an [http.ServeMux] configured to serve the interactive graph viewer and collection data.
func New(analysis *models.Analysis) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(indexTemplate); err != nil {
			slog.ErrorContext(r.Context(), "failed to write index template", "error", err)
		}
	})
	mux.HandleFunc("GET /collection.json.gz", func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if err := json.NewEncoder(gz).Encode(analysis); err != nil {
			slog.ErrorContext(r.Context(), "failed to encode analysis", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if err := gz.Close(); err != nil {
			slog.ErrorContext(r.Context(), "failed to close gzip writer", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(buf.Bytes()); err != nil {
			slog.ErrorContext(r.Context(), "failed to write response", "error", err)
		}
	})
	return mux
}

// Serve starts the HTTP server and waits for context cancellation to shut down gracefully.
func Serve(ctx context.Context, server *http.Server) error {
	addr := server.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	return ServeListener(ctx, server, ln)
}

// ServeListener starts serving on the provided listener and shuts down gracefully on context cancellation.
func ServeListener(ctx context.Context, server *http.Server, ln net.Listener) error {
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to gracefully shut down server: %w", err)
		}
		return nil
	}
}
