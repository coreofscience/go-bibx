package models

import (
	"github.com/coreofscience/go-bibx/models"
)

// NodeSummary represents a work node within an analysis graph, safe for MCP jsonschema reflection.
type NodeSummary struct {
	ID        string   `json:"id" jsonschema:"Unique node identifier in the citation graph"`
	Category  string   `json:"category" jsonschema:"Assigned Tree of Science category ('root', 'trunk', 'leaf', or 'none')"`
	Rootness  float64  `json:"rootness" jsonschema:"Normalized score in [0, 1] measuring foundational/seminal impact (log-scaled relative to the top root in the collection). A high score does not depend on the work mentioning the query"`
	Trunkness float64  `json:"trunkness" jsonschema:"Normalized SAP score in [0, 1] measuring structural bridge centrality between roots and leaves (log-scaled relative to the top trunk in the collection)"`
	Leafness  float64  `json:"leafness" jsonschema:"Normalized score in [0, 1] measuring modern state-of-the-art relevance (log-scaled relative to the top leaf in the collection)"`
	Title     *string  `json:"title,omitempty" jsonschema:"Work title"`
	Authors   []string `json:"authors,omitempty" jsonschema:"Author names"`
	DOI       *string  `json:"doi,omitempty" jsonschema:"Digital Object Identifier (DOI)"`
	IDs       []string `json:"ids,omitempty" jsonschema:"All known identifiers for the work (e.g., 'openalex:...', 'doi:...', 'simple:...')"`
}

// NewNodeSummaryFromModel converts a [models.Node] to a schema-safe [NodeSummary].
func NewNodeSummaryFromModel(n *models.Node) *NodeSummary {
	if n == nil {
		return nil
	}
	summary := &NodeSummary{
		ID:        n.ID,
		Category:  string(n.Category),
		Rootness:  n.Rootness,
		Trunkness: n.Trunkness,
		Leafness:  n.Leafness,
	}
	if n.Work != nil {
		summary.Title = n.Work.Title
		summary.Authors = n.Work.Authors
		summary.DOI = n.Work.DOI
		if n.Work.IDs != nil {
			summary.IDs = n.Work.IDs.Items()
		}
	}
	return summary
}
