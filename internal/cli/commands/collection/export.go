package collection

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func newExportCommand() *cli.Command {
	formats := collections.NewSet(renderers.AnalysisFormats()...)
	supportedFormats := strings.Join(formats.Items(), ", ")

	return &cli.Command{
		Name:      "export",
		Aliases:   []string{"dump"},
		Usage:     "Export works in a collection to BibTeX, JSON, or markdown",
		ArgsUsage: "[<id|name>]",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:      "target",
				UsageText: "collection ID or name alias (defaults to latest)",
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   fmt.Sprintf("format to export collection in (%s)", supportedFormats),
				Value:   "bibtex",
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
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "output file path (defaults to stdout)",
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
			anaSvc := services.NewAnalysisService(db, openalex.NewRestyClient())

			target := c.StringArg("target")
			col, err := colSvc.Resolve(ctx, target)
			if err != nil {
				if errors.Is(err, services.ErrCollectionNotFound) {
					if target == "" {
						return errors.New("no collections found")
					}
					return fmt.Errorf("collection not found: %s", target)
				}
				return fmt.Errorf("failed to resolve collection: %w", err)
			}

			analysis, err := anaSvc.LoadAnalysis(ctx, col.ID)
			if err != nil {
				return fmt.Errorf("failed to load collection analysis: %w", err)
			}

			var writer io.Writer = os.Stdout
			outputPath := c.String("output")
			if outputPath != "" {
				file, err := os.Create(outputPath)
				if err != nil {
					return fmt.Errorf("failed to create output file: %w", err)
				}
				defer func() {
					_ = file.Close()
				}()
				writer = file
			}

			renderer, err := renderers.NewAnalysisRendererWithFormat(c.String("format"))
			if err != nil {
				return fmt.Errorf("failed to create renderer: %w", err)
			}

			if err := renderer.RenderAnalysis(writer, analysis); err != nil {
				return fmt.Errorf("failed to render collection: %w", err)
			}

			return nil
		},
	}
}
