package search

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/coreofscience/go-bibx/cli/renderers"
	"github.com/coreofscience/go-bibx/cli/servers/viewer"
	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/urfave/cli/v3"
)

var (
	formats = collections.NewSet(renderers.AnalysisFormats()...)
)

func New() *cli.Command {
	supportedFormats := strings.Join(formats.Items(), ", ")
	return &cli.Command{
		Name:      "search",
		Usage:     "Perform a semantic search on the bibx collection",
		ArgsUsage: "<query>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"l"},
				Usage:   "number of nodes to start the graph algorithm with",
				Value:   10,
				Validator: func(value int) error {
					if value < 0 {
						return fmt.Errorf("limit must be greater than or equal to 0")
					}
					return nil
				},
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   fmt.Sprintf("format to output results in (%s)", supportedFormats),
				Value:   "simple",
				Validator: func(value string) error {
					if value == "" {
						return fmt.Errorf("format is required")
					}
					if !formats.Contains(value) {
						return fmt.Errorf("invalid format, must be one of: %s", supportedFormats)
					}
					return nil
				},
			},
			&cli.BoolFlag{
				Name:    "view",
				Aliases: []string{"v"},
				Usage:   "visualize the search results",
				Value:   false,
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "port to serve the visualization on",
				Value:   8080,
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
			analysisPath, ok := utils.GetAnalysisPath(ctx)
			if !ok {
				return fmt.Errorf("analysis path not set")
			}
			analysisService, err := services.NewOpenAlexAnalysisServiceFromConfig(
				&services.OpenAlexAnalysisServiceConfig{
					AnalysisPath: analysisPath,
				},
			)
			if err != nil {
				return fmt.Errorf("failed to create analysis service: %w", err)
			}
			renderer, err := renderers.NewAnalysisRendererWithFormat(c.String("format"))
			if err != nil {
				return fmt.Errorf("failed to create renderer: %w", err)
			}
			query := c.StringArg("query")
			if query == "" {
				return errors.New("query is required")
			}
			analysis, err := analysisService.Search(ctx, query, c.Int("limit"))
			if err != nil {
				return fmt.Errorf("failed to perform search: %w", err)
			}
			if err := renderer.RenderAnalysis(os.Stdout, analysis); err != nil {
				return fmt.Errorf("failed to render results: %w", err)
			}
			if c.Bool("view") {
				mux := viewer.New(analysis)
				port := c.Int("port")
				slog.InfoContext(ctx, "starting visualization server", "port", port)
				if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
					return fmt.Errorf("failed to start server: %w", err)
				}
			}
			return nil
		},
	}
}
