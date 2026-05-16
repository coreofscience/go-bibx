package renderers

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/coreofscience/go-bibx/models"
)

// JSONRenderer renders results as JSON
type JSONRenderer struct {
	writer io.Writer
}

// NewJSONRenderer creates a new JSONRenderer with the given writer
func NewJSONRenderer(writer io.Writer) *JSONRenderer {
	return &JSONRenderer{writer: writer}
}

// Render implements the [Renderer] interface
func (e *JSONRenderer) Render(results []*models.Result) error {
	encoder := json.NewEncoder(e.writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("error encoding results: %w", err)
	}
	return nil
}
