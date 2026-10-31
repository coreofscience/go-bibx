package models

import (
	"fmt"
	"iter"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
)

// Collection represents a collection of works with methods to manage them.
type Collection struct {
	works Works
}

// NewCollection creates a new Collection instance with deduplicated works.
func NewCollection(a Works) (*Collection, error) {
	a, err := a.Deduplicate()
	if err != nil {
		return nil, fmt.Errorf("failed to deduplicate works: %w", err)
	}
	return &Collection{works: a}, nil
}

// Len returns the number of works in the collection.
func (c *Collection) Len() int {
	if c == nil || c.works == nil {
		return 0
	}
	return len(c.works)
}

// Main returns an iterator over the main works in the collection, excluding duplicates.
func (c *Collection) Main() iter.Seq[*Work] {
	return func(yield func(*Work) bool) {
		seen := make(map[*Work]struct{}, 0)
		for _, work := range c.works {
			if _, ok := seen[work]; ok {
				continue
			}
			seen[work] = struct{}{}
			if !yield(work) {
				return
			}
		}
	}
}

// All returns an iterator over all works in the collection, including their references.
func (c *Collection) All() iter.Seq[*Work] {
	return func(yield func(*Work) bool) {
		seen := make(map[*Work]struct{}, 0)
		for _, work := range c.works {
			if _, ok := seen[work]; ok {
				continue
			}
			seen[work] = struct{}{}
			if !yield(work) {
				return
			}
			for _, ref := range work.References {
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

// Keep filters the collection, returning a new [Collection] containing only works whose keys match the provided labels.
func (c *Collection) Keep(labels ...string) (*Collection, error) {
	toKeep := collections.NewSet(labels...)
	newWorks := make([]*Work, 0, len(c.works))
	for _, work := range c.works {
		key := work.Key()
		if key == nil || !toKeep.Contains(*key) {
			continue
		}
		newWork := work.KeepReferences(labels...)
		newReferences := make([]*Work, 0, len(newWork.References))
		for _, ref := range newWork.References {
			newReferences = append(newReferences, ref.KeepReferences(labels...))
		}
		newWork.References = newReferences
		newWorks = append(newWorks, newWork)
	}
	return NewCollection(newWorks)
}

// CitationGraph returns a directed graph representing the citation relationships between works in the collection.
func (c *Collection) CitationGraph() (gograph.Graph[string], error) {
	graph := gograph.New[string](gograph.Directed())
	for _, work := range c.works {
		workKey := work.Key()
		if workKey == nil {
			continue
		}
		workVertex := gograph.NewVertex(*workKey)
		graph.AddVertex(workVertex)
		for _, ref := range work.References {
			refKey := ref.Key()
			if refKey == nil {
				continue
			}
			refVertex := gograph.NewVertex(*refKey)
			_, err := graph.AddEdge(workVertex, refVertex)
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
