package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"uuid"

	"github.com/coreofscience/go-bibx/algorithms"
	"github.com/coreofscience/go-bibx/clients/openalex"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/repository"
)

// InquireOptions specifies optional configuration for storing inquiry results.
type InquireOptions struct {
	Force bool
	Name  string
}

// InquireResult encapsulates the stored or matched collection and whether it was newly queried.
type InquireResult struct {
	Collection *Collection
	IsNew      bool
}

// AnalysisService defines operations for literature inquiry, graph algorithms, and analysis queries.
type AnalysisService interface {
	Inquire(ctx context.Context, query string, limit int, opts InquireOptions) (*InquireResult, error)
	StoreAnalysis(ctx context.Context, collectionID uuid.UUID, a *models.Analysis) error
	LoadAnalysis(ctx context.Context, collectionID uuid.UUID) (*models.Analysis, error)
	QueryCategory(ctx context.Context, collectionID uuid.UUID, category string, limit int) ([]*models.Result, error)
	Search(ctx context.Context, collectionID uuid.UUID, query string, limit int) (*models.Analysis, error)
	GetArticles(ctx context.Context, collectionID uuid.UUID, ids []string) ([]*models.Article, []string, error)
}

// OpenAlexAnalysisService coordinates literature fetching from OpenAlex,
// running SAP and Steiner algorithms, and managing analysis persistence via SQLite.
type OpenAlexAnalysisService struct {
	db             *sql.DB
	queries        *repository.Queries
	openalexClient openalex.Client
}

// NewAnalysisService creates a new [OpenAlexAnalysisService] backed by the provided [sql.DB] and [openalex.Client].
func NewAnalysisService(
	db *sql.DB,
	openalexClient openalex.Client,
) *OpenAlexAnalysisService {
	return &OpenAlexAnalysisService{
		db:             db,
		queries:        repository.New(db),
		openalexClient: openalexClient,
	}
}

// NewAnalysisServiceWithQueries creates a new [OpenAlexAnalysisService] with explicit [*repository.Queries].
func NewAnalysisServiceWithQueries(
	db *sql.DB,
	queries *repository.Queries,
	openalexClient openalex.Client,
) *OpenAlexAnalysisService {
	return &OpenAlexAnalysisService{
		db:             db,
		queries:        queries,
		openalexClient: openalexClient,
	}
}

// Inquire coordinates fetching literature from OpenAlex, running SAP algorithms,
// and storing the resulting analysis in SQLite with query idempotency support.
func (s *OpenAlexAnalysisService) Inquire(
	ctx context.Context,
	query string,
	limit int,
	opts InquireOptions,
) (*InquireResult, error) {
	queryHash := ComputeQueryHash(query, limit)

	existingRow, err := s.queries.GetCollectionByHash(ctx, queryHash[:])
	var existing *Collection
	if err == nil {
		existing = toCollection(existingRow)
		if !opts.Force {
			slog.InfoContext(ctx, "collection already exists for query; skipping remote inquiry",
				"collection_id", existing.ID,
				"query", query,
				"limit", limit,
			)
			return &InquireResult{
				Collection: existing,
				IsNew:      false,
			}, nil
		}
		slog.InfoContext(ctx, "collection already exists for query; force re-run enabled",
			"collection_id", existing.ID,
			"query", query,
		)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing collection: %w", err)
	}

	slog.InfoContext(ctx, "fetching initial articles")
	works, err := s.openalexClient.ListRecentArticles(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent articles: %w", err)
	}
	articles := make(models.Articles, len(works))
	for i, w := range works {
		articles[i] = openalex.WorkToArticle(w)
	}

	slog.InfoContext(ctx, "creating and cleaning collection")
	collection, err := models.NewCollection(articles)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}
	collection, err = collection.Clean()
	if err != nil {
		return nil, fmt.Errorf("failed to clean collection: %w", err)
	}
	slog.InfoContext(ctx, "enriching collection")
	collection, err = s.enrich(ctx, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich collection: %w", err)
	}
	collection, err = collection.Clean()
	if err != nil {
		return nil, fmt.Errorf("failed to clean enriched collection: %w", err)
	}

	graph, err := collection.CitationGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to create citation graph: %w", err)
	}

	slog.InfoContext(ctx, "applying sap algorithm")
	sap, err := algorithms.NewSapAlgorithm(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to create sap algorithm: %w", err)
	}
	result, err := sap.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run sap algorithm: %w", err)
	}

	nodes := make([]*models.Node, 0, collection.Len())
	for article := range collection.All() {
		articleKey := article.Key()
		if articleKey == nil {
			slog.WarnContext(ctx, "article without key, skipping", "label", article.Label)
			continue
		}
		key := *articleKey
		category := result.Categories[key]
		if category == "" {
			category = algorithms.CategoryNone
		}
		nodes = append(nodes, &models.Node{
			ID:        key,
			Article:   article,
			Category:  models.Category(category),
			Rootness:  result.Rootness[key],
			Trunkness: result.Trunkness[key],
			Leafness:  result.Leafness[key],
		})
	}
	links := make([]*models.Link, 0, collection.Len())
	for _, edge := range graph.AllEdges() {
		links = append(links, &models.Link{
			Source: edge.Source().Label(),
			Target: edge.Destination().Label(),
			Weight: 1.0,
		})
	}
	analysis := &models.Analysis{
		Nodes: nodes,
		Links: links,
	}

	var colName *string
	if trimmed := strings.TrimSpace(opts.Name); trimmed != "" {
		colName = &trimmed
	}

	colID := uuid.NewV7()
	if existing != nil {
		colID = existing.ID
		if colName != nil {
			if err := s.queries.UpdateCollectionName(ctx, repository.UpdateCollectionNameParams{
				Name: toNullString(colName),
				ID:   colID,
			}); err != nil {
				if strings.Contains(err.Error(), "UNIQUE constraint failed") {
					return nil, ErrCollectionExists
				}
				return nil, fmt.Errorf("failed to update collection name: %w", err)
			}
		}
	} else {
		var nameNull sql.NullString
		if colName != nil {
			nameNull = sql.NullString{String: *colName, Valid: true}
		}
		if err := s.queries.CreateCollection(ctx, repository.CreateCollectionParams{
			ID:           colID,
			QueryHash:    queryHash[:],
			Query:        NormalizeQuery(query),
			ResultLimit:  int64(limit),
			Parameters:   nil,
			Name:         nameNull,
			ArticleCount: sql.NullInt64{Int64: int64(len(nodes)), Valid: true},
		}); err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return nil, ErrCollectionExists
			}
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	slog.InfoContext(ctx, "storing analysis", "collection_id", colID)
	if err := s.StoreAnalysis(ctx, colID, analysis); err != nil {
		return nil, fmt.Errorf("failed to store analysis: %w", err)
	}

	savedRow, err := s.queries.GetCollectionByID(ctx, colID)
	if err != nil {
		return nil, fmt.Errorf("failed to load saved collection: %w", err)
	}
	return &InquireResult{
		Collection: toCollection(savedRow),
		IsNew:      true,
	}, nil
}

