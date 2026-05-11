package articles

import (
	"bytes"
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/coreofscience/go-bibx/internal/collections"
	"gopkg.in/yaml.v3"
)

//go:embed templates/article.md
var articleTemplateRaw string

var articleTemplate = template.Must(template.New("article").Funcs(template.FuncMap{
	"wrap": func(limit int, indent int, v any) string {
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
		words := strings.Fields(s)
		if len(words) == 0 {
			return ""
		}
		var result strings.Builder
		lineLen := 0
		for i, word := range words {
			if lineLen+len(word)+1 > limit && lineLen > 0 {
				result.WriteString("\n" + strings.Repeat(" ", indent))
				lineLen = indent
			} else if i > 0 {
				result.WriteString(" ")
				lineLen++
			}
			result.WriteString(word)
			lineLen += len(word)
		}
		return result.String()
	},
	"join": func(sep string, items []string) string {
		return strings.Join(items, sep)
	},
	"frontMatter": func(a *Article) string {
		if a == nil {
			return ""
		}
		m := a.FrontMatter()
		data, err := yaml.Marshal(m)
		if err != nil {
			return "---\nerror: failed to marshal frontmatter\n---"
		}
		return "---\n" + string(data) + "---"
	},
}).Parse(articleTemplateRaw))

type Article struct {
	Label      string                   `json:"label"`
	IDs        *collections.Set[string] `json:"ids,omitempty"`
	Authors    []string                 `json:"authors,omitempty"`
	Year       *int                     `json:"year,omitempty"`
	Title      *string                  `json:"title,omitempty"`
	Journal    *string                  `json:"journal,omitempty"`
	Volume     *string                  `json:"volume,omitempty"`
	Issue      *string                  `json:"issue,omitempty"`
	Page       *string                  `json:"page,omitempty"`
	DOI        *string                  `json:"doi,omitempty"`
	Permalink  *string                  `json:"permalink,omitempty"`
	TimesCited *int                     `json:"times_cited,omitempty"`
	Keywords   *collections.Set[string] `json:"keywords,omitempty"`
	Abstract   *string                  `json:"abstract,omitempty"`
	References References               `json:"references,omitempty"`
	Rich       bool                     `json:"rich"`
}

type FrontMatter struct {
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
}

type ArticleOption func(*Article)

func WithReferences(references []*Article) ArticleOption {
	return func(a *Article) {
		a.References = references
	}
}

func (a *Article) Clone(options ...ArticleOption) *Article {
	if a == nil {
		return nil
	}
	copied := &Article{
		Label:      a.Label,
		IDs:        a.IDs.Clone(),
		Authors:    slices.Clone(a.Authors),
		Year:       a.Year,
		Title:      a.Title,
		Journal:    a.Journal,
		Volume:     a.Volume,
		Issue:      a.Issue,
		Page:       a.Page,
		DOI:        a.DOI,
		Permalink:  a.Permalink,
		TimesCited: a.TimesCited,
		Keywords:   a.Keywords.Clone(),
		Abstract:   a.Abstract,
		References: slices.Clone(a.References),
		Rich:       a.Rich,
	}
	for _, option := range options {
		option(copied)
	}
	return copied
}

