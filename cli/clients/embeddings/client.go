package embeddings

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/ollama/ollama/api"
)

const (
	modelID            = "hf.co/ggml-org/embeddinggemma-300M-GGUF"
	maxTextsPerRequest = 100
)

type Client interface {
	Sync(ctx context.Context) error
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedMany(ctx context.Context, texts []string) ([][]float32, error)
}

type OllamaClient struct {
	client *api.Client
}

func NewOllamaClient() (*OllamaClient, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}
	return &OllamaClient{
		client: client,
	}, nil
}

// Sync implements the [Client] interface.
func (c *OllamaClient) Sync(ctx context.Context) error {
	err := c.client.Pull(ctx, &api.PullRequest{
		Model: modelID,
	}, func(progress api.ProgressResponse) error {
		if progress.Total == 0 || progress.Completed == progress.Total {
			slog.Info("pulling model", "model", modelID, "status", progress.Status)
			return nil
		}
		percent := float64(progress.Completed) / float64(progress.Total)
		slog.Debug("pulling model", "model", modelID, "status", progress.Status, "progress", fmt.Sprintf("%.2f%%", percent*100))
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}
	return nil
}

// Embed implements the [Client] interface.
func (c *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := c.client.Embed(ctx, &api.EmbedRequest{
		Model: modelID,
		Input: []string{text},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return resp.Embeddings[0], nil
}

// EmbedMany implements the [Client] interface.
func (c *OllamaClient) EmbedMany(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, 0, len(texts))
	chunks := utils.Chunks(texts, maxTextsPerRequest)
	for i, chunk := range chunks {
		resp, err := c.client.Embed(ctx, &api.EmbedRequest{
			Model: modelID,
			Input: chunk,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get embeddings: %w", err)
		}
		if len(resp.Embeddings) != len(chunk) {
			return nil, fmt.Errorf("number of embeddings returned does not match number of input texts")
		}
		results = append(results, resp.Embeddings...)
		slog.Debug("done embedding chunk", "chunk", i+1, "total", len(chunks))
	}
	return results, nil
}
