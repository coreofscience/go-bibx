package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/coreofscience/go-bibx/cli/clients/openalex"
	"github.com/coreofscience/go-bibx/cli/renderers"
	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/models"
)

type AnalysisService interface {
	Store(ctx context.Context, query string, limit int) error
	Query(ctx context.Context, category string, limit int, format string) error
}

type OpenAlexAnalysisService struct {
	openalexClient openalex.Client
	analysisRepo   repos.AnalysisRepo
	renderers      map[string]renderers.Renderer
}

func NewOpenAlexAnalysisService(
	openalexClient openalex.Client,
	analysisRepo repos.AnalysisRepo,
	rendererMap map[string]renderers.Renderer,
) *OpenAlexAnalysisService {
	return &OpenAlexAnalysisService{
		openalexClient: openalexClient,
		analysisRepo:   analysisRepo,
		renderers:      rendererMap,
	}
}

func (s *OpenAlexAnalysisService) Store(
	ctx context.Context,
	query string,
	limit int,
) error {
	works, err := s.openalexClient.ListRecentArticles(ctx, query, limit)
	if err != nil {
		return fmt.Errorf("failed to list recent articles: %w", err)
	}
	articles := make(models.Articles, len(works))
	for i, w := range works {
		articles[i] = openalex.WorkToArticle(w)
	}
	collection, err := models.NewCollection(articles)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	collection, err = collection.Clean()
	if err != nil {
		return fmt.Errorf("failed to clean collection: %w", err)
	}
	collection, err = s.enrich(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to enrich collection: %w", err)
	}
	analysis, err := models.NewAnalysis(collection)
	if err != nil {
		return fmt.Errorf("failed to create analysis: %w", err)
	}
	return s.analysisRepo.Store(ctx, analysis)
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
	slog.Debug("enriching articles", "count", len(toEnrich))
	ids := make([]string, 0, len(toEnrich))
	for _, article := range toEnrich {
		id, ok := article.ID("openalex")
		if ok {
			ids = append(ids, id)
		}
	}
	slog.Debug("listing works by ids", "count", len(ids), "articles", len(toEnrich))
	works, err := s.openalexClient.ListArticlesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list articles by ids: %w", err)
	}
	idToArticle := make(map[string]*models.Article, len(works))
	slog.Debug("mapping works to articles", "count", len(works))
	for _, work := range works {
		idToArticle[work.ID] = openalex.WorkToArticle(work)
	}
	newArticles := make(models.Articles, 0)
	discarded := 0
	for article := range c.Main() {
		if !article.Rich {
			slog.Warn("found a main work still to enrich, which is weird")
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
				slog.Warn("found a reference without an openalex id, skipping enrichment", "label", ref.Label)
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
	slog.Debug("enriched articles", "enriched", len(newArticles), "discarded", discarded)
	return models.NewCollection(newArticles)
}

func (s *OpenAlexAnalysisService) Query(
	ctx context.Context,
	category string,
	limit int,
	format string,
) error {
	analysis, err := s.analysisRepo.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load analysis: %w", err)
	}
	results, err := analysis.Query(category, limit)
	if err != nil {
		return fmt.Errorf("failed to query analysis: %w", err)
	}
	renderer, ok := s.renderers[format]
	if !ok {
		return fmt.Errorf("unsupported format: %s", format)
	}
	err = renderer.Render(results)
	if err != nil {
		return fmt.Errorf("failed to render results: %w", err)
	}
	return nil
}
