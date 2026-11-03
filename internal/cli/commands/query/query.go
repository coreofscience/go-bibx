package query

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/coreofscience/go-bibx/clients/openalex"
	"github.com/coreofscience/go-bibx/internal/cli/renderers"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
	"github.com/urfave/cli/v3"
)

var (
	formats    = collections.NewSet(renderers.ResultFormats()...)
	categories = collections.NewSet("root", "trunk", "leaf")
)

// New returns the query subcommand which retrieves top works by category score.
func New() *cli.Command {
	supportedFormats := strings.Join(formats.Items(), ", ")
	supportedCategories := strings.Join(categories.Items(), ", ")
	return &cli.Command{
		Name:      "query",
		Usage:     "List the top root, trunk or leaf works in a collection",
		ArgsUsage: "<root|trunk|leaf>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"l"},
				Usage:   "number of results to return",
				Value:   5,
				Validator: func(value int) error {
					if value < 1 {
						return fmt.Errorf("limit must be greater than 0")
					}
					return nil
				},
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   fmt.Sprintf("format to output results in (%s)", supportedFormats),
				Value:   "json",
				Validator: func(value string) error {
					if value == "" {
						return errors.New("format is required")
					}
					if !formats.Contains(value) {
						return fmt.Errorf("invalid format, must be one of: %s", supportedFormats)
					}
					return nil
				},
			},
			&cli.StringFlag{
				Name:    "collection",
				Aliases: []string{"c"},
				Usage:   "collection ID or name alias to query (defaults to latest)",
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:      "category",
				UsageText: fmt.Sprintf("category to list (%s)", supportedCategories),
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			category := c.StringArg("category")
			if category == "" {
				return fmt.Errorf("category argument is required (%s)", supportedCategories)
			}
			if !categories.Contains(category) {
				return fmt.Errorf("invalid category %q, must be one of: %s", category, supportedCategories)
			}

			dbPath, ok := utils.GetDBPath(ctx)
			if !ok {
				return errors.New("database path not set")
			}
			db, err := repository.OpenAndMigrateSQLite(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				_ = db.Close()
			}()

			colSvc := services.NewCollectionService(db)
			anaSvc := services.NewAnalysisService(db, openalex.NewRestyClient())

			renderer, err := renderers.NewResultRendererWithFormat(c.String("format"))
			if err != nil {
				return fmt.Errorf("failed to create renderer: %w", err)
			}

			colTarget := c.String("collection")
			col, err := colSvc.Resolve(ctx, colTarget)
			if err != nil {
				return fmt.Errorf("failed to resolve collection %q: %w", colTarget, err)
			}

			results, err := anaSvc.QueryCategory(ctx, col.ID, category, c.Int("limit"))
			if err != nil {
				return fmt.Errorf("failed to query: %w", err)
			}
			if err := renderer.RenderResults(os.Stdout, results); err != nil {
				return fmt.Errorf("failed to render results: %w", err)
			}
			return nil
		},
	}
}
