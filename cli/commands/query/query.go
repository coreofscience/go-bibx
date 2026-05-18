package query

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/coreofscience/go-bibx/cli/clients/openalex"
	"github.com/coreofscience/go-bibx/cli/renderers"
	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/internal/collections"
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
				Name:  "root",
				Usage: "root directory where the collection is stored",
				Value: ".",
			},
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
				Name:  "top",
				Usage: "number of top results to return",
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
			openalexClient := openalex.NewRestyClient()
			analysisPath := path.Join(c.String("root"), ".bibx", "collection.json.gz")
			analysisRepo := repos.NewFileAnalysisRepo(analysisPath)
			service := services.NewOpenAlexAnalysisService(
				openalexClient,
				analysisRepo,
			)
			results, err := service.Query(ctx, c.String("category"), c.Int("top"))
			if err != nil {
				return fmt.Errorf("failed to query: %w", err)
			}
			format := c.String("format")
			var renderer renderers.Renderer
			switch format {
			case "json":
				renderer = renderers.NewJSONRenderer()
			case "markdown", "simple", "reference":
				renderer, err = renderers.NewMarkdownRenderer(format)
			default:
				return fmt.Errorf("unsupported format: %s", format)
			}
			if err != nil {
				return fmt.Errorf("failed to create renderer: %w", err)
			}
			if err := renderer.RenderResults(os.Stdout, results); err != nil {
				return fmt.Errorf("failed to render results: %w", err)
			}
			return nil
		},
	}
}
