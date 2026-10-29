package models

import "github.com/coreofscience/go-bibx/models"

// ArticleSummary represents the article metadata returned in analysis tool results,
// safe for MCP jsonschema reflection.
type ArticleSummary struct {
	Label      string   `json:"label" jsonschema:"Primary article identifier in the collection (e.g., 'openalex:W...')"`
	Title      *string  `json:"title,omitempty" jsonschema:"Article title"`
	Authors    []string `json:"authors,omitempty" jsonschema:"Author names"`
	Year       *int     `json:"year,omitempty" jsonschema:"Publication year"`
	Journal    *string  `json:"journal,omitempty" jsonschema:"Publication venue or journal name"`
	Volume     *string  `json:"volume,omitempty" jsonschema:"Journal volume"`
	Issue      *string  `json:"issue,omitempty" jsonschema:"Journal issue"`
	Page       *string  `json:"page,omitempty" jsonschema:"Page numbers or range"`
	DOI        *string  `json:"doi,omitempty" jsonschema:"Digital Object Identifier (DOI)"`
	Permalink  *string  `json:"permalink,omitempty" jsonschema:"Canonical URL to the article"`
	TimesCited *int     `json:"times_cited,omitempty" jsonschema:"Global citation count reported by OpenAlex"`
	Keywords   []string `json:"keywords,omitempty" jsonschema:"Associated keywords or concepts"`
	Abstract   *string  `json:"abstract,omitempty" jsonschema:"Article abstract text"`
	IDs        []string `json:"ids,omitempty" jsonschema:"All known identifiers for the article (e.g., 'openalex:...', 'doi:...', 'simple:...')"`
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
