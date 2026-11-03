// Package test provides helpers for tests that need a real,
// migrated SQLite database.
package test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/coreofscience/go-bibx/repository"
)

var (
	templateOnce sync.Once
	template     []byte
	templateErr  error
)

// migratedTemplate migrates a database once per test binary and returns its
// contents. Running the migrations is the slow part of opening a test
// database, especially under the race detector, so tests copy these bytes
// instead.
func migratedTemplate() ([]byte, error) {
	templateOnce.Do(func() {
		template, templateErr = buildTemplate()
	})
	return template, templateErr
}

func buildTemplate() ([]byte, error) {
	dir, err := os.MkdirTemp("", "bibx-template-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create template directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	path := filepath.Join(dir, "template.db")
	db, err := repository.OpenAndMigrateSQLite(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open template database: %w", err)
	}
	// Closing checkpoints the WAL into the main file, so reading it alone is enough.
	if err := db.Close(); err != nil {
		return nil, fmt.Errorf("failed to close template database: %w", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template database: %w", err)
	}
	return contents, nil
}

// NewDBPath returns the path to a fresh, fully migrated database file inside
// t.TempDir().
func NewDBPath(t testing.TB) string {
	t.Helper()
	contents, err := migratedTemplate()
	if err != nil {
		t.Fatalf("failed to build template database: %v", err)
	}
	path := filepath.Join(t.TempDir(), "test.db")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("failed to write test database: %v", err)
	}
	return path
}

// NewDB opens a fresh, fully migrated database that is closed when the test
// finishes.
func NewDB(t testing.TB) *sql.DB {
	t.Helper()
	db, err := repository.OpenSQLite(NewDBPath(t))
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close test database: %v", err)
		}
	})
	return db
}
