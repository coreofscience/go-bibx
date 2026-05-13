package renderers

import (
	"fmt"
	"io"
	"text/template"

	"github.com/coreofscience/go-bibx/models"
)

type MarkdownRenderer struct {
	template *template.Template
	writer   io.Writer
}

func NewMarkdownRenderer(writer io.Writer) (*MarkdownRenderer, error) {
	templ := template.New("")
	templ, err := templ.Funcs(template.FuncMap{
		"wrap":        wrap,
		"join":        join,
		"frontMatter": frontMatter,
		"render":      render(templ),
	}).ParseFS(templatesFS, "templates/*.md")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	return &MarkdownRenderer{
		template: templ,
		writer:   writer,
	}, nil
}

// Render implements the [Renderer] interface
func (e *MarkdownRenderer) Render(results []*models.Result) error {
	for _, result := range results {
		if err := e.template.ExecuteTemplate(
			e.writer,
			"article.md",
			result.Article,
		); err != nil {
			return fmt.Errorf("error rendering template: %w", err)
		}
		if _, err := e.writer.Write([]byte("\n\n---\n\n")); err != nil {
			return fmt.Errorf("error writing separator: %w", err)
		}
	}
	return nil
}
