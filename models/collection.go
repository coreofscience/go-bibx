package models

import (
	"encoding/json"
	"fmt"
	"iter"
	"slices"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
)

// Collection represents a collection of articles with methods to manage them.
type Collection struct {
	articles Articles
}

// NewCollection creates a new Collection instance with deduplicated articles.
func NewCollection(a Articles) (*Collection, error) {
	a, err := a.Deduplicate()
	if err != nil {
		return nil, fmt.Errorf("failed to deduplicate articles: %w", err)
	}
	return &Collection{articles: a}, nil
}

// Len returns the number of articles in the collection.
func (c *Collection) Len() int {
	if c == nil || c.articles == nil {
		return 0
	}
	return len(c.articles)
}

// Main returns an iterator over the main articles in the collection, excluding duplicates.
func (c *Collection) Main() iter.Seq[*Article] {
	return func(yield func(*Article) bool) {
		seen := make(map[*Article]struct{}, 0)
		for _, article := range c.articles {
			if _, ok := seen[article]; ok {
				continue
			}
			seen[article] = struct{}{}
			if !yield(article) {
				return
			}
		}
	}
}

// All returns an iterator over all articles in the collection, including their references.
func (c *Collection) All() iter.Seq[*Article] {
	return func(yield func(*Article) bool) {
		seen := make(map[*Article]struct{}, 0)
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

func (c *Collection) Keep(labels ...string) (*Collection, error) {
	toKeep := collections.NewSet(labels...)
	newArticles := make([]*Article, 0, len(c.articles))
	for _, article := range c.articles {
		if !toKeep.Contains(*article.Key()) {
			continue
		}
		newArticle := article.Clone()
		shouldCleanUpReferences := slices.ContainsFunc(
			newArticle.References,
			func(a *Article) bool { return !toKeep.Contains(*a.Key()) },
		)
		if shouldCleanUpReferences {
			newReferences := make([]*Article, 0, len(newArticle.References))
			for _, ref := range newArticle.References {
				if toKeep.Contains(*ref.Key()) {
					newReferences = append(newReferences, ref)
				}
			}
			newArticle.References = newReferences
		}
		newArticles = append(newArticles, newArticle)
	}
	return NewCollection(newArticles)
}

// Purge removes articles from the collection by their IDs.
func (c *Collection) Purge(ids ...string) (*Collection, error) {
	toRemove := collections.NewSet(ids...)
	newArticles := make([]*Article, 0, len(c.articles))
	for _, article := range c.articles {
		if article.IDs.Intersect(toRemove).Len() > 0 {
			continue
		}
		newArticle := article.PurgeReferences(ids...)
		newReferences := make([]*Article, 0, len(newArticle.References))
		for _, ref := range newArticle.References {
			newReferences = append(newReferences, ref.PurgeReferences(ids...))
		}
		newArticle.References = newReferences
		newArticles = append(newArticles, newArticle)
	}
	return NewCollection(newArticles)
}

// CitationGraph returns a directed graph representing the citation relationships between articles in the collection.
func (c *Collection) CitationGraph() (gograph.Graph[string], error) {
	graph := gograph.New[string](gograph.Directed())
	for _, article := range c.articles {
		articleKey := article.Key()
		if articleKey == nil {
			continue
		}
		articleVertex := gograph.NewVertex(*articleKey)
		for _, ref := range article.References {
			refKey := ref.Key()
			if refKey == nil {
				continue
			}
			refVertex := gograph.NewVertex(*refKey)
			_, err := graph.AddEdge(articleVertex, refVertex)
			if err != nil {
				return nil, fmt.Errorf("failed to add edge: %w", err)
			}
		}
	}
	return graph, nil
}

// Clean returns a clean version of the largest connected component of the collection.
func (c *Collection) Clean() (*Collection, error) {
	graph, err := c.CitationGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to build citation graph: %w", err)
	}
	graph, err = graphs.RemoveCycles(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to remove cycles from citation graph: %w", err)
	}
	graph, err = graphs.RemoveDangling(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to remove dangling vertices from citation graph: %w", err)
	}
	undirected, err := graphs.Undirected(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to convert citation graph to undirected: %w", err)
	}
	giant, err := graphs.Giant(undirected)
	if err != nil {
		return nil, fmt.Errorf("failed to compute giant component of citation graph: %w", err)
	}
	toKeep := make([]string, 0, giant.Order())
	for _, vertex := range giant.GetAllVertices() {
		toKeep = append(toKeep, vertex.Label())
	}
	return c.Keep(toKeep...)
}

// MarshalJSON implements the json.Marshaler interface for Collection.
func (c *Collection) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.articles)
}
