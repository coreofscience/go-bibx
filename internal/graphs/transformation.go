package graphs

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/connectivity"
)

// Mimic returns a new graph with the same properties as the original graph.
func Mimic[K comparable](g gograph.Graph[K], overrideOptions ...gograph.GraphOptionFunc) gograph.Graph[K] {
	options := make([]gograph.GraphOptionFunc, 0)
	if g.IsDirected() {
		options = append(options, gograph.Directed())
	}
	if g.IsWeighted() {
		options = append(options, gograph.Weighted())
	}
	if g.IsAcyclic() {
		options = append(options, gograph.Acyclic())
	}
	options = append(options, overrideOptions...)
	return gograph.New[K](options...)
}

// Invert inverts the direction of the edges in the graph.
func Invert[V comparable](graph gograph.Graph[V]) gograph.Graph[V] {
	if !graph.IsDirected() {
		return graph
	}
	inverted := Mimic(graph)

	for _, v := range graph.GetAllVertices() {
		inverted.AddVertex(gograph.NewVertex(v.Label()))
	}

	for _, edge := range graph.AllEdges() {
		source := gograph.NewVertex(edge.Source().Label())
		destination := gograph.NewVertex(edge.Destination().Label())
		_, err := inverted.AddEdge(
			destination,
			source,
			gograph.WithEdgeWeight(edge.Weight()),
		)
		if err != nil {
			slog.Error("error adding edge to inverted graph", "error", err)
		}
	}
	return inverted
}

// Keep returns a graph with only the provided vertices and their connecting edges.
func Keep[V comparable](g gograph.Graph[V], toKeep []V) (gograph.Graph[V], error) {
	newGraph := Mimic(g)
	toKeepSet := make(map[V]struct{}, len(toKeep))
	for _, v := range toKeep {
		toKeepSet[v] = struct{}{}
	}

	for _, v := range g.GetAllVertices() {
		if _, found := toKeepSet[v.Label()]; found {
			newGraph.AddVertex(gograph.NewVertex(v.Label()))
		}
	}

	for _, edge := range g.AllEdges() {
		sourceLabel := edge.Source().Label()
		destLabel := edge.Destination().Label()
		if _, found := toKeepSet[sourceLabel]; !found {
			continue
		}
		if _, found := toKeepSet[destLabel]; !found {
			continue
		}
		source := gograph.NewVertex(sourceLabel)
		dest := gograph.NewVertex(destLabel)
		if edge := newGraph.GetEdge(source, dest); edge != nil {
			continue
		}
		_, err := newGraph.AddEdge(source, dest, gograph.WithEdgeWeight(edge.Weight()))
		if err != nil {
			return nil, fmt.Errorf("failed to add edge from %v to %v: %w", sourceLabel, destLabel, err)
		}
	}
	return newGraph, nil
}

// Purge removes the provided vertices from the graph.
func Purge[V comparable](g gograph.Graph[V], toRemove []V) (gograph.Graph[V], error) {
	newGraph := Mimic(g)
	toRemoveSet := make(map[V]struct{}, len(toRemove))
	for _, v := range toRemove {
		toRemoveSet[v] = struct{}{}
	}

	for _, v := range g.GetAllVertices() {
		if _, found := toRemoveSet[v.Label()]; !found {
			newGraph.AddVertex(gograph.NewVertex(v.Label()))
		}
	}

	for _, edge := range g.AllEdges() {
		sourceLabel := edge.Source().Label()
		destLabel := edge.Destination().Label()
		if _, found := toRemoveSet[sourceLabel]; found {
			continue
		}
		if _, found := toRemoveSet[destLabel]; found {
			continue
		}
		source := gograph.NewVertex(sourceLabel)
		dest := gograph.NewVertex(destLabel)
		if edge := newGraph.GetEdge(source, dest); edge != nil {
			continue
		}
		_, err := newGraph.AddEdge(source, dest, gograph.WithEdgeWeight(edge.Weight()))
		if err != nil {
			return nil, fmt.Errorf("failed to add edge from %v to %v: %w", sourceLabel, destLabel, err)
		}
	}
	return newGraph, nil
}

// Undirected returns a new graph with the same vertices and edges as the
// original graph, but with all edges undirected.
func Undirected[V comparable](g gograph.Graph[V]) (gograph.Graph[V], error) {
	if !g.IsDirected() {
		return g, nil
	}
	// Create a new graph with the same properties as the original graph, but undirected
	options := make([]gograph.GraphOptionFunc, 0)
	if g.IsWeighted() {
		options = append(options, gograph.Weighted())
	}
	if g.IsAcyclic() {
		options = append(options, gograph.Acyclic())
	}
	newGraph := gograph.New[V](options...)

	for _, v := range g.GetAllVertices() {
		newGraph.AddVertex(gograph.NewVertex(v.Label()))
	}

	for _, edge := range g.AllEdges() {
		sourceLabel := edge.Source().Label()
		destLabel := edge.Destination().Label()
		source := gograph.NewVertex(sourceLabel)
		dest := gograph.NewVertex(destLabel)
		if edge := g.GetEdge(dest, source); edge != nil {
			continue
		}
		_, err := newGraph.AddEdge(source, dest, gograph.WithEdgeWeight(edge.Weight()))
		if err != nil {
			return nil, fmt.Errorf("failed to add edge from %v to %v: %w", sourceLabel, destLabel, err)
		}
	}
	return newGraph, nil
}

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

// RemoveDangling returns a new graph with no dangling vertices
func RemoveDangling[V comparable](g gograph.Graph[V]) (gograph.Graph[V], error) {
	dangling := make([]V, 0)
	for _, vertex := range g.GetAllVertices() {
		if vertex.OutDegree() == 0 && vertex.InDegree() == 1 {
			dangling = append(dangling, vertex.Label())
		}
	}
	slog.Debug("removing dangling vertices", "numDangling", len(dangling), "order", g.Order())
	newGraph, err := Purge(g, dangling)
	if err != nil {
		return nil, fmt.Errorf("failed to purge vertices: %w", err)
	}
	return newGraph, nil
}
