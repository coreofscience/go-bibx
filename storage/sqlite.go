package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/repository"
	_ "modernc.org/sqlite" // Pure-Go SQLite driver
)

// SQLiteStore implements [Store] backed by a pure-Go SQLite database and [repository.Queries].
type SQLiteStore struct {
	db      *sql.DB
	queries *repository.Queries
}

// NewSQLiteStore opens or creates a SQLite database at the specified path,
// applies WAL mode and foreign key constraints, runs embedded migrations, and returns an initialized [*SQLiteStore].
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := repository.OpenAndMigrateSQLite(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open and migrate sqlite database: %w", err)
	}

	return &SQLiteStore{
		db:      db,
		queries: repository.New(db),
	}, nil
}

// NewSQLiteStoreWithDB creates a [*SQLiteStore] wrapping an existing [*sql.DB].
// If runMigration is true, it executes pending embedded migrations.
func NewSQLiteStoreWithDB(db *sql.DB, runMigration bool) (*SQLiteStore, error) {
	if runMigration {
		if err := repository.Migrate(db); err != nil {
			return nil, fmt.Errorf("failed to run database migrations: %w", err)
		}
	}
	return &SQLiteStore{
		db:      db,
		queries: repository.New(db),
	}, nil
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
	c, err := s.queries.GetCollectionByHash(ctx, hash[:])
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to get collection by hash: %w", err)
	}
	return toCollection(c), nil
}

// FindCollectionByID finds a collection by its UUID.
func (s *SQLiteStore) FindCollectionByID(ctx context.Context, id uuid.UUID) (*Collection, error) {
	c, err := s.queries.GetCollectionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to get collection by id: %w", err)
	}
	return toCollection(c), nil
}

