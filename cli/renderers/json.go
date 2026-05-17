package renderers

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/coreofscience/go-bibx/models"
)

// JSONRenderer renders results as JSON
type JSONRenderer struct {
}

// NewJSONRenderer creates a new JSONRenderer
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// RenderResults implements the [Renderer] interface
func (e *JSONRenderer) RenderResults(w io.Writer, results []*models.Result) error {
	return e.render(w, results, "results")
}

// RenderArticle implements the [Renderer] interface
func (e *JSONRenderer) RenderArticle(w io.Writer, article *models.Article) error {
	return e.render(w, article, "article")
}

// RenderAnalysis implements [Renderer].
func (e *JSONRenderer) RenderAnalysis(w io.Writer, analysis *models.Analysis) error {
	return e.render(w, analysis, "analysis")
}

func (e *JSONRenderer) render(w io.Writer, item any, what string) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(item); err != nil {
		return fmt.Errorf("error encoding %s: %w", what, err)
	}
	return nil
}
