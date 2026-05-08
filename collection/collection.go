package collection

import (
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"slices"

	"github.com/coreofscience/go-bibx/articles"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/connectivity"
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

// All returns an iterator over all articles in the collection, including their references.
func (c *Collection) All() iter.Seq[*articles.Article] {
	return func(yield func(*articles.Article) bool) {
		seen := make(map[*articles.Article]struct{}, 0)
		for _, article := range c.articles {
			if _, ok := seen[article]; ok {
				continue
			}
			seen[article] = struct{}{}
			if !yield(article) {
				return
			}
			for _, ref := range article.References {
				if _, ok := seen[ref]; ok {
					continue
				}
				seen[ref] = struct{}{}
				if !yield(ref) {
					return
				}
			}
		}
	}
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
func (c *Collection) CitationPairs() iter.Seq2[*articles.Article, *articles.Article] {
	return func(yield func(*articles.Article, *articles.Article) bool) {
		seen := make(map[*articles.Article]struct{})
		for _, article := range c.articles {
			if _, ok := seen[article]; ok {
				continue
			}
			seen[article] = struct{}{}
			for _, ref := range article.References {
				if !yield(article, ref) {
					return
				}
			}
		}
	}
}

// Purge removes articles from the collection by their IDs.
func (c *Collection) Purge(ids ...string) (*Collection, error) {
	toRemove := collections.NewSet(ids...)
	newArticles := make([]*articles.Article, 0, len(c.articles))
	for _, article := range c.articles {
		if toRemove.Contains(*article.Key()) {
			continue
		}
		newArticle := article.Copy()
		shouldPurgeReferences := slices.ContainsFunc(
			newArticle.References,
			func(a *articles.Article) bool { return toRemove.Contains(*a.Key()) },
		)
		if shouldPurgeReferences {
			newReferences := make([]*articles.Article, 0, len(newArticle.References))
			for _, ref := range newArticle.References {
				if toRemove.Contains(*ref.Key()) {
					continue
				}
				newReferences = append(newReferences, ref)
			}
			newArticle.References = newReferences
		}
		newArticles = append(newArticles, newArticle)
	}
	return New(newArticles)
}

// Remove cycles in the references of articles in the collection.
func (c *Collection) RemoveCycles() (*Collection, error) {
	graph := gograph.New[string](gograph.Directed())
	for article := range c.All() {
		articleVertex := gograph.NewVertex(*article.Key())
		for _, ref := range article.References {
			refVertex := gograph.NewVertex(*ref.Key())
			_, err := graph.AddEdge(articleVertex, refVertex)
			if err != nil {
				return nil, fmt.Errorf("failed to add edge to citation graph: %w", err)
			}
		}
	}
	cycles := slices.DeleteFunc(connectivity.Tarjan(graph), func(cycle []*gograph.Vertex[string]) bool {
		return len(cycle) == 1
	})
	if len(cycles) == 0 {
		slog.Debug("no cycles found in graph")
		return c, nil
	}
	slog.Warn("found cycles in the citation graph", "numCycles", len(cycles))
	toRemove := make([]string, 0, len(cycles))
	for _, cycle := range cycles {
		for _, vertex := range cycle {
			toRemove = append(toRemove, vertex.Label())
		}
	}
	slog.Debug("removing articles", "numArticles", len(toRemove))
	return c.Purge(toRemove...)
}

// MarshalJSON implements the json.Marshaler interface for Collection.
func (c *Collection) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.articles)
}
