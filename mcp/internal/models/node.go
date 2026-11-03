package models

import (
	"github.com/coreofscience/go-bibx/models"
)

// NodeSummary represents a work node within an analysis graph, safe for MCP jsonschema reflection.
type NodeSummary struct {
	ID        string       `json:"id" jsonschema:"Unique node identifier in the citation graph"`
	Category  string       `json:"category" jsonschema:"Assigned Tree of Science category ('root', 'trunk', 'leaf', or 'none')"`
	Rootness  float64      `json:"rootness" jsonschema:"Score measuring foundational/seminal impact (in-degree from collection citation graph). A high score does not depend on the work mentioning the query"`
	Trunkness float64      `json:"trunkness" jsonschema:"SAP score measuring structural bridge centrality between roots and leaves"`
	Leafness  float64      `json:"leafness" jsonschema:"Score measuring modern state-of-the-art relevance (citation paths reaching roots)"`
	Work      *WorkSummary `json:"work,omitempty" jsonschema:"Bibliographic metadata for the work"`
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
		Work:      NewWorkSummaryFromModel(n.Work),
	}
}
