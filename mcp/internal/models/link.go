package models

import (
	"github.com/coreofscience/go-bibx/models"
)

// LinkSummary represents a directed citation edge between two nodes in an analysis graph.
type LinkSummary struct {
	Source string  `json:"source" jsonschema:"Citing node ID"`
	Target string  `json:"target" jsonschema:"Cited node ID"`
	Weight float64 `json:"weight" jsonschema:"Edge weight in the citation network"`
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
