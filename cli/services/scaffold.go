package services

import (
	"context"
	"fmt"

	"github.com/coreofscience/go-bibx/cli/repos"
)

// ScaffoldService defines methods for project initialization and template scaffolding.
type ScaffoldService interface {
	Scaffold(ctx context.Context) error
}

// DefaultScaffoldService provides a standard implementation of [ScaffoldService].
type DefaultScaffoldService struct {
	scaffoldRepo repos.ScaffoldRepo
}

// NewDefaultScaffoldService creates a new [DefaultScaffoldService] using the provided [repos.ScaffoldRepo].
func NewDefaultScaffoldService(scaffoldRepo repos.ScaffoldRepo) *DefaultScaffoldService {
	return &DefaultScaffoldService{
		scaffoldRepo: scaffoldRepo,
	}
}

// DefaultScaffoldServiceConfig holds configuration options for [DefaultScaffoldService].
type DefaultScaffoldServiceConfig struct {
	ScaffoldPath string
}

// NewDefaultScaffoldServiceWithConfig creates a new [DefaultScaffoldService] configured with the target scaffold directory.
func NewDefaultScaffoldServiceWithConfig(
	config *DefaultScaffoldServiceConfig,
) *DefaultScaffoldService {
	scaffoldRepo := repos.NewTemplateScaffoldRepo(config.ScaffoldPath)
	return NewDefaultScaffoldService(scaffoldRepo)
}

// Scaffold invokes the repository to scaffold templates into the target directory.
func (s *DefaultScaffoldService) Scaffold(ctx context.Context) error {
	if err := s.scaffoldRepo.Scaffold(); err != nil {
		return fmt.Errorf("failed to scaffold: %w", err)
	}
	return nil
}
