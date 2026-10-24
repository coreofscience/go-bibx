package storage

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
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

// FindCollectionByHash finds a collection matching the given query hash.
func (s *SQLiteStore) FindCollectionByHash(ctx context.Context, hash [32]byte) (*Collection, error) {
	query := `
		SELECT id, query_hash, query, result_limit, parameters, name, article_count, created_at, updated_at
		FROM collections
		WHERE query_hash = ?`
	row := s.db.QueryRowContext(ctx, query, hash[:])
	return scanCollection(row)
}

// FindCollectionByID finds a collection by its UUID.
func (s *SQLiteStore) FindCollectionByID(ctx context.Context, id uuid.UUID) (*Collection, error) {
	query := `
		SELECT id, query_hash, query, result_limit, parameters, name, article_count, created_at, updated_at
		FROM collections
		WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, UUIDToBytes(id))
	return scanCollection(row)
}

// FindCollectionByName finds a collection by its human-friendly name alias.
func (s *SQLiteStore) FindCollectionByName(ctx context.Context, name string) (*Collection, error) {
	query := `
		SELECT id, query_hash, query, result_limit, parameters, name, article_count, created_at, updated_at
		FROM collections
		WHERE name = ?`
	row := s.db.QueryRowContext(ctx, query, name)
	return scanCollection(row)
}

// ListCollections returns all collections ordered by creation timestamp descending.
func (s *SQLiteStore) ListCollections(ctx context.Context) ([]*Collection, error) {
	query := `
		SELECT id, query_hash, query, result_limit, parameters, name, article_count, created_at, updated_at
		FROM collections
		ORDER BY created_at DESC, id DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var cols []*Collection
	for rows.Next() {
		col, err := scanCollectionRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan collection: %w", err)
		}
		cols = append(cols, col)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during collection rows iteration: %w", err)
	}
	return cols, nil
}

