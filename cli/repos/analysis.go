package repos

import (
	"context"
	"fmt"

	"github.com/coreofscience/go-bibx/internal/formats"
	"github.com/coreofscience/go-bibx/models"
)

type AnalysisRepo interface {
	Load(ctx context.Context) (*models.Analysis, error)
	Store(ctx context.Context, a *models.Analysis) error
}

type FileAnalysisRepo struct {
	FileName string
}

func NewFileAnalysisRepo(fileName string) *FileAnalysisRepo {
	return &FileAnalysisRepo{
		FileName: fileName,
	}
}

func (r *FileAnalysisRepo) Load(ctx context.Context) (*models.Analysis, error) {
	var analysis models.Analysis
	if err := formats.LoadGzipJSON(r.FileName, &analysis); err != nil {
		return nil, fmt.Errorf("error loading analysis: %w", err)
	}
	return &analysis, nil
}

func (r *FileAnalysisRepo) Store(ctx context.Context, a *models.Analysis) error {
	if err := formats.StoreGzipJSON(r.FileName, a); err != nil {
		return fmt.Errorf("error storing analysis: %w", err)
	}
	return nil
}
