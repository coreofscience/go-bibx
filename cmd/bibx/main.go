package main

import (
	"context"
	"log/slog"
	"os"
	"path"

	"github.com/coreofscience/go-bibx/cli/commands"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := cli.Command{
		Name:  "bibx",
		Usage: "bibx is a CLI tool for managing bibliographic collections",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "enable verbose logging",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "root",
				Usage: "path to the root directory",
				Value: ".",
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			utils.SetDefaultLogger(c.Bool("verbose"))
			analysisPath := path.Join(c.String("root"), ".bibx", "collection.json.gz")
			searchPath := path.Join(c.String("root"), ".bibx", "search.json.gz")
			ctx = context.WithValue(ctx, utils.RootDirKey, c.String("root"))
			ctx = context.WithValue(ctx, utils.AnalysisPathKey, analysisPath)
			ctx = context.WithValue(ctx, utils.SearchPathKey, searchPath)
			return ctx, nil
		},
		Commands: commands.New(),
	}
	utils.SetDefaultLogger(false)
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
