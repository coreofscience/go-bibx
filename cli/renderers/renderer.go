package renderers

import (
	"github.com/coreofscience/go-bibx/models"
)

type Renderer interface {
	// RenderResults renders the graph in a specific format.
	RenderResults(result []*models.Result) error
}
