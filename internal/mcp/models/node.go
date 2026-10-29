package models

import (
	"github.com/coreofscience/go-bibx/models"
)

// NodeSummary represents an article node within an analysis graph, safe for MCP jsonschema reflection.
type NodeSummary struct {
	ID        string          `json:"id" jsonschema:"Unique node identifier in the citation graph"`
	Category  string          `json:"category" jsonschema:"Assigned Tree of Science category ('root', 'trunk', 'leaf', or 'none')"`
	Rootness  float64         `json:"rootness" jsonschema:"Score measuring foundational/seminal impact (in-degree from collection citation graph)"`
	Trunkness float64         `json:"trunkness" jsonschema:"SAP score measuring structural bridge centrality between roots and leaves"`
	Leafness  float64         `json:"leafness" jsonschema:"Score measuring modern state-of-the-art relevance (citation paths reaching roots)"`
	Article   *ArticleSummary `json:"article,omitempty" jsonschema:"Bibliographic metadata for the article"`
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
		Article:   NewArticleSummaryFromModel(n.Article),
	}
}
