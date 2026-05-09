package collection

import (
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"slices"

	"github.com/coreofscience/go-bibx/articles"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/graphs"
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

func (c *Collection) Keep(labels []string) (*Collection, error) {
	toKeep := collections.NewSet(labels...)
	newArticles := make([]*articles.Article, 0, len(c.articles))
	for _, article := range c.articles {
		if !toKeep.Contains(*article.Key()) {
			continue
		}
		newArticle := article.Copy()
		shouldCleanUpReferences := slices.ContainsFunc(
			newArticle.References,
			func(a *articles.Article) bool { return !toKeep.Contains(*a.Key()) },
		)
		if shouldCleanUpReferences {
			newReferences := make([]*articles.Article, 0, len(newArticle.References))
			for _, ref := range newArticle.References {
				if toKeep.Contains(*ref.Key()) {
					newReferences = append(newReferences, ref)
				}
			}
			newArticle.References = newReferences
		}
		newArticles = append(newArticles, newArticle)
	}
	return New(newArticles)
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

// CitationGraph returns a directed graph representing the citation relationships between articles in the collection.
func (c *Collection) CitationGraph() (gograph.Graph[string], error) {
	graph := gograph.New[string](gograph.Directed())
	for _, article := range c.articles {
		articleVertex := gograph.NewVertex(*article.Key())
		for _, ref := range article.References {
			refVertex := gograph.NewVertex(*ref.Key())
			_, err := graph.AddEdge(articleVertex, refVertex)
			if err != nil {
				return nil, fmt.Errorf("failed to add edge: %w", err)
			}
		}
	}
	return graph, nil
}

// CitationGraph returns a directed graph representing the citation relationships between articles in the collection.
func (c *Collection) UndirectedCitationGraph() (gograph.Graph[string], error) {
	graph := gograph.New[string]()
	for _, article := range c.articles {
		vertexKey := article.Key()
		if vertexKey == nil {
			continue
		}
		articleVertex := gograph.NewVertex(*vertexKey)
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

// RemoveCycles removes cycles in the references of articles in the collection.
func (c *Collection) RemoveCycles() (*Collection, error) {
	graph, err := c.CitationGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to build citation graph: %w", err)
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

// RemoveIrrelevant removes articles that are not referenced by other articles.
func (c *Collection) RemoveIrrelevant() (*Collection, error) {
	// giant.remove_nodes_from(
	//     [n for n in giant if giant.in_degree(n) == 1 and giant.out_degree(n) == 0]
	// )
	graph, err := c.CitationGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to build citation graph: %w", err)
	}
	toRemove := make([]string, 0)
	for _, vertex := range graph.GetAllVertices() {
		if vertex.InDegree() == 1 && vertex.OutDegree() == 0 {
			toRemove = append(toRemove, vertex.Label())
		}
	}
	slog.Debug("removing articles", "numArticles", len(toRemove), "outOf", graph.Order())
	return c.Purge(toRemove...)
}

// Split splits the collection into connected components and returns a slice of collections.
func (c *Collection) Split() ([]*Collection, error) {
	graph, err := c.UndirectedCitationGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to build citation graph: %w", err)
	}
	wccs, err := graphs.WeaklyConnectedComponents(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to find weakly connected components: %w", err)
	}
	wccsLabels := make([][]string, 0, len(wccs))
	for _, wcc := range wccs {
		labels := make([]string, 0, len(wcc))
		for _, vertex := range wcc {
			labels = append(labels, vertex.Label())
		}
		wccsLabels = append(wccsLabels, labels)
	}
	slices.SortFunc(wccsLabels, func(a []string, b []string) int {
		return len(b) - len(a)
	})
	slog.Debug("found sub collections", "size", len(wccsLabels), "largest", len(wccsLabels[0]), "smallest", len(wccsLabels[len(wccsLabels)-1]))
	collections := make([]*Collection, 0, len(wccsLabels))
	for _, labels := range wccsLabels {
		collection, err := c.Keep(labels)
		if err != nil {
			return nil, fmt.Errorf("failed to keep labels: %w", err)
		}
		collections = append(collections, collection)
	}
	return collections, nil
}

// MarshalJSON implements the json.Marshaler interface for Collection.
func (c *Collection) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.articles)
}
