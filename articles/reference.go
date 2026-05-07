package articles

import "github.com/coreofscience/go-bibx/internal/collections"

// Reference represents a reference to an article.
type Reference struct {
	Label string                   `json:"label"`
	IDs   *collections.Set[string] `json:"ids"`
	Title *string                  `json:"title,omitempty"`
}
