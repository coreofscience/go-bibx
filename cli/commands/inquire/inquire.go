package inquire

import (
	"context"
	"log/slog"
	"os"
	"path"

	"github.com/coreofscience/go-bibx/cli/clients/openalex"
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
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "verbose output",
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
			utils.SetDefaultLogger(c.Bool("verbose"))
			p := path.Join(c.String("root"), ".bibx", "collection.json.gz")
			if _, err := os.Stat(p); err == nil && !c.Bool("force") {
				slog.Error("file already exists", "path", p)
				os.Exit(1)
			}
			query := c.StringArg("query")
			limit := c.Int("limit")
			openalexClient := openalex.NewRestyClient()
			analysisRepo := repos.NewFileAnalysisRepo(p)
			service := services.NewOpenAlexAnalysisService(
				openalexClient,
				analysisRepo,
				nil,
			)
			if err := service.Store(context.Background(), query, limit); err != nil {
				slog.Error("failed to store analysis", "error", err)
				os.Exit(1)
			}
			return nil
		},
	}
}
