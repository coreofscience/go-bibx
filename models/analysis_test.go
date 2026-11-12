package models_test

import (
	"math"
	"testing"

	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeScore(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0.0, models.NormalizeScore(0, 100))
	assert.Equal(t, 0.0, models.NormalizeScore(-5, 100))
	assert.Equal(t, 0.0, models.NormalizeScore(10, 0))
	assert.Equal(t, 0.0, models.NormalizeScore(10, -1))
	assert.Equal(t, 1.0, models.NormalizeScore(100, 100))
	assert.Equal(t, 1.0, models.NormalizeScore(150, 100))
	assert.Equal(t, 1.0, models.NormalizeScore(1, 1))

	expected := math.Log1p(100) / math.Log1p(10000)
	assert.InDelta(t, expected, models.NormalizeScore(100, 10000), 1e-12)
}

func TestNormalizeResults(t *testing.T) {
	t.Parallel()

	assert.Empty(t, models.NormalizeResults(nil))

	w1 := &models.Work{Label: "W1"}
	w2 := &models.Work{Label: "W2"}
	results := []*models.Result{
		{Score: 10000, Work: w1},
		{Score: 100, Work: w2},
		nil,
	}

	normalized := models.NormalizeResults(results)
	require.Len(t, normalized, 3)
	assert.Equal(t, 1.0, normalized[0].Score)
	assert.Same(t, w1, normalized[0].Work)
	assert.InDelta(t, math.Log1p(100)/math.Log1p(10000), normalized[1].Score, 1e-12)
	assert.Same(t, w2, normalized[1].Work)
	assert.Nil(t, normalized[2])

	// Original slice must not be mutated
	assert.Equal(t, 10000.0, results[0].Score)
	assert.Equal(t, 100.0, results[1].Score)
}

func TestAnalysis_NormalizeAndQuery(t *testing.T) {
	t.Parallel()

	var nilAnalysis *models.Analysis
	assert.Nil(t, nilAnalysis.Normalize())

	w1 := &models.Work{Label: "R1"}
	w2 := &models.Work{Label: "R2"}
	w3 := &models.Work{Label: "T1"}
	w4 := &models.Work{Label: "T2"}
	w5 := &models.Work{Label: "L1"}

	analysis := &models.Analysis{
		Nodes: []*models.Node{
			{ID: "R1", Category: models.CategoryRoot, Rootness: 50, Work: w1},
			{ID: "R2", Category: models.CategoryRoot, Rootness: 10, Work: w2},
			{ID: "T1", Category: models.CategoryTrunk, Trunkness: 1_000_000, Work: w3},
			{ID: "T2", Category: models.CategoryTrunk, Trunkness: 1_000, Work: w4},
			{ID: "L1", Category: models.CategoryLeaf, Leafness: 250, Work: w5},
			nil,
		},
		Links: []*models.Link{
			{Source: "L1", Target: "T1", Weight: 1.0},
		},
	}

	norm := analysis.Normalize()
	require.NotNil(t, norm)
	require.Len(t, norm.Nodes, 6)
	require.Len(t, norm.Links, 1)

	assert.Equal(t, 1.0, norm.Nodes[0].Rootness)
	assert.InDelta(t, math.Log1p(10)/math.Log1p(50), norm.Nodes[1].Rootness, 1e-12)
	assert.Equal(t, 0.0, norm.Nodes[0].Trunkness)
	assert.Equal(t, 0.0, norm.Nodes[0].Leafness)

	assert.Equal(t, 1.0, norm.Nodes[2].Trunkness)
	assert.InDelta(t, math.Log1p(1_000)/math.Log1p(1_000_000), norm.Nodes[3].Trunkness, 1e-12)

	assert.Equal(t, 1.0, norm.Nodes[4].Leafness)
	assert.Nil(t, norm.Nodes[5])

	// Original analysis remains unmutated
	assert.Equal(t, 50.0, analysis.Nodes[0].Rootness)
	assert.Equal(t, 1_000_000.0, analysis.Nodes[2].Trunkness)

	// Query returns normalized scores relative to full category max even when top=1
	topTrunks, err := analysis.Query("trunk", 2)
	require.NoError(t, err)
	require.Len(t, topTrunks, 2)
	assert.Equal(t, 1.0, topTrunks[0].Score)
	assert.InDelta(t, math.Log1p(1_000)/math.Log1p(1_000_000), topTrunks[1].Score, 1e-12)
}
