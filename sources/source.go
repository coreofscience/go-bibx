package sources

import (
	"github.com/coreofscience/go-bibx/models"
)

type Source interface {
	// Build creates a new collection instance form the supplied files.
	Build() (*models.Collection, error)
}
