package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/coreofscience/go-bibx/clients"
	"github.com/coreofscience/go-bibx/utils"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	openalexClient := clients.NewOpenAlexClient()
	recentWorks, err := openalexClient.ListRecentArticles(
		context.Background(),
		&clients.ListRecentArticlesParams{
			Query: "bit patterned media",
			Limit: utils.NewRef(100),
		},
	)
	if err != nil {
		slog.Error("failed to list recent articles", "error", err)
		return
	}
	referencedWorks := make([]string, 0, len(recentWorks))
	for _, work := range recentWorks {
		if work.ReferencedWorks != nil {
			referencedWorks = append(referencedWorks, work.ReferencedWorks...)
		}
	}
	worksByID, err := openalexClient.ListArticlesByIDs(
		context.Background(),
		&clients.ListArticlesByIDsParams{
			IDs: referencedWorks,
		},
	)
	if err != nil {
		slog.Error("failed to list articles by IDs", "error", err)
		return
	}
	works := append(recentWorks, worksByID...)
	worksJSON, err := json.MarshalIndent(works, "", "  ")
	if err != nil {
		slog.Error("failed to marshal works to JSON", "error", err)
	}
	fmt.Println(string(worksJSON))
}