// CreateCollection inserts a new collection record into the database.
func (s *SQLiteStore) CreateCollection(ctx context.Context, col *Collection) error {
	if col == nil {
		return errors.New("cannot create nil collection")
	}
	if col.ID == uuid.Nil() {
		col.ID = uuid.NewV7()
	}

	query := `
		INSERT INTO collections (id, query_hash, query, result_limit, parameters, name, article_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query,
		UUIDToBytes(col.ID),
		col.QueryHash[:],
		col.Query,
		col.ResultLimit,
		col.Parameters,
		col.Name,
		col.ArticleCount,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrCollectionExists
		}
		return fmt.Errorf("failed to create collection: %w", err)
	}
	return nil
}

// DeleteCollection removes a collection and cascades to its nodes and links.
func (s *SQLiteStore) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM collections WHERE id = ?"
	res, err := s.db.ExecContext(ctx, query, UUIDToBytes(id))
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return ErrCollectionNotFound
	}
	return nil
}

// StoreAnalysis persists an analysis graph (articles, scored nodes, links, and FTS5 entries) for a collection.
func (s *SQLiteStore) StoreAnalysis(ctx context.Context, collectionID uuid.UUID, a *models.Analysis) error {
	if a == nil {
		return errors.New("cannot store nil analysis")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 1. Verify collection exists
	var count int
	err = tx.QueryRowContext(ctx, "SELECT count(*) FROM collections WHERE id = ?", UUIDToBytes(collectionID)).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify collection existence: %w", err)
	}
	if count == 0 {
		return ErrCollectionNotFound
	}

	// Prepare statements for performance
	articleStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO articles (id, label, title, journal, year, volume, issue, page, doi, permalink, times_cited, abstract, rich, authors, keywords, ids)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(label) DO UPDATE SET
			title=COALESCE(excluded.title, articles.title),
			journal=COALESCE(excluded.journal, articles.journal),
			year=COALESCE(excluded.year, articles.year),
			volume=COALESCE(excluded.volume, articles.volume),
			issue=COALESCE(excluded.issue, articles.issue),
			page=COALESCE(excluded.page, articles.page),
			doi=COALESCE(excluded.doi, articles.doi),
			permalink=COALESCE(excluded.permalink, articles.permalink),
			times_cited=COALESCE(excluded.times_cited, articles.times_cited),
			abstract=COALESCE(excluded.abstract, articles.abstract),
			rich=excluded.rich OR articles.rich,
			authors=COALESCE(excluded.authors, articles.authors),
			keywords=COALESCE(excluded.keywords, articles.keywords),
			ids=COALESCE(excluded.ids, articles.ids)
		RETURNING id`)
	if err != nil {
		return fmt.Errorf("failed to prepare article statement: %w", err)
	}
	defer func() {
		_ = articleStmt.Close()
	}()

	delFTSStmt, err := tx.PrepareContext(ctx, "DELETE FROM articles_fts WHERE article_id = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare fts delete statement: %w", err)
	}
	defer func() {
		_ = delFTSStmt.Close()
	}()

	insertFTSStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO articles_fts (article_id, title, abstract, keywords, authors)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare fts insert statement: %w", err)
	}
	defer func() {
		_ = insertFTSStmt.Close()
	}()

	nodeStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO nodes (id, collection_id, article_id, category, rootness, trunkness, leafness, community)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(collection_id, article_id) DO UPDATE SET
			category=excluded.category,
			rootness=excluded.rootness,
			trunkness=excluded.trunkness,
			leafness=excluded.leafness,
			community=excluded.community
		RETURNING id`)
	if err != nil {
		return fmt.Errorf("failed to prepare node statement: %w", err)
	}
	defer func() {
		_ = nodeStmt.Close()
	}()

	// Map node string identifiers to their database UUIDs for linking
	nodeIDToUUID := make(map[string]uuid.UUID, len(a.Nodes))

	// 2. Insert/update Articles & Nodes
	for _, node := range a.Nodes {
		resolvedNodeID, err := persistNode(ctx, collectionID, node, articleStmt, delFTSStmt, insertFTSStmt, nodeStmt)
		if err != nil {
			return err
		}

		nodeIDToUUID[node.ID] = resolvedNodeID
		if node.Article != nil && node.Article.Label != "" {
			nodeIDToUUID[node.Article.Label] = resolvedNodeID
		}
		if node.Article != nil {
			if key := node.Article.Key(); key != nil {
				nodeIDToUUID[*key] = resolvedNodeID
			}
		}
	}

	// 3. Insert Links
	if err := persistLinks(ctx, collectionID, a.Links, nodeIDToUUID, tx); err != nil {
		return err
	}

	// 4. Update collection article count
	_, err = tx.ExecContext(ctx, `
		UPDATE collections
		SET article_count = (SELECT count(*) FROM nodes WHERE collection_id = ?),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		UUIDToBytes(collectionID), UUIDToBytes(collectionID),
	)
	if err != nil {
		return fmt.Errorf("failed to update collection metadata: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit analysis transaction: %w", err)
	}

	return nil
}

