package models

import "github.com/coreofscience/go-bibx/models"

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
