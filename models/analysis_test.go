package models_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalysis_Search(t *testing.T) {
	nodeExact := &models.Node{
		ID:        "exact",
		Embedding: []float32{1.0, 0.0},
	}
	nodeOrthogonal := &models.Node{
		ID:        "orthogonal",
		Embedding: []float32{0.0, 1.0},
	}
	nodeOpposite := &models.Node{
		ID:        "opposite",
		Embedding: []float32{-1.0, 0.0},
	}

	analysis := &models.Analysis{
		Nodes: []*models.Node{nodeOrthogonal, nodeOpposite, nodeExact},
	}

	// Query identical to nodeExact
	queryVec := []float32{1.0, 0.0}

	results, err := analysis.Search(queryVec, 10)
	require.NoError(t, err)

	// Previously score == 0 was skipped, so nodeExact would be excluded!
	// Now nodeExact MUST be included as the first (best) match.
	require.Len(t, results, 3)
	assert.Equal(t, "exact", results[0].ID)
	assert.Equal(t, "orthogonal", results[1].ID)
	assert.Equal(t, "opposite", results[2].ID)
}

func TestAnalysis_Search_Limit(t *testing.T) {
	node1 := &models.Node{ID: "n1", Embedding: []float32{1.0, 0.0}}
	node2 := &models.Node{ID: "n2", Embedding: []float32{0.0, 1.0}}

	analysis := &models.Analysis{
		Nodes: []*models.Node{node1, node2},
	}

	results, err := analysis.Search([]float32{1.0, 0.0}, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "n1", results[0].ID)
}

func TestAnalysis_Search_InvalidVectorLength(t *testing.T) {
	analysis := &models.Analysis{
		Nodes: []*models.Node{
			{ID: "n1", Embedding: []float32{1.0, 0.0}},
		},
	}

	_, err := analysis.Search([]float32{1.0}, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid vector length")
}

func TestAnalysis_Search_EmptyNodes(t *testing.T) {
	analysis := &models.Analysis{
		Nodes: nil,
	}

	results, err := analysis.Search([]float32{1.0, 0.0}, 10)
	require.NoError(t, err)
	assert.Nil(t, results)
}