func persistNode(
	ctx context.Context,
	collectionID uuid.UUID,
	node *models.Node,
	articleStmt, delFTSStmt, insertFTSStmt, nodeStmt *sql.Stmt,
) (uuid.UUID, error) {
	art := node.Article
	if art == nil {
		art = &models.Article{
			Label: node.ID,
		}
	}
	if art.Label == "" {
		if key := art.Key(); key != nil {
			art.Label = *key
		} else {
			art.Label = node.ID
		}
	}

	authorsJSON, _ := json.Marshal(art.Authors)
	var keywordsJSON, idsJSON []byte
	if art.Keywords != nil {
		keywordsJSON, _ = json.Marshal(art.Keywords.Items())
	}
	if art.IDs != nil {
		idsJSON, _ = json.Marshal(art.IDs.Items())
	}

	newArtID := uuid.NewV7()
	var resolvedArtIDBytes []byte
	err := articleStmt.QueryRowContext(ctx,
		UUIDToBytes(newArtID),
		art.Label,
		art.Title,
		art.Journal,
		art.Year,
		art.Volume,
		art.Issue,
		art.Page,
		art.DOI,
		art.Permalink,
		art.TimesCited,
		art.Abstract,
		art.Rich,
		authorsJSON,
		keywordsJSON,
		idsJSON,
	).Scan(&resolvedArtIDBytes)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("failed to persist article %s: %w", art.Label, err)
	}

	resolvedArtID, err := ScanUUID(resolvedArtIDBytes)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("failed to parse article id: %w", err)
	}

	if _, err := delFTSStmt.ExecContext(ctx, UUIDToBytes(resolvedArtID)); err != nil {
		return uuid.Nil(), fmt.Errorf("failed to clean fts for article %s: %w", art.Label, err)
	}

	titleStr := ""
	if art.Title != nil {
		titleStr = *art.Title
	}
	abstractStr := ""
	if art.Abstract != nil {
		abstractStr = *art.Abstract
	}
	keywordsStr := ""
	if art.Keywords != nil {
		keywordsStr = strings.Join(art.Keywords.Items(), " ")
	}
	authorsStr := strings.Join(art.Authors, " ")

	if _, err := insertFTSStmt.ExecContext(ctx, UUIDToBytes(resolvedArtID), titleStr, abstractStr, keywordsStr, authorsStr); err != nil {
		return uuid.Nil(), fmt.Errorf("failed to index fts for article %s: %w", art.Label, err)
	}

	newNodeID := uuid.NewV7()
	var resolvedNodeIDBytes []byte
	err = nodeStmt.QueryRowContext(ctx,
		UUIDToBytes(newNodeID),
		UUIDToBytes(collectionID),
		UUIDToBytes(resolvedArtID),
		string(node.Category),
		node.Rootness,
		node.Trunkness,
		node.Leafness,
		node.Community,
	).Scan(&resolvedNodeIDBytes)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("failed to persist node %s: %w", node.ID, err)
	}

	resolvedNodeID, err := ScanUUID(resolvedNodeIDBytes)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("failed to parse node id: %w", err)
	}

	return resolvedNodeID, nil
}

func persistLinks(
	ctx context.Context,
	collectionID uuid.UUID,
	links []*models.Link,
	nodeIDToUUID map[string]uuid.UUID,
	tx *sql.Tx,
) error {
	if len(links) == 0 {
		return nil
	}

	linkStmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO links (collection_id, source_id, target_id, weight)
		VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare link statement: %w", err)
	}
	defer func() {
		_ = linkStmt.Close()
	}()

	for _, link := range links {
		sourceUUID, okSource := nodeIDToUUID[link.Source]
		targetUUID, okTarget := nodeIDToUUID[link.Target]
		if !okSource || !okTarget {
			continue
		}

		weight := link.Weight
		if weight == 0 {
			weight = 1.0
		}

		if _, err := linkStmt.ExecContext(ctx, UUIDToBytes(collectionID), UUIDToBytes(sourceUUID), UUIDToBytes(targetUUID), weight); err != nil {
			return fmt.Errorf("failed to insert link: %w", err)
		}
	}

	return nil
}

