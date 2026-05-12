package query

import (
	"context"
	"log/slog"
	"os"

	"github.com/coreofscience/go-bibx/cli/clients/openalex"
	"github.com/coreofscience/go-bibx/cli/renderers"
	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/urfave/cli/v3"
)

var (
	formats    = collections.NewSet("reference", "markdown", "json")
	categories = collections.NewSet("root", "trunk", "leaf")
)

func New() *cli.Command {

	return &cli.Command{
		Name:  "query",
		Usage: "query the bibx collection for relevant articles",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "file",
				Usage: "path to the bibx collection file",
				Value: ".bibx/collection.json.gz",
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
				Usage: "format to output results in (reference, markdown, json, etc.)",
				Value: "json",
				Validator: func(value string) error {
					if value == "" {
						return cli.Exit("format is required", 1)
					}
					if !formats.Contains(value) {
						return cli.Exit("invalid format, must be one of: reference, markdown, json", 1)
					}
					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "enable verbose output",
				Value: false,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			utils.SetDefaultLogger(c.Bool("verbose"))
			openalexClient := openalex.NewRestyClient()
			analysisRepo := repos.NewFileAnalysisRepo(
				c.String("file"),
			)
			markdownRenderer, err := renderers.NewMarkdownRenderer(os.Stdout)
			if err != nil {
				slog.Error("failed to create markdown renderer", "error", err)
				os.Exit(1)
			}
			referenceRenderer, err := renderers.NewMarkdownReferenceRenderer(os.Stdout)
			if err != nil {
				slog.Error("failed to create markdown renderer", "error", err)
				os.Exit(1)
			}
			service := services.NewOpenAlexAnalysisService(
				openalexClient,
				analysisRepo,
				map[string]renderers.Renderer{
					"markdown":  markdownRenderer,
					"reference": referenceRenderer,
					"json":      renderers.NewJSONRenderer(os.Stdout),
				},
			)
			err = service.Query(ctx, c.String("category"), c.Int("top"), c.String("format"))
			if err != nil {
				slog.Error("failed to query analysis", "error", err)
				os.Exit(1)
			}
			return nil
		},
	}
}
