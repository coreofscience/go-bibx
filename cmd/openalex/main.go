package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/coreofscience/go-bibx/sources"
	"github.com/urfave/cli/v3"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
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
				Value: 500,
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "verbose output",
				Value: false,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			sqlite_vec.Auto()

			db, err := sql.Open("sqlite3", ":memory:")
			if err != nil {
				slog.Error("failed to open database", "error", err)
				return err
			}
			defer db.Close()

			var vecVersion string
			err = db.QueryRow("select vec_version()").Scan(&vecVersion)
			if err != nil {
				slog.Error("failed to get vec version", "error", err)
				return err
			}
			slog.Info("vec loaded", "version", vecVersion)

			logLevel := slog.LevelInfo
			if c.Bool("verbose") {
				logLevel = slog.LevelDebug
			}
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: logLevel,
			}))
			slog.SetDefault(logger)
			collection, err := sources.NewOpenAlexSource(
				c.String("query"),
				sources.WithLimit(c.Int("limit")),
				sources.WithEnrichReferences(sources.EnrichReferencesCommon),
			).Build(context.Background())
			if err != nil {
				slog.Error("failed to build collection", "error", err)
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
