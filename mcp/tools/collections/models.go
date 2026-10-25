package collections

import "time"

// CollectionSummary is a version of [storage.Collection] that safe to infer.
type CollectionSummary struct {
	ID           string    `json:"id"`
	Query        string    `json:"query"`
	ResultLimit  int       `json:"result_limit"`
	Name         *string   `json:"name,omitempty"`
	ArticleCount int       `json:"article_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
