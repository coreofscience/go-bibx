package collection

import (
	"fmt"

	"github.com/coreofscience/go-bibx/articles"
)

// Collection represents a collection of articles with methods to manage them.
type Collection struct {
	Articles           articles.Articles `json:"articles"`
	EnrichedReferences articles.Articles `json:"enrichedReferences,omitempty"`
}

// New creates a new Collection instance with deduplicated articles.
func New(articles articles.Articles, enrichedReferences articles.Articles) (*Collection, error) {
	articles, err := articles.Deduplicate()
	if err != nil {
		return nil, fmt.Errorf("failed to deduplicate articles: %w", err)
	}
	enrichedReferences, err = enrichedReferences.Deduplicate()
	if err != nil {
		return nil, fmt.Errorf("failed to deduplicate enriched references: %w", err)
	}
	return &Collection{
		Articles:           articles,
		EnrichedReferences: enrichedReferences,
	}, nil
}

// Len returns the number of articles in the collection.
func (c *Collection) Len() int {
	if c == nil || c.Articles == nil {
		return 0
	}
	return len(c.Articles)
}

// Merge merges another collection into the current collection.
func (c *Collection) Merge(other *Collection) (*Collection, error) {
	if c == nil {
		return other, nil
	}
	if other == nil {
		return c, nil
	}
	mergedArticles := append(c.Articles, other.Articles...)
	mergedArticles, err := mergedArticles.Deduplicate()
	if err != nil {
		return nil, fmt.Errorf("failed to deduplicate merged articles: %w", err)
	}
	return &Collection{Articles: mergedArticles}, nil
}
