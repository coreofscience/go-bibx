package sources

import (
	"context"
	"fmt"

	"github.com/coreofscience/go-bibx/articles"
	"github.com/coreofscience/go-bibx/collection"
	"github.com/coreofscience/go-bibx/internal/clients/openalex"
	"github.com/coreofscience/go-bibx/internal/collections"
)

const (
	commonReferences = 400
	mostReferences   = 2000
)

type EnrichReferences string

const (
	EnrichReferencesNone   EnrichReferences = "none"
	EnrichReferencesCommon EnrichReferences = "common"
	EnrichReferencesMost   EnrichReferences = "most"
	EnrichReferencesAll    EnrichReferences = "all"
)

type openAlexSource struct {
	query  string
	limit  int
	enrich EnrichReferences
	client openalex.Client
}

type OpenAlexSourceOption func(*openAlexSource)

func WithLimit(limit int) OpenAlexSourceOption {
	return func(s *openAlexSource) {
		s.limit = limit
	}
}

func WithEnrichReferences(enrich EnrichReferences) OpenAlexSourceOption {
	return func(s *openAlexSource) {
		s.enrich = enrich
	}
}

func WithClient(client openalex.Client) OpenAlexSourceOption {
	return func(s *openAlexSource) {
		s.client = client
	}
}

func NewOpenAlexSource(query string, options ...OpenAlexSourceOption) Source {
	s := &openAlexSource{
		query:  query,
		limit:  100,
		enrich: EnrichReferencesNone,
		client: openalex.NewRestyClient(),
	}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *openAlexSource) Build(ctx context.Context) (*collection.Collection, error) {
	works, err := s.client.ListRecentArticles(ctx, s.query, s.limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent articles: %w", err)
	}
	cache := make(map[string]*openalex.Work)
	for _, work := range works {
		cache[work.ID] = &work
	}
	allReferences := make([]string, 0)
	for _, work := range works {
		if work.ReferencedWorks != nil {
			allReferences = append(allReferences, work.ReferencedWorks...)
		}
	}
	missing := collections.NewSet[string]()
	if s.enrich != EnrichReferencesNone && len(allReferences) > 0 {
		referenceCounter := collections.NewCounter(allReferences...)
		var toFetch []string
		switch s.enrich {
		case EnrichReferencesCommon:
			mostCommon := referenceCounter.MostCommon(commonReferences)
			for _, item := range mostCommon {
				toFetch = append(toFetch, item.Item)
			}
		case EnrichReferencesMost:
			mostCommon := referenceCounter.MostCommon(mostReferences)
			for _, item := range mostCommon {
				toFetch = append(toFetch, item.Item)
			}
		case EnrichReferencesAll:
			toFetch = allReferences
		}
		for _, refID := range toFetch {
			if _, exists := cache[refID]; !exists {
				missing.Add(refID)
			}
		}
	}
	referencedWorks, err := s.client.ListArticlesByIDs(ctx, missing.Items())
	if err != nil {
		return nil, fmt.Errorf("failed to list articles by IDs: %w", err)
	}
	for _, work := range referencedWorks {
		cache[work.ID] = &work
	}
	articleCache := make(map[string]*articles.Article)
	for id, work := range cache {
		articleCache[id] = openalex.WorkToArticle(work)
	}
	articles := make([]*articles.Article, 0, len(articleCache))
	for _, work := range works {
		article := articleCache[work.ID]
		for i, reference := range work.ReferencedWorks {
			if refArticle, exists := articleCache[reference]; exists {
				article.References[i] = refArticle
			}
		}
		articles = append(articles, article)
	}
	collection, err := collection.New(articles)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}
	return collection, nil
}
