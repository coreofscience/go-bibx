package articles

import (
	"fmt"
	"slices"
)

// Reference represents a reference to an article.
type Reference struct {
	Label string            `json:"label"`
	IDs   map[string]string `json:"ids"`
	Title *string           `json:"title,omitempty"`
}

// SortedIDs returns a sorted slice of the Article's IDs in the format "source:id".
func (a *Reference) SortedIDs() []string {
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
