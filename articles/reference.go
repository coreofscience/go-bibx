package articles

import "github.com/coreofscience/go-bibx/internal/collections"

// Reference represents a reference to an article.
type Reference struct {
	Label string                   `json:"label"`
	IDs   *collections.Set[string] `json:"ids"`
	Title *string                  `json:"title,omitempty"`
}

// Key returns the first ID of the Reference if it exists.
func (r *Reference) Key() *string {
	if r == nil || r.IDs.Len() == 0 {
		return nil
	}
	items := r.IDs.Items()
	return &items[0]
}
