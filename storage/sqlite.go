package storage

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite" // Pure-Go SQLite driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// runMigrations executes all pending embedded migrations against the database.
func runMigrations(db *sql.DB) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source driver: %w", err)
	}

	dbDriver, err := sqlite.WithInstance(db, &sqlite.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return fmt.Errorf("failed to create sqlite migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", dbDriver)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// SQLiteStore implements [Store] backed by a pure-Go SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or creates a SQLite database at the specified path,
// applies WAL mode and foreign key constraints, runs embedded migrations,
// and returns an initialized [*SQLiteStore].
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set sqlite pragmas: %w", err)
	}

	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// NewSQLiteStoreWithDB creates a [*SQLiteStore] wrapping an existing [*sql.DB].
// If runMigration is true, it executes pending embedded migrations.
func NewSQLiteStoreWithDB(db *sql.DB, runMigration bool) (*SQLiteStore, error) {
	if runMigration {
		if err := runMigrations(db); err != nil {
			return nil, fmt.Errorf("failed to run database migrations: %w", err)
		}
	}
	return &SQLiteStore{db: db}, nil
}

// DB returns the underlying [*sql.DB] instance.
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// Close closes the underlying SQLite database connection.
func (s *SQLiteStore) Close() error {
	if s.db == nil {
		return nil
	}
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close sqlite store: %w", err)
	}
	return nil
}
