package services

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/coreofscience/go-bibx/cli/clients/embeddings"
	"github.com/coreofscience/go-bibx/cli/renderers"
	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/internal/vector"
	"github.com/coreofscience/go-bibx/models"
)

type SearchService interface {
	// Store creates an stores a new search graph.
	Store(ctx context.Context) error

	// Search performs a search for the given query and returns a list of results.
	Search(ctx context.Context, query string, limit int) ([]*models.Result, error)
}

type SemanticSearchService struct {
	analysisRepo     repos.AnalysisRepo
	searchRepo       repos.SearchRepo
	embeddingsClient embeddings.Client
	renderer         renderers.Renderer
}

func NewSemanticSearchService(
	analysisRepo repos.AnalysisRepo,
	searchRepo repos.SearchRepo,
	embeddingsClient embeddings.Client,
	renderer renderers.Renderer,
) *SemanticSearchService {
	return &SemanticSearchService{
		analysisRepo:     analysisRepo,
		searchRepo:       searchRepo,
		embeddingsClient: embeddingsClient,
		renderer:         renderer,
	}
}

// Store implements the [SearchService] interface
func (s *SemanticSearchService) Store(ctx context.Context) error {
	analysis, err := s.analysisRepo.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load analysis: %w", err)
	}
	v := vector.NewDumbVectors[string]()
	totalNodes := len(analysis.Nodes)
	slog.InfoContext(ctx, "embedding all the nodes in the graph", "count", totalNodes)
	texts := make([]string, 0, totalNodes)
	for _, node := range analysis.Nodes {
		var buffer bytes.Buffer
		err := s.renderer.RenderArticle(&buffer, node.Article)
		if err != nil {
			return fmt.Errorf("failed to render article: %w", err)
		}
		texts = append(texts, buffer.String())
	}
	vecs, err := s.embeddingsClient.EmbedMany(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to embed articles: %w", err)
	}
	for vec, node := range utils.Zip(vecs, analysis.Nodes) {
		err := v.Add(vector.NewNode(node.ID, vec))
		if err != nil {
			return fmt.Errorf("failed to add node: %w", err)
		}
	}
	if err := s.searchRepo.Store(ctx, v); err != nil {
		return fmt.Errorf("failed to store search graph: %w", err)
	}
	return nil
}

// Search implements the [SearchService] interface
func (s *SemanticSearchService) Search(ctx context.Context, query string, limit int) ([]*models.Result, error) {
	v, err := s.embeddingsClient.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	graph, err := s.searchRepo.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load search graph: %w", err)
	}
	analysis, err := s.analysisRepo.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load analysis: %w", err)
	}
	articlesByID := make(map[string]*models.Article)
	for _, node := range analysis.Nodes {
		articlesByID[node.ID] = node.Article
	}
	neighbors, err := graph.Search(v, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search graph: %w", err)
	}
	results := make([]*models.Result, len(neighbors))
	for i, neighbor := range neighbors {
		if article, ok := articlesByID[neighbor.Key]; ok {
			results[i] = &models.Result{
				Score:   float64(i),
				Article: article,
			}
		}
	}
	return results, nil
}
