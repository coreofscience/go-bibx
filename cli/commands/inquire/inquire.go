package inquire

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/internal/utils"

	"github.com/urfave/cli/v3"
)

// New returns the inquire subcommand which fetches works from OpenAlex and performs analysis.
func New() *cli.Command {
	return &cli.Command{
		Name:      "inquire",
		Usage:     "Inquire for a research topic",
		ArgsUsage: "<query>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "force re-fetching and recalculation of existing query results",
				Value:   false,
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"l"},
				Usage:   "number of initial results to fetch",
				Value:   200,
				Validator: func(value int) error {
					if value < 1 {
						return fmt.Errorf("limit must be greater than 0")
					}
					if value > 1000 {
						return fmt.Errorf("limit must be less than or equal to 1000")
					}
					return nil
				},
			},
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "optional name alias for the collection",
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
			dbPath, ok := utils.GetDBPath(ctx)
			if !ok {
				return errors.New("database path not set")
			}

			force := c.Bool("force")
			analysisServiceConfig := &services.OpenAlexAnalysisServiceConfig{
				DBPath: dbPath,
			}
			analysisService, err := services.NewOpenAlexAnalysisServiceFromConfig(analysisServiceConfig)
			if err != nil {
				return fmt.Errorf("failed to create analysis service: %w", err)
			}
			defer func() {
				_ = analysisService.StoreBackend().Close()
			}()

			query := c.StringArg("query")
			if query == "" {
				return errors.New("query argument is required")
			}
			limit := c.Int("limit")

			name := c.String("name")
			slog.InfoContext(ctx, "starting inquiry", "query", query, "limit", limit)
			res, err := analysisService.StoreWithOptions(ctx, query, limit, services.StoreOptions{
				Force: force,
				Name:  name,
			})
			if err != nil {
				return fmt.Errorf("failed to store analysis: %w", err)
			}

			if !res.IsNew {
				fmt.Printf("Collection already exists: %s (%d articles). Use --force to re-run inquiry.\n", res.Collection.ID, res.Collection.ArticleCount)
				return nil
			}

			slog.InfoContext(ctx, "inquiry completed",
				"collection_id", res.Collection.ID,
				"articles", res.Collection.ArticleCount,
			)
			fmt.Printf("Collection ready: %s (articles: %d)\n", res.Collection.ID, res.Collection.ArticleCount)
			return nil
		},
	}
}
