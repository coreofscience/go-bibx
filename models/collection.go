package models

import "encoding/json"

// Collection represents a collection of articles with methods to manage them.
type Collection struct {
	articles Articles
}

// NewCollection creates a new Collection instance with deduplicated articles.
func NewCollection(articles Articles) *Collection {
	return &Collection{articles: articles.Deduplicate()}
}

// Len returns the number of articles in the collection.
func (c *Collection) Len() int {
	if c == nil || c.articles == nil {
		return 0
	}
	return len(c.articles)
}

// Merge merges another collection into the current collection.
func (c *Collection) Merge(other *Collection) *Collection {
	if c == nil {
		return other
	}
	if other == nil {
		return c
	}
	mergedArticles := append(c.articles, other.articles...)
	return &Collection{articles: mergedArticles.Deduplicate()}
}

// Citation pairs returns a list of citation pairs from the articles in the collection.
func (c *Collection) CitationPairs() [][2]*Article {
	if c == nil || c.articles == nil {
		return nil
	}
	var pairs [][2]*Article
	seen := make(map[*Article]bool)
	for _, article := range c.articles {
		if article == nil || article.IDs.Len() == 0 || seen[article] {
			continue
		}
		seen[article] = true
		for _, ref := range article.References {
			if ref == nil || ref.IDs.Len() == 0 {
				continue
			}
			pairs = append(pairs, [2]*Article{article, ref})
		}
	}
	return pairs
}

// MarshalJSON implements the json.Marshaler interface for Collection.
func (c *Collection) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.articles)
}
