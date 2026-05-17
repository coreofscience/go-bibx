package graphs

import (
	"log/slog"

	"github.com/hmdsefi/gograph"
)

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
