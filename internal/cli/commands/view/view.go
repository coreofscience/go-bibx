package view

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/coreofscience/go-bibx/clients/openalex"
	"github.com/coreofscience/go-bibx/internal/cli/servers/viewer"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
	"github.com/urfave/cli/v3"
)

// New returns the view subcommand which launches the browser-based visualization server.
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
			&cli.StringFlag{
				Name:    "collection",
				Aliases: []string{"col", "c"},
				Usage:   "collection ID or name alias to view (defaults to latest)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			port := c.Int("port")

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

			colTarget := c.String("collection")
			col, err := colSvc.Resolve(ctx, colTarget)
			if err != nil {
				if errors.Is(err, services.ErrCollectionNotFound) {
					if colTarget != "" {
						return fmt.Errorf("collection not found: %s", colTarget)
					}
					return errors.New("no collections found; run 'bibx inquire' first")
				}
				return fmt.Errorf("failed to resolve collection: %w", err)
			}

			analysis, err := anaSvc.LoadAnalysis(ctx, col.ID)
			if err != nil {
				return fmt.Errorf("failed to load analysis from database: %w", err)
			}

			mux := viewer.New(analysis)
			slog.InfoContext(ctx, "visualization started", "port", port, "collection_id", col.ID)
			server := &http.Server{
				Addr:    fmt.Sprintf(":%d", port),
				Handler: mux,
			}
			if err := viewer.Serve(ctx, server); err != nil {
				return fmt.Errorf("failed to start visualization server: %w", err)
			}
			return nil
		},
	}
}
