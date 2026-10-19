package services_test

import (
	"context"
	"testing"

	"github.com/coreofscience/go-bibx/cli/clients/openalex"
	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/internal/texter"
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

type mockEmbeddingsClient struct {
	capturedTexts []string
	embedManyFunc func(texts []string) ([][]float32, error)
	embedFunc     func(text string) ([]float32, error)
}

func (m *mockEmbeddingsClient) Sync(ctx context.Context) error {
	return nil
}

func (m *mockEmbeddingsClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.embedFunc != nil {
		return m.embedFunc(text)
	}
	return []float32{1.0, 0.0}, nil
}

func (m *mockEmbeddingsClient) EmbedMany(ctx context.Context, texts []string) ([][]float32, error) {
	m.capturedTexts = append(m.capturedTexts, texts...)
	if m.embedManyFunc != nil {
		return m.embedManyFunc(texts)
	}
	res := make([][]float32, len(texts))
	for i := range texts {
		res[i] = []float32{float32(i + 1), 0.0}
	}
	return res, nil
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

func TestOpenAlexAnalysisService_Store_EmbeddingTextsAllocation(t *testing.T) {
	title1 := "First Paper"
	title2 := "Second Paper"
	title3 := "Third Paper"
	title4 := "Fourth Paper"

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
	embedClient := &mockEmbeddingsClient{}
	analysisRepo := &mockAnalysisRepo{}
	ttr := texter.NewDefaultArticleTexter()

	svc := services.NewOpenAlexAnalysisService(oaClient, analysisRepo, embedClient, ttr)

	err := svc.Store(context.Background(), "test query", 10)
	require.NoError(t, err)

	// Verify that texts passed to EmbedMany matches the number of articles in collection
	// and does not have leading empty strings from make([]string, collection.Len())
	require.Len(t, embedClient.capturedTexts, 4)
	for _, txt := range embedClient.capturedTexts {
		assert.NotEmpty(t, txt)
	}

	// Verify that the stored analysis has nodes with matching embeddings
	require.NotNil(t, analysisRepo.storedAnalysis)
	require.Len(t, analysisRepo.storedAnalysis.Nodes, 4)
	for i, node := range analysisRepo.storedAnalysis.Nodes {
		assert.Equal(t, []float32{float32(i + 1), 0.0}, node.Embedding)
	}

	// Test Query
	results, err := svc.Query(context.Background(), "root", 2)
	require.NoError(t, err)
	assert.NotNil(t, results)

	// Test Search
	embedClient.embedFunc = func(text string) ([]float32, error) {
		return []float32{1.0, 0.0}, nil
	}
	searchResults, err := svc.Search(context.Background(), "First Paper", 2)
	require.NoError(t, err)
	assert.NotNil(t, searchResults)
}

