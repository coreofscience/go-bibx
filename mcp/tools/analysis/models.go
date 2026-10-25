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
	res := &Result{
		Score: r.Score,
	}
	if r.Article != nil {
		art := r.Article
		var keywords []string
		if art.Keywords != nil {
			keywords = art.Keywords.Items()
		}
		var ids []string
		if art.IDs != nil {
			ids = art.IDs.Items()
		}
		res.Article = &ArticleSummary{
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
	return res
}
