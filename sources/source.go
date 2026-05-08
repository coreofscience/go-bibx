package sources

import (
	"context"

	"github.com/coreofscience/go-bibx/collection"
)

type Source interface {
	// Build creates a new collection instance from the supplied files.
	Build(ctx context.Context) (*collection.Collection, error)
}
