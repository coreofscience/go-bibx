package renderers

import (
	"fmt"
	"io"

	"github.com/coreofscience/go-bibx/models"
)

type ResultRenderer interface {
	// RenderResults renders the graph in a specific format.
	RenderResults(w io.Writer, result []*models.Result) error
}

type AnalysisRenderer interface {
	// RenderAnalysis renders an analysis in a specific format.
	RenderAnalysis(w io.Writer, analysis *models.Analysis) error
}

type Renderer interface {
	ResultRenderer
	AnalysisRenderer
}

func NewResultRendererWithFormat(format string) (ResultRenderer, error) {
	return NewRendererWithFormat(format)
}

func NewAnalysisRendererWithFormat(format string) (AnalysisRenderer, error) {
	switch format {
	case "filename":
		return NewFilenameRenderer(), nil
	default:
		return NewRendererWithFormat(format)
	}
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

func AnalysisFormats() []string {
	return []string{"filename", "json", "simple", "markdown", "reference"}
}

func ResultFormats() []string {
	return []string{"json", "simple", "markdown", "reference"}
}
