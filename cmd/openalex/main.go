package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/coreofscience/go-bibx/sources"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	collection, err := sources.NewOpenAlexSource("bit patterned media").Build(context.Background())
	if err != nil {
		slog.Error("failed to build collection", "error", err)
		return
	}
	collectionJSON, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		slog.Error("failed to marshal works to JSON", "error", err)
	}
	fmt.Println(string(collectionJSON))
}
