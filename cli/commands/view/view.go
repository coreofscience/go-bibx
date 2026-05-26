package view

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/cli/servers/viewer"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/urfave/cli/v3"
)

func New() *cli.Command {
	return &cli.Command{
		Name:  "view",
		Usage: "Visualize the analysis results in a web browser",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "port to listen on",
				Value:   8080,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			port := c.Int("port")
			analysisPath, ok := utils.GetAnalysisPath(ctx)
			if !ok {
				return fmt.Errorf("analysis path not set")
			}
			analysisRepo := repos.NewFileAnalysisRepo(analysisPath)
			analysis, err := analysisRepo.Load(ctx)
			if err != nil {
				return fmt.Errorf("failed to load analysis from file: %w", err)
			}
			mux := viewer.New(analysis)
			slog.InfoContext(ctx, "visualization started", "port", port)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
				return fmt.Errorf("failed to start visualization server: %w", err)
			}
			return nil
		},
	}
}
