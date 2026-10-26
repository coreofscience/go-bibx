package models

import "github.com/coreofscience/go-bibx/models"

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
