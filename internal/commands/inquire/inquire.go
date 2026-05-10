package inquire

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"github.com/coreofscience/go-bibx/analysis"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/sources"
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
			path := path.Join(c.String("root"), ".bibx", "collection.json.gz")
			if _, err := os.Stat(path); err == nil && !c.Bool("force") {
				slog.Error("file already exists", "path", path)
				os.Exit(1)
			}
			query := c.StringArg("query")
			slog.Debug("inquiring for collection", "query", query)
			collection, err := sources.NewOpenAlexSource(
				query,
				sources.WithLimit(c.Int("limit")),
				sources.WithEnrichReferences(sources.EnrichReferencesNone),
			).Build(context.Background())
			if err != nil {
				slog.Error("failed to build collection", "error", err)
				os.Exit(1)
			}
			collection, err = collection.RemoveCycles()
			if err != nil {
				slog.Error("failed to remove cycles", "error", err)
				os.Exit(1)
			}
			collection, err = collection.RemoveDangling()
			if err != nil {
				slog.Error("failed to remove dangling articles", "error", err)
				os.Exit(1)
			}
			collection, err = collection.Giant()
			if err != nil {
				slog.Error("failed to find giant collection", "error", err)
				os.Exit(1)
			}
			collection, err = collection.Enrich(ctx)
			if err != nil {
				slog.Error("failed to enrich collection", "error", err)
				os.Exit(1)
			}
			result := analysis.New(collection)
			dir := filepath.Dir(path)
			slog.Debug("creating output directory", "dir", dir)
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				slog.Error("failed to create output directory", "error", err)
				os.Exit(1)
			}
			slog.Debug("creating output file", "path", path)
			file, err := os.Create(path)
			if err != nil {
				slog.Error("failed to create output file", "error", err)
				os.Exit(1)
			}
			defer func() {
				err = file.Close()
				if err != nil {
					slog.Error("failed to close output file", "error", err)
					os.Exit(1)
				}
			}()
			gz := gzip.NewWriter(file)
			defer func() {
				err = gz.Close()
				if err != nil {
					slog.Error("failed to close gzip writer", "error", err)
					os.Exit(1)
				}
			}()
			if err := json.NewEncoder(gz).Encode(result); err != nil {
				slog.Error("failed to encode result", "error", err)
				os.Exit(1)
			}
			return nil
		},
	}
}
