package articles

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type Article struct {
	Label      string            `json:"label"`
	IDs        map[string]string `json:"ids"`
	Authors    []string          `json:"authors"`
	Year       *int              `json:"year"`
	Title      *string           `json:"title"`
	Journal    *string           `json:"journal"`
	Volume     *string           `json:"volume"`
	Issue      *string           `json:"issue"`
	Page       *string           `json:"page"`
	DOI        *string           `json:"doi"`
	Permalink  *string           `json:"permalink"`
	TimesCited *int              `json:"timesCited"`
	Keywords   []string          `json:"keywords"`
	Abstract   *string           `json:"abstract"`
	References []*Reference      `json:"references"`
}

// SortedIDs returns a sorted slice of the Article's IDs in the format "source:id".
func (a *Article) SortedIDs() []string {
	if a == nil || a.IDs == nil {
		return nil
	}
	ids := make([]string, 0, len(a.IDs))
	for source, id := range a.IDs {
		ids = append(ids, fmt.Sprintf("%s:%s", source, id))
	}
	slices.Sort(ids)
	return ids
}

// Merge creates a new Article by merging the fields of the current Article with another Article.
func (a *Article) Merge(other *Article) *Article {
	if a == nil {
		return other
	}
	if other == nil {
		return a
	}
	newIDs := mergeMaps(a.IDs, other.IDs)
	merged := &Article{
		Label:      *keepLongestString(a.Label, other.Label),
		IDs:        newIDs,
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
		Keywords:   keepLongestSlice(a.Keywords, other.Keywords),
		References: keepLongestSlice(a.References, other.References),
	}

	return merged
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

// Reference returns the reference of the Article.
func (a *Article) Reference() *Reference {
	if a == nil {
		return nil
	}
	return &Reference{
		Label: a.Label,
		IDs:   a.IDs,
		Title: a.Title,
	}
}

// AddSimpleId adds a simple ID to the Article's IDs set.
func (a *Article) AddSimpleId() *Article {
	if a == nil {
		return nil
	}
	simpleId := a.SimpleId()
	if simpleId != nil && *simpleId != "" {
		a.IDs["simple"] = *simpleId
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

func mergeMaps[K comparable, V any](ms ...map[K]V) map[K]V {
	var newMap map[K]V
	for _, m := range ms {
		if m != nil {
			if newMap == nil {
				newMap = make(map[K]V)
			}
			maps.Copy(newMap, m)
		}
	}
	return newMap
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
