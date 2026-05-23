package services

import (
	"context"
	"fmt"

	"github.com/coreofscience/go-bibx/cli/repos"
)

type ScaffoldService interface {
	Scaffold(ctx context.Context) error
}

type DefaultScaffoldService struct {
	scaffoldRepo repos.ScaffoldRepo
}

func NewDefaultScaffoldService(scaffoldRepo repos.ScaffoldRepo) *DefaultScaffoldService {
	return &DefaultScaffoldService{
		scaffoldRepo: scaffoldRepo,
	}
}

type DefaultScaffoldServiceConfig struct {
	ScaffoldPath string
}

func NewDefaultScaffoldServiceWithConfig(
	config *DefaultScaffoldServiceConfig,
) *DefaultScaffoldService {
	scaffoldRepo := repos.NewTemplateScaffoldRepo(config.ScaffoldPath)
	return NewDefaultScaffoldService(scaffoldRepo)
}

func (s *DefaultScaffoldService) Scaffold(ctx context.Context) error {
	if err := s.scaffoldRepo.Scaffold(); err != nil {
		return fmt.Errorf("failed to scaffold: %w", err)
	}
	return nil
}
