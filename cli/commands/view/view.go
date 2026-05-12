package view

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/coreofscience/go-bibx/cli/servers/viewer"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/urfave/cli/v3"
)

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
			mux := viewer.New(file)
			slog.Info("visualization started", "port", port)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
				slog.Error("failed to start server", "error", err)
				return err
			}
			return nil
		},
	}
}
