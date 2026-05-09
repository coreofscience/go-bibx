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
			collection, err = collection.RemoveDangling()
			if err != nil {
				slog.Error("failed to remove dangling articles", "error", err)
				return err
			}
			collection, err = collection.Giant()
			if err != nil {
				slog.Error("failed to find giant collection", "error", err)
				return err
			}
			collection, err = collection.Enrich(ctx)
			if err != nil {
				slog.Error("failed to enrich collection", "error", err)
				return err
			}
			citationGraph, err := collection.CitationGraph()
			if err != nil {
				slog.Error("failed to build citation graph", "error", err)
				return err
			}
			slog.Debug("found a collection", "mainArticleCount", collection.Len(), "nodes", citationGraph.Order(), "edges", citationGraph.Size())
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