// LoadAnalysis reconstructs the entire models.Analysis graph from the database for a collection.
func (s *SQLiteStore) LoadAnalysis(ctx context.Context, collectionID uuid.UUID) (*models.Analysis, error) {
	// Verify collection exists
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM collections WHERE id = ?", UUIDToBytes(collectionID)).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to verify collection existence: %w", err)
	}
	if count == 0 {
		return nil, ErrCollectionNotFound
	}

	// Load nodes joined with articles
	nodesQuery := `
		SELECT
			n.id, n.category, n.rootness, n.trunkness, n.leafness, n.community,
			a.id, a.label, a.title, a.journal, a.year, a.volume, a.issue, a.page,
			a.doi, a.permalink, a.times_cited, a.abstract, a.rich,
			a.authors, a.keywords, a.ids
		FROM nodes n
		JOIN articles a ON a.id = n.article_id
		WHERE n.collection_id = ?`

	rows, err := s.db.QueryContext(ctx, nodesQuery, UUIDToBytes(collectionID))
	if err != nil {
		return nil, fmt.Errorf("failed to query collection nodes: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var nodes []*models.Node
	nodeUUIDToLabel := make(map[string]string)

	for rows.Next() {
		var (
			nodeIDBytes []byte
			category    string
			rootness    float64
			trunkness   float64
			leafness    float64
			community   int
			artIDBytes  []byte
			label       string
			title       sql.NullString
			journal     sql.NullString
			year        sql.NullInt64
			volume      sql.NullString
			issue       sql.NullString
			page        sql.NullString
			doi         sql.NullString
			permalink   sql.NullString
			timesCited  sql.NullInt64
			abstract    sql.NullString
			rich        bool
			authorsRaw  []byte
			keywordsRaw []byte
			idsRaw      []byte
		)

		err := rows.Scan(
			&nodeIDBytes, &category, &rootness, &trunkness, &leafness, &community,
			&artIDBytes, &label, &title, &journal, &year, &volume, &issue, &page,
			&doi, &permalink, &timesCited, &abstract, &rich,
			&authorsRaw, &keywordsRaw, &idsRaw,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node row: %w", err)
		}

		art := buildArticle(label, title, journal, year, volume, issue, page, doi, permalink, timesCited, abstract, rich, authorsRaw, keywordsRaw, idsRaw)
		nodeID := label
		if key := art.Key(); key != nil {
			nodeID = *key
		}

		node := &models.Node{
			ID:        nodeID,
			Category:  models.Category(category),
			Rootness:  rootness,
			Trunkness: trunkness,
			Leafness:  leafness,
			Community: community,
			Article:   art,
		}
		nodes = append(nodes, node)
		nodeUUIDToLabel[string(nodeIDBytes)] = nodeID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during node rows iteration: %w", err)
	}

	// Load links
	linksQuery := `
		SELECT source_id, target_id, weight
		FROM links
		WHERE collection_id = ?`

	linkRows, err := s.db.QueryContext(ctx, linksQuery, UUIDToBytes(collectionID))
	if err != nil {
		return nil, fmt.Errorf("failed to query collection links: %w", err)
	}
	defer func() {
		_ = linkRows.Close()
	}()

	var links []*models.Link
	for linkRows.Next() {
		var (
			sourceBytes []byte
			targetBytes []byte
			weight      float64
		)
		if err := linkRows.Scan(&sourceBytes, &targetBytes, &weight); err != nil {
			return nil, fmt.Errorf("failed to scan link row: %w", err)
		}
		sourceID, ok1 := nodeUUIDToLabel[string(sourceBytes)]
		targetID, ok2 := nodeUUIDToLabel[string(targetBytes)]
		if ok1 && ok2 {
			links = append(links, &models.Link{
				Source: sourceID,
				Target: targetID,
				Weight: weight,
			})
		}
	}
	if err := linkRows.Err(); err != nil {
		return nil, fmt.Errorf("error during link rows iteration: %w", err)
	}

	return &models.Analysis{
		Nodes: nodes,
		Links: links,
	}, nil
}

// QueryCategory retrieves scored results for a category using indexed B-Tree ordering.
func (s *SQLiteStore) QueryCategory(ctx context.Context, collectionID uuid.UUID, category string, limit int) ([]*models.Result, error) {
	if limit <= 0 {
		return nil, nil
	}

	var scoreCol string
	switch models.Category(category) {
	case models.CategoryTrunk:
		scoreCol = "n.trunkness"
	case models.CategoryLeaf:
		scoreCol = "n.leafness"
	default:
		scoreCol = "n.rootness"
	}

	query := fmt.Sprintf(`
		SELECT
			%s,
			a.id, a.label, a.title, a.journal, a.year, a.volume, a.issue, a.page,
			a.doi, a.permalink, a.times_cited, a.abstract, a.rich,
			a.authors, a.keywords, a.ids
		FROM nodes n
		JOIN articles a ON a.id = n.article_id
		WHERE n.collection_id = ? AND %s > 0
		ORDER BY %s DESC
		LIMIT ?`, scoreCol, scoreCol, scoreCol)

	rows, err := s.db.QueryContext(ctx, query, UUIDToBytes(collectionID), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query category: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var results []*models.Result
	for rows.Next() {
		var (
			score       float64
			artIDBytes  []byte
			label       string
			title       sql.NullString
			journal     sql.NullString
			year        sql.NullInt64
			volume      sql.NullString
			issue       sql.NullString
			page        sql.NullString
			doi         sql.NullString
			permalink   sql.NullString
			timesCited  sql.NullInt64
			abstract    sql.NullString
			rich        bool
			authorsRaw  []byte
			keywordsRaw []byte
			idsRaw      []byte
		)

		err := rows.Scan(
			&score,
			&artIDBytes, &label, &title, &journal, &year, &volume, &issue, &page,
			&doi, &permalink, &timesCited, &abstract, &rich,
			&authorsRaw, &keywordsRaw, &idsRaw,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category result: %w", err)
		}

		art := buildArticle(label, title, journal, year, volume, issue, page, doi, permalink, timesCited, abstract, rich, authorsRaw, keywordsRaw, idsRaw)
		results = append(results, &models.Result{
			Score:   score,
			Article: art,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during category rows iteration: %w", err)
	}

	return results, nil
}

// Search executes a keyword full-text search against the collection's articles using SQLite FTS5 BM25.
func (s *SQLiteStore) Search(ctx context.Context, collectionID uuid.UUID, query string, limit int) ([]*models.Node, error) {
	if limit <= 0 {
		return nil, nil
	}

	ftsQuery := cleanFTS5Query(query)
	if ftsQuery == "" {
		return nil, nil
	}

	searchSQL := `
		SELECT
			n.id, n.category, n.rootness, n.trunkness, n.leafness, n.community,
			a.id, a.label, a.title, a.journal, a.year, a.volume, a.issue, a.page,
			a.doi, a.permalink, a.times_cited, a.abstract, a.rich,
			a.authors, a.keywords, a.ids,
			bm25(articles_fts) AS rank
		FROM articles_fts
		JOIN nodes n ON n.article_id = articles_fts.article_id
		JOIN articles a ON a.id = n.article_id
		WHERE n.collection_id = ? AND articles_fts MATCH ?
		ORDER BY rank
		LIMIT ?`

	rows, err := s.db.QueryContext(ctx, searchSQL, UUIDToBytes(collectionID), ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search fts: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var nodes []*models.Node
	for rows.Next() {
		var (
			nodeIDBytes []byte
			category    string
			rootness    float64
			trunkness   float64
			leafness    float64
			community   int
			artIDBytes  []byte
			label       string
			title       sql.NullString
			journal     sql.NullString
			year        sql.NullInt64
			volume      sql.NullString
			issue       sql.NullString
			page        sql.NullString
			doi         sql.NullString
			permalink   sql.NullString
			timesCited  sql.NullInt64
			abstract    sql.NullString
			rich        bool
			authorsRaw  []byte
			keywordsRaw []byte
			idsRaw      []byte
			rank        float64
		)

		err := rows.Scan(
			&nodeIDBytes, &category, &rootness, &trunkness, &leafness, &community,
			&artIDBytes, &label, &title, &journal, &year, &volume, &issue, &page,
			&doi, &permalink, &timesCited, &abstract, &rich,
			&authorsRaw, &keywordsRaw, &idsRaw,
			&rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search node: %w", err)
		}

		art := buildArticle(label, title, journal, year, volume, issue, page, doi, permalink, timesCited, abstract, rich, authorsRaw, keywordsRaw, idsRaw)
		nodeID := label
		if key := art.Key(); key != nil {
			nodeID = *key
		}

		nodes = append(nodes, &models.Node{
			ID:        nodeID,
			Category:  models.Category(category),
			Rootness:  rootness,
			Trunkness: trunkness,
			Leafness:  leafness,
			Community: community,
			Article:   art,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during search rows iteration: %w", err)
	}

	return nodes, nil
}

// cleanFTS5Query prepares an input string for safe SQLite FTS5 matching by quoting individual terms.
func cleanFTS5Query(query string) string {
	words := strings.Fields(query)
	if len(words) == 0 {
		return ""
	}
	cleaned := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.ReplaceAll(w, `"`, "")
		if w != "" {
			cleaned = append(cleaned, `"`+w+`"`)
		}
	}
	return strings.Join(cleaned, " ")
}

// scanCollection scans a single row into a [*Collection].
func scanCollection(row *sql.Row) (*Collection, error) {
	var (
		idBytes      []byte
		hashBytes    []byte
		query        string
		resultLimit  int
		parameters   []byte
		name         sql.NullString
		articleCount int
		createdAt    time.Time
		updatedAt    time.Time
	)

	err := row.Scan(&idBytes, &hashBytes, &query, &resultLimit, &parameters, &name, &articleCount, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to scan collection row: %w", err)
	}

	id, err := ScanUUID(idBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to scan collection uuid: %w", err)
	}

	var hash [32]byte
	copy(hash[:], hashBytes)

	var namePtr *string
	if name.Valid {
		namePtr = &name.String
	}

	return &Collection{
		ID:           id,
		QueryHash:    hash,
		Query:        query,
		ResultLimit:  resultLimit,
		Parameters:   parameters,
		Name:         namePtr,
		ArticleCount: articleCount,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

// scanCollectionRow scans the current row from [*sql.Rows] into a [*Collection].
func scanCollectionRow(rows *sql.Rows) (*Collection, error) {
	var (
		idBytes      []byte
		hashBytes    []byte
		query        string
		resultLimit  int
		parameters   []byte
		name         sql.NullString
		articleCount int
		createdAt    time.Time
		updatedAt    time.Time
	)

	err := rows.Scan(&idBytes, &hashBytes, &query, &resultLimit, &parameters, &name, &articleCount, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan collection row: %w", err)
	}

	id, err := ScanUUID(idBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to scan collection uuid: %w", err)
	}

	var hash [32]byte
	copy(hash[:], hashBytes)

	var namePtr *string
	if name.Valid {
		namePtr = &name.String
	}

	return &Collection{
		ID:           id,
		QueryHash:    hash,
		Query:        query,
		ResultLimit:  resultLimit,
		Parameters:   parameters,
		Name:         namePtr,
		ArticleCount: articleCount,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

// buildArticle constructs a [*models.Article] from scanned SQL fields.
func buildArticle(
	label string,
	title, journal sql.NullString,
	year sql.NullInt64,
	volume, issue, page, doi, permalink sql.NullString,
	timesCited sql.NullInt64,
	abstract sql.NullString,
	rich bool,
	authorsRaw, keywordsRaw, idsRaw []byte,
) *models.Article {
	var authors []string
	if len(authorsRaw) > 0 {
		_ = json.Unmarshal(authorsRaw, &authors)
	}

	var keywords *collections.Set[string]
	if len(keywordsRaw) > 0 {
		var kwList []string
		if err := json.Unmarshal(keywordsRaw, &kwList); err == nil && len(kwList) > 0 {
			keywords = collections.NewSet(kwList...)
		}
	}

	var ids *collections.Set[string]
	if len(idsRaw) > 0 {
		var idList []string
		if err := json.Unmarshal(idsRaw, &idList); err == nil && len(idList) > 0 {
			ids = collections.NewSet(idList...)
		}
	}

	art := &models.Article{
		Label:    label,
		Authors:  authors,
		Keywords: keywords,
		IDs:      ids,
		Rich:     rich,
	}

	if title.Valid {
		art.Title = &title.String
	}
	if journal.Valid {
		art.Journal = &journal.String
	}
	if year.Valid {
		y := int(year.Int64)
		art.Year = &y
	}
	if volume.Valid {
		art.Volume = &volume.String
	}
	if issue.Valid {
		art.Issue = &issue.String
	}
	if page.Valid {
		art.Page = &page.String
	}
	if doi.Valid {
		art.DOI = &doi.String
	}
	if permalink.Valid {
		art.Permalink = &permalink.String
	}
	if timesCited.Valid {
		tc := int(timesCited.Int64)
		art.TimesCited = &tc
	}
	if abstract.Valid {
		art.Abstract = &abstract.String
	}

	return art
}
