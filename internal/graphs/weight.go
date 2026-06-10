package graphs

import (
	"log/slog"
	"math"

	"github.com/hmdsefi/gograph"
)

// InvertWeight returns a new graph with the same vertices and edges as the original graph,
// but with the weights inverted using an exponential kernel function.
func InvertWeight[V comparable](graph gograph.Graph[V]) gograph.Graph[V] {
	inverted := Mimic(graph)

	for _, v := range graph.GetAllVertices() {
		inverted.AddVertex(gograph.NewVertex(v.Label()))
	}

	for _, edge := range graph.AllEdges() {
		source := gograph.NewVertex(edge.Source().Label())
		destination := gograph.NewVertex(edge.Destination().Label())

		weight := edge.Weight()
		invertedWeight := math.Exp(-weight)

		_, err := inverted.AddEdge(
			source,
			destination,
			gograph.WithEdgeWeight(invertedWeight),
		)
		if err != nil {
			slog.Error("error adding edge to inverted weight graph", "error", err)
		}
	}

	return inverted
}
