package sources

import (
	"context"

	"github.com/coreofscience/go-bibx/models"
)

type Source interface {
	// Build creates a new collection instance form the supplied files.
	Build(ctx context.Context) (*models.Collection, error)
}
