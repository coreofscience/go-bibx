package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/coreofscience/go-bibx/algorithms"
	"github.com/coreofscience/go-bibx/cli/clients/openalex"
	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/coreofscience/go-bibx/models"
)

// OpenAlexAnalysisService coordinates literature fetching from OpenAlex,
// running SAP and Leiden algorithms, and managing analysis storage.
type OpenAlexAnalysisService struct {
	openalexClient openalex.Client
	analysisRepo   repos.AnalysisRepo
}

// NewOpenAlexAnalysisService creates a new [OpenAlexAnalysisService] with the specified dependencies.
func NewOpenAlexAnalysisService(
	openalexClient openalex.Client,
	analysisRepo repos.AnalysisRepo,
) *OpenAlexAnalysisService {
	return &OpenAlexAnalysisService{
		openalexClient: openalexClient,
		analysisRepo:   analysisRepo,
	}
}

// OpenAlexAnalysisServiceConfig holds configuration options for [OpenAlexAnalysisService].
type OpenAlexAnalysisServiceConfig struct {
	AnalysisPath string
}

// NewOpenAlexAnalysisServiceFromConfig creates a new [OpenAlexAnalysisService] initialized with default dependencies and the provided config.
func NewOpenAlexAnalysisServiceFromConfig(config *OpenAlexAnalysisServiceConfig) (*OpenAlexAnalysisService, error) {
	openalexClient := openalex.NewRestyClient()
	analysisRepo := repos.NewFileAnalysisRepo(config.AnalysisPath)
	return NewOpenAlexAnalysisService(
		openalexClient,
		analysisRepo,
	), nil
}

// Store coordinates fetching literature, running community detection and SAP algorithms, and storing the resulting analysis.
func (s *OpenAlexAnalysisService) Store(
	ctx context.Context,
	query string,
	limit int,
) error {
	slog.InfoContext(ctx, "fetching initial articles")
	works, err := s.openalexClient.ListRecentArticles(ctx, query, limit)
	if err != nil {
		return fmt.Errorf("failed to list recent articles: %w", err)
	}
	articles := make(models.Articles, len(works))
	for i, w := range works {
		articles[i] = openalex.WorkToArticle(w)
	}

	slog.InfoContext(ctx, "creating and cleaning collection")
	collection, err := models.NewCollection(articles)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	collection, err = collection.Clean()
	if err != nil {
		return fmt.Errorf("failed to clean collection: %w", err)
	}
	slog.InfoContext(ctx, "enriching collection")
	collection, err = s.enrich(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to enrich collection: %w", err)
	}

	graph, err := collection.CitationGraph()
	if err != nil {
		return fmt.Errorf("failed to create citation graph: %w", err)
	}

	slog.InfoContext(ctx, "applying sap algorithm")
	sap, err := algorithms.NewSapAlgorithm(graph)
	if err != nil {
		return fmt.Errorf("failed to create sap algorithm: %w", err)
	}
	result, err := sap.Run()
	if err != nil {
		return fmt.Errorf("failed to run sap algorithm: %w", err)
	}

	nodes := make([]*models.Node, 0, collection.Len())
	for article := range collection.All() {
		articleKey := article.Key()
		if articleKey == nil {
			slog.WarnContext(ctx, "article without key, skipping", "label", article.Label)
			continue
		}
		key := *articleKey
		nodes = append(nodes, &models.Node{
			ID:        key,
			Article:   article,
			Category:  models.Category(result.Categories[key]),
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

	slog.InfoContext(ctx, "applying leiden algorithm")
	citationGraph, err := analysis.CitationGraph()
	if err != nil {
		return fmt.Errorf("failed to get citation graph: %w", err)
	}
	undirectedGraph, err := graphs.Undirected(citationGraph)
	if err != nil {
		return fmt.Errorf("failed to make graph undirected: %w", err)
	}
	communities := algorithms.NewLeiden(undirectedGraph).Run()
	for _, node := range analysis.Nodes {
		node.Community = communities[node.ID]
	}

	slog.InfoContext(ctx, "storing analysis")
	err = s.analysisRepo.Store(ctx, analysis)
	if err != nil {
		return fmt.Errorf("failed to store analysis: %w", err)
	}
	return nil
}

// Query retrieves top articles by Tree of Science category from the analysis repository.
func (s *OpenAlexAnalysisService) Query(
	ctx context.Context,
	category string,
	limit int,
) ([]*models.Result, error) {
	analysis, err := s.analysisRepo.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load analysis: %w", err)
	}
	results, err := analysis.Query(category, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query analysis: %w", err)
	}
	return results, nil
}

// Search executes a keyword graph search using BM25 and pseudo-Steiner graph reduction.
func (s *OpenAlexAnalysisService) Search(
	ctx context.Context,
	query string,
	limit int,
) (*models.Analysis, error) {
	analysis, err := s.analysisRepo.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load analysis: %w", err)
	}
	searchResults, err := analysis.Search(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search analysis: %w", err)
	}
	if len(searchResults) == 0 {
		return &models.Analysis{}, nil
	}
	citationGraph, err := analysis.CitationGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to get citation graph: %w", err)
	}
	pseudoStainer, err := algorithms.NewPseudoSteiner(citationGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to create pseudo stainer: %w", err)
	}
	terminals := make([]string, 0, len(searchResults))
	for _, result := range searchResults {
		terminals = append(terminals, result.ID)
	}
	relationGraph, err := pseudoStainer.Run(terminals)
	if err != nil {
		return nil, fmt.Errorf("failed to run pseudo stainer: %w", err)
	}
	ids := make([]string, 0, relationGraph.Order())
	for _, vertex := range relationGraph.GetAllVertices() {
		ids = append(ids, vertex.Label())
	}
	return analysis.Keep(ids), nil
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
