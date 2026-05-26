package main

import (
	"context"
	"log/slog"
	"os"

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
				Name:    "root",
				Aliases: []string{"r"},
				Usage:   "path to the root directory",
				Value:   ".",
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			utils.SetDefaultLogger(c.Bool("verbose"))
			ctx = utils.WithRootDir(ctx, c.String("root"))
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
