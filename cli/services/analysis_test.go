package services_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/coreofscience/go-bibx/clients/openalex"
	"github.com/coreofscience/go-bibx/cli/services"
	openalexmocks "github.com/coreofscience/go-bibx/internal/mocks/clients/openalex"
	"github.com/coreofscience/go-bibx/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

	oaClient := openalexmocks.NewMockClient(t)
	oaClient.EXPECT().
		ListRecentArticles(mock.Anything, mock.Anything, mock.Anything).
		Return(worksMapToSlice(w1, w2, w3, w4), nil).
		Times(4)
	oaClient.EXPECT().
		ListArticlesByIDs(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, ids []string) ([]*openalex.Work, error) {
			var results []*openalex.Work
			for _, id := range ids {
				if w, ok := worksMap[id]; ok {
					results = append(results, w)
				}
			}
			return results, nil
		}).
		Maybe()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "service_test.db")
	store, err := storage.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, store.Close())
	}()

	svc := services.NewOpenAlexAnalysisService(oaClient, store)

	// 1. Initial Store
	col, err := svc.Store(context.Background(), "test query", 10)
	require.NoError(t, err)
	require.NotNil(t, col)
	oaClient.AssertNumberOfCalls(t, "ListRecentArticles", 1)
	assert.Equal(t, 4, col.ArticleCount)

	// 2. Query Idempotency: re-running identical query (with whitespace variation) skips remote fetch
	col2, err := svc.Store(context.Background(), "  test   query  ", 10)
	require.NoError(t, err)
	require.NotNil(t, col2)
	assert.Equal(t, col.ID, col2.ID, "idempotent query must return the existing collection")
	oaClient.AssertNumberOfCalls(t, "ListRecentArticles", 1)

	// 3. Force re-run: with force=true, OpenAlex is called again
	col3, err := svc.Store(context.Background(), "test query", 10, true)
	require.NoError(t, err)
	require.NotNil(t, col3)
	oaClient.AssertNumberOfCalls(t, "ListRecentArticles", 2)

	// 4. Test Query category (defaults to latest collection)
	results, err := svc.Query(context.Background(), "root", 2)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// 5. Test Search with matching keyword query (via SQLite FTS5)
	searchResults, err := svc.Search(context.Background(), "First Paper", 2)
	require.NoError(t, err)
	require.NotNil(t, searchResults)
	assert.NotEmpty(t, searchResults.Nodes)

	// 6. Test Search with zero matches returns empty analysis
	emptyResults, err := svc.Search(context.Background(), "nonexistentqueryxyz", 2)
	require.NoError(t, err)
	require.NotNil(t, emptyResults)
	assert.Empty(t, emptyResults.Nodes)
	assert.Empty(t, emptyResults.Links)

	// 7. Test StoreWithOptions with custom name
	namedRes, err := svc.StoreWithOptions(context.Background(), "named query", 10, services.StoreOptions{
		Name: "custom-alias",
	})
	require.NoError(t, err)
	require.True(t, namedRes.IsNew)
	require.NotNil(t, namedRes.Collection.Name)
	assert.Equal(t, "custom-alias", *namedRes.Collection.Name)

	// Re-run idempotent check with StoreWithOptions
	cachedRes, err := svc.StoreWithOptions(context.Background(), "named query", 10, services.StoreOptions{})
	require.NoError(t, err)
	assert.False(t, cachedRes.IsNew)
	assert.Equal(t, namedRes.Collection.ID, cachedRes.Collection.ID)

	// Force re-run with updated name updates the existing collection's name
	renamedRes, err := svc.StoreWithOptions(context.Background(), "named query", 10, services.StoreOptions{
		Force: true,
		Name:  "renamed-alias",
	})
	require.NoError(t, err)
	assert.True(t, renamedRes.IsNew)
	assert.Equal(t, namedRes.Collection.ID, renamedRes.Collection.ID)
	require.NotNil(t, renamedRes.Collection.Name)
	assert.Equal(t, "renamed-alias", *renamedRes.Collection.Name)
}

func TestOpenAlexAnalysisService_ReCleansAfterEnrichDiscardsMissingReferences(t *testing.T) {
	// W1 and W2 cite W3 (surviving initial Clean), and W4 & W5 only connect to the giant component via W_MISSING.
	// When enrich discards W_MISSING, re-running Clean() removes the now-disconnected W4 & W5 and dangling nodes.
	w1 := &openalex.Work{
		ID:              "https://openalex.org/W1",
		Title:           new("Paper 1"),
		IDs:             map[string]string{"openalex": "https://openalex.org/W1"},
		ReferencedWorks: []string{"https://openalex.org/W3", "https://openalex.org/W_MISSING"},
	}
	w2 := &openalex.Work{
		ID:              "https://openalex.org/W2",
		Title:           new("Paper 2"),
		IDs:             map[string]string{"openalex": "https://openalex.org/W2"},
		ReferencedWorks: []string{"https://openalex.org/W3"},
	}
	w3 := &openalex.Work{
		ID:    "https://openalex.org/W3",
		Title: new("Paper 3"),
		IDs:   map[string]string{"openalex": "https://openalex.org/W3"},
	}
	w4 := &openalex.Work{
		ID:              "https://openalex.org/W4",
		Title:           new("Isolated After Enrich"),
		IDs:             map[string]string{"openalex": "https://openalex.org/W4"},
		ReferencedWorks: []string{"https://openalex.org/W_MISSING"},
	}

	oaClient := openalexmocks.NewMockClient(t)
	oaClient.EXPECT().
		ListRecentArticles(mock.Anything, mock.Anything, mock.Anything).
		Return(worksMapToSlice(w1, w2, w3, w4), nil).
		Once()
	oaClient.EXPECT().
		ListArticlesByIDs(mock.Anything, mock.Anything).
		Return([]*openalex.Work{}, nil). // W_MISSING is not returned by OpenAlex
		Once()

	tempDir := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(tempDir, "reclean_test.db"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, store.Close())
	}()

	svc := services.NewOpenAlexAnalysisService(oaClient, store)
	col, err := svc.Store(context.Background(), "reclean query", 10)
	require.NoError(t, err)
	require.NotNil(t, col)
	// W4 became disconnected after W_MISSING was discarded during enrichment, so only W1, W2, W3 remain
	assert.Equal(t, 3, col.ArticleCount)

	loaded, err := store.LoadAnalysis(context.Background(), col.ID)
	require.NoError(t, err)
	for _, n := range loaded.Nodes {
		assert.NotEmpty(t, n.Category)
		assert.NotEqual(t, "openalex:https://openalex.org/W4", n.ID)
	}
}

func worksMapToSlice(works ...*openalex.Work) []*openalex.Work {
	return works
}
