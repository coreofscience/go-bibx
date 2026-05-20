package renderers

import (
	"fmt"
	"io"

	"github.com/coreofscience/go-bibx/models"
)

type Renderer interface {
	// RenderResults renders the graph in a specific format.
	RenderResults(w io.Writer, result []*models.Result) error

	// RenderAnalysis renders an analysis in a specific format.
	RenderAnalysis(w io.Writer, analysis *models.Analysis) error
}

func NewRendererWithFormat(format string) (Renderer, error) {
	switch format {
	case "json":
		return NewJSONRenderer(), nil
	case "simple", "markdown", "reference":
		return NewMarkdownRenderer(format)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
