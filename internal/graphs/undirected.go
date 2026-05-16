package graphs

import (
	"fmt"

	"github.com/hmdsefi/gograph"
)

// Undirected returns a new graph with the same vertices and edges as the
// original graph, but with all edges undirected.
func Undirected[V comparable](g gograph.Graph[V]) (gograph.Graph[V], error) {
	if !g.IsDirected() {
		return g, nil
	}
	// Create a new graph with the same properties as the original graph
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
