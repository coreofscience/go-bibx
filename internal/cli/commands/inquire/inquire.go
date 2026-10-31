package inquire

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/coreofscience/go-bibx/clients/openalex"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
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
			db, err := repository.OpenAndMigrateSQLite(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				_ = db.Close()
			}()

			analysisService := services.NewAnalysisService(db, openalex.NewRestyClient())

			query := c.StringArg("query")
			if query == "" {
				return errors.New("query argument is required")
			}
			limit := c.Int("limit")

			name := c.String("name")
			slog.InfoContext(ctx, "starting inquiry", "query", query, "limit", limit)
			res, err := analysisService.Inquire(ctx, query, limit, services.InquireOptions{
				Force: force,
				Name:  name,
			})
			if err != nil {
				return fmt.Errorf("failed to store analysis: %w", err)
			}

			if !res.IsNew {
				fmt.Printf("Collection already exists: %s (%d works). Use --force to re-run inquiry.\n", res.Collection.ID, res.Collection.WorkCount)
				return nil
			}

			slog.InfoContext(ctx, "inquiry completed",
				"collection_id", res.Collection.ID,
				"works", res.Collection.WorkCount,
			)
			fmt.Printf("Collection ready: %s (works: %d)\n", res.Collection.ID, res.Collection.WorkCount)
			return nil
		},
	}
}
