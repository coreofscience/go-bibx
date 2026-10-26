package models

import (
	"fmt"
	"slices"
	"strings"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
)

// Article represents a bibliographic work, containing citation metadata, identifiers,
// author information, and references to cited articles.
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

// ArticleOption specifies a functional option for configuring an [Article].
type ArticleOption func(*Article)

// WithReferences sets the references for an [Article].
func WithReferences(references []*Article) ArticleOption {
	return func(a *Article) {
		a.References = references
	}
}

// Clone returns a deep copy of the [Article], applying any provided [ArticleOption] modifiers.
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

// Merge creates a new Article by merging the fields of the current Article with another Article.
func (a *Article) Merge(other *Article) *Article {
	if a == nil {
		return other
	}
	if other == nil {
		return a
	}
	if !a.Rich && other.Rich {
		return other.Merge(a)
	}
	merged := &Article{
		Label:      utils.KeepLongestString(a.Label, other.Label),
		IDs:        a.IDs.Union(other.IDs),
		Authors:    utils.KeepLongestSlice(a.Authors, other.Authors),
		Year:       utils.Keep(a.Year, other.Year),
		Title:      utils.Keep(a.Title, other.Title),
		Journal:    utils.Keep(a.Journal, other.Journal),
		Volume:     utils.Keep(a.Volume, other.Volume),
		Issue:      utils.Keep(a.Issue, other.Issue),
		Page:       utils.Keep(a.Page, other.Page),
		DOI:        utils.Keep(a.DOI, other.DOI),
		Permalink:  utils.Keep(a.Permalink, other.Permalink),
		TimesCited: utils.Keep(a.TimesCited, other.TimesCited),
		Keywords:   a.Keywords.Union(other.Keywords),
		Abstract:   utils.Keep(a.Abstract, other.Abstract),
		References: utils.KeepLongestSlice(a.References, other.References),
		Rich:       a.Rich || other.Rich,
	}
	return merged
}

// ID extracts the identifier value for the specified prefix source (e.g. "doi", "openalex").
// Returns the identifier and true if found, or an empty string and false otherwise.
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

// PurgeReferences returns a cloned [Article] with any references matching the given IDs removed.
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

// KeepReferences returns a cloned [Article] retaining only references that match at least one of the given IDs.
func (a *Article) KeepReferences(ids ...string) *Article {
	if a == nil {
		return nil
	}
	idSet := collections.NewSet(ids...)
	shouldDiscard := slices.ContainsFunc(a.References, func(ref *Article) bool {
		return ref.IDs.Intersect(idSet).Len() == 0
	})
	if !shouldDiscard {
		return a
	}
	newReferences := make([]*Article, 0, len(a.References))
	for _, ref := range a.References {
		if ref.IDs.Intersect(idSet).Len() > 0 {
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
	firstName, _, _ := strings.Cut(author, " ")
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
		if a.IDs == nil {
			a.IDs = collections.NewSet[string]()
		}
		a.IDs.Add(id)
	}
	return a
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