func (a *Article) FrontMatter() *FrontMatter {
	keywords := a.Keywords.Items()
	return &FrontMatter{
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
}

// Merge creates a new Article by merging the fields of the current Article with another Article.
func (a *Article) Merge(other *Article) *Article {
	if a == nil {
		return other
	}
	if other == nil {
		return a
	}
	merged := &Article{
		Label:      *keepLongestString(a.Label, other.Label),
		IDs:        a.IDs.Union(other.IDs),
		Authors:    keepLongestSlice(a.Authors, other.Authors),
		Year:       keep(a.Year, other.Year),
		Title:      keep(a.Title, other.Title),
		Journal:    keep(a.Journal, other.Journal),
		Volume:     keep(a.Volume, other.Volume),
		Issue:      keep(a.Issue, other.Issue),
		Page:       keep(a.Page, other.Page),
		DOI:        keep(a.DOI, other.DOI),
		Permalink:  keep(a.Permalink, other.Permalink),
		TimesCited: keep(a.TimesCited, other.TimesCited),
		Keywords:   a.Keywords.Union(other.Keywords),
		References: keepLongestSlice(a.References, other.References),
		Rich:       a.Rich || other.Rich,
	}
	return merged
}

func (a *Article) ID(source string) (string, bool) {
	if a == nil || a.IDs.Len() == 0 {
		return "", false
	}
	items := a.IDs.Items()
	for _, id := range items {
		if id, found := strings.CutPrefix(id, fmt.Sprintf("%s:", source)); found {
			return id, true
		}
	}
	return "", false
}

func (a *Article) PurgeReferences(ids ...string) *Article {
	if a == nil {
		return nil
	}
	idSet := collections.NewSet(ids...)
	shouldPurge := slices.ContainsFunc(a.References, func(ref *Article) bool {
		return ref.IDs.Intersect(idSet).Len() > 0
	})
	if !shouldPurge {
		return a
	}
	newReferences := make([]*Article, 0, len(a.References))
	for _, ref := range a.References {
		if ref.IDs.Intersect(idSet).Len() == 0 {
			newReferences = append(newReferences, ref)
		}
	}
	return a.Clone(WithReferences(newReferences))
}

// Key returns the first ID of the Article if it exists.
func (a *Article) Key() *string {
	if a == nil || a.IDs.Len() == 0 {
		return nil
	}
	items := a.IDs.Items()
	return &items[0]
}

// SimpleLabel returns a simplified label for the Article.
func (a *Article) SimpleLabel() *string {
	if a == nil {
		return nil
	}
	parts := make([]string, 0)
	if len(a.Authors) > 0 {
		parts = append(parts, strings.ReplaceAll(a.Authors[0], ",", ""))
	}
	if a.Year != nil {
		parts = append(parts, fmt.Sprintf("%d", *a.Year))
	}
	if a.Journal != nil {
		parts = append(parts, *a.Journal)
	}
	if a.Volume != nil {
		parts = append(parts, fmt.Sprintf("V%s", *a.Volume))
	}
	if a.Page != nil {
		parts = append(parts, fmt.Sprintf("P%s", *a.Page))
	}
	if a.DOI != nil {
		parts = append(parts, fmt.Sprintf("DOI %s", *a.DOI))
	}
	if len(parts) == 0 {
		return nil
	}
	result := strings.Join(parts, ", ")
	return &result
}

// SimpleId returns the first author's name and the year.
func (a *Article) SimpleId() *string {
	if a == nil {
		return nil
	}
	if len(a.Authors) == 0 || a.Year == nil {
		return nil
	}
	author := a.Authors[0]
	firstName := strings.Split(author, " ")[0]
	name := strings.ReplaceAll(firstName, ",", "")
	result := strings.ToLower(fmt.Sprintf("%s%d", name, *a.Year))
	return &result
}

// Permalink returns the permalink of the Article if it exists.
func (a *Article) GetPermalink() *string {
	if a == nil {
		return nil
	}
	if a.Permalink != nil && *a.Permalink != "" {
		return a.Permalink
	}
	if a.DOI != nil && *a.DOI != "" {
		result := fmt.Sprintf("https://doi.org/%s", *a.DOI)
		return &result
	}
	return nil
}

// AddSimpleId adds a simple ID to the Article's IDs set.
func (a *Article) AddSimpleId() *Article {
	if a == nil {
		return nil
	}
	simpleId := a.SimpleId()
	if simpleId != nil && *simpleId != "" {
		id := fmt.Sprintf("simple:%s", *simpleId)
		a.IDs.Add(id)
	}
	return a
}

// ToMarkdown returns the Article as a Markdown string.
func (a *Article) ToMarkdown() string {
	var buf bytes.Buffer
	if err := articleTemplate.Execute(&buf, a); err != nil {
		return fmt.Sprintf("# %s\n", a.Label)
	}
	return buf.String()
}

// SetSimpleLabel sets a simplified label for the Article.
func (a *Article) SetSimpleLabel() *Article {
	if a == nil {
		return nil
	}
	simpleLabel := a.SimpleLabel()
	if simpleLabel != nil && *simpleLabel != "" {
		a.Label = *simpleLabel
	}
	return a
}

func keep[T any](a, b *T) *T {
	if a != nil {
		return a
	}
	return b
}

func keepLongestString(a, b string) *string {
	if len(a) > len(b) {
		return &a
	}
	if len(b) > len(a) {
		return &b
	}
	if a > b {
		return &b
	}
	return &a
}

func keepLongestSlice[T any](a, b []T) []T {
	if len(a) >= len(b) {
		return a
	}
	return b
}
