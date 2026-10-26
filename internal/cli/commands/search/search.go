package search

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/coreofscience/go-bibx/clients/openalex"
	"github.com/coreofscience/go-bibx/internal/cli/renderers"
	"github.com/coreofscience/go-bibx/internal/cli/servers/viewer"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
	"github.com/urfave/cli/v3"
)

var (
	formats = collections.NewSet(renderers.AnalysisFormats()...)
)

// New returns the search subcommand which performs keyword graph search on the collection.
func New() *cli.Command {
	supportedFormats := strings.Join(formats.Items(), ", ")
	return &cli.Command{
		Name:      "search",
		Usage:     "Perform a keyword search on the bibx collection",
		ArgsUsage: "<query>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"l"},
				Usage:   "number of nodes to start the graph algorithm with",
				Value:   10,
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
				Value:   "simple",
				Validator: func(value string) error {
					if value == "" {
						return fmt.Errorf("format is required")
					}
					if !formats.Contains(value) {
						return fmt.Errorf("invalid format, must be one of: %s", supportedFormats)
					}
					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "view",
				Aliases: []string{"v"},
				Usage:   "visualize the search results",
				Value:   false,
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "port to serve the visualization on",
				Value:   8080,
			},
			&cli.StringFlag{
				Name:    "collection",
				Aliases: []string{"col", "c"},
				Usage:   "collection ID or name alias to search (defaults to latest)",
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
			db, err := repository.OpenAndMigrateSQLite(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				_ = db.Close()
			}()

			colSvc := services.NewCollectionService(db)
			analysisService := services.NewAnalysisService(db, openalex.NewRestyClient())

			renderer, err := renderers.NewAnalysisRendererWithFormat(c.String("format"))
			if err != nil {
				return fmt.Errorf("failed to create renderer: %w", err)
			}
			query := c.StringArg("query")
			if query == "" {
				return errors.New("query is required")
			}

			colTarget := c.String("collection")
			col, err := colSvc.Resolve(ctx, colTarget)
			if err != nil {
				return fmt.Errorf("failed to resolve collection %q: %w", colTarget, err)
			}

			analysis, err := analysisService.Search(ctx, col.ID, query, c.Int("limit"))
			if err != nil {
				return fmt.Errorf("failed to perform search: %w", err)
			}
			if c.Bool("view") {
				mux := viewer.New(analysis)
				port := c.Int("port")
				slog.InfoContext(ctx, "starting visualization server", "port", port)
				server := &http.Server{
					Addr:    fmt.Sprintf(":%d", port),
					Handler: mux,
				}
				if err := viewer.Serve(ctx, server); err != nil {
					return fmt.Errorf("failed to start server: %w", err)
				}
			} else {
				if err := renderer.RenderAnalysis(os.Stdout, analysis); err != nil {
					return fmt.Errorf("failed to render results: %w", err)
				}
			}
			return nil
		},
	}
}
