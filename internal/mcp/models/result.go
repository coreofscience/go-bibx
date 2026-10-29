package models

import "github.com/coreofscience/go-bibx/models"

// Result represents a scored article in a Tree of Science category query.
type Result struct {
	Score   float64         `json:"score" jsonschema:"Tree of Science category score (unnormalized SAP metric; higher means stronger fit)"`
	Article *ArticleSummary `json:"article,omitempty" jsonschema:"Bibliographic metadata for the scored article"`
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
