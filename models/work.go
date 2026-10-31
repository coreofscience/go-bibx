package models

import (
	"fmt"
	"slices"
	"strings"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
)

// Work represents a bibliographic work, containing citation metadata, identifiers,
// author information, and references to cited works.
type Work struct {
	Label      string                   `json:"label"`
	IDs        *collections.Set[string] `json:"ids,omitempty"`
	Authors    []string                 `json:"authors,omitempty"`
	Year       *int                     `json:"year,omitempty"`
	Title      *string                  `json:"title,omitempty"`
	Venue      *string                  `json:"venue,omitempty"`
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

// WorkOption specifies a functional option for configuring an [Work].
type WorkOption func(*Work)

// WithReferences sets the references for an [Work].
func WithReferences(references []*Work) WorkOption {
	return func(w *Work) {
		w.References = references
	}
}

// Clone returns a deep copy of the [Work], applying any provided [WorkOption] modifiers.
func (w *Work) Clone(options ...WorkOption) *Work {
	if w == nil {
		return nil
	}
	copied := &Work{
		Label:      w.Label,
		IDs:        w.IDs.Clone(),
		Authors:    slices.Clone(w.Authors),
		Year:       w.Year,
		Title:      w.Title,
		Venue:      w.Venue,
		Volume:     w.Volume,
		Issue:      w.Issue,
		Page:       w.Page,
		DOI:        w.DOI,
		Permalink:  w.Permalink,
		TimesCited: w.TimesCited,
		Keywords:   w.Keywords.Clone(),
		Abstract:   w.Abstract,
		References: slices.Clone(w.References),
		Rich:       w.Rich,
	}
	for _, option := range options {
		option(copied)
	}
	return copied
}

// Merge creates a new Work by merging the fields of the current Work with another Work.
func (w *Work) Merge(other *Work) *Work {
	if w == nil {
		return other
	}
	if other == nil {
		return w
	}
	if !w.Rich && other.Rich {
		return other.Merge(w)
	}
	merged := &Work{
		Label:      utils.KeepLongestString(w.Label, other.Label),
		IDs:        w.IDs.Union(other.IDs),
		Authors:    utils.KeepLongestSlice(w.Authors, other.Authors),
		Year:       utils.Keep(w.Year, other.Year),
		Title:      utils.Keep(w.Title, other.Title),
		Venue:      utils.Keep(w.Venue, other.Venue),
		Volume:     utils.Keep(w.Volume, other.Volume),
		Issue:      utils.Keep(w.Issue, other.Issue),
		Page:       utils.Keep(w.Page, other.Page),
		DOI:        utils.Keep(w.DOI, other.DOI),
		Permalink:  utils.Keep(w.Permalink, other.Permalink),
		TimesCited: utils.Keep(w.TimesCited, other.TimesCited),
		Keywords:   w.Keywords.Union(other.Keywords),
		Abstract:   utils.Keep(w.Abstract, other.Abstract),
		References: utils.KeepLongestSlice(w.References, other.References),
		Rich:       w.Rich || other.Rich,
	}
	return merged
}

// ID extracts the identifier value for the specified prefix source (e.g. "doi", "openalex").
// Returns the identifier and true if found, or an empty string and false otherwise.
func (w *Work) ID(source string) (string, bool) {
	if w == nil || w.IDs.Len() == 0 {
		return "", false
	}
	items := w.IDs.Items()
	for _, id := range items {
		if id, found := strings.CutPrefix(id, fmt.Sprintf("%s:", source)); found {
			return id, true
		}
	}
	return "", false
}

// PurgeReferences returns a cloned [Work] with any references matching the given IDs removed.
func (w *Work) PurgeReferences(ids ...string) *Work {
	if w == nil {
		return nil
	}
	idSet := collections.NewSet(ids...)
	shouldPurge := slices.ContainsFunc(w.References, func(ref *Work) bool {
		return ref.IDs.Intersect(idSet).Len() > 0
	})
	if !shouldPurge {
		return w
	}
	newReferences := make([]*Work, 0, len(w.References))
	for _, ref := range w.References {
		if ref.IDs.Intersect(idSet).Len() == 0 {
			newReferences = append(newReferences, ref)
		}
	}
	return w.Clone(WithReferences(newReferences))
}

// KeepReferences returns a cloned [Work] retaining only references that match at least one of the given IDs.
func (w *Work) KeepReferences(ids ...string) *Work {
	if w == nil {
		return nil
	}
	idSet := collections.NewSet(ids...)
	shouldDiscard := slices.ContainsFunc(w.References, func(ref *Work) bool {
		return ref.IDs.Intersect(idSet).Len() == 0
	})
	if !shouldDiscard {
		return w
	}
	newReferences := make([]*Work, 0, len(w.References))
	for _, ref := range w.References {
		if ref.IDs.Intersect(idSet).Len() > 0 {
			newReferences = append(newReferences, ref)
		}
	}
	return w.Clone(WithReferences(newReferences))
}

// Key returns the first ID of the Work if it exists.
func (w *Work) Key() *string {
	if w == nil || w.IDs.Len() == 0 {
		return nil
	}
	items := w.IDs.Items()
	return &items[0]
}

// SimpleLabel returns a simplified label for the Work.
func (w *Work) SimpleLabel() *string {
	if w == nil {
		return nil
	}
	parts := make([]string, 0)
	if len(w.Authors) > 0 {
		parts = append(parts, strings.ReplaceAll(w.Authors[0], ",", ""))
	}
	if w.Year != nil {
		parts = append(parts, fmt.Sprintf("%d", *w.Year))
	}
	if w.Venue != nil {
		parts = append(parts, *w.Venue)
	}
	if w.Volume != nil {
		parts = append(parts, fmt.Sprintf("V%s", *w.Volume))
	}
	if w.Page != nil {
		parts = append(parts, fmt.Sprintf("P%s", *w.Page))
	}
	if w.DOI != nil {
		parts = append(parts, fmt.Sprintf("DOI %s", *w.DOI))
	}
	if len(parts) == 0 {
		return nil
	}
	result := strings.Join(parts, ", ")
	return &result
}

// SimpleId returns the first author's name and the year.
func (w *Work) SimpleId() *string {
	if w == nil {
		return nil
	}
	if len(w.Authors) == 0 || w.Year == nil {
		return nil
	}
	author := w.Authors[0]
	firstName, _, _ := strings.Cut(author, " ")
	name := strings.ReplaceAll(firstName, ",", "")
	result := strings.ToLower(fmt.Sprintf("%s%d", name, *w.Year))
	return &result
}

// Permalink returns the permalink of the Work if it exists.
func (w *Work) GetPermalink() *string {
	if w == nil {
		return nil
	}
	if w.Permalink != nil && *w.Permalink != "" {
		return w.Permalink
	}
	if w.DOI != nil && *w.DOI != "" {
		result := fmt.Sprintf("https://doi.org/%s", *w.DOI)
		return &result
	}
	return nil
}

// AddSimpleId adds a simple ID to the Work's IDs set.
func (w *Work) AddSimpleId() *Work {
	if w == nil {
		return nil
	}
	simpleId := w.SimpleId()
	if simpleId != nil && *simpleId != "" {
		id := fmt.Sprintf("simple:%s", *simpleId)
		if w.IDs == nil {
			w.IDs = collections.NewSet[string]()
		}
		w.IDs.Add(id)
	}
	return w
}

// SetSimpleLabel sets a simplified label for the Work.
func (w *Work) SetSimpleLabel() *Work {
	if w == nil {
		return nil
	}
	simpleLabel := w.SimpleLabel()
	if simpleLabel != nil && *simpleLabel != "" {
		w.Label = *simpleLabel
	}
	return w
}
