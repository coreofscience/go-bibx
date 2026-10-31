package models

import (
	"time"

	"github.com/coreofscience/go-bibx/services"
)

// CollectionSummary is a version of [services.Collection] that is safe to infer.
type CollectionSummary struct {
	ID          string    `json:"id" jsonschema:"Unique collection UUID"`
	Query       string    `json:"query" jsonschema:"Search query used to build the collection"`
	ResultLimit int       `json:"result_limit" jsonschema:"Maximum number of initial seed works requested from OpenAlex"`
	Name        *string   `json:"name,omitempty" jsonschema:"Optional human-readable alias for the collection"`
	WorkCount   int       `json:"work_count" jsonschema:"Total number of works in the analyzed citation graph"`
	CreatedAt   time.Time `json:"created_at" jsonschema:"Creation timestamp (RFC 3339)"`
	UpdatedAt   time.Time `json:"updated_at" jsonschema:"Last update timestamp (RFC 3339)"`
}

// NewCollectionSummaryFromModel converts a [services.Collection] to a schema-safe [CollectionSummary].
func NewCollectionSummaryFromModel(c *services.Collection) *CollectionSummary {
	if c == nil {
		return nil
	}
	return &CollectionSummary{
		ID:          c.ID.String(),
		Query:       c.Query,
		ResultLimit: c.ResultLimit,
		Name:        c.Name,
		WorkCount:   c.WorkCount,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
