package repos

import (
	"compress/gzip"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/coder/hnsw"
)

type SearchRepo interface {
	Load(ctx context.Context) (*hnsw.Graph[string], error)
	Store(ctx context.Context, graph *hnsw.Graph[string]) error
}

type FileSearchRepo struct {
	Path string
}

func NewFileSearchRepo(path string) *FileSearchRepo {
	return &FileSearchRepo{Path: path}
}

func (r *FileSearchRepo) Load(ctx context.Context) (*hnsw.Graph[string], error) {
	graph := hnsw.NewGraph[string]()
	file, err := os.Open(r.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			slog.Error("failed to close file", "error", err)
		}
	}()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		err := gz.Close()
		if err != nil {
			slog.Error("failed to close gzip reader", "error", err)
		}
	}()
	err = graph.Import(gz)
	if err != nil {
		return nil, fmt.Errorf("failed to import graph: %w", err)
	}
	return graph, nil
}

func (r *FileSearchRepo) Store(ctx context.Context, graph *hnsw.Graph[string]) error {
	dir := filepath.Dir(r.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	file, err := os.Create(r.Path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			slog.Error("failed to close file", "error", err)
		}
	}()
	gz := gzip.NewWriter(file)
	defer func() {
		err := gz.Close()
		if err != nil {
			slog.Error("failed to close gzip writer", "error", err)
		}
	}()
	err = graph.Export(gz)
	if err != nil {
		return fmt.Errorf("failed to export graph: %w", err)
	}
	return nil
}
