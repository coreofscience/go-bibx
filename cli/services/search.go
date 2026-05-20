package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/coreofscience/go-bibx/algorithms"
	"github.com/coreofscience/go-bibx/cli/clients/embeddings"
	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/internal/texter"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/internal/vector"
	"github.com/coreofscience/go-bibx/models"
)

type SemanticSearchService struct {
	analysisRepo     repos.AnalysisRepo
	searchRepo       repos.SearchRepo
	embeddingsClient embeddings.Client
	texter           texter.ArticleTexter
}

func NewSemanticSearchService(
	analysisRepo repos.AnalysisRepo,
	searchRepo repos.SearchRepo,
	embeddingsClient embeddings.Client,
	ttr texter.ArticleTexter,
) *SemanticSearchService {
	return &SemanticSearchService{
		analysisRepo:     analysisRepo,
		searchRepo:       searchRepo,
		embeddingsClient: embeddingsClient,
		texter:           ttr,
	}
}

type SemanticSearchServiceConfig struct {
	AnalysisPath string
	SearchPath   string
}

func NewSemanticSearchServiceFromConfig(cfg *SemanticSearchServiceConfig) (*SemanticSearchService, error) {
	analysisRepo := repos.NewFileAnalysisRepo(cfg.AnalysisPath)
	searchRepo := repos.NewFileSearchRepo(cfg.SearchPath)
	embeddingsClient, err := embeddings.NewOllamaClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings client: %w", err)
	}
	ttr := texter.NewDefaultArticleTexter()
	return NewSemanticSearchService(
		analysisRepo,
		searchRepo,
		embeddingsClient,
		ttr,
	), nil
}

func (s *SemanticSearchService) Store(ctx context.Context) error {
	analysis, err := s.analysisRepo.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load analysis: %w", err)
	}
	totalNodes := len(analysis.Nodes)
	slog.InfoContext(ctx, "embedding all the nodes in the graph", "count", totalNodes)
	texts := make([]string, 0, totalNodes)
	for _, node := range analysis.Nodes {
		text := s.texter.ExtractText(node.Article)
		texts = append(texts, text)
	}
	vecs, err := s.embeddingsClient.EmbedMany(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to embed articles: %w", err)
	}
	v := vector.NewDumbVectors[string]()
	vecByID := make(map[string][]float32)
	for vec, node := range utils.Zip(vecs, analysis.Nodes) {
		vecByID[node.ID] = vec
		err := v.Add(vector.NewNode(node.ID, vec))
		if err != nil {
			return fmt.Errorf("failed to add node: %w", err)
		}
	}
	for _, link := range analysis.Links {
		weight := vector.CosineDistance(vecByID[link.Source], vecByID[link.Target])
		v.Link(vector.NewLink(link.Source, link.Target, weight))
	}
	if err := s.searchRepo.Store(ctx, v); err != nil {
		return fmt.Errorf("failed to store search graph: %w", err)
	}
	return nil
}

func (s *SemanticSearchService) Search(ctx context.Context, query string, limit int) (*models.Analysis, error) {
	vec, err := s.embeddingsClient.Embed(ctx, s.texter.CleanText(query))
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	searchEngine, err := s.searchRepo.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load search graph: %w", err)
	}
	analysis, err := s.analysisRepo.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load analysis: %w", err)
	}
	searchResults, err := searchEngine.Search(vec, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search graph: %w", err)
	}
	citationGraph, err := searchEngine.Graph()
	if err != nil {
		return nil, fmt.Errorf("failed to build citation graph: %w", err)
	}
	quasiStainer, err := algorithms.NewPseudoSteiner(citationGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to create quasi-stainer: %w", err)
	}
	terminals := make([]string, len(searchResults))
	for i, result := range searchResults {
		terminals[i] = string(result.Key)
	}
	relationGraph, err := quasiStainer.Run(terminals)
	if err != nil {
		return nil, fmt.Errorf("failed to run quasi-stainer: %w", err)
	}
	ids := make([]string, 0, relationGraph.Order())
	for _, vertex := range relationGraph.GetAllVertices() {
		ids = append(ids, vertex.Label())
	}
	return analysis.Keep(ids), nil
}
