package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/coreofscience/go-bibx/cli/clients/openalex"
	"github.com/coreofscience/go-bibx/cli/repos"
	"github.com/coreofscience/go-bibx/models"
)

type AnalysisService interface {
	Store(ctx context.Context, query string, limit int) error
}

type OpenAlexAnalysisService struct {
	openalexClient openalex.Client
	analysisRepo   repos.AnalysisRepo
}

func NewOpenAlexAnalysisService(
	openalexClient openalex.Client,
	analysisRepo repos.AnalysisRepo,
) *OpenAlexAnalysisService {
	return &OpenAlexAnalysisService{
		openalexClient: openalexClient,
		analysisRepo:   analysisRepo,
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
		articles[i] = openalex.WorkToArticle(&w)
	}
	collection, err := models.NewCollection(articles)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	collection, err = collection.RemoveCycles()
	if err != nil {
		return fmt.Errorf("failed to remove cycles: %w", err)
	}
	collection, err = collection.RemoveDangling()
	if err != nil {
		return fmt.Errorf("failed to remove dangling articles: %w", err)
	}
	collection, err = collection.Giant()
	if err != nil {
		return fmt.Errorf("failed to find giant collection: %w", err)
	}
	collection, err = s.enrich(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to enrich collection: %w", err)
	}
	analysis := models.NewAnalysis(collection)
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
	slog.Debug("listing works by ids", "count", len(ids))
	works, err := s.openalexClient.ListArticlesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list articles by ids: %w", err)
	}
	idToArticle := make(map[string]*models.Article, len(works))
	for _, work := range works {
		idToArticle[work.ID] = openalex.WorkToArticle(&work)
	}
	newArticles := make(models.Articles, 0)
	for article := range c.Main() {
		if !article.Rich {
			slog.Warn("found a main work still to enrich, which is weird")
		}
		newArticle := article.Clone()
		newReferences := make(models.References, 0, len(article.References))
		for _, ref := range article.References {
			id, ok := ref.ID("openalex")
			if !ok {
				newReferences = append(newReferences, ref)
				continue
			}
			if enriched, ok := idToArticle[id]; ok {
				newReferences = append(newReferences, enriched)
			} else {
				newReferences = append(newReferences, ref)
			}
		}
		newArticle.References = newReferences
		newArticles = append(newArticles, newArticle)
	}
	return models.NewCollection(newArticles)
}
