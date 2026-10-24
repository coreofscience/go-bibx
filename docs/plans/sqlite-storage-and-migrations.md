# Implementation Plan: SQLite Storage & Embedded Migrations

## 1. Overview & Goals

This plan outlines the implementation steps to transition `go-bibx` from single-file storage (`.bibx/analysis.json.gz`) to a multi-collection, normalized **pure-Go SQLite** database (`.bibx/bibx.db`), with embedded database migrations via `golang-migrate`, standard library UUIDv7 support (Go 1.27), and a library-first "Bring Your Own Storage" (BYOS) interface.

### Key Objectives
1. **Multi-Collection Support**: Persist and manage multiple research topics and inquiries within a single database file.
2. **Embedded Migrations**: Embed versioned SQL migration scripts into the binary using `//go:embed` and execute them automatically via `github.com/golang-migrate/migrate/v4`.
3. **CGO-Free Portability**: Ensure all drivers (`modernc.org/sqlite`) respect `CGO_ENABLED=0` to comply with the GoReleaser matrix across Linux, macOS, and Windows.
4. **Time-Ordered UUIDv7**: Use Go 1.27 standard library `uuid.NewV7()` stored as 16-byte `BLOB(16)` for all primary and foreign keys.
5. **Query Idempotency**: Fingerprint inquiries with a normalized canonical hash (`SHA-256`) to skip redundant API requests and graph processing when the same query and settings are re-run.
6. **Library-First BYOS**: Define a clean `storage.Store` interface and co-located `storage.SQLiteStore` allowing developers to bring any storage backend or custom `*sql.DB` when using `go-bibx` as a library.

---

## 2. Target File Structure

```
go-bibx/
├── cmd/bibx/main.go
├── storage/
│   ├── store.go                      # Store interface, domain entities, query hasher
│   ├── sqlite.go                     # SQLiteStore implementation & constructor
│   ├── sqlite_test.go                # Integration tests (WAL, migrations, CRUD, FTS)
│   └── migrations/
│       ├── 000001_init_schema.up.sql
│       └── 000001_init_schema.down.sql
├── internal/
│   └── utils/
│       └── context.go                # DB path resolution & context helpers
├── cli/
│   ├── services/
│   │   ├── analysis.go               # Updated to use storage.Store & idempotency check
│   │   └── analysis_test.go
│   └── commands/
│       ├── commands.go
│       ├── inquire/inquire.go        # Supports idempotency, --db, and --force
│       ├── collection/               # New 'collection' subcommands
│       │   ├── collection.go
│       │   ├── list.go
│       │   └── delete.go
│       ├── query/query.go            # Uses storage.Store.QueryCategory
│       ├── search/search.go          # Uses storage.Store.Search (FTS5)
│       └── view/view.go              # Loads collection from SQLiteStore
└── docs/plans/
    └── sqlite-storage-and-migrations.md
```

---

## 3. Database Schema (`000001_init_schema.up.sql`)

