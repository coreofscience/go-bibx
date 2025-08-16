package models

import (
	"fmt"
	"sort"
	"strings"

	"github.com/coreofscience/go-bibx/collections"
)

type Article struct {
	Label      string
	IDs        collections.Set[string]
	Authors    []string
	Year       *int
	Title      *string
	Journal    *string
	Volume     *string
	Issue      *string
	Page       *string
	DOI        *string
	Permalink  *string
	TimesCited *int
	Keywords   []string
	References []*Article
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
		Label:      *keepLongest(a.Label, other.Label),
		IDs:        *a.IDs.Union(&other.IDs),
		Authors:    append(a.Authors, other.Authors...),
		Year:       keep(a.Year, other.Year),
		Title:      keep(a.Title, other.Title),
		Journal:    keep(a.Journal, other.Journal),
		Volume:     keep(a.Volume, other.Volume),
		Issue:      keep(a.Issue, other.Issue),
		Page:       keep(a.Page, other.Page),
		DOI:        keep(a.DOI, other.DOI),
		Permalink:  keep(a.Permalink, other.Permalink),
		TimesCited: keep(a.TimesCited, other.TimesCited),
	}

	if len(a.Keywords) > 0 {
		merged.Keywords = a.Keywords
	} else {
		merged.Keywords = other.Keywords
	}

	if len(a.References) > 0 {
		merged.References = a.References
	} else {
		merged.References = other.References
	}

	return merged
}

// Key returns the first ID of the Article if it exists.
func (a *Article) Key() *string {
	if a == nil || a.IDs.Len() == 0 {
		return nil
	}
	items := a.IDs.Items()
	sort.Slice(items, func(i, j int) bool {
		// TODO: Maybe use the longest ID as the key?
		return *items[i] < *items[j]
	})
	return items[0]
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
	resunt := strings.Join(parts, ", ")
	return &resunt
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
	result := fmt.Sprintf("%s%d", name, *a.Year)
	return &result
}

// Permalink returns the permalink of the Article if it exists.
func (a *Article) GetPermalink() *string {
	if a == nil || a.Permalink == nil {
		return nil
	}
	if a.DOI != nil && *a.DOI != "" {
		result := fmt.Sprintf("https://doi.org/%s", *a.DOI)
		return &result
	}
	return a.Permalink
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

func keepLongest(a, b string) *string {
	if len(a) >= len(b) {
		return &a
	}
	return &b
}
