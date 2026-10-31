package renderers

import (
	"fmt"
	"io"

	"github.com/coreofscience/go-bibx/models"
)

// BibtexRenderer renders results and analyses in BibTeX format.
type BibtexRenderer struct{}

// NewBibtexRenderer creates a new [BibtexRenderer].
func NewBibtexRenderer() *BibtexRenderer {
	return &BibtexRenderer{}
}

// RenderResults implements the [ResultRenderer] interface.
func (b *BibtexRenderer) RenderResults(w io.Writer, results []*models.Result) error {
	works := make(models.Works, 0, len(results))
	for _, res := range results {
		if res != nil && res.Work != nil {
			works = append(works, res.Work)
		}
	}
	output := works.ToBibTeX()
	if output != "" {
		if _, err := io.WriteString(w, output); err != nil {
			return fmt.Errorf("error writing bibtex results: %w", err)
		}
	}
	return nil
}

// RenderAnalysis implements the [AnalysisRenderer] interface.
func (b *BibtexRenderer) RenderAnalysis(w io.Writer, analysis *models.Analysis) error {
	if analysis == nil {
		return nil
	}
	works := make(models.Works, 0, len(analysis.Nodes))
	for _, node := range analysis.Nodes {
		if node != nil && node.Work != nil {
			works = append(works, node.Work)
		}
	}
	output := works.ToBibTeX()
	if output != "" {
		if _, err := io.WriteString(w, output); err != nil {
			return fmt.Errorf("error writing bibtex analysis: %w", err)
		}
	}
	return nil
}
