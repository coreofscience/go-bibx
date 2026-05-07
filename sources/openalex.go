package sources

import (
	"context"
	"fmt"
	"strings"

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
	EnrichReferencesBasic  EnrichReferences = "basic"
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
		enrich: EnrichReferencesBasic,
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
	existingWorkIDs := collections.NewSet[string]()
	for _, work := range works {
		existingWorkIDs.Add(work.ID)
	}
	allReferences := make([]string, 0)
	for _, work := range works {
		if work.ReferencedWorks != nil {
			allReferences = append(allReferences, work.ReferencedWorks...)
		}
	}
	missing := collections.NewSet[string]()
	if s.enrich != EnrichReferencesBasic && len(allReferences) > 0 {
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
			if !existingWorkIDs.Contains(refID) {
				missing.Add(refID)
			}
		}
	}
	referencedWorks, err := s.client.ListArticlesByIDs(ctx, missing.Items())
	if err != nil {
		return nil, fmt.Errorf("failed to list articles by IDs: %w", err)
	}
	arts := make([]*articles.Article, len(works))
	for i, work := range works {
		arts[i] = workToArticle(&work)
	}
	enrichedReferences := make(articles.Articles, len(referencedWorks))
	for i, work := range referencedWorks {
		enrichedReferences[i] = workToArticle(&work)
	}
	collection, err := collection.New(arts, enrichedReferences)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}
	return collection, nil
}

func workToArticle(work *openalex.Work) *articles.Article {
	ids := make(map[string]string)
	for source, id := range work.IDs {
		realID := id
		if source == "doi" {
			realID = extractDOI(id)
		}
		ids[source] = realID
	}
	var authors []string
	for _, author := range work.Authorships {
		authors = append(authors, invertName(author.Author.DisplayName))
	}
	var journal *string
	if work.PrimaryLocation != nil && work.PrimaryLocation.Source != nil {
		journal = &work.PrimaryLocation.Source.DisplayName
	}
	var doi *string
	if work.DOI != nil {
		doiVal := extractDOI(*work.DOI)
		doi = &doiVal
	}
	var permalink *string
	if work.PrimaryLocation != nil && work.PrimaryLocation.LandingPageUrl != nil {
		permalink = work.PrimaryLocation.LandingPageUrl
	}
	references := make([]*articles.Reference, len(work.ReferencedWorks))
	for i, reference := range work.ReferencedWorks {
		references[i] = referenceToReference(reference)
	}
	keywords := make([]string, len(work.Keywords))
	for i, keyword := range work.Keywords {
		keywords[i] = keyword.DisplayName
	}
	abstract := invertAbstract(work.AbstractInvertedIndex)
	return &articles.Article{
		Label:      work.ID,
		IDs:        ids,
		Authors:    authors,
		Year:       work.PublicationYear,
		Title:      work.Title,
		Journal:    journal,
		Volume:     work.Biblio.Volume,
		Issue:      work.Biblio.Issue,
		Page:       work.Biblio.FirstPage,
		DOI:        doi,
		Permalink:  permalink,
		TimesCited: &work.CitedByCount,
		References: references,
		Keywords:   keywords,
		Abstract:   &abstract,
	}
}

func referenceToReference(reference string) *articles.Reference {
	return &articles.Reference{
		Label: reference,
		IDs: map[string]string{
			"openalex": reference,
		},
	}
}

func extractDOI(doi string) string {
	doi = strings.TrimSpace(doi)
	prefixes := []string{
		"https://doi.org/",
		"http://doi.org/",
		"https://dx.doi.org/",
		"http://dx.doi.org/",
		"doi:",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(doi), prefix) {
			return doi[len(prefix):]
		}
	}
	return doi
}

func invertName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, ",") {
		return name
	}
	parts := strings.Fields(name)
	if len(parts) < 2 {
		return name
	}
	lastName := parts[len(parts)-1]
	firstNames := parts[:len(parts)-1]
	return fmt.Sprintf("%s, %s", lastName, strings.Join(firstNames, " "))
}

func invertAbstract(abstractInvertedIndex *map[string][]int) string {
	if abstractInvertedIndex == nil {
		return ""
	}
	length := 0
	for _, indices := range *abstractInvertedIndex {
		maxIndex := 0
		for _, index := range indices {
			if index > maxIndex {
				maxIndex = index
			}
		}
		if maxIndex > length {
			length = maxIndex
		}
	}
	words := make([]string, length+1)
	for word, indices := range *abstractInvertedIndex {
		for _, index := range indices {
			words[index] = word
		}
	}
	return strings.Join(words, " ")
}
