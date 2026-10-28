package renderers

import (
	"fmt"
	"io"

	"github.com/coreofscience/go-bibx/models"
)

// ResultRenderer renders a list of scored query results to an output writer.
type ResultRenderer interface {
	// RenderResults renders the graph in a specific format.
	RenderResults(w io.Writer, result []*models.Result) error
}

// AnalysisRenderer renders a complete [models.Analysis] dataset to an output writer.
type AnalysisRenderer interface {
	// RenderAnalysis renders an analysis in a specific format.
	RenderAnalysis(w io.Writer, analysis *models.Analysis) error
}

// Renderer combines [ResultRenderer] and [AnalysisRenderer] interfaces.
type Renderer interface {
	ResultRenderer
	AnalysisRenderer
}

// NewResultRendererWithFormat creates a [ResultRenderer] for the specified format.
func NewResultRendererWithFormat(format string) (ResultRenderer, error) {
	return NewRendererWithFormat(format)
}

// NewAnalysisRendererWithFormat creates an [AnalysisRenderer] for the specified format.
func NewAnalysisRendererWithFormat(format string) (AnalysisRenderer, error) {
	switch format {
	case "filename":
		return NewFilenameRenderer(), nil
	default:
		return NewRendererWithFormat(format)
	}
}

// NewRendererWithFormat creates a unified [Renderer] for formats supporting both results and analysis.
func NewRendererWithFormat(format string) (Renderer, error) {
	switch format {
	case "json":
		return NewJSONRenderer(), nil
	case "bibtex":
		return NewBibtexRenderer(), nil
	case "simple", "markdown", "reference":
		return NewMarkdownRenderer(format)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// AnalysisFormats returns the list of all supported output formats for analysis rendering.
func AnalysisFormats() []string {
	return []string{"filename", "json", "simple", "markdown", "reference", "bibtex"}
}

// ResultFormats returns the list of all supported output formats for result rendering.
func ResultFormats() []string {
	return []string{"json", "simple", "markdown", "reference", "bibtex"}
}
