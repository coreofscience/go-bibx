package query

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/coreofscience/go-bibx/cli/renderers"
	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/urfave/cli/v3"
)

var (
	formats    = collections.NewSet("reference", "markdown", "json", "simple")
	categories = collections.NewSet("root", "trunk", "leaf")
)

func New() *cli.Command {

	return &cli.Command{
		Name:  "query",
		Usage: "query the bibx collection for relevant articles",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "category",
				Usage:    "category to filter by",
				Required: true,
				Validator: func(value string) error {
					if value == "" {
						return cli.Exit("category is required", 1)
					}
					if !categories.Contains(value) {
						return cli.Exit("invalid category, must be one of: root, trunk, leaf", 1)
					}
					return nil
				},
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "number of results to return",
				Value: 5,
			},
			&cli.StringFlag{
				Name:  "format",
				Usage: "format to output results in (simple, reference, markdown, json, etc.)",
				Value: "json",
				Validator: func(value string) error {
					if value == "" {
						return errors.New("format is required")
					}
					if !formats.Contains(value) {
						return fmt.Errorf("invalid format, must be one of: simple, reference, markdown, json")
					}
					return nil
				},
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			analysisPath, ok := utils.GetAnalysisPath(ctx)
			if !ok {
				return fmt.Errorf("analysis path not set")
			}
			analysisServiceConfig := &services.OpenAlexAnalysisServiceConfig{
				AnalysisPath: analysisPath,
			}
			service := services.NewOpenAlexAnalysisServiceFromConfig(analysisServiceConfig)
			renderer, err := renderers.NewRendererWithFormat(c.String("format"))
			if err != nil {
				return fmt.Errorf("failed to create renderer: %w", err)
			}
			results, err := service.Query(ctx, c.String("category"), c.Int("limit"))
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
