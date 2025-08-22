package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/coreofscience/go-bibx/clients"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	openalexClient := clients.NewOpenAlexClient(&clients.NewOpenAlexClientParams{})
	works, err := openalexClient.ListRecentArticles(
		context.Background(),
		&clients.ListRecentArticlesParams{
			Query: "bit patterned media",
			Limit: nil,
		},
	)
	if err != nil {
		slog.Error("failed to list recent articles", "error", err)
		return
	}
	worksJSON, err := json.MarshalIndent(works, "", "  ")
	if err != nil {
		slog.Error("failed to marshal works to JSON", "error", err)
	}
	fmt.Println(string(worksJSON))
}
