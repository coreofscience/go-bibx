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
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("error encoding results: %w", err)
	}
	return nil
}

// RenderArticle implements the [Renderer] interface
func (e *JSONRenderer) RenderArticle(w io.Writer, article *models.Article) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(article); err != nil {
		return fmt.Errorf("error encoding article: %w", err)
	}
	return nil
}

// RenderAnalysis implements [Renderer].
func (e *JSONRenderer) RenderAnalysis(w io.Writer, analysis *models.Analysis) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(analysis); err != nil {
		return fmt.Errorf("error encoding article: %w", err)
	}
	return nil
}
