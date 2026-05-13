package renderers

import (
	"fmt"
	"io"
	"text/template"

	"github.com/coreofscience/go-bibx/models"
)

type MarkdownReferenceRenderer struct {
	template *template.Template
}

func NewMarkdownReferenceRenderer() (*MarkdownReferenceRenderer, error) {
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
	return &MarkdownReferenceRenderer{
		template: templ,
	}, nil
}

// RenderResults implements the [Renderer] interface
func (e *MarkdownReferenceRenderer) RenderResults(w io.Writer, results []*models.Result) error {
	for _, result := range results {
		if err := e.template.ExecuteTemplate(
			w,
			"reference.md",
			result.Article,
		); err != nil {
			return fmt.Errorf("error rendering template: %w", err)
		}
		if _, err := w.Write([]byte("\n\n---\n\n")); err != nil {
			return fmt.Errorf("error writing separator: %w", err)
		}
	}
	return nil
}

// RenderArticle implements the [Renderer] interface
func (e *MarkdownReferenceRenderer) RenderArticle(w io.Writer, article *models.Article) error {
	if err := e.template.ExecuteTemplate(
		w,
		"reference.md",
		article,
	); err != nil {
		return fmt.Errorf("error rendering template: %w", err)
	}
	return nil
}
