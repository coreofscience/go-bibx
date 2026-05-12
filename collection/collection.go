package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"slices"

	"github.com/coreofscience/go-bibx/articles"
	"github.com/coreofscience/go-bibx/internal/clients/openalex"
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
func New(a articles.Articles) (*Collection, error) {
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
func (c *Collection) Main() iter.Seq[*articles.Article] {
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
		}
	}
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

func (c *Collection) Keep(labels ...string) (*Collection, error) {
	toKeep := collections.NewSet(labels...)
	newArticles := make([]*articles.Article, 0, len(c.articles))
	for _, article := range c.articles {
		if !toKeep.Contains(*article.Key()) {
			continue
		}
		newArticle := article.KeepReferences(labels...)
		newReferences := make([]*articles.Article, 0, len(newArticle.References))
		for _, ref := range newArticle.References {
			newReferences = append(newReferences, ref.KeepReferences(labels...))
		}
		newArticle.References = newReferences
		newArticles = append(newArticles, newArticle)
	}
	return New(newArticles)
}

// Purge removes articles from the collection by their IDs.
func (c *Collection) Purge(ids ...string) (*Collection, error) {
	toRemove := collections.NewSet(ids...)
	newArticles := make([]*articles.Article, 0, len(c.articles))
	for _, article := range c.articles {
		if article.IDs.Intersect(toRemove).Len() > 0 {
			continue
		}
		newArticle := article.PurgeReferences(ids...)
		newReferences := make([]*articles.Article, 0, len(newArticle.References))
		for _, ref := range newArticle.References {
			newReferences = append(newReferences, ref.PurgeReferences(ids...))
		}
		newArticle.References = newReferences
		newArticles = append(newArticles, newArticle)
	}
	return New(newArticles)
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
	cycles := slices.DeleteFunc(
		connectivity.Tarjan(graph),
		func(cycle []*gograph.Vertex[string]) bool {
			return len(cycle) == 1
		},
	)
	selfLoops := make([]string, 0, len(cycles))
	for _, edge := range graph.AllEdges() {
		if edge.Source().Label() == edge.Destination().Label() {
			selfLoops = append(selfLoops, edge.Source().Label())
		}
	}
	slog.Debug("found self-loops", "numSelfLoops", len(selfLoops))
	if len(cycles) == 0 && len(selfLoops) == 0 {
		slog.Debug("no cycles or self-loops found in graph")
		return c, nil
	}
	slog.Warn("found cycles in the citation graph", "numCycles", len(cycles), "numSelfLoops", len(selfLoops))
	toRemove := make([]string, 0, len(cycles))
	for _, cycle := range cycles {
		for _, vertex := range cycle {
			toRemove = append(toRemove, vertex.Label())
		}
	}
	toRemove = append(toRemove, selfLoops...)
	slog.Debug("removing articles", "numArticles", len(toRemove))
	return c.Purge(toRemove...)
}

// RemoveDangling removes articles that are not referenced by other articles.
func (c *Collection) RemoveDangling() (*Collection, error) {
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
	slog.Debug("removing dangling articles", "numArticles", len(toRemove), "outOf", graph.Order())
	return c.Purge(toRemove...)
}

// Giant splits the collection into connected components and returns a slice of collections.
func (c *Collection) Giant() (*Collection, error) {
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
	if len(wccsLabels) == 0 {
		return nil, fmt.Errorf("no sub collections found")
	}
	return c.Keep(wccsLabels[0]...)
}

func (c *Collection) Enrich(ctx context.Context) (*Collection, error) {
	toEnrich := make(articles.Articles, 0, len(c.articles))
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
	// TODO: Use different clients
	slog.Debug("listing works by ids", "count", len(ids))
	client := openalex.NewRestyClient()
	works, err := client.ListArticlesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list articles by ids: %w", err)
	}
	idToArticle := make(map[string]*articles.Article, len(works))
	for _, work := range works {
		idToArticle[work.ID] = openalex.WorkToArticle(&work)
	}
	newArticles := make(articles.Articles, 0, len(c.articles))
	for _, article := range c.articles {
		if !article.Rich {
			slog.Warn("found a main work still to enrich, which is weird")
		}
		newArticle := article.Clone()
		newReferences := make(articles.References, 0, len(article.References))
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
	return New(newArticles)
}

// MarshalJSON implements the json.Marshaler interface for Collection.
func (c *Collection) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.articles)
}
