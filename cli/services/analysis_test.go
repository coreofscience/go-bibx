package services_test

import (
	"context"
	"testing"

	"github.com/coreofscience/go-bibx/cli/clients/openalex"
	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOpenAlexClient struct {
	recentWorks []*openalex.Work
	worksByID   map[string]*openalex.Work
}

func (m *mockOpenAlexClient) ListRecentArticles(ctx context.Context, query string, limit int) ([]*openalex.Work, error) {
	return m.recentWorks, nil
}

func (m *mockOpenAlexClient) ListArticlesByIDs(ctx context.Context, ids []string) ([]*openalex.Work, error) {
	var results []*openalex.Work
	for _, id := range ids {
		if w, ok := m.worksByID[id]; ok {
			results = append(results, w)
		}
	}
	return results, nil
}

type mockAnalysisRepo struct {
	storedAnalysis *models.Analysis
}

func (m *mockAnalysisRepo) Load(ctx context.Context) (*models.Analysis, error) {
	return m.storedAnalysis, nil
}

func (m *mockAnalysisRepo) Store(ctx context.Context, a *models.Analysis) error {
	m.storedAnalysis = a
	return nil
}

func TestOpenAlexAnalysisService_StoreAndSearch(t *testing.T) {
	title1 := "First Paper on Machine Learning"
	title2 := "Second Paper on Deep Learning"
	title3 := "Third Paper on Neural Networks"
	title4 := "Fourth Paper on Foundation Models"

	w1 := &openalex.Work{
		ID:              "https://openalex.org/W1",
		Title:           &title1,
		IDs:             map[string]string{"openalex": "https://openalex.org/W1"},
		ReferencedWorks: []string{"https://openalex.org/W2", "https://openalex.org/W3"},
	}
	w2 := &openalex.Work{
		ID:              "https://openalex.org/W2",
		Title:           &title2,
		IDs:             map[string]string{"openalex": "https://openalex.org/W2"},
		ReferencedWorks: []string{"https://openalex.org/W4"},
	}
	w3 := &openalex.Work{
		ID:              "https://openalex.org/W3",
		Title:           &title3,
		IDs:             map[string]string{"openalex": "https://openalex.org/W3"},
		ReferencedWorks: []string{"https://openalex.org/W4"},
	}
	w4 := &openalex.Work{
		ID:    "https://openalex.org/W4",
		Title: &title4,
		IDs:   map[string]string{"openalex": "https://openalex.org/W4"},
	}

	worksMap := map[string]*openalex.Work{
		"https://openalex.org/W1": w1,
		"https://openalex.org/W2": w2,
		"https://openalex.org/W3": w3,
		"https://openalex.org/W4": w4,
	}

	oaClient := &mockOpenAlexClient{
		recentWorks: []*openalex.Work{w1, w2, w3, w4},
		worksByID:   worksMap,
	}
	analysisRepo := &mockAnalysisRepo{}

	svc := services.NewOpenAlexAnalysisService(oaClient, analysisRepo)

	err := svc.Store(context.Background(), "test query", 10)
	require.NoError(t, err)

	// Verify that the stored analysis has 4 nodes
	require.NotNil(t, analysisRepo.storedAnalysis)
	require.Len(t, analysisRepo.storedAnalysis.Nodes, 4)

	// Verify all links have weight 1.0
	require.NotEmpty(t, analysisRepo.storedAnalysis.Links)
	for _, link := range analysisRepo.storedAnalysis.Links {
		assert.Equal(t, 1.0, link.Weight)
	}

	// Test Query
	results, err := svc.Query(context.Background(), "root", 2)
	require.NoError(t, err)
	assert.NotNil(t, results)

	// Test Search with matching keyword query
	searchResults, err := svc.Search(context.Background(), "First Paper", 2)
	require.NoError(t, err)
	require.NotNil(t, searchResults)
	assert.NotEmpty(t, searchResults.Nodes)

	// Test Search with zero matches returns empty analysis
	emptyResults, err := svc.Search(context.Background(), "nonexistentqueryxyz", 2)
	require.NoError(t, err)
	require.NotNil(t, emptyResults)
	assert.Empty(t, emptyResults.Nodes)
	assert.Empty(t, emptyResults.Links)
}
