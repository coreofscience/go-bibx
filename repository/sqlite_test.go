package repository_test

import (
	"path/filepath"
	"testing"

	"github.com/coreofscience/go-bibx/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenSQLite_CreatesParentDirectories(t *testing.T) {
	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "deeply", "nested", ".bibx", "bibx.db")

	db, err := repository.OpenSQLite(nestedPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
	require.NoError(t, err)
	assert.Equal(t, "wal", journalMode)
}

func TestMigrate_RunsMigrations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_migrations.db")

	db, err := repository.OpenSQLite(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	err = repository.Migrate(db)
	require.NoError(t, err)

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

	var version int
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	require.NoError(t, err)
	assert.Equal(t, 2, version)
	assert.False(t, dirty)

	// Idempotency check: running Migrate again does nothing
	err = repository.Migrate(db)
	require.NoError(t, err)
}

func TestOpenAndMigrateSQLite(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_open_and_migrate.db")

	db, err := repository.OpenAndMigrateSQLite(dbPath)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	var version int
	err = db.QueryRow("SELECT version FROM schema_migrations").Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, 2, version)
}
