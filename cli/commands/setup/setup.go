package setup

import (
	"context"
	"fmt"
	"log/slog"

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
				return fmt.Errorf("failed to create Ollama client: %w", err)
			}
			if err := client.Sync(ctx); err != nil {
				return fmt.Errorf("failed to sync Ollama client: %w", err)
			}
			slog.Info("setup completed successfully")
			return nil
		},
	}
}
