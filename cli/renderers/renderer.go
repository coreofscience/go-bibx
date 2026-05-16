package renderers

import "github.com/coreofscience/go-bibx/models"

type Renderer interface {
	// Render renders the graph in a specific format and returns the result as a string.
	Render(result []*models.Result) error
}