// FindCollectionByName finds a collection by its human-friendly name alias.
func (s *SQLiteStore) FindCollectionByName(ctx context.Context, name string) (*Collection, error) {
	c, err := s.queries.GetCollectionByName(ctx, sql.NullString{String: name, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to get collection by name: %w", err)
	}
	return toCollection(c), nil
}

// ListCollections returns all collections ordered by creation timestamp descending.
func (s *SQLiteStore) ListCollections(ctx context.Context) ([]*Collection, error) {
	items, err := s.queries.ListCollections(ctx, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	cols := make([]*Collection, len(items))
	for i, item := range items {
		cols[i] = toCollection(item)
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

	var nameNull sql.NullString
	if col.Name != nil {
		nameNull = sql.NullString{String: *col.Name, Valid: true}
	}

	err := s.queries.CreateCollection(ctx, repository.CreateCollectionParams{
		ID:           col.ID,
		QueryHash:    col.QueryHash[:],
		Query:        col.Query,
		ResultLimit:  int64(col.ResultLimit),
		Parameters:   col.Parameters,
		Name:         nameNull,
		ArticleCount: sql.NullInt64{Int64: int64(col.ArticleCount), Valid: true},
	})
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
	if _, err := s.queries.GetCollectionByID(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCollectionNotFound
		}
		return fmt.Errorf("failed to check collection before deletion: %w", err)
	}

	if err := s.queries.DeleteCollectionById(ctx, id); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	return nil
}

// UpdateCollectionName updates the name alias of an existing collection.
func (s *SQLiteStore) UpdateCollectionName(ctx context.Context, id uuid.UUID, name *string) error {
	if _, err := s.queries.GetCollectionByID(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCollectionNotFound
		}
		return fmt.Errorf("failed to check collection before updating name: %w", err)
	}

	if err := s.queries.UpdateCollectionName(ctx, repository.UpdateCollectionNameParams{
		Name: toNullString(name),
		ID:   id,
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrCollectionExists
		}
		return fmt.Errorf("failed to update collection name: %w", err)
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

	qtx := s.queries.WithTx(tx)

	// 1. Verify collection exists
	if _, err := qtx.GetCollectionByID(ctx, collectionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCollectionNotFound
		}
		return fmt.Errorf("failed to verify collection existence: %w", err)
	}

	// 2. Clear any prior links and nodes for this collection
	if err := qtx.DeleteLinksByCollectionID(ctx, collectionID); err != nil {
		return fmt.Errorf("failed to clear existing collection links: %w", err)
	}
	if err := qtx.DeleteNodesByCollectionID(ctx, collectionID); err != nil {
		return fmt.Errorf("failed to clear existing collection nodes: %w", err)
	}

	// 3. Insert/update Articles & Nodes
	nodeIDToUUID := make(map[string]uuid.UUID, len(a.Nodes))
	for _, node := range a.Nodes {
		resolvedNodeID, err := persistNode(ctx, qtx, collectionID, node)
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

	// 4. Insert Links
	if err := persistLinks(ctx, qtx, collectionID, a.Links, nodeIDToUUID); err != nil {
		return err
	}

	// 5. Update collection article count
	if err := qtx.UpdateArticleCount(ctx, repository.UpdateArticleCountParams{
		CollectionID: collectionID,
		ID:           collectionID,
	}); err != nil {
		return fmt.Errorf("failed to update collection metadata: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit analysis transaction: %w", err)
	}

	return nil
}

func persistNode(
	ctx context.Context,
	qtx *repository.Queries,
	collectionID uuid.UUID,
	node *models.Node,
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

	var authorsJSON, keywordsJSON, idsJSON []byte
	if len(art.Authors) > 0 {
		authorsJSON, _ = json.Marshal(art.Authors)
	}
	if art.Keywords.Len() > 0 {
		keywordsJSON, _ = json.Marshal(art.Keywords.Items())
	}
	if art.IDs.Len() > 0 {
		idsJSON, _ = json.Marshal(art.IDs.Items())
	}

	resolvedArtID, err := qtx.UpsertArticle(ctx, repository.UpsertArticleParams{
		ID:         uuid.NewV7(),
		Label:      art.Label,
		Title:      toNullString(art.Title),
		Journal:    toNullString(art.Journal),
		Year:       toNullInt64(art.Year),
		Volume:     toNullString(art.Volume),
		Issue:      toNullString(art.Issue),
		Page:       toNullString(art.Page),
		Doi:        toNullString(art.DOI),
		Permalink:  toNullString(art.Permalink),
		TimesCited: toNullInt64(art.TimesCited),
		Abstract:   toNullString(art.Abstract),
		Rich:       sql.NullBool{Bool: art.Rich, Valid: true},
		Authors:    authorsJSON,
		Keywords:   keywordsJSON,
		Ids:        idsJSON,
	})
	if err != nil {
		return uuid.Nil(), fmt.Errorf("failed to upsert article %s: %w", art.Label, err)
	}

	resolvedNodeID, err := qtx.UpsertNode(ctx, repository.UpsertNodeParams{
		ID:           uuid.NewV7(),
		CollectionID: collectionID,
		ArticleID:    resolvedArtID,
		Category:     string(node.Category),
		Rootness:     node.Rootness,
		Trunkness:    node.Trunkness,
		Leafness:     node.Leafness,
		Community:    int64(node.Community),
	})
	if err != nil {
		return uuid.Nil(), fmt.Errorf("failed to upsert node %s: %w", node.ID, err)
	}

	return resolvedNodeID, nil
}

func persistLinks(
	ctx context.Context,
	qtx *repository.Queries,
	collectionID uuid.UUID,
	links []*models.Link,
	nodeIDToUUID map[string]uuid.UUID,
) error {
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

		err := qtx.UpsertLink(ctx, repository.UpsertLinkParams{
			CollectionID: collectionID,
			SourceID:     sourceUUID,
			TargetID:     targetUUID,
			Weight:       sql.NullFloat64{Float64: weight, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to insert link: %w", err)
		}
	}
	return nil
}

// LoadAnalysis reconstructs the entire models.Analysis graph from the database for a collection.
func (s *SQLiteStore) LoadAnalysis(ctx context.Context, collectionID uuid.UUID) (*models.Analysis, error) {
	if _, err := s.queries.GetCollectionByID(ctx, collectionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to verify collection existence: %w", err)
	}

	nodeRows, err := s.queries.ListNodes(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection nodes: %w", err)
	}

	nodes := make([]*models.Node, 0, len(nodeRows))
	nodeUUIDToLabel := make(map[uuid.UUID]string, len(nodeRows))

	for _, r := range nodeRows {
		art := buildArticle(r.Label, r.Title, r.Journal, r.Year, r.Volume, r.Issue, r.Page, r.Doi, r.Permalink, r.TimesCited, r.Abstract, r.Rich.Bool, r.Authors, r.Keywords, r.Ids)
		nodeID := r.Label
		if key := art.Key(); key != nil {
			nodeID = *key
		}

		nodes = append(nodes, &models.Node{
			ID:        nodeID,
			Category:  models.Category(r.Category),
			Rootness:  r.Rootness,
			Trunkness: r.Trunkness,
			Leafness:  r.Leafness,
			Community: int(r.Community),
			Article:   art,
		})
		nodeUUIDToLabel[r.ID] = nodeID
	}

	linkRows, err := s.queries.ListLinks(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection links: %w", err)
	}

	links := make([]*models.Link, 0, len(linkRows))
	for _, r := range linkRows {
		sourceID, okSource := nodeUUIDToLabel[r.SourceID]
		targetID, okTarget := nodeUUIDToLabel[r.TargetID]
		if okSource && okTarget {
			weight := 1.0
			if r.Weight.Valid {
				weight = r.Weight.Float64
			}
			links = append(links, &models.Link{
				Source: sourceID,
				Target: targetID,
				Weight: weight,
			})
		}
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

	switch models.Category(category) {
	case models.CategoryTrunk:
		rows, err := s.queries.ListTrunks(ctx, repository.ListTrunksParams{
			CollectionID: collectionID,
			Limit:        int64(limit),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to query trunks: %w", err)
		}
		results := make([]*models.Result, len(rows))
		for i, r := range rows {
			art := buildArticle(r.Label, r.Title, r.Journal, r.Year, r.Volume, r.Issue, r.Page, r.Doi, r.Permalink, r.TimesCited, r.Abstract, r.Rich.Bool, r.Authors, r.Keywords, r.Ids)
			results[i] = &models.Result{Score: r.Trunkness, Article: art}
		}
		return results, nil

	case models.CategoryLeaf:
		rows, err := s.queries.ListLeaves(ctx, repository.ListLeavesParams{
			CollectionID: collectionID,
			Limit:        int64(limit),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to query leaves: %w", err)
		}
		results := make([]*models.Result, len(rows))
		for i, r := range rows {
			art := buildArticle(r.Label, r.Title, r.Journal, r.Year, r.Volume, r.Issue, r.Page, r.Doi, r.Permalink, r.TimesCited, r.Abstract, r.Rich.Bool, r.Authors, r.Keywords, r.Ids)
			results[i] = &models.Result{Score: r.Leafness, Article: art}
		}
		return results, nil

	default: // Root
		rows, err := s.queries.ListRoots(ctx, repository.ListRootsParams{
			CollectionID: collectionID,
			Limit:        int64(limit),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to query roots: %w", err)
		}
		results := make([]*models.Result, len(rows))
		for i, r := range rows {
			art := buildArticle(r.Label, r.Title, r.Journal, r.Year, r.Volume, r.Issue, r.Page, r.Doi, r.Permalink, r.TimesCited, r.Abstract, r.Rich.Bool, r.Authors, r.Keywords, r.Ids)
			results[i] = &models.Result{Score: r.Rootness, Article: art}
		}
		return results, nil
	}
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

	rows, err := s.queries.Search(ctx, repository.SearchParams{
		ArticlesFts:  ftsQuery,
		CollectionID: collectionID,
		Limit:        int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search fts: %w", err)
	}

	nodes := make([]*models.Node, len(rows))
	for i, r := range rows {
		art := buildArticle(r.Label, r.Title, r.Journal, r.Year, r.Volume, r.Issue, r.Page, r.Doi, r.Permalink, r.TimesCited, r.Abstract, r.Rich.Bool, r.Authors, r.Keywords, r.Ids)
		nodeID := r.Label
		if key := art.Key(); key != nil {
			nodeID = *key
		}
		nodes[i] = &models.Node{
			ID:        nodeID,
			Category:  models.Category(r.Category),
			Rootness:  r.Rootness,
			Trunkness: r.Trunkness,
			Leafness:  r.Leafness,
			Community: int(r.Community),
			Article:   art,
		}
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

func toCollection(c repository.Collection) *Collection {
	var hash [32]byte
	if b, ok := c.QueryHash.([]byte); ok {
		copy(hash[:], b)
	}

	var namePtr *string
	if c.Name.Valid {
		namePtr = &c.Name.String
	}

	var createdAt, updatedAt time.Time
	if c.CreatedAt.Valid {
		createdAt = c.CreatedAt.Time.UTC()
	}
	if c.UpdatedAt.Valid {
		updatedAt = c.UpdatedAt.Time.UTC()
	}

	return &Collection{
		ID:           c.ID,
		QueryHash:    hash,
		Query:        c.Query,
		ResultLimit:  int(c.ResultLimit),
		Parameters:   c.Parameters,
		Name:         namePtr,
		ArticleCount: int(c.ArticleCount.Int64),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func toNullInt64(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
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
