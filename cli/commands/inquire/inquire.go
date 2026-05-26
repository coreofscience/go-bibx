package inquire

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/internal/utils"

	"github.com/urfave/cli/v3"
)

func New() *cli.Command {
	return &cli.Command{
		Name:      "inquire",
		Usage:     "Inquire for a research topic",
		ArgsUsage: "<query>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "force overwrite of existing results",
				Value:   false,
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"l"},
				Usage:   "number of initial results to fetch",
				Value:   200,
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:      "query",
				UsageText: "search query for the collection",
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			analysisPath, ok := utils.GetAnalysisPath(ctx)
			if !ok {
				return fmt.Errorf("analysis path not set")
			}
			searchPath, ok := utils.GetSearchPath(ctx)
			if !ok {
				return fmt.Errorf("search path not set")
			}
			force := c.Bool("force")
			if _, err := os.Stat(analysisPath); err == nil && !force {
				return fmt.Errorf("file already exists: %s", analysisPath)
			}
			if _, err := os.Stat(searchPath); err == nil && !force {
				return fmt.Errorf("file already exists: %s", searchPath)
			}
			analysisServiceConfig := &services.OpenAlexAnalysisServiceConfig{
				AnalysisPath: analysisPath,
			}
			analysisService := services.NewOpenAlexAnalysisServiceFromConfig(analysisServiceConfig)
			searchServiceConfig := &services.SemanticSearchServiceConfig{
				AnalysisPath: analysisPath,
				SearchPath:   searchPath,
			}
			searchService, err := services.NewSemanticSearchServiceFromConfig(searchServiceConfig)
			if err != nil {
				return fmt.Errorf("failed to create search service: %w", err)
			}

			query := c.StringArg("query")
			if query == "" {
				return errors.New("query argument is required")
			}
			limit := c.Int("limit")
			if limit <= 0 {
				return errors.New("limit must be greater than 0")
			}

			slog.InfoContext(ctx, "starting inquiry", "query", query, "limit", limit)
			if err := analysisService.Store(ctx, query, limit); err != nil {
				return fmt.Errorf("failed to store analysis: %w", err)
			}

			slog.InfoContext(ctx, "building semantic search index")
			if err := searchService.Store(ctx); err != nil {
				return fmt.Errorf("failed to store search results: %w", err)
			}

			return nil
		},
	}
}
