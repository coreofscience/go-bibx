package renderers

import (
	"fmt"
	"io"

	"github.com/coreofscience/go-bibx/models"
)

// FilenameRenderer renders analysis node filenames to an output writer.
type FilenameRenderer struct{}

// NewFilenameRenderer creates a new [FilenameRenderer] instance.
func NewFilenameRenderer() *FilenameRenderer {
	return &FilenameRenderer{}
}

// RenderAnalysis outputs the generated markdown filename for each node in the analysis.
func (r *FilenameRenderer) RenderAnalysis(w io.Writer, analysis *models.Analysis) error {
	for _, node := range analysis.Nodes {
		if node.Article == nil {
			continue
		}
		filename := node.Filename()
		if _, err := fmt.Fprintln(w, filename); err != nil {
			return fmt.Errorf("failed to write filename: %w", err)
		}
	}
	return nil
}
