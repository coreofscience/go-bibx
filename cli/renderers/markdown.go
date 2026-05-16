package renderers

import (
	"fmt"
	"io"
	"text/template"

	"github.com/coreofscience/go-bibx/models"
)

type MarkdownRenderer struct {
	template *template.Template
	format   string
}

func NewMarkdownRenderer(format string) (*MarkdownRenderer, error) {
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
		format:   format,
	}, nil
}

// RenderResults implements the [Renderer] interface
func (e *MarkdownRenderer) RenderResults(w io.Writer, results []*models.Result) error {
	for _, result := range results {
		if err := e.template.ExecuteTemplate(
			w,
			fmt.Sprintf("%s.md", e.format),
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
func (e *MarkdownRenderer) RenderArticle(w io.Writer, article *models.Article) error {
	if err := e.template.ExecuteTemplate(
		w,
		fmt.Sprintf("%s.md", e.format),
		article,
	); err != nil {
		return fmt.Errorf("error rendering template: %w", err)
	}
	return nil
}
