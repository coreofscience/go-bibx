package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		stop()
	}()
	if err := cmd.Run(ctx, os.Args); err != nil {
		if errors.Is(err, context.Canceled) {
			os.Exit(130)
		}
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