```sql
-- Collections Table
CREATE TABLE collections (
    id            BLOB(16) PRIMARY KEY,      -- uuid.NewV7() binary
    query_hash    BLOB(32) UNIQUE NOT NULL,  -- SHA-256 idempotency key
    query         TEXT NOT NULL,             -- Normalized search query
    result_limit  INTEGER NOT NULL,          -- Requested result limit
    parameters    JSON,                      -- Additional algorithm / API parameters
    name          TEXT,                      -- Optional human alias (e.g., 'quantum-2026')
    article_count INTEGER DEFAULT 0,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_collections_created_at ON collections(created_at DESC);
CREATE INDEX idx_collections_name ON collections(name);

-- Articles Table (Normalized Bibliographic Data)
CREATE TABLE articles (
    id          BLOB(16) PRIMARY KEY,        -- uuid.NewV7() binary
    label       TEXT UNIQUE NOT NULL,        -- Natural key (e.g., 'openalex:W...')
    title       TEXT,
    journal     TEXT,
    year        INTEGER,
    volume      TEXT,
    issue       TEXT,
    page        TEXT,
    doi         TEXT,
    permalink   TEXT,
    times_cited INTEGER,
    abstract    TEXT,
    rich        BOOLEAN DEFAULT 0,
    authors     JSON,                        -- JSON array of author strings
    keywords    JSON,                        -- JSON array of keyword strings
    ids         JSON,                        -- JSON object of source identifiers
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_articles_label ON articles(label);
CREATE INDEX idx_articles_doi ON articles(doi);
CREATE INDEX idx_articles_year ON articles(year);

-- Scored Nodes Table
CREATE TABLE nodes (
    id            BLOB(16) PRIMARY KEY,      -- uuid.NewV7() binary
    collection_id BLOB(16) NOT NULL,
    article_id    BLOB(16) NOT NULL,
    category      TEXT NOT NULL,             -- 'root', 'trunk', 'leaf'
    rootness      REAL NOT NULL,
    trunkness     REAL NOT NULL,
    leafness      REAL NOT NULL,
    community     INTEGER NOT NULL,
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
    UNIQUE(collection_id, article_id)
);

CREATE INDEX idx_nodes_collection_category ON nodes(collection_id, category);
CREATE INDEX idx_nodes_collection_rootness ON nodes(collection_id, rootness DESC);
CREATE INDEX idx_nodes_collection_trunkness ON nodes(collection_id, trunkness DESC);
CREATE INDEX idx_nodes_collection_leafness ON nodes(collection_id, leafness DESC);

-- Citation Graph Links Table
CREATE TABLE links (
    collection_id BLOB(16) NOT NULL,
    source_id     BLOB(16) NOT NULL,         -- References nodes(id)
    target_id     BLOB(16) NOT NULL,         -- References nodes(id)
    weight        REAL DEFAULT 1.0,
    PRIMARY KEY (collection_id, source_id, target_id),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
    FOREIGN KEY (source_id) REFERENCES nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (target_id) REFERENCES nodes(id) ON DELETE CASCADE
);

CREATE INDEX idx_links_collection ON links(collection_id);

-- FTS5 Full-Text Search Virtual Table
CREATE VIRTUAL TABLE articles_fts USING fts5(
    article_id UNINDEXED,                    -- Join back to articles(id)
    title,
    abstract,
    keywords,
    authors,
    tokenize = 'porter unicode61'
);
```

---

## 4. Phased Implementation Steps

### Phase 1: Toolchain Upgrades & Dependency Management
- [x] Upgrade `go.mod` to Go 1.27 (`go 1.27.0`).
- [x] Add `github.com/golang-migrate/migrate/v4` with components:
  - `github.com/golang-migrate/migrate/v4/source/iofs`
  - `github.com/golang-migrate/migrate/v4/database/sqlite`
- [x] Add `modernc.org/sqlite` pure-Go driver.
- [x] Run `go mod tidy` and verify `CGO_ENABLED=0 go test ./...` builds cleanly.

### Phase 2: Core Storage Interface & Utilities (`storage`)
- [x] Create `storage/store.go`:
  - Define `Store` interface (`FindCollectionByHash`, `FindCollectionByID`, `FindCollectionByName`, `ListCollections`, `CreateCollection`, `DeleteCollection`, `StoreAnalysis`, `LoadAnalysis`, `QueryCategory`, `Search`, `Close`).
  - Define `Collection` struct with `ID uuid.UUID`, `QueryHash [32]byte`, `Query`, `ResultLimit`, etc.
  - Implement `NormalizeQuery(q string) string` (lowercases, trims, and replaces `\s+` with a single space `" "`).
  - Implement `ComputeQueryHash(query string, limit int) [32]byte`.
  - Implement UUID helper functions for scanning and serializing Go 1.27 `uuid.UUID` to/from SQLite `BLOB(16)`.
- [x] Add unit tests in `storage/store_test.go` verifying query normalization and hash determinism.

### Phase 3: Embedded Migrations & Schema Initialization
- [x] Create `storage/migrations/000001_init_schema.up.sql` with full schema.
- [x] Create `storage/migrations/000001_init_schema.down.sql` with cleanup drops.
- [x] In `storage/sqlite.go`, embed migrations using:
  ```go
  //go:embed migrations/*.sql
  var migrationsFS embed.FS
  ```
- [x] Implement `runMigrations(db *sql.DB) error` utilizing `iofs.New` and `sqlite.WithInstance`.

### Phase 4: `SQLiteStore` Implementation & Testing
- [x] Create `SQLiteStore` struct wrapping `*sql.DB`.
- [x] Implement `NewSQLiteStore(path string) (*SQLiteStore, error)`:
  - Open database via `sql.Open("sqlite", path)`.
  - Set pragmas: `PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`.
  - Execute `runMigrations(db)`.
- [x] Implement `NewSQLiteStoreWithDB(db *sql.DB, runMigration bool) (*SQLiteStore, error)`.
- [x] Implement `CollectionStore` methods on `SQLiteStore`:
  - `FindCollectionByHash`, `FindCollectionByID`, `FindCollectionByName`, `ListCollections`, `CreateCollection`, `DeleteCollection`.
