package collection

import (
	"encoding/json"
	"fmt"

	"github.com/coreofscience/go-bibx/articles"
)

// Collection represents a collection of articles with methods to manage them.
type Collection struct {
	articles articles.Articles
}

// New creates a new Collection instance with deduplicated articles.
func New(articles articles.Articles) (*Collection, error) {
	articles, err := articles.Deduplicate()
	if err != nil {
		return nil, fmt.Errorf("failed to deduplicate articles: %w", err)
	}
	return &Collection{articles: articles}, nil
}

// Len returns the number of articles in the collection.
func (c *Collection) Len() int {
	if c == nil || c.articles == nil {
		return 0
	}
	return len(c.articles)
}

// Merge merges another collection into the current collection.
func (c *Collection) Merge(other *Collection) (*Collection, error) {
	if c == nil {
		return other, nil
	}
	if other == nil {
		return c, nil
	}
	mergedArticles := append(c.articles, other.articles...)
	mergedArticles, err := mergedArticles.Deduplicate()
	if err != nil {
		return nil, fmt.Errorf("failed to deduplicate merged articles: %w", err)
	}
	return &Collection{articles: mergedArticles}, nil
}

// Citation pairs returns a list of citation pairs from the articles in the collection.
func (c *Collection) CitationPairs() [][2]*articles.Article {
	if c == nil || c.articles == nil {
		return nil
	}
	var pairs [][2]*articles.Article
	seen := make(map[*articles.Article]bool)
	for _, article := range c.articles {
		if article == nil || article.IDs.Len() == 0 || seen[article] {
			continue
		}
		seen[article] = true
		for _, ref := range article.References {
			if ref == nil || ref.IDs.Len() == 0 {
				continue
			}
			pairs = append(pairs, [2]*articles.Article{article, ref})
		}
	}
	return pairs
}

// MarshalJSON implements the json.Marshaler interface for Collection.
func (c *Collection) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.articles)
}
