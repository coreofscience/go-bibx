package storage_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"uuid"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSQLiteStore_Migrations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_bibx.db")

	store, err := storage.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, store.Close())
	}()

	db := store.DB()
	require.NotNil(t, db)

	// 1. Verify schema tables were created
	expectedTables := []string{
		"collections",
		"articles",
		"nodes",
		"links",
		"articles_fts",
		"schema_migrations",
	}

	for _, table := range expectedTables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		require.NoError(t, err, "table %s should exist", table)
		assert.Equal(t, table, name)
	}

	// 2. Verify schema_migrations status
	var version int
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	require.NoError(t, err)
	assert.Equal(t, 1, version)
	assert.False(t, dirty)

	// 3. Verify idempotency
	secondStore, err := storage.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, secondStore.Close())
}

func TestSQLiteStore_CollectionCRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_collection_crud.db")

	store, err := storage.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, store.Close())
	}()

	ctx := context.Background()

	name1 := "quantum-computing"
	hash1 := storage.ComputeQueryHash("quantum computing", 100)
	col1 := &storage.Collection{
		ID:          uuid.NewV7(),
		QueryHash:   hash1,
		Query:       "quantum computing",
		ResultLimit: 100,
		Name:        &name1,
	}

	// 1. Create collection
	err = store.CreateCollection(ctx, col1)
	require.NoError(t, err)

	// 2. Create collection with duplicate hash fails
	err = store.CreateCollection(ctx, &storage.Collection{
		ID:          uuid.NewV7(),
		QueryHash:   hash1,
		Query:       "quantum computing",
		ResultLimit: 100,
	})
	require.ErrorIs(t, err, storage.ErrCollectionExists)

	// 3. Create nil collection fails
	err = store.CreateCollection(ctx, nil)
	require.Error(t, err)

	// 4. Find by hash
	foundHash, err := store.FindCollectionByHash(ctx, hash1)
	require.NoError(t, err)
	assert.Equal(t, col1.ID, foundHash.ID)
	assert.Equal(t, col1.Query, foundHash.Query)
	assert.Equal(t, name1, *foundHash.Name)

	// 5. Find by non-existent hash
	_, err = store.FindCollectionByHash(ctx, storage.ComputeQueryHash("non-existent", 10))
	require.ErrorIs(t, err, storage.ErrCollectionNotFound)

	// 6. Find by ID
	foundID, err := store.FindCollectionByID(ctx, col1.ID)
	require.NoError(t, err)
	assert.Equal(t, col1.ID, foundID.ID)

	// 7. Find by non-existent ID
	_, err = store.FindCollectionByID(ctx, uuid.NewV7())
	require.ErrorIs(t, err, storage.ErrCollectionNotFound)

	// 8. Find by name
	foundName, err := store.FindCollectionByName(ctx, name1)
	require.NoError(t, err)
	assert.Equal(t, col1.ID, foundName.ID)

	// 9. Find by non-existent name
	_, err = store.FindCollectionByName(ctx, "does-not-exist")
	require.ErrorIs(t, err, storage.ErrCollectionNotFound)

	// 10. List collections
	hash2 := storage.ComputeQueryHash("deep learning", 50)
	col2 := &storage.Collection{
		ID:          uuid.NewV7(),
		QueryHash:   hash2,
		Query:       "deep learning",
		ResultLimit: 50,
	}
	require.NoError(t, store.CreateCollection(ctx, col2))

	list, err := store.ListCollections(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, col2.ID, list[0].ID) // Ordered by created_at DESC

	// 11. Delete collection
	err = store.DeleteCollection(ctx, col1.ID)
	require.NoError(t, err)

	_, err = store.FindCollectionByID(ctx, col1.ID)
	require.ErrorIs(t, err, storage.ErrCollectionNotFound)

	// 12. Delete non-existent collection
	err = store.DeleteCollection(ctx, col1.ID)
	require.ErrorIs(t, err, storage.ErrCollectionNotFound)
}

