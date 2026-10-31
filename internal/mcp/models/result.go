package models

import "github.com/coreofscience/go-bibx/models"

// Result represents a scored work in a Tree of Science category query.
type Result struct {
	Score float64      `json:"score" jsonschema:"Tree of Science category score (unnormalized SAP metric; higher means stronger fit)"`
	Work  *WorkSummary `json:"work,omitempty" jsonschema:"Bibliographic metadata for the scored work"`
}

// NewResultFromModel converts a [models.Result] to a schema-safe [Result].
func NewResultFromModel(r *models.Result) *Result {
	if r == nil {
		return nil
	}
	return &Result{
		Score: r.Score,
		Work:  NewWorkSummaryFromModel(r.Work),
	}
}
