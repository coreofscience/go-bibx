package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/coreofscience/go-bibx/internal/commands"
	"github.com/urfave/cli/v3"
)

func main() {
	cli := cli.Command{
		Name:     "bibx",
		Usage:    "bibx is a CLI tool for managing bibliographic collections",
		Commands: commands.New(),
	}
	if err := cli.Run(context.Background(), os.Args); err != nil {
		slog.Error("error running main program", "error", err)
		os.Exit(1)
	}
}
