package models_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalysis_Search(t *testing.T) {
	title1 := "Deep Graph Neural Networks"
	title2 := "Convolutional Neural Networks"
	abstract2 := "Discussion on graph representations."
	title3 := "Quantum Computing Algorithms"

	node1 := &models.Node{
		ID: "n1",
		Article: &models.Article{
			Title: &title1,
		},
	}
	node2 := &models.Node{
		ID: "n2",
		Article: &models.Article{
			Title:    &title2,
			Abstract: &abstract2,
		},
	}
	node3 := &models.Node{
		ID: "n3",
		Article: &models.Article{
			Title: &title3,
		},
	}

	analysis := &models.Analysis{
		Nodes: []*models.Node{node2, node3, node1},
	}

	// Query "graph neural" matches n1 (both terms in title + phrase boost) and n2 (title + abstract)
	results, err := analysis.Search("graph neural", 10)
	require.NoError(t, err)

	require.Len(t, results, 2)
	assert.Equal(t, "n1", results[0].ID)
	assert.Equal(t, "n2", results[1].ID)
}

func TestAnalysis_Search_KeywordsAndAuthors(t *testing.T) {
	title1 := "Search Engine Architectures"
	title2 := "General Systems"

	node1 := &models.Node{
		ID: "n1",
		Article: &models.Article{
			Title:   &title1,
			Authors: []string{"Gerard Salton"},
		},
	}
	node2 := &models.Node{
		ID: "n2",
		Article: &models.Article{
			Title:    &title2,
			Keywords: collections.NewSet("salton", "bm25"),
		},
	}

	analysis := &models.Analysis{
		Nodes: []*models.Node{node1, node2},
	}

	results, err := analysis.Search("salton", 10)
	require.NoError(t, err)
	require.Len(t, results, 2)
}

func TestAnalysis_Search_Limit(t *testing.T) {
	title1 := "Network Science One"
	title2 := "Network Science Two"

	node1 := &models.Node{
		ID:      "n1",
		Article: &models.Article{Title: &title1},
	}
	node2 := &models.Node{
		ID:      "n2",
		Article: &models.Article{Title: &title2},
	}

	analysis := &models.Analysis{
		Nodes: []*models.Node{node1, node2},
	}

	results, err := analysis.Search("network", 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestAnalysis_Search_EmptyNodes(t *testing.T) {
	analysis := &models.Analysis{
		Nodes: nil,
	}

	results, err := analysis.Search("query", 10)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestAnalysis_Search_NoMatches(t *testing.T) {
	title := "Network Science"
	analysis := &models.Analysis{
		Nodes: []*models.Node{
			{
				ID:      "n1",
				Article: &models.Article{Title: &title},
			},
		},
	}

	results, err := analysis.Search("nonexistent", 10)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestAnalysis_Search_NilArticle(t *testing.T) {
	analysis := &models.Analysis{
		Nodes: []*models.Node{
			{
				ID:      "n1",
				Article: nil,
			},
		},
	}

	results, err := analysis.Search("query", 10)
	require.NoError(t, err)
	assert.Nil(t, results)
}
