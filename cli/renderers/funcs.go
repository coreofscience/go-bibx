package renderers

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/models"
	"go.yaml.in/yaml/v4"
)

func wrap(limit int, v any) string {
	var s string
	switch t := v.(type) {
	case string:
		s = t
	case *string:
		if t == nil {
			return ""
		}
		s = *t
	default:
		return ""
	}
	return utils.WrapText(s, limit)
}

func join(sep string, items []string) string {
	return strings.Join(items, sep)
}

func frontMatter(a *models.Article) string {
	if a == nil {
		return ""
	}
	keywords := a.Keywords.Items()
	m := &struct {
		Title      *string   `yaml:"title,omitempty"`
		Journal    *string   `yaml:"journal,omitempty"`
		Volume     *string   `yaml:"volume,omitempty"`
		Issue      *string   `yaml:"issue,omitempty"`
		Page       *string   `yaml:"page,omitempty"`
		DOI        *string   `yaml:"doi,omitempty"`
		Permalink  *string   `yaml:"permalink,omitempty"`
		TimesCited *int      `yaml:"times_cited,omitempty"`
		Keywords   *[]string `yaml:"keywords,omitempty"`
		Rich       bool      `yaml:"rich"`
	}{
		Title:      a.Title,
		Journal:    a.Journal,
		Volume:     a.Volume,
		Issue:      a.Issue,
		Page:       a.Page,
		DOI:        a.DOI,
		Permalink:  a.Permalink,
		TimesCited: a.TimesCited,
		Keywords:   &keywords,
		Rich:       a.Rich,
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return "---\nerror: failed to marshal frontmatter\n---"
	}
	return "---\n" + string(data) + "---"
}

func render(templ *template.Template) func(name string, data any) (string, error) {
	return func(name string, data any) (string, error) {
		var buf bytes.Buffer
		if err := templ.ExecuteTemplate(&buf, name, data); err != nil {
			return "", fmt.Errorf("failed to render template: %w", err)
		}
		return buf.String(), nil
	}
}
