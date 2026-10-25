package analysis

import (
	"github.com/coreofscience/go-bibx/models"
)

// ArticleSummary represents the article metadata returned in analysis tool results,
// safe for MCP jsonschema reflection.
type ArticleSummary struct {
	Label      string   `json:"label"`
	Title      *string  `json:"title,omitempty"`
	Authors    []string `json:"authors,omitempty"`
	Year       *int     `json:"year,omitempty"`
	Journal    *string  `json:"journal,omitempty"`
	Volume     *string  `json:"volume,omitempty"`
	Issue      *string  `json:"issue,omitempty"`
	Page       *string  `json:"page,omitempty"`
	DOI        *string  `json:"doi,omitempty"`
	Permalink  *string  `json:"permalink,omitempty"`
	TimesCited *int     `json:"times_cited,omitempty"`
	Keywords   []string `json:"keywords,omitempty"`
	Abstract   *string  `json:"abstract,omitempty"`
	IDs        []string `json:"ids,omitempty"`
}

// NewArticleSummaryFromModel converts a [models.Article] to a schema-safe [ArticleSummary].
func NewArticleSummaryFromModel(art *models.Article) *ArticleSummary {
	if art == nil {
		return nil
	}
	var keywords []string
	if art.Keywords != nil {
		keywords = art.Keywords.Items()
	}
	var ids []string
	if art.IDs != nil {
		ids = art.IDs.Items()
	}
	return &ArticleSummary{
		Label:      art.Label,
		Title:      art.Title,
		Authors:    art.Authors,
		Year:       art.Year,
		Journal:    art.Journal,
		Volume:     art.Volume,
		Issue:      art.Issue,
		Page:       art.Page,
		DOI:        art.DOI,
		Permalink:  art.Permalink,
		TimesCited: art.TimesCited,
		Keywords:   keywords,
		Abstract:   art.Abstract,
		IDs:        ids,
	}
}

// Result represents a scored article in a Tree of Science category query.
type Result struct {
	Score   float64         `json:"score"`
	Article *ArticleSummary `json:"article,omitempty"`
}

// NewResultFromModel converts a [models.Result] to a schema-safe [Result].
func NewResultFromModel(r *models.Result) *Result {
	if r == nil {
		return nil
	}
	return &Result{
		Score:   r.Score,
		Article: NewArticleSummaryFromModel(r.Article),
	}
}

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
