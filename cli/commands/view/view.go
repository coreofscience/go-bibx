package view

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/cli/servers/viewer"
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
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			port := c.Int("port")
			fileName := c.String("file")
			analysisRepo := repos.NewFileAnalysisRepo(fileName)
			analysis, err := analysisRepo.Load(ctx)
			if err != nil {
				return fmt.Errorf("failed to load analysis from file: %w", err)
			}
			mux := viewer.New(analysis)
			slog.Info("visualization started", "port", port)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
				return fmt.Errorf("failed to start visualization server: %w", err)
			}
			return nil
		},
	}
}
