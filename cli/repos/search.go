package repos

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/coreofscience/go-bibx/internal/vector"
)

type SearchRepo interface {
	Load(ctx context.Context) (*vector.DumbVectors[string], error)
	Store(ctx context.Context, v *vector.DumbVectors[string]) error
}

type FileSearchRepo struct {
	Path string
}

func NewFileSearchRepo(path string) *FileSearchRepo {
	return &FileSearchRepo{Path: path}
}

func (r *FileSearchRepo) Load(ctx context.Context) (*vector.DumbVectors[string], error) {
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
	var vectors *vector.DumbVectors[string]
	if err := json.NewDecoder(gz).Decode(&vectors); err != nil {
		return nil, fmt.Errorf("failed to decode vectors: %w", err)
	}
	return vectors, nil
}

func (r *FileSearchRepo) Store(ctx context.Context, v *vector.DumbVectors[string]) error {
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
	if err := json.NewEncoder(gz).Encode(v); err != nil {
		return fmt.Errorf("failed to encode vectors: %w", err)
	}
	return nil
}
