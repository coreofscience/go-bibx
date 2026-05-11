package query

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/coreofscience/go-bibx/analysis"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/urfave/cli/v3"
)

func New() *cli.Command {
	categories := collections.NewSet("root", "trunk", "leaf")
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
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "enable verbose output",
				Value: false,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			utils.SetDefaultLogger(c.Bool("verbose"))
			a, err := analysis.Load(c.String("file"))
			if err != nil {
				slog.Error("failed to load analysis", "error", err)
				os.Exit(1)
			}
			results, err := a.Query(c.String("category"), c.Int("top"))
			if err != nil {
				slog.Error("failed to query analysis", "error", err)
				os.Exit(1)
			}
			for _, result := range results {
				fmt.Println(result.Article.ToMarkdown())
			}
			return nil
		},
	}
}