- [x] Implement `AnalysisStore` methods on `SQLiteStore`:
  - `StoreAnalysis`: Inserts/updates `articles` (deduplicated by `label`), inserts `nodes`, `links`, and populates `articles_fts`.
  - `LoadAnalysis`: Reconstructs `models.Analysis` from `nodes`, `articles`, and `links` for a given `collection_id`.
  - `QueryCategory`: Direct indexed SQL query (`SELECT ... FROM nodes JOIN articles ... ORDER BY ... LIMIT ?`).
  - `Search`: SQL query using `articles_fts MATCH ?` with `bm25(articles_fts)`.
- [x] Write integration test suite `storage/sqlite_test.go`:
  - Test migration execution and schema creation in an in-memory/temp SQLite file.
  - Test query hash lookup and collection idempotency.
  - Test storing and re-loading an analysis graph.
  - Test category queries and FTS5 search ranking.

### Phase 5: Service Refactoring & Idempotency Integration
- [x] Update `cli/services/analysis.go`:
  - Replace `repos.AnalysisRepo` with `storage.Store`.
  - Update `OpenAlexAnalysisService.Store(ctx, query, limit, force)`:
    1. Compute `queryHash = storage.ComputeQueryHash(query, limit)`.
    2. Check `existing, err := s.store.FindCollectionByHash(ctx, queryHash)`.
    3. If found and `!force`, return early with existing collection reference (skipping OpenAlex API calls and SAP/Leiden computation).
    4. If not found or `force`, fetch works, compute algorithms, create `Collection` record with new `uuid.NewV7()`, and call `s.store.StoreAnalysis(...)`.
  - Update `Query` to delegate to `s.store.QueryCategory`.
  - Update `Search` to leverage `s.store.Search`.
- [x] Update `cli/services/analysis_test.go` with mock store verifying idempotency behavior.

### Phase 6: CLI Commands & UX Expansion
- [ ] Update `cmd/bibx/main.go` and `internal/utils/context.go`:
  - Add global `--db` flag (default: `.bibx/bibx.db`).
  - Update context helpers to resolve active database path.
- [ ] Update `cli/commands/inquire/inquire.go`:
  - Support `--force` to override idempotency.
  - Display helpful message when skipping existing queries (`Collection already exists... Use --force to re-run`).
- [ ] Add `cli/commands/collection/`:
  - `bibx collection list`: Print tabular list of collections (UUIDv7, name, article count, query, created_at).
  - `bibx collection show <id|name>`: Display collection summary.
  - `bibx collection delete <id|name>`: Delete collection and cascade remove nodes/links.
- [ ] Update `cli/commands/query/query.go` & `search/search.go`:
  - Add optional `--collection` flag (defaults to latest collection if omitted).
- [ ] Update `cli/commands/view/view.go`:
  - Load requested (or latest) collection from `SQLiteStore` and pass to viewer server.

### Phase 7: Backward Compatibility & Migration Utility
- [ ] Detect legacy `.bibx/analysis.json.gz`:
  - If `.bibx/bibx.db` does not exist and `.bibx/analysis.json.gz` exists, automatically offer or perform migration into collection `legacy`.
- [ ] Add explicit command: `bibx storage migrate` to import legacy gzip files.
- [ ] Verify test suite and run static checks:
  - `go test -v ./...`
  - `golangci-lint run`
  - `CGO_ENABLED=0 go build -o /tmp/bibx ./cmd/bibx`

---

## 5. Verification & Acceptance Criteria

1. **Compilation**: `CGO_ENABLED=0 go build ./cmd/bibx` succeeds on Linux, macOS, and Windows.
2. **Migration Idempotency**: Starting the application against a clean directory initializes `.bibx/bibx.db` and runs `000001_init_schema.up.sql` with `schema_migrations` tracking version 1.
3. **Query Idempotency**:
   - `bibx inquire "quantum machine learning" --limit 50` fetches OpenAlex and populates database.
   - `bibx inquire "  quantum   machine   learning  " --limit 50` detects identical `query_hash` and finishes without network calls.
   - `bibx inquire "quantum machine learning" --limit 50 --force` re-fetches and updates the collection.
4. **Performance & Granularity**: `bibx query --category root --limit 5` executes directly against SQL indices without loading all collection nodes into memory.
5. **BYOS Library Swappability**: Developers using `go-bibx` as a module can implement `storage.Store` or pass a custom `*sql.DB` to `storage.NewSQLiteStoreWithDB`.
