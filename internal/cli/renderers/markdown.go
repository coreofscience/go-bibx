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

// MarkdownRenderer renders works and analysis results using embedded markdown templates.
type MarkdownRenderer struct {
	template *template.Template
	format   string
}

// NewMarkdownRenderer creates a new [MarkdownRenderer] with the specified template format.
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
		if err := e.Render(w, result.Work, e.format); err != nil {
			return err
		}
	}
	return nil
}

// RenderAnalysis implements [Renderer].
func (e *MarkdownRenderer) RenderAnalysis(w io.Writer, analysis *models.Analysis) error {
	if analysis == nil {
		return nil
	}
	nodeByID := make(map[string]*models.Work, len(analysis.Nodes))
	for _, node := range analysis.Nodes {
		if node != nil && node.Work != nil {
			nodeByID[node.ID] = node.Work
		}
	}
	refsBySource := make(map[string][]*models.Work)
	for _, link := range analysis.Links {
		if link == nil {
			continue
		}
		if targetArt, ok := nodeByID[link.Target]; ok {
			refsBySource[link.Source] = append(refsBySource[link.Source], targetArt)
		}
	}
	for _, node := range analysis.Nodes {
		if node == nil || node.Work == nil {
			continue
		}
		work := node.Work
		if len(work.References) == 0 && len(refsBySource[node.ID]) > 0 {
			work = work.Clone(models.WithReferences(refsBySource[node.ID]))
		}
		if err := e.Render(w, work, e.format); err != nil {
			return err
		}
	}
	return nil
}

// Render executes the template corresponding to format on the provided item and writes the output to w.
func (e *MarkdownRenderer) Render(w io.Writer, item any, format string) error {
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
