package graphs

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/connectivity"
)

// RemoveCycles returns a graph with its cycles removed. The returned graph is a DAG.
func RemoveCycles[V comparable](g gograph.Graph[V]) (gograph.Graph[V], error) {
	if !g.IsDirected() {
		return nil, errors.New("input graph must be directed")
	}
	cycles := slices.DeleteFunc(
		connectivity.Tarjan(g),
		func(cycle []*gograph.Vertex[V]) bool {
			return len(cycle) <= 1
		},
	)
	selfLoops := make([]*gograph.Vertex[V], 0)
	for _, edge := range g.AllEdges() {
		if edge.Source().Label() == edge.Destination().Label() {
			selfLoops = append(selfLoops, edge.Source())
		}
	}
	if len(cycles) == 0 && len(selfLoops) == 0 {
		slog.Debug("no cycles found in the graph")
		return g, nil
	}
	slog.Warn("found cycles in the citation graph", "numCycles", len(cycles), "numSelfLoops", len(selfLoops))
	toRemove := make([]V, 0, len(cycles)+len(selfLoops))
	for _, cycle := range cycles {
		for _, vertex := range cycle {
			toRemove = append(toRemove, vertex.Label())
		}
	}
	for _, vertex := range selfLoops {
		toRemove = append(toRemove, vertex.Label())
	}
	slog.Debug("removing cycles from the graph", "removing", len(toRemove))
	newGraph, err := Purge(g, toRemove)
	if err != nil {
		return nil, fmt.Errorf("failed to remove cycles from the graph: %w", err)
	}
	return newGraph, nil
}
