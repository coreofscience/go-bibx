package renderers

import (
	"embed"
	"fmt"
	"io"
	"text/template"

	"github.com/coreofscience/go-bibx/models"
)

//go:embed templates/*.md
var templatesFS embed.FS

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
		if err := e.render(w, result.Article, e.format); err != nil {
			return err
		}
	}
	return nil
}

// RenderAnalysis implements [Renderer].
func (e *MarkdownRenderer) RenderAnalysis(w io.Writer, analysis *models.Analysis) error {
	for _, node := range analysis.Nodes {
		if err := e.render(w, node.Article, e.format); err != nil {
			return err
		}
	}
	return nil
}

func (e *MarkdownRenderer) render(w io.Writer, item any, format string) error {
	if err := e.template.ExecuteTemplate(
		w,
		fmt.Sprintf("%s.md", format),
		item,
	); err != nil {
		return fmt.Errorf("error rendering template: %w", err)
	}
	if _, err := w.Write([]byte("\n\n---\n\n")); err != nil {
		return fmt.Errorf("error writing separator: %w", err)
	}
	return nil
}
