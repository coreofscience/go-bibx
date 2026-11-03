package models

import "github.com/coreofscience/go-bibx/models"

// WorkSummary represents the work metadata returned in analysis tool results,
// safe for MCP jsonschema reflection.
type WorkSummary struct {
	Label      string   `json:"label" jsonschema:"Primary work identifier in the collection (e.g., 'openalex:W...')"`
	Title      *string  `json:"title,omitempty" jsonschema:"Work title"`
	Authors    []string `json:"authors,omitempty" jsonschema:"Author names"`
	Year       *int     `json:"year,omitempty" jsonschema:"Publication year"`
	Venue      *string  `json:"venue,omitempty" jsonschema:"Publication venue, such as a journal or conference"`
	Volume     *string  `json:"volume,omitempty" jsonschema:"Journal volume"`
	Issue      *string  `json:"issue,omitempty" jsonschema:"Journal issue"`
	Page       *string  `json:"page,omitempty" jsonschema:"Page numbers or range"`
	DOI        *string  `json:"doi,omitempty" jsonschema:"Digital Object Identifier (DOI)"`
	Permalink  *string  `json:"permalink,omitempty" jsonschema:"Canonical URL to the work"`
	TimesCited *int     `json:"times_cited,omitempty" jsonschema:"Global citation count reported by OpenAlex"`
	Keywords   []string `json:"keywords,omitempty" jsonschema:"Associated keywords or concepts"`
	Abstract   *string  `json:"abstract,omitempty" jsonschema:"Work abstract text"`
	IDs        []string `json:"ids,omitempty" jsonschema:"All known identifiers for the work (e.g., 'openalex:...', 'doi:...', 'simple:...')"`
}

// NewWorkSummaryFromModel converts a [models.Work] to a schema-safe [WorkSummary].
func NewWorkSummaryFromModel(work *models.Work) *WorkSummary {
	if work == nil {
		return nil
	}
	var keywords []string
	if work.Keywords != nil {
		keywords = work.Keywords.Items()
	}
	var ids []string
	if work.IDs != nil {
		ids = work.IDs.Items()
	}
	return &WorkSummary{
		Label:      work.Label,
		Title:      work.Title,
		Authors:    work.Authors,
		Year:       work.Year,
		Venue:      work.Venue,
		Volume:     work.Volume,
		Issue:      work.Issue,
		Page:       work.Page,
		DOI:        work.DOI,
		Permalink:  work.Permalink,
		TimesCited: work.TimesCited,
		Keywords:   keywords,
		Abstract:   work.Abstract,
		IDs:        ids,
	}
}
