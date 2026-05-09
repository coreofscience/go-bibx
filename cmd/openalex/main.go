package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/coreofscience/go-bibx/sources"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := cli.Command{
		Name:  "openalex",
		Usage: "fetch works from OpenAlex",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "query",
				Usage: "search query",
				Value: "bit patterned media",
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "maximum number of results",
				Value: 200,
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "verbose output",
				Value: false,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			logLevel := slog.LevelInfo
			if c.Bool("verbose") {
				logLevel = slog.LevelDebug
			}
			logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
				Level:      logLevel,
				TimeFormat: time.Kitchen,
			}))
			slog.SetDefault(logger)
			collection, err := sources.NewOpenAlexSource(
				c.String("query"),
				sources.WithLimit(c.Int("limit")),
				sources.WithEnrichReferences(sources.EnrichReferencesNone),
			).Build(context.Background())
			if err != nil {
				slog.Error("failed to build collection", "error", err)
				return err
			}
			collection, err = collection.RemoveCycles()
			if err != nil {
				slog.Error("failed to remove cycles", "error", err)
				return err
			}
			collection, err = collection.RemoveIrrelevant()
			if err != nil {
				slog.Error("failed to remove irrelevant articles", "error", err)
				return err
			}
			collections, err := collection.Split()
			if err != nil {
				slog.Error("failed to split collection", "error", err)
				return err
			}

			if len(collections) < 1 {
				slog.Warn("we found no collections", "count", len(collections))
				fmt.Println("{}")
				return nil
			}
			for _, c := range collections {
				graph, err := c.CitationGraph()
				if err != nil {
					slog.Error("failed to build citation graph", "error", err)
					return err
				}
				slog.Debug("found a collection", "mainArticleCount", c.Len(), "nodes", graph.Order(), "edges", graph.Size())
			}
			collection = collections[0]
			collection, err = collection.Enrich(ctx)
			if err != nil {
				slog.Error("failed to enrich collection", "error", err)
				return err
			}
			collectionJSON, err := json.MarshalIndent(collection, "", "  ")
			if err != nil {
				slog.Error("failed to marshal works to JSON", "error", err)
			}
			fmt.Println(string(collectionJSON))
			return nil
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("failed to run command", "error", err)
		os.Exit(1)
	}
}
