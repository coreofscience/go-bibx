package setup

import (
	"context"
	"log/slog"
	"os"

	"github.com/coreofscience/go-bibx/cli/clients/embeddings"
	"github.com/urfave/cli/v3"
)

func New() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Setup the application by downloading necessary files and configurations.",
		Action: func(ctx context.Context, c *cli.Command) error {
			client, err := embeddings.NewOllamaClient()
			if err != nil {
				slog.Error("failed to create embeddings client", "error", err)
				os.Exit(1)
			}
			if err := client.Sync(ctx); err != nil {
				slog.Error("failed to sync embeddings model", "error", err)
				os.Exit(1)
			}
			slog.Info("setup completed successfully")
			return nil
		},
	}
}
