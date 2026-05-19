package inquire

import (
	"context"
	"fmt"
	"os"

	"github.com/coreofscience/go-bibx/cli/clients/embeddings"
	"github.com/coreofscience/go-bibx/cli/clients/openalex"
	"github.com/coreofscience/go-bibx/cli/renderers"
	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/internal/utils"

	"github.com/urfave/cli/v3"
)

func New() *cli.Command {
	return &cli.Command{
		Name:      "inquire",
		Usage:     "inquire for a research topic",
		ArgsUsage: "<query>",
		Flags: []cli.Flag{
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
			analysisPath := ctx.Value(utils.AnalysisPathKey).(string)
			searchPath := ctx.Value(utils.SearchPathKey).(string)
			force := c.Bool("force")
			if _, err := os.Stat(analysisPath); err == nil && !force {
				return fmt.Errorf("file already exists: %s", analysisPath)
			}
			if _, err := os.Stat(searchPath); err == nil && !force {
				return fmt.Errorf("file already exists: %s", searchPath)
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
				return fmt.Errorf("failed to store analysis: %w", err)
			}
			searchRepo := repos.NewFileSearchRepo(searchPath)
			embeddingsClient, err := embeddings.NewOllamaClient()
			if err != nil {
				return fmt.Errorf("failed to create embeddings client: %w", err)
			}
			markdownRenderer, err := renderers.NewMarkdownRenderer("simple")
			if err != nil {
				return fmt.Errorf("failed to create markdown renderer: %w", err)
			}
			searchService := services.NewSemanticSearchService(
				analysisRepo,
				searchRepo,
				embeddingsClient,
				markdownRenderer,
			)
			if err := searchService.Store(ctx); err != nil {
				return fmt.Errorf("failed to store search results: %w", err)
			}
			return nil
		},
	}
}
