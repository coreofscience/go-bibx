package models

import (
	"errors"
	"fmt"
	"iter"
	"log/slog"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
)

// Works represents a collection of [Work] pointers with helper methods for graph construction and deduplication.
type Works []*Work

// All returns a slice of all works including their references
func (w *Works) All() iter.Seq[*Work] {
	return func(yield func(*Work) bool) {
		if w == nil {
			return
		}
		seen := make(map[*Work]struct{})
		for _, work := range *w {
			if _, ok := seen[work]; ok {
				continue
			}
			seen[work] = struct{}{}
			if !yield(work) {
				return
			}
			for _, reference := range work.References {
				if _, ok := seen[reference]; ok {
					continue
				}
				seen[reference] = struct{}{}
				if !yield(reference) {
					return
				}
			}
		}
	}
}

// UniqueById returns a map of unique works by their IDs.
func (w *Works) UniqueById() (map[string]*Work, error) {
	if w == nil || *w == nil {
		return nil, errors.New("works is nil or empty")
	}
	graph := gograph.New[string]()
	idToWork := make(map[string][]*Work)
	for work := range w.All() {
		if work == nil || work.IDs == nil || work.IDs.Len() == 0 {
			continue
		}
		allIDs := work.IDs.Items()
		firstID := allIDs[0]
		remainingIDs := allIDs[1:]
		firstVertex := gograph.NewVertex(firstID)
		_, _ = graph.AddEdge(firstVertex, firstVertex)
		idToWork[firstID] = append(idToWork[firstID], work)
		for _, id := range remainingIDs {
			vertex := gograph.NewVertex(id)
			_, _ = graph.AddEdge(firstVertex, vertex)
			idToWork[id] = append(idToWork[id], work)
		}
	}
	unique := make(map[string]*Work, 0)
	wccs, err := graphs.WeaklyConnectedComponents(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate weakly connected components: %w", err)
	}
	if len(wccs) == 0 {
		return unique, nil
	}
	smallest, biggest := len(*w), 0
	for _, sc := range wccs {
		if len(sc) < smallest {
			smallest = len(sc)
		}
		if len(sc) > biggest {
			biggest = len(sc)
		}
	}
	slog.Debug("found works to merge", "count", len(wccs), "smallest", smallest, "biggest", biggest)
	for _, sc := range wccs {
		if len(sc) == 0 {
			continue
		}
		visited := make(map[*Work]bool)
		connectedWorks := make([]*Work, 0, len(sc))
		ids := make([]string, 0, len(sc))
		for _, vertex := range sc {
			id := vertex.Label()
			ids = append(ids, id)
			works, exists := idToWork[id]
			if !exists {
				slog.Warn("no works found for ID", "id", id)
				continue
			}
			for _, work := range works {
				if work == nil || visited[work] {
					continue
				}
				visited[work] = true
				connectedWorks = append(connectedWorks, work)
			}
		}
		if len(connectedWorks) == 0 {
			continue
		}
		if len(connectedWorks) == 1 {
			for _, id := range ids {
				if id == "" {
					continue
				}
				if _, exists := unique[id]; !exists {
					unique[id] = connectedWorks[0]
				}
			}
			continue
		}
		firstWork := connectedWorks[0]
		remainingWorks := connectedWorks[1:]
		for _, work := range remainingWorks {
			firstWork = firstWork.Merge(work)
		}
		for _, id := range ids {
			if id == "" {
				continue
			}
			if _, exists := unique[id]; !exists {
				unique[id] = firstWork
			}
		}
	}
	return unique, nil
}

// Deduplicate returns a slice of unique works, preserving the order of the first occurrence of each work.
func (w *Works) Deduplicate() (Works, error) {
	if w == nil || *w == nil {
		return nil, errors.New("works are empty")
	}
	uniqueMap, err := w.UniqueById()
	if err != nil {
		return nil, fmt.Errorf("failed to get unique works: %w", err)
	}
	// Create the list of unique works
	uniqueWorks := make(Works, 0, len(*w))
	seen := make(map[*Work]bool)
	for _, work := range *w {
		if work == nil || work.IDs.Len() == 0 {
			continue
		}
		firstID := work.IDs.Items()[0]
		uniqueWork, exists := uniqueMap[firstID]
		if !exists || uniqueWork == nil || seen[uniqueWork] {
			continue
		}
		uniqueWorks = append(uniqueWorks, uniqueWork)
		seen[uniqueWork] = true
	}

	// Replace the references in each work with their deduplicated versions
	for _, work := range uniqueWorks {
		if work == nil || work.References == nil || len(work.References) == 0 {
			continue
		}
		dedupedReferences := make([]*Work, 0, len(work.References))
		seenRefs := make(map[*Work]bool)
		for _, ref := range work.References {
			if ref == nil || ref.IDs.Len() == 0 {
				continue
			}
			firstRefID := ref.IDs.Items()[0]
			dedupedRef, exists := uniqueMap[firstRefID]
			if !exists || dedupedRef == nil || seenRefs[dedupedRef] {
				continue
			}
			dedupedReferences = append(dedupedReferences, dedupedRef)
			seenRefs[dedupedRef] = true
		}
		work.References = dedupedReferences
	}

	return uniqueWorks, nil
}
