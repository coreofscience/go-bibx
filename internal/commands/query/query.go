package query

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/coreofscience/go-bibx/analysis"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/render"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/urfave/cli/v3"
)

var (
	formats    = collections.NewSet("markdown", "json")
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
				Usage: "format to output results in (markdown, json, etc.)",
				Value: "markdown",
				Validator: func(value string) error {
					if value == "" {
						return cli.Exit("format is required", 1)
					}
					if !formats.Contains(value) {
						return cli.Exit("invalid format, must be one of: markdown, json", 1)
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
			switch c.String("format") {
			case "json":
				bytes, err := json.Marshal(results)
				if err != nil {
					slog.Error("failed to marshal results", "error", err)
					os.Exit(1)
				}
				fmt.Println(string(bytes))
				return nil
			case "markdown":
				renderer, err := render.NewMarkdownRenderer()
				for _, result := range results {
					if err != nil {
						slog.Error("failed to create markdown encoder", "error", err)
						os.Exit(1)
					}
					if !result.Article.Rich {
						continue
					}
					fmt.Println(renderer.Render(result.Article))
				}
				return nil
			default:
				return cli.Exit("invalid format, must be one of: markdown, json", 1)
			}
		},
	}
}
