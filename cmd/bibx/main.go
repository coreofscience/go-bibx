package main

import (
	"context"
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
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			utils.SetDefaultLogger(c.Bool("verbose"))
			return ctx, nil
		},
		Commands: commands.New(),
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
}
