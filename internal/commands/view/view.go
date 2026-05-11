package view

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/urfave/cli/v3"
)

//go:embed templates/index.html
var indexTemplate []byte

func New() *cli.Command {
	return &cli.Command{
		Name:  "view",
		Usage: "visualize a collection graph",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "file",
				Usage: "path to the collection graph file",
				Value: ".bibx/collection.json.gz",
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "port to listen on",
				Value: 8080,
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "enable verbose logging",
				Value: false,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			utils.SetDefaultLogger(c.Bool("verbose"))
			port := c.Int("port")
			file := c.String("file")
			http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write(indexTemplate); err != nil {
					slog.Error("failed to write index template", "error", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			})
			http.HandleFunc("GET /collection.json.gz", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
				http.ServeFile(w, r, file)
			})
			http.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				w.Header().Set("X-Accel-Buffering", "no")
				w.Header().Set("Cache-Control", "no-cache")

				// Just keep the connection open.
				// When the server restarts, this connection breaks.
				<-r.Context().Done()
			})
			slog.Info("visualization started", "port", port)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
				slog.Error("failed to start server", "error", err)
				return err
			}
			return nil
		},
	}
}
