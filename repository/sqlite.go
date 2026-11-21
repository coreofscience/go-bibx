package repository

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite" // Pure-Go SQLite driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate executes all pending embedded migrations against the database.
func Migrate(db *sql.DB) error {
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

// OpenSQLite opens a SQLite database connection at the given path,
// ensures its parent directory exists, and configures recommended PRAGMAs
// (WAL mode, foreign keys, and busy_timeout) on every connection in the pool.
// It does NOT automatically run migrations, leaving schema management decoupled.
func OpenSQLite(path string) (*sql.DB, error) {
	cleanPath := path
	if after, ok := strings.CutPrefix(cleanPath, "file:"); ok {
		cleanPath = after
		if idx := strings.Index(cleanPath, "?"); idx != -1 {
			cleanPath = cleanPath[:idx]
		}
	}

	if cleanPath != ":memory:" && cleanPath != "" {
		dir := filepath.Dir(cleanPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create database directory: %w", err)
			}
		}
	}

	dsn := sqliteDSN(path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to sqlite database: %w", err)
	}

	return db, nil
}

type sqlitePragma struct {
	name  string
	param string
}

var defaultSQLitePragmas = []sqlitePragma{
	{name: "journal_mode", param: "journal_mode(WAL)"},
	{name: "foreign_keys", param: "foreign_keys(1)"},
	{name: "busy_timeout", param: "busy_timeout(5000)"},
}

func sqliteDSN(path string) string {
	base, rawQuery, hasQuery := strings.Cut(path, "?")
	existing := make(map[string]bool)
	if hasQuery {
		if values, err := url.ParseQuery(rawQuery); err == nil {
			for _, p := range values["_pragma"] {
				if name := pragmaName(p); name != "" {
					existing[name] = true
				}
			}
		}
	}

	var missing []string
	for _, dp := range defaultSQLitePragmas {
		if !existing[dp.name] {
			missing = append(missing, "_pragma="+dp.param)
		}
	}
	if len(missing) == 0 {
		return path
	}

	added := strings.Join(missing, "&")
	if !hasQuery || rawQuery == "" {
		return base + "?" + added
	}
	return path + "&" + added
}

// pragmaName extracts the lowercase pragma name from a modernc.org/sqlite
// _pragma value such as "busy_timeout(5000)" or "busy_timeout=5000".
func pragmaName(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.IndexAny(raw, "(="); i != -1 {
		raw = raw[:i]
	}
	return strings.ToLower(strings.TrimSpace(raw))
}

// OpenAndMigrateSQLite opens the SQLite database at the given path and runs pending migrations.
// This is a convenience wrapper for application entrypoints.
func OpenAndMigrateSQLite(path string) (*sql.DB, error) {
	db, err := OpenSQLite(path)
	if err != nil {
		return nil, err
	}

	if err := Migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	return db, nil
}
