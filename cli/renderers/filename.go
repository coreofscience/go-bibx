package renderers

import (
	"fmt"
	"io"

	"github.com/coreofscience/go-bibx/models"
)

type FilenameRenderer struct{}

func NewFilenameRenderer() *FilenameRenderer {
	return &FilenameRenderer{}
}

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
