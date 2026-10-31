package repository_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/coreofscience/go-bibx/repository"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
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
		"works",
		"nodes",
		"links",
		"works_fts",
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
	assert.Equal(t, 3, version)
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
	assert.Equal(t, 3, version)
}

// newMigrator returns a migrator that can move db to any schema version.
func newMigrator(t *testing.T, db *sql.DB) *migrate.Migrate {
	t.Helper()
	sourceDriver, err := iofs.New(os.DirFS("migrations"), ".")
	require.NoError(t, err)
	dbDriver, err := sqlite.WithInstance(db, &sqlite.Config{MigrationsTable: "schema_migrations"})
	require.NoError(t, err)
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", dbDriver)
	require.NoError(t, err)
	return m
}

func TestMigrate_RenamesDatabaseWithoutSearchTriggers(t *testing.T) {
	db, err := repository.OpenSQLite(filepath.Join(t.TempDir(), "test_no_triggers.db"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()
	m := newMigrator(t, db)

	// Databases created by earlier versions of 000001 have no search triggers.
	require.NoError(t, m.Migrate(2))
	_, err = db.Exec(`
		DROP TRIGGER articles_ai;
		DROP TRIGGER articles_ad;
		DROP TRIGGER articles_au;
		INSERT INTO articles (id, label, title) VALUES (x'03', 'old', 'Kafka logs');
	`)
	require.NoError(t, err)

	require.NoError(t, m.Migrate(3))

	var title string
	err = db.QueryRow("SELECT title FROM works_fts WHERE works_fts MATCH 'kafka'").Scan(&title)
	require.NoError(t, err)
	assert.Equal(t, "Kafka logs", title)
}

func TestMigrate_RenamesArticlesToWorks(t *testing.T) {
	db, err := repository.OpenSQLite(filepath.Join(t.TempDir(), "test_rename.db"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()
	m := newMigrator(t, db)

	search := func(table, term string) []string {
		t.Helper()
		rows, err := db.Query("SELECT title FROM "+table+" WHERE "+table+" MATCH ? ORDER BY title", term)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, rows.Close())
		}()
		titles := []string{}
		for rows.Next() {
			var title string
			require.NoError(t, rows.Scan(&title))
			titles = append(titles, title)
		}
		require.NoError(t, rows.Err())
		return titles
	}

	require.NoError(t, m.Migrate(2))
	_, err = db.Exec(`
		INSERT INTO collections (id, query_hash, query, result_limit, article_count)
		VALUES (x'01', x'02', 'kafka', 10, 1);
		INSERT INTO articles (id, label, title, journal) VALUES (x'03', 'old', 'Kafka logs', 'NetDB');
		INSERT INTO nodes (id, collection_id, article_id, category, rootness, trunkness, leafness)
		VALUES (x'04', x'01', x'03', 'root', 1, 0, 0);
	`)
	require.NoError(t, err)

	require.NoError(t, m.Migrate(3))

	var venue string
	var workCount int
	err = db.QueryRow(`
		SELECT w.venue, c.work_count
		FROM nodes n
		JOIN works w ON w.id = n.work_id
		JOIN collections c ON c.id = n.collection_id
	`).Scan(&venue, &workCount)
	require.NoError(t, err)
	assert.Equal(t, "NetDB", venue)
	assert.Equal(t, 1, workCount)

	_, err = db.Exec("INSERT INTO works (id, label, title) VALUES (x'05', 'new', 'Kafka streams')")
	require.NoError(t, err)
	assert.Equal(t, []string{"Kafka logs", "Kafka streams"}, search("works_fts", "kafka"))

	_, err = db.Exec("UPDATE works SET title = 'Spark logs' WHERE label = 'old'")
	require.NoError(t, err)
	assert.Equal(t, []string{"Kafka streams"}, search("works_fts", "kafka"))

	_, err = db.Exec("DELETE FROM works WHERE label = 'new'")
	require.NoError(t, err)
	assert.Empty(t, search("works_fts", "kafka"))

	require.NoError(t, m.Migrate(2))

	var journal string
	var articleCount int
	err = db.QueryRow(`
		SELECT a.journal, c.article_count
		FROM nodes n
		JOIN articles a ON a.id = n.article_id
		JOIN collections c ON c.id = n.collection_id
	`).Scan(&journal, &articleCount)
	require.NoError(t, err)
	assert.Equal(t, "NetDB", journal)
	assert.Equal(t, 1, articleCount)
	assert.Equal(t, []string{"Spark logs"}, search("articles_fts", "spark"))

	var leftovers int
	err = db.QueryRow(`
		SELECT count(*) FROM sqlite_master
		WHERE name LIKE 'works%' OR name LIKE 'idx_works%'
	`).Scan(&leftovers)
	require.NoError(t, err)
	assert.Zero(t, leftovers)
}
