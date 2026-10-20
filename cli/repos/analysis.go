package repos

import (
	"context"
	"fmt"

	"github.com/coreofscience/go-bibx/internal/formats"
	"github.com/coreofscience/go-bibx/models"
)

// AnalysisRepo provides methods to load and persist an [models.Analysis].
type AnalysisRepo interface {
	Load(ctx context.Context) (*models.Analysis, error)
	Store(ctx context.Context, a *models.Analysis) error
}

// FileAnalysisRepo implements [AnalysisRepo] using a gzip-compressed JSON file.
type FileAnalysisRepo struct {
	FileName string
}

// NewFileAnalysisRepo creates a new [FileAnalysisRepo] for the given file path.
func NewFileAnalysisRepo(fileName string) *FileAnalysisRepo {
	return &FileAnalysisRepo{
		FileName: fileName,
	}
}

// Load loads and decodes the [models.Analysis] from the configured file.
func (r *FileAnalysisRepo) Load(ctx context.Context) (*models.Analysis, error) {
	var analysis models.Analysis
	if err := formats.LoadGzipJSON(r.FileName, &analysis); err != nil {
		return nil, fmt.Errorf("error loading analysis: %w", err)
	}
	return &analysis, nil
}

// Store saves the provided [models.Analysis] to the configured file.
func (r *FileAnalysisRepo) Store(ctx context.Context, a *models.Analysis) error {
	if err := formats.StoreGzipJSON(r.FileName, a); err != nil {
		return fmt.Errorf("error storing analysis: %w", err)
	}
	return nil
}