func TestSQLiteStore_AnalysisPersistenceAndQuery(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_analysis.db")

	store, err := storage.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, store.Close())
	}()

	ctx := context.Background()

	// 1. Create a collection first
	colID := uuid.NewV7()
	hash := storage.ComputeQueryHash("graph algorithms", 10)
	require.NoError(t, store.CreateCollection(ctx, &storage.Collection{
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
		Community: 1,
		Article: &models.Article{
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
		Community: 1,
		Article: &models.Article{
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
		Community: 1,
		Article: &models.Article{
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
	err = store.StoreAnalysis(ctx, uuid.NewV7(), analysis)
	require.ErrorIs(t, err, storage.ErrCollectionNotFound)

	// 4. StoreAnalysis: nil analysis fails
	err = store.StoreAnalysis(ctx, colID, nil)
	require.Error(t, err)

	// 5. StoreAnalysis: successful storage
	err = store.StoreAnalysis(ctx, colID, analysis)
	require.NoError(t, err)

	// Verify article count updated on collection
	col, err := store.FindCollectionByID(ctx, colID)
	require.NoError(t, err)
	assert.Equal(t, 3, col.ArticleCount)

	// 6. LoadAnalysis: successful reconstruction
	loaded, err := store.LoadAnalysis(ctx, colID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.Len(t, loaded.Nodes, 3)
	require.Len(t, loaded.Links, 2)

	// Verify nodes content
	nodeMap := make(map[string]*models.Node)
	for _, n := range loaded.Nodes {
		nodeMap[n.ID] = n
	}

	loadedW1 := nodeMap["openalex:W1"]
	require.NotNil(t, loadedW1)
	assert.Equal(t, models.CategoryRoot, loadedW1.Category)
	assert.InDelta(t, 0.95, loadedW1.Rootness, 0.001)
	assert.Equal(t, "Seminal Graph Paper", *loadedW1.Article.Title)
	assert.Equal(t, 2024, *loadedW1.Article.Year)
	assert.Equal(t, "10.1000/1", *loadedW1.Article.DOI)
	assert.ElementsMatch(t, []string{"Turing, Alan", "Shannon, Claude"}, loadedW1.Article.Authors)
	assert.True(t, loadedW1.Article.Keywords.Contains("foundations"))
	assert.True(t, loadedW1.Article.IDs.Contains("openalex:W1"))

	// Verify links
	assert.Equal(t, "openalex:W2", loaded.Links[0].Source)
	assert.Equal(t, "openalex:W1", loaded.Links[0].Target)
	assert.Equal(t, 1.0, loaded.Links[0].Weight)

	// 7. LoadAnalysis: non-existent collection fails
	_, err = store.LoadAnalysis(ctx, uuid.NewV7())
	require.ErrorIs(t, err, storage.ErrCollectionNotFound)

	// 8. QueryCategory: Root
	rootResults, err := store.QueryCategory(ctx, colID, "root", 5)
	require.NoError(t, err)
	require.Len(t, rootResults, 2) // W1 (0.95) and W2 (0.1) have rootness > 0
	assert.Equal(t, "Seminal Graph Paper", *rootResults[0].Article.Title)
	assert.InDelta(t, 0.95, rootResults[0].Score, 0.001)

	// QueryCategory: Leaf
	leafResults, err := store.QueryCategory(ctx, colID, "leaf", 1)
	require.NoError(t, err)
	require.Len(t, leafResults, 1)
	assert.Equal(t, "Frontier Graph Neural Network", *leafResults[0].Article.Title)
	assert.InDelta(t, 0.92, leafResults[0].Score, 0.001)

	// QueryCategory with limit <= 0
	emptyResults, err := store.QueryCategory(ctx, colID, "root", 0)
	require.NoError(t, err)
	assert.Nil(t, emptyResults)

	// 9. Search via FTS5
	searchResults, err := store.Search(ctx, colID, "neural networks", 10)
	require.NoError(t, err)
	require.Len(t, searchResults, 1)
	assert.Equal(t, "openalex:W3", searchResults[0].ID)
	assert.Equal(t, "Frontier Graph Neural Network", *searchResults[0].Article.Title)

	// Search matching author
	authorResults, err := store.Search(ctx, colID, "Turing", 10)
	require.NoError(t, err)
	require.Len(t, authorResults, 1)
	assert.Equal(t, "openalex:W1", authorResults[0].ID)

	// Search matching keyword
	kwResults, err := store.Search(ctx, colID, "bridges", 10)
	require.NoError(t, err)
	require.Len(t, kwResults, 1)
	assert.Equal(t, "openalex:W2", kwResults[0].ID)

	// Search with special characters (safe quoting)
	safeResults, err := store.Search(ctx, colID, "neural-networks*", 10)
	require.NoError(t, err)
	assert.NotNil(t, safeResults)

	// Search with limit <= 0 or empty query
	assert.Nil(t, mustSearch(t, store, colID, "", 10))
	assert.Nil(t, mustSearch(t, store, colID, "neural", 0))

	// 10. Verify DeleteCollection cascades and removes nodes/links
	require.NoError(t, store.DeleteCollection(ctx, colID))
	_, err = store.LoadAnalysis(ctx, colID)
	require.ErrorIs(t, err, storage.ErrCollectionNotFound)
}

func mustSearch(t *testing.T, store storage.Store, colID uuid.UUID, query string, limit int) []*models.Node {
	t.Helper()
	res, err := store.Search(context.Background(), colID, query, limit)
	require.NoError(t, err)
	return res
}

func TestNewSQLiteStoreWithDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "custom_db.db")

	rawDB, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rawDB.Close())
	}()

	store, err := storage.NewSQLiteStoreWithDB(rawDB, true)
	require.NoError(t, err)
	require.NotNil(t, store)

	var version int
	err = rawDB.QueryRow("SELECT version FROM schema_migrations").Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, 1, version)
}
