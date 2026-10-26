package models

import (
	"time"

	"github.com/coreofscience/go-bibx/services"
)

// CollectionSummary is a version of [services.Collection] that is safe to infer.
type CollectionSummary struct {
	ID           string    `json:"id"`
	Query        string    `json:"query"`
	ResultLimit  int       `json:"result_limit"`
	Name         *string   `json:"name,omitempty"`
	ArticleCount int       `json:"article_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewCollectionSummaryFromModel converts a [services.Collection] to a schema-safe [CollectionSummary].
func NewCollectionSummaryFromModel(c *services.Collection) *CollectionSummary {
	if c == nil {
		return nil
	}
	return &CollectionSummary{
		ID:           c.ID.String(),
		Query:        c.Query,
		ResultLimit:  c.ResultLimit,
		Name:         c.Name,
		ArticleCount: c.ArticleCount,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}