// StoreAnalysis persists an analysis graph (articles, scored nodes, links, and FTS5 entries) for a collection.
func (s *OpenAlexAnalysisService) StoreAnalysis(ctx context.Context, collectionID uuid.UUID, a *models.Analysis) error {
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

// LoadAnalysis reconstructs the entire models.Analysis graph from the database for a collection.
func (s *OpenAlexAnalysisService) LoadAnalysis(ctx context.Context, collectionID uuid.UUID) (*models.Analysis, error) {
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
func (s *OpenAlexAnalysisService) QueryCategory(ctx context.Context, collectionID uuid.UUID, category string, limit int) ([]*models.Result, error) {
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

// Search executes a keyword graph search using SQLite FTS5 and pseudo-Steiner graph reduction.
func (s *OpenAlexAnalysisService) Search(ctx context.Context, collectionID uuid.UUID, query string, limit int) (*models.Analysis, error) {
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
	if len(rows) == 0 {
		return &models.Analysis{}, nil
	}

	analysis, err := s.LoadAnalysis(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load analysis for graph search: %w", err)
	}

	citationGraph, err := analysis.CitationGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to get citation graph: %w", err)
	}
	pseudoSteiner, err := algorithms.NewPseudoSteiner(citationGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to create pseudo steiner: %w", err)
	}
	terminals := make([]string, 0, len(rows))
	for _, r := range rows {
		art := buildArticle(r.Label, r.Title, r.Journal, r.Year, r.Volume, r.Issue, r.Page, r.Doi, r.Permalink, r.TimesCited, r.Abstract, r.Rich.Bool, r.Authors, r.Keywords, r.Ids)
		nodeID := r.Label
		if key := art.Key(); key != nil {
			nodeID = *key
		}
		terminals = append(terminals, nodeID)
	}
	relationGraph, err := pseudoSteiner.Run(terminals)
	if err != nil {
		return nil, fmt.Errorf("failed to run pseudo steiner: %w", err)
	}
	ids := make([]string, 0, relationGraph.Order())
	for _, vertex := range relationGraph.GetAllVertices() {
		ids = append(ids, vertex.Label())
	}
	return analysis.Keep(ids), nil
}

// GetArticles retrieves articles belonging to a collection matching the provided IDs or labels.
// If ids is empty or nil, all articles belonging to the collection are returned.
// Any requested ID that cannot be resolved is returned in the notFound slice.
func (s *OpenAlexAnalysisService) GetArticles(ctx context.Context, collectionID uuid.UUID, ids []string) ([]*models.Article, []string, error) {
	analysis, err := s.LoadAnalysis(ctx, collectionID)
	if err != nil {
		return nil, nil, err
	}
	if analysis == nil || len(analysis.Nodes) == 0 {
		return nil, ids, nil
	}

	if len(ids) == 0 {
		articles := make([]*models.Article, 0, len(analysis.Nodes))
		for _, node := range analysis.Nodes {
			if node != nil && node.Article != nil {
				articles = append(articles, node.Article)
			}
		}
		return articles, nil, nil
	}

	foundArticles := make([]*models.Article, 0, len(ids))
	seen := make(map[*models.Article]bool)
	var notFound []string

	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		var matched *models.Article
		for _, node := range analysis.Nodes {
			if matchArticle(node, trimmed) {
				matched = node.Article
				break
			}
		}
		if matched != nil {
			if !seen[matched] {
				seen[matched] = true
				foundArticles = append(foundArticles, matched)
			}
		} else {
			notFound = append(notFound, trimmed)
		}
	}

	return foundArticles, notFound, nil
}

func matchArticle(node *models.Node, target string) bool {
	if node == nil {
		return false
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}

	// 1. Direct match on node ID
	if strings.EqualFold(node.ID, target) {
		return true
	}

	art := node.Article
	if art == nil {
		return false
	}

	// 2. Direct match on article Label
	if strings.EqualFold(art.Label, target) {
		return true
	}

	// 3. Match on SimpleId (e.g. "vaswani2017")
	if sID := art.SimpleId(); sID != nil && strings.EqualFold(*sID, target) {
		return true
	}

	// 4. Match on DOI
	if art.DOI != nil && *art.DOI != "" {
		cleanArtDOI := utils.ExtractDOI(*art.DOI)
		cleanTargetDOI := utils.ExtractDOI(target)
		if strings.EqualFold(cleanArtDOI, cleanTargetDOI) {
			return true
		}
	}

	// 5. Match on IDs set
	if art.IDs != nil {
		for _, id := range art.IDs.Items() {
			if matchID(id, target) {
				return true
			}
		}
	}

	// 6. Match on Key
	if key := art.Key(); key != nil && matchID(*key, target) {
		return true
	}

	return false
}

func matchID(storedID, target string) bool {
	if strings.EqualFold(storedID, target) {
		return true
	}

	// Compare with stripped source prefix (e.g. "openalex:", "doi:")
	_, strippedStored, hasStoredPrefix := strings.Cut(storedID, ":")
	_, strippedTarget, hasTargetPrefix := strings.Cut(target, ":")

	if hasStoredPrefix && strings.EqualFold(strippedStored, target) {
		return true
	}
	if hasTargetPrefix && strings.EqualFold(storedID, strippedTarget) {
		return true
	}
	if hasStoredPrefix && hasTargetPrefix && strings.EqualFold(strippedStored, strippedTarget) {
		return true
	}

	// Compare with trimmed OpenAlex URL prefix
	oaStored := strings.TrimPrefix(storedID, "openalex:")
	oaStored = strings.TrimPrefix(oaStored, "https://openalex.org/")
	oaTarget := strings.TrimPrefix(target, "openalex:")
	oaTarget = strings.TrimPrefix(oaTarget, "https://openalex.org/")
	if oaStored != "" && strings.EqualFold(oaStored, oaTarget) {
		return true
	}

	return false
}

func (s *OpenAlexAnalysisService) enrich(
	ctx context.Context,
	c *models.Collection,
) (*models.Collection, error) {
	toEnrich := make(models.Articles, 0)
	for article := range c.All() {
		if !article.Rich {
			toEnrich = append(toEnrich, article)
		}
	}
	slog.DebugContext(ctx, "enriching articles", "count", len(toEnrich))
	ids := make([]string, 0, len(toEnrich))
	for _, article := range toEnrich {
		id, ok := article.ID("openalex")
		if ok {
			ids = append(ids, id)
		}
	}
	slog.DebugContext(ctx, "listing works by ids", "count", len(ids), "articles", len(toEnrich))
	works, err := s.openalexClient.ListArticlesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list articles by ids: %w", err)
	}
	idToArticle := make(map[string]*models.Article, len(works))
	slog.DebugContext(ctx, "mapping works to articles", "count", len(works))
	for _, work := range works {
		idToArticle[work.ID] = openalex.WorkToArticle(work)
	}
	newArticles := make(models.Articles, 0)
	discarded := 0
	for article := range c.Main() {
		if !article.Rich {
			slog.WarnContext(ctx, "found a main work still to enrich, which is weird")
		}
		newArticle := article.Clone()
		newReferences := make(models.References, 0, len(article.References))
		for _, ref := range article.References {
			if ref.Rich {
				newReferences = append(newReferences, ref)
				continue
			}
			id, ok := ref.ID("openalex")
			if !ok {
				slog.WarnContext(ctx, "found a reference without an openalex id, skipping enrichment", "label", ref.Label)
				newReferences = append(newReferences, ref)
				continue
			}
			if enriched, ok := idToArticle[id]; ok {
				newReferences = append(newReferences, enriched)
			} else {
				discarded++
			}
		}
		newArticle.References = newReferences
		newArticles = append(newArticles, newArticle)
	}
	slog.DebugContext(ctx, "enriched articles", "enriched", len(newArticles), "discarded", discarded)
	collection, err := models.NewCollection(newArticles)
	if err != nil {
		return nil, fmt.Errorf("failed to create enriched collection: %w", err)
	}
	return collection, nil
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
	if art.Keywords != nil && art.Keywords.Len() > 0 {
		keywordsJSON, _ = json.Marshal(art.Keywords.Items())
	}
	if art.IDs != nil && art.IDs.Len() > 0 {
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
