package viewer

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/coreofscience/go-bibx/models"
)

//go:embed templates/index.html
var indexTemplate []byte

func New(analysis *models.Analysis) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(indexTemplate); err != nil {
			slog.ErrorContext(r.Context(), "failed to write index template", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
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
