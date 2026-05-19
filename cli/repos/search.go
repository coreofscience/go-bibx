package repos

import (
	"context"
	"fmt"

	"github.com/coreofscience/go-bibx/internal/formats"
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
	var vectors *vector.DumbVectors[string]
	if err := formats.LoadGzipJSON(r.Path, &vectors); err != nil {
		return nil, fmt.Errorf("failed to load vectors: %w", err)
	}
	return vectors, nil
}

func (r *FileSearchRepo) Store(ctx context.Context, v *vector.DumbVectors[string]) error {
	if err := formats.StoreGzipJSON(r.Path, v); err != nil {
		return fmt.Errorf("failed to store vectors: %w", err)
	}
	return nil
}
