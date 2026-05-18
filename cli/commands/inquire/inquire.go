package inquire

import (
	"context"
	"log/slog"
	"os"
	"path"

	"github.com/coreofscience/go-bibx/cli/clients/embeddings"
	"github.com/coreofscience/go-bibx/cli/clients/openalex"
	"github.com/coreofscience/go-bibx/cli/renderers"
	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/cli/services"

	"github.com/urfave/cli/v3"
)

func New() *cli.Command {
	return &cli.Command{
		Name:      "inquire",
		Usage:     "inquire for a research topic",
		ArgsUsage: "<query>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "root",
				Usage: "root directory store the results",
				Value: ".",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "force overwrite of existing results",
				Value: false,
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "number of initial results to fetch",
				Value: 200,
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
			analysisPath := path.Join(c.String("root"), ".bibx", "collection.json.gz")
			searchPath := path.Join(c.String("root"), ".bibx", "search.json.gz")
			force := c.Bool("force")
			if _, err := os.Stat(analysisPath); err == nil && !force {
				slog.Error("file already exists", "path", analysisPath)
				os.Exit(1)
			}
			if _, err := os.Stat(searchPath); err == nil && !force {
				slog.Error("file already exists", "path", searchPath)
				os.Exit(1)
			}
			query := c.StringArg("query")
			limit := c.Int("limit")
			openalexClient := openalex.NewRestyClient()
			analysisRepo := repos.NewFileAnalysisRepo(analysisPath)
			analysisService := services.NewOpenAlexAnalysisService(
				openalexClient,
				analysisRepo,
			)
			if err := analysisService.Store(context.Background(), query, limit); err != nil {
				slog.Error("failed to store analysis", "error", err)
				os.Exit(1)
			}
			searchRepo := repos.NewFileSearchRepo(searchPath)
			embeddingsClient, err := embeddings.NewOllamaClient()
			if err != nil {
				slog.Error("failed to create embeddings client", "error", err)
				os.Exit(1)
			}
			markdownRenderer, err := renderers.NewMarkdownRenderer("simple")
			if err != nil {
				slog.Error("failed to create markdown renderer", "error", err)
				os.Exit(1)
			}
			searchService := services.NewSemanticSearchService(
				analysisRepo,
				searchRepo,
				embeddingsClient,
				markdownRenderer,
			)
			if err := searchService.Store(ctx); err != nil {
				slog.Error("failed to store search graph", "error", err)
				os.Exit(1)
			}
			return nil
		},
	}
}
