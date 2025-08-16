package sources

import (
	"io"

	"github.com/coreofscience/go-bibx/models"
)

type Source interface {
	// Build creates a new collection instance form the supplied files.
	Build(files []io.Reader) (*models.Collection, error)
}
