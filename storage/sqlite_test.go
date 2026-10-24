package storage_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"uuid"

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

	// 3. Verify idempotency: opening again or wrapping with migration should succeed without error
	secondStore, err := storage.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, secondStore.Close())
}

func TestSQLiteStore_SchemaIntegrityAndFTS(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_integrity.db")

	store, err := storage.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, store.Close())
	}()

	db := store.DB()

	colID := uuid.NewV7()
	queryHash := storage.ComputeQueryHash("deep learning", 50)

	// Insert Collection
	_, err = db.Exec(`
		INSERT INTO collections (id, query_hash, query, result_limit, name, article_count)
		VALUES (?, ?, ?, ?, ?, ?)`,
		storage.UUIDToBytes(colID), queryHash[:], "deep learning", 50, "dl-collection", 1,
	)
	require.NoError(t, err)

	// Duplicate query_hash should fail due to UNIQUE constraint
	_, err = db.Exec(`
		INSERT INTO collections (id, query_hash, query, result_limit)
		VALUES (?, ?, ?, ?)`,
		storage.UUIDToBytes(uuid.NewV7()), queryHash[:], "deep learning", 50,
	)
	require.Error(t, err, "duplicate query_hash must violate unique constraint")

	// Insert Article
	artID := uuid.NewV7()
	_, err = db.Exec(`
		INSERT INTO articles (id, label, title, year, abstract, authors)
		VALUES (?, ?, ?, ?, ?, ?)`,
		storage.UUIDToBytes(artID), "openalex:W123", "Deep Learning Foundations", 2026, "A paper about neural networks", `["Goodfellow", "Bengio"]`,
	)
	require.NoError(t, err)

	// Insert FTS entry
	_, err = db.Exec(`
		INSERT INTO articles_fts (article_id, title, abstract, authors)
		VALUES (?, ?, ?, ?)`,
		storage.UUIDToBytes(artID), "Deep Learning Foundations", "A paper about neural networks", "Goodfellow Bengio",
	)
	require.NoError(t, err)

	// Query FTS
	var matchTitle string
	err = db.QueryRow("SELECT title FROM articles_fts WHERE articles_fts MATCH ?", "neural networks").Scan(&matchTitle)
	require.NoError(t, err)
	assert.Equal(t, "Deep Learning Foundations", matchTitle)

	// Insert Node
	nodeID := uuid.NewV7()
	_, err = db.Exec(`
		INSERT INTO nodes (id, collection_id, article_id, category, rootness, trunkness, leafness, community)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		storage.UUIDToBytes(nodeID), storage.UUIDToBytes(colID), storage.UUIDToBytes(artID), "root", 0.95, 0.1, 0.0, 1,
	)
	require.NoError(t, err)

	// Insert Link referencing node as both source and target (self-link or with another node)
	nodeID2 := uuid.NewV7()
	artID2 := uuid.NewV7()
	_, err = db.Exec(`
		INSERT INTO articles (id, label, title) VALUES (?, ?, ?)`,
		storage.UUIDToBytes(artID2), "openalex:W456", "Neural Networks",
	)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO nodes (id, collection_id, article_id, category, rootness, trunkness, leafness, community)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		storage.UUIDToBytes(nodeID2), storage.UUIDToBytes(colID), storage.UUIDToBytes(artID2), "leaf", 0.1, 0.2, 0.85, 1,
	)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO links (collection_id, source_id, target_id, weight)
		VALUES (?, ?, ?, ?)`,
		storage.UUIDToBytes(colID), storage.UUIDToBytes(nodeID), storage.UUIDToBytes(nodeID2), 1.0,
	)
	require.NoError(t, err)

	// Verify Cascade Delete when collection is deleted
	_, err = db.Exec("DELETE FROM collections WHERE id = ?", storage.UUIDToBytes(colID))
	require.NoError(t, err)

	var nodeCount int
	err = db.QueryRow("SELECT count(*) FROM nodes WHERE collection_id = ?", storage.UUIDToBytes(colID)).Scan(&nodeCount)
	require.NoError(t, err)
	assert.Equal(t, 0, nodeCount, "nodes should be cascade deleted with collection")

	var linkCount int
	err = db.QueryRow("SELECT count(*) FROM links WHERE collection_id = ?", storage.UUIDToBytes(colID)).Scan(&linkCount)
	require.NoError(t, err)
	assert.Equal(t, 0, linkCount, "links should be cascade deleted with collection")
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
