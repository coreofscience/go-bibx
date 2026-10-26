package models

import (
	"github.com/coreofscience/go-bibx/models"
)

// LinkSummary represents a directed citation edge between two nodes in an analysis graph.
type LinkSummary struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Weight float64 `json:"weight"`
}

// NewLinkSummaryFromModel converts a [models.Link] to a schema-safe [LinkSummary].
func NewLinkSummaryFromModel(l *models.Link) *LinkSummary {
	if l == nil {
		return nil
	}
	return &LinkSummary{
		Source: l.Source,
		Target: l.Target,
		Weight: l.Weight,
	}
}
