package repos

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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
	file, err := os.Open(r.FileName)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer func() {
		err = file.Close()
		if err != nil {
			slog.Error("error closing file", "error", err)
		}
	}()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("error creating gzip reader: %w", err)
	}
	defer func() {
		err = gz.Close()
		if err != nil {
			slog.Error("error closing gzip writer", "error", err)
		}
	}()
	var analysis models.Analysis
	if err := json.NewDecoder(gz).Decode(&analysis); err != nil {
		return nil, fmt.Errorf("error decoding file: %w", err)
	}
	return &analysis, nil
}

func (r *FileAnalysisRepo) Store(ctx context.Context, a *models.Analysis) error {
	dir := filepath.Dir(r.FileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}
	file, err := os.Create(r.FileName)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer func() {
		err = file.Close()
		if err != nil {
			slog.Error("error closing file", "error", err)
		}
	}()
	gz := gzip.NewWriter(file)
	defer func() {
		err = gz.Close()
		if err != nil {
			slog.Error("error closing gzip writer", "error", err)
		}
	}()
	if err := json.NewEncoder(gz).Encode(a); err != nil {
		return fmt.Errorf("error encoding analysis: %w", err)
	}
	return nil
}
