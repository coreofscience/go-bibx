package services

import (
	"context"
	"fmt"

	"github.com/coreofscience/go-bibx/cli/repos"
)

type RawFileService interface {
	Store(ctx context.Context) error
}

type MarkdownFileService struct {
	analysisRepo repos.AnalysisRepo
	markdownRepo repos.MarkdownRepo
}

func NewMarkdownFileService(
	analysisRepo repos.AnalysisRepo,
	markdownRepo repos.MarkdownRepo,
) *MarkdownFileService {
	return &MarkdownFileService{
		analysisRepo: analysisRepo,
		markdownRepo: markdownRepo,
	}
}

type MarkdownFileServiceConfig struct {
	AnalysisPath string
	MarkdownDir  string
}

func NewMarkdownFileServiceWithConfig(config *MarkdownFileServiceConfig) (*MarkdownFileService, error) {
	analysisRepo := repos.NewFileAnalysisRepo(config.AnalysisPath)
	markdownRepo := repos.NewFolderMarkdownRepo(config.MarkdownDir)
	return NewMarkdownFileService(analysisRepo, markdownRepo), nil
}

func (s *MarkdownFileService) Store(ctx context.Context) error {
	analysis, err := s.analysisRepo.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load analysis: %w", err)
	}
	if err := s.markdownRepo.Store(ctx, analysis); err != nil {
		return fmt.Errorf("failed to store markdown files: %w", err)
	}
	return nil
}
