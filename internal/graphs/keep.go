package graphs

import (
	"fmt"

	"github.com/hmdsefi/gograph"
)

// Keep returns a graph with only the provided vertices and their connecting edges.
func Keep[V comparable](g gograph.Graph[V], toKeep []V) (gograph.Graph[V], error) {
	// Create a new graph with the same properties as the original graph
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
	newGraph := gograph.New[V](options...)
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
		if edge := newGraph.GetEdge(dest, source); edge != nil {
			continue
		}
		_, err := newGraph.AddEdge(source, dest, gograph.WithEdgeWeight(edge.Weight()))
		if err != nil {
			return nil, fmt.Errorf("failed to add edge from %v to %v: %w", sourceLabel, destLabel, err)
		}
	}
	return newGraph, nil
}
