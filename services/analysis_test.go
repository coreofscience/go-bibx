package services_test

import (
	"context"
	"path/filepath"
	"testing"
	"uuid"

	"github.com/coreofscience/go-bibx/clients/openalex"
	"github.com/coreofscience/go-bibx/internal/collections"
	openalexmocks "github.com/coreofscience/go-bibx/internal/mocks/clients/openalex"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOpenAlexAnalysisService_InquireAndSearch(t *testing.T) {
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
		SearchWorks(mock.Anything, mock.Anything, mock.Anything).
		Return(worksMapToSlice(w1, w2, w3, w4), nil).
		Times(4)
	oaClient.EXPECT().
		ListWorksByIDs(mock.Anything, mock.Anything).
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
	db, err := repository.OpenAndMigrateSQLite(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	svc := services.NewAnalysisService(db, oaClient)

	// 1. Initial Inquire
	res, err := svc.Inquire(context.Background(), "test query", 10, services.InquireOptions{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.True(t, res.IsNew)
	assert.Equal(t, 4, res.SeedCount)
	col := res.Collection
	oaClient.AssertNumberOfCalls(t, "SearchWorks", 1)
	assert.Equal(t, 4, col.WorkCount)

	// 2. Query Idempotency: re-running identical query (with whitespace variation) skips remote fetch
	res2, err := svc.Inquire(context.Background(), "  test   query  ", 10, services.InquireOptions{})
	require.NoError(t, err)
	require.NotNil(t, res2)
	assert.False(t, res2.IsNew)
	assert.Zero(t, res2.SeedCount, "cached collections do not report a seed count")
	assert.Equal(t, col.ID, res2.Collection.ID, "idempotent query must return the existing collection")
	oaClient.AssertNumberOfCalls(t, "SearchWorks", 1)

	// 3. Force re-run: with force=true, OpenAlex is called again
	res3, err := svc.Inquire(context.Background(), "test query", 10, services.InquireOptions{Force: true})
	require.NoError(t, err)
	require.NotNil(t, res3)
	assert.True(t, res3.IsNew)
	oaClient.AssertNumberOfCalls(t, "SearchWorks", 2)

	// 4. Test QueryCategory
	results, err := svc.QueryCategory(context.Background(), col.ID, "root", 2)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// 5. Test Search with matching keyword query (via SQLite FTS5)
	searchResults, err := svc.Search(context.Background(), col.ID, "First Paper", 2)
	require.NoError(t, err)
	require.NotNil(t, searchResults)
	assert.NotEmpty(t, searchResults.Nodes)

	// 6. Test Search with zero matches returns empty analysis
	emptyResults, err := svc.Search(context.Background(), col.ID, "nonexistentqueryxyz", 2)
	require.NoError(t, err)
	require.NotNil(t, emptyResults)
	assert.Empty(t, emptyResults.Nodes)
	assert.Empty(t, emptyResults.Links)

	// 7. Test Inquire with custom name
	namedRes, err := svc.Inquire(context.Background(), "named query", 10, services.InquireOptions{
		Name: "custom-alias",
	})
	require.NoError(t, err)
	require.True(t, namedRes.IsNew)
	require.NotNil(t, namedRes.Collection.Name)
	assert.Equal(t, "custom-alias", *namedRes.Collection.Name)

	// Re-run idempotent check
	cachedRes, err := svc.Inquire(context.Background(), "named query", 10, services.InquireOptions{})
	require.NoError(t, err)
	assert.False(t, cachedRes.IsNew)
	assert.Equal(t, namedRes.Collection.ID, cachedRes.Collection.ID)

	// Force re-run with updated name updates the existing collection's name
	renamedRes, err := svc.Inquire(context.Background(), "named query", 10, services.InquireOptions{
		Force: true,
		Name:  "renamed-alias",
	})
	require.NoError(t, err)
	require.True(t, renamedRes.IsNew)
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
		SearchWorks(mock.Anything, mock.Anything, mock.Anything).
		Return(worksMapToSlice(w1, w2, w3, w4), nil).
		Once()
	oaClient.EXPECT().
		ListWorksByIDs(mock.Anything, mock.Anything).
		Return([]*openalex.Work{}, nil). // W_MISSING is not returned by OpenAlex
		Once()

	tempDir := t.TempDir()
	db, err := repository.OpenAndMigrateSQLite(filepath.Join(tempDir, "reclean_test.db"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	svc := services.NewAnalysisService(db, oaClient)
	res, err := svc.Inquire(context.Background(), "reclean query", 10, services.InquireOptions{})
	require.NoError(t, err)
	require.NotNil(t, res)
	col := res.Collection
	// W4 became disconnected after W_MISSING was discarded during enrichment, so only W1, W2, W3 remain
	assert.Equal(t, 3, col.WorkCount)

	loaded, err := svc.LoadAnalysis(context.Background(), col.ID)
	require.NoError(t, err)
	for _, n := range loaded.Nodes {
		assert.NotEmpty(t, n.Category)
		assert.NotEqual(t, "openalex:https://openalex.org/W4", n.ID)
	}
}

func TestOpenAlexAnalysisService_AnalysisPersistenceAndQuery(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_analysis_persistence.db")

	db, err := repository.OpenAndMigrateSQLite(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	colSvc := services.NewCollectionService(db)
	oaClient := openalexmocks.NewMockClient(t)
	anaSvc := services.NewAnalysisService(db, oaClient)
	ctx := context.Background()

	// 1. Create a collection first
	colID := uuid.NewV7()
	hash := services.ComputeQueryHash("graph algorithms", 10)
	require.NoError(t, colSvc.Create(ctx, &services.Collection{
		ID:          colID,
		QueryHash:   hash,
		Query:       "graph algorithms",
		ResultLimit: 10,
	}))

	// 2. Prepare sample models.Analysis
	year1 := 2024
	year2 := 2025
	year3 := 2026
	title1 := "Seminal Graph Paper"
	title2 := "Connecting Theory on Graphs"
	title3 := "Frontier Graph Neural Network"
	doi1 := "10.1000/1"
	abstract1 := "This foundational paper presents seminal graph theory."
	abstract2 := "This structural paper bridges root concepts to frontier models."
	abstract3 := "This frontier study applies neural networks to citation graphs."

	node1 := &models.Node{
		ID:        "openalex:W1",
		Category:  models.CategoryRoot,
		Rootness:  0.95,
		Trunkness: 0.1,
		Leafness:  0.0,
		Work: &models.Work{
			Label:    "openalex:W1",
			Title:    &title1,
			Year:     &year1,
			DOI:      &doi1,
			Abstract: &abstract1,
			Authors:  []string{"Turing, Alan", "Shannon, Claude"},
			Keywords: collections.NewSet("graph theory", "foundations"),
			IDs:      collections.NewSet("openalex:W1"),
			Rich:     true,
		},
	}

	node2 := &models.Node{
		ID:        "openalex:W2",
		Category:  models.CategoryTrunk,
		Rootness:  0.1,
		Trunkness: 0.88,
		Leafness:  0.2,
		Work: &models.Work{
			Label:    "openalex:W2",
			Title:    &title2,
			Year:     &year2,
			Abstract: &abstract2,
			Authors:  []string{"Dijkstra, Edsger"},
			Keywords: collections.NewSet("network bridges", "shortest paths"),
			IDs:      collections.NewSet("openalex:W2"),
			Rich:     true,
		},
	}

	node3 := &models.Node{
		ID:        "openalex:W3",
		Category:  models.CategoryLeaf,
		Rootness:  0.0,
		Trunkness: 0.2,
		Leafness:  0.92,
		Work: &models.Work{
			Label:    "openalex:W3",
			Title:    &title3,
			Year:     &year3,
			Abstract: &abstract3,
			Authors:  []string{"LeCun, Yann"},
			Keywords: collections.NewSet("graph neural networks", "deep learning"),
			IDs:      collections.NewSet("openalex:W3"),
			Rich:     true,
		},
	}

	links := []*models.Link{
		{Source: "openalex:W2", Target: "openalex:W1", Weight: 1.0},
		{Source: "openalex:W3", Target: "openalex:W2", Weight: 1.0},
	}

	analysis := &models.Analysis{
		Nodes: []*models.Node{node1, node2, node3},
		Links: links,
	}

	// 3. StoreAnalysis: non-existent collection fails
	err = anaSvc.StoreAnalysis(ctx, uuid.NewV7(), analysis)
	require.ErrorIs(t, err, services.ErrCollectionNotFound)

	// 4. StoreAnalysis: nil analysis fails
	err = anaSvc.StoreAnalysis(ctx, colID, nil)
	require.Error(t, err)

	// 5. StoreAnalysis: successful storage
	err = anaSvc.StoreAnalysis(ctx, colID, analysis)
	require.NoError(t, err)

	// Verify work count updated on collection
	col, err := colSvc.FindByID(ctx, colID)
	require.NoError(t, err)
	assert.Equal(t, 3, col.WorkCount)

	// 6. LoadAnalysis: successful reconstruction
	loaded, err := anaSvc.LoadAnalysis(ctx, colID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.Len(t, loaded.Nodes, 3)
	require.Len(t, loaded.Links, 2)

	// Verify QueryCategory
	rootResults, err := anaSvc.QueryCategory(ctx, colID, "root", 5)
	require.NoError(t, err)
	require.Len(t, rootResults, 2)
	assert.Equal(t, "Seminal Graph Paper", *rootResults[0].Work.Title)

	// Verify Search
	searchResults, err := anaSvc.Search(ctx, colID, "neural networks", 10)
	require.NoError(t, err)
	require.NotEmpty(t, searchResults.Nodes)

	// 7. Verify re-storing a smaller analysis removes stale nodes/links and preserves existing work authors/keywords when nil
	stubNode1 := &models.Node{
		ID:        "openalex:W1",
		Category:  models.CategoryRoot,
		Rootness:  0.80,
		Trunkness: 0.0,
		Leafness:  0.0,
		Work: &models.Work{
			Label:    "openalex:W1",
			Authors:  nil,
			Keywords: collections.NewSet[string](),
			IDs:      collections.NewSet("openalex:W1"),
			Rich:     false,
		},
	}
	smallerAnalysis := &models.Analysis{
		Nodes: []*models.Node{stubNode1, node2},
		Links: []*models.Link{
			{Source: "openalex:W2", Target: "openalex:W1", Weight: 1.0},
		},
	}
	require.NoError(t, anaSvc.StoreAnalysis(ctx, colID, smallerAnalysis))

	colAfterUpdate, err := colSvc.FindByID(ctx, colID)
	require.NoError(t, err)
	assert.Equal(t, 2, colAfterUpdate.WorkCount, "stale node W3 should be removed from collection count")

	reloaded, err := anaSvc.LoadAnalysis(ctx, colID)
	require.NoError(t, err)
	require.Len(t, reloaded.Nodes, 2, "stale node W3 should be removed")
	require.Len(t, reloaded.Links, 1, "stale link W3->W2 should be removed")

	// Search matching keyword
	kwResults, err := anaSvc.Search(ctx, colID, "bridges", 10)
	require.NoError(t, err)
	require.NotEmpty(t, kwResults.Nodes)
}

func TestOpenAlexAnalysisService_SearchHighLimitPerformance(t *testing.T) {
	tempDir := t.TempDir()
	db, err := repository.OpenAndMigrateSQLite(filepath.Join(tempDir, "high_limit_test.db"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	colSvc := services.NewCollectionService(db)
	oaClient := openalexmocks.NewMockClient(t)
	anaSvc := services.NewAnalysisService(db, oaClient)
	ctx := context.Background()

	colID := uuid.NewV7()
	hash := services.ComputeQueryHash("high limit test", 100)
	require.NoError(t, colSvc.Create(ctx, &services.Collection{
		ID:          colID,
		QueryHash:   hash,
		Query:       "high limit test",
		ResultLimit: 100,
	}))

	numNodes := 100
	nodes := make([]*models.Node, numNodes)
	links := make([]*models.Link, 0, numNodes)

	for i := 0; i < numNodes; i++ {
		lbl := filepath.Join("W", uuid.NewV7().String())
		title := "Graph Paper " + lbl
		abstract := "Topic research on quantum computing and physics"
		nodes[i] = &models.Node{
			ID:        lbl,
			Category:  models.CategoryTrunk,
			Trunkness: 0.5,
			Work: &models.Work{
				Label:    lbl,
				Title:    &title,
				Abstract: &abstract,
				Rich:     true,
			},
		}
		if i > 0 {
			links = append(links, &models.Link{
				Source: lbl,
				Target: nodes[i-1].ID,
				Weight: 1.0,
			})
		}
	}

	analysis := &models.Analysis{
		Nodes: nodes,
		Links: links,
	}

	require.NoError(t, anaSvc.StoreAnalysis(ctx, colID, analysis))

	// Perform search with limit 100
	res, err := anaSvc.Search(ctx, colID, "quantum physics", 100)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.NotEmpty(t, res.Nodes)
}

func TestOpenAlexAnalysisService_GetWorks(t *testing.T) {
	tempDir := t.TempDir()
	db, err := repository.OpenAndMigrateSQLite(filepath.Join(tempDir, "get_works_test.db"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	colSvc := services.NewCollectionService(db)
	oaClient := openalexmocks.NewMockClient(t)
	anaSvc := services.NewAnalysisService(db, oaClient)
	ctx := context.Background()

	t.Run("non-existent collection returns ErrCollectionNotFound", func(t *testing.T) {
		unknownID := uuid.NewV7()
		works, notFound, err := anaSvc.GetWorks(ctx, unknownID, []string{"W1"})
		require.ErrorIs(t, err, services.ErrCollectionNotFound)
		assert.Nil(t, works)
		assert.Nil(t, notFound)
	})

	// Setup a collection with test works
	colID := uuid.NewV7()
	hash := services.ComputeQueryHash("work lookup test", 10)
	require.NoError(t, colSvc.Create(ctx, &services.Collection{
		ID:          colID,
		QueryHash:   hash,
		Query:       "work lookup test",
		ResultLimit: 10,
	}))

	year1 := 2017
	title1 := "Attention Is All You Need"
	doi1 := "10.5555/3295222"
	permalink1 := "https://doi.org/10.5555/3295222"

	art1 := &models.Work{
		Label:     "https://openalex.org/W100",
		Title:     &title1,
		Authors:   []string{"Vaswani, Ashish", "Shazeer, Noam"},
		Year:      &year1,
		DOI:       &doi1,
		Permalink: &permalink1,
		IDs:       collections.NewSet("openalex:https://openalex.org/W100", "doi:10.5555/3295222"),
		Rich:      true,
	}

	year2 := 2018
	title2 := "Deep Residual Learning"
	doi2 := "10.1109/CVPR.2016.90"

	art2 := &models.Work{
		Label:   "custom-resnet-label",
		Title:   &title2,
		Authors: []string{"He, Kaiming", "Zhang, Xiangyu"},
		Year:    &year2,
		DOI:     &doi2,
		IDs:     collections.NewSet("openalex:W200"),
		Rich:    true,
	}

	analysis := &models.Analysis{
		Nodes: []*models.Node{
			{
				ID:       "openalex:https://openalex.org/W100",
				Category: models.CategoryRoot,
				Rootness: 0.9,
				Work:     art1,
			},
			{
				ID:        "node-resnet-id",
				Category:  models.CategoryTrunk,
				Trunkness: 0.8,
				Work:      art2,
			},
		},
		Links: []*models.Link{
			{
				Source: "openalex:https://openalex.org/W100",
				Target: "node-resnet-id",
				Weight: 1.0,
			},
		},
	}

	require.NoError(t, anaSvc.StoreAnalysis(ctx, colID, analysis))

	t.Run("empty ids returns all works in collection", func(t *testing.T) {
		works, notFound, err := anaSvc.GetWorks(ctx, colID, nil)
		require.NoError(t, err)
		assert.Empty(t, notFound)
		require.Len(t, works, 2)
	})

	t.Run("match by node ID", func(t *testing.T) {
		works, notFound, err := anaSvc.GetWorks(ctx, colID, []string{"openalex:W200"})
		require.NoError(t, err)
		assert.Empty(t, notFound)
		require.Len(t, works, 1)
		assert.Equal(t, "custom-resnet-label", works[0].Label)
	})

	t.Run("match by work label", func(t *testing.T) {
		works, notFound, err := anaSvc.GetWorks(ctx, colID, []string{"custom-resnet-label"})
		require.NoError(t, err)
		assert.Empty(t, notFound)
		require.Len(t, works, 1)
		assert.Equal(t, "Deep Residual Learning", *works[0].Title)
	})

	t.Run("match by SimpleId", func(t *testing.T) {
		works, notFound, err := anaSvc.GetWorks(ctx, colID, []string{"vaswani2017"})
		require.NoError(t, err)
		assert.Empty(t, notFound)
		require.Len(t, works, 1)
		assert.Equal(t, "Attention Is All You Need", *works[0].Title)
	})

	t.Run("match by DOI variations", func(t *testing.T) {
		// Clean DOI
		works, notFound, err := anaSvc.GetWorks(ctx, colID, []string{"10.5555/3295222"})
		require.NoError(t, err)
		assert.Empty(t, notFound)
		require.Len(t, works, 1)

		// Prefixed with doi:
		works, notFound, err = anaSvc.GetWorks(ctx, colID, []string{"doi:10.5555/3295222"})
		require.NoError(t, err)
		assert.Empty(t, notFound)
		require.Len(t, works, 1)

		// Prefixed with https://doi.org/
		works, notFound, err = anaSvc.GetWorks(ctx, colID, []string{"https://doi.org/10.5555/3295222"})
		require.NoError(t, err)
		assert.Empty(t, notFound)
		require.Len(t, works, 1)
	})

	t.Run("match by OpenAlex ID variations", func(t *testing.T) {
		// W-number only
		works, notFound, err := anaSvc.GetWorks(ctx, colID, []string{"W100"})
		require.NoError(t, err)
		assert.Empty(t, notFound)
		require.Len(t, works, 1)

		// OpenAlex URL
		works, notFound, err = anaSvc.GetWorks(ctx, colID, []string{"https://openalex.org/W100"})
		require.NoError(t, err)
		assert.Empty(t, notFound)
		require.Len(t, works, 1)

		// Prefixed W-number
		works, notFound, err = anaSvc.GetWorks(ctx, colID, []string{"openalex:W200"})
		require.NoError(t, err)
		assert.Empty(t, notFound)
		require.Len(t, works, 1)
	})

	t.Run("partial match returns found and notFound", func(t *testing.T) {
		works, notFound, err := anaSvc.GetWorks(ctx, colID, []string{"W100", "unknown-id-xyz", "vaswani2017"})
		require.NoError(t, err)
		require.Len(t, notFound, 1)
		assert.Equal(t, "unknown-id-xyz", notFound[0])
		// Note that W100 and vaswani2017 are the same work, so deduplication yields 1 work
		require.Len(t, works, 1)
	})
}

func worksMapToSlice(works ...*openalex.Work) []*openalex.Work {
	return works
}
