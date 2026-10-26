package models

import (
	"github.com/coreofscience/go-bibx/models"
)

// NodeSummary represents an article node within an analysis graph, safe for MCP jsonschema reflection.
type NodeSummary struct {
	ID        string          `json:"id"`
	Category  string          `json:"category"`
	Rootness  float64         `json:"rootness"`
	Trunkness float64         `json:"trunkness"`
	Leafness  float64         `json:"leafness"`
	Community int             `json:"community"`
	Article   *ArticleSummary `json:"article,omitempty"`
}

// NewNodeSummaryFromModel converts a [models.Node] to a schema-safe [NodeSummary].
func NewNodeSummaryFromModel(n *models.Node) *NodeSummary {
	if n == nil {
		return nil
	}
	return &NodeSummary{
		ID:        n.ID,
		Category:  string(n.Category),
		Rootness:  n.Rootness,
		Trunkness: n.Trunkness,
		Leafness:  n.Leafness,
		Community: n.Community,
		Article:   NewArticleSummaryFromModel(n.Article),
	}
}
