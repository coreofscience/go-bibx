package search

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/coreofscience/go-bibx/cli/clients/embeddings"
	"github.com/coreofscience/go-bibx/cli/renderers"
	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/cli/servers/viewer"
	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/texter"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/urfave/cli/v3"
)

var (
	formats = collections.NewSet("reference", "markdown", "json", "simple")
)

func New() *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "semantic search for a research topic",
		ArgsUsage: "<query>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "limit",
				Usage: "number of top results to return",
				Value: 5,
			},
			&cli.StringFlag{
				Name:  "format",
				Usage: "format to output results in (simple, reference, markdown, json, etc.)",
				Value: "simple",
				Validator: func(value string) error {
					if value == "" {
						return cli.Exit("format is required", 1)
					}
					if !formats.Contains(value) {
						return cli.Exit("invalid format, must be one of: simple, reference, markdown, json", 1)
					}
					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "view",
				Usage: "visualize the search results",
				Value: false,
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "port to serve the visualization on",
				Value: 8080,
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
			query := c.StringArg("query")
			analysisPath, ok := utils.GetAnalysisPath(ctx)
			if !ok {
				return fmt.Errorf("analysis path not set")
			}
			searchPath, ok := utils.GetSearchPath(ctx)
			if !ok {
				return fmt.Errorf("search path not set")
			}
			embeddingsClient, err := embeddings.NewOllamaClient()
			if err != nil {
				return fmt.Errorf("failed to create embeddings client: %w", err)
			}
			analysisRepo := repos.NewFileAnalysisRepo(analysisPath)
			searchRepo := repos.NewFileSearchRepo(searchPath)
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
			ttr := texter.NewDefaultArticleTexter()
			searchService := services.NewSemanticSearchService(
				analysisRepo,
				searchRepo,
				embeddingsClient,
				ttr,
			)
			limit := int(c.Int("limit"))
			analysis, err := searchService.Search(ctx, query, limit)
			if err != nil {
				return fmt.Errorf("failed to perform search: %w", err)
			}
			if c.Bool("view") {
				mux := viewer.New(analysis)
				port := c.Int("port")
				slog.Info("starting visualization server", "port", port)
				if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
					return fmt.Errorf("failed to start server: %w", err)
				}
			}
			if err := renderer.RenderAnalysis(os.Stdout, analysis); err != nil {
				return fmt.Errorf("failed to render results: %w", err)
			}
			return nil
		},
	}
}
