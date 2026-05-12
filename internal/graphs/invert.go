package graphs

import "github.com/hmdsefi/gograph"

// Invert inverts the direction of the edges in the graph.
func Invert[V comparable](graph gograph.Graph[V]) gograph.Graph[V] {
	if !graph.IsDirected() {
		return graph
	}
	inverted := gograph.New[V](gograph.Directed())
	for _, edge := range graph.AllEdges() {
		source := gograph.NewVertex(edge.Source().Label())
		destination := gograph.NewVertex(edge.Destination().Label())
		if edge := inverted.GetEdge(destination, source); edge != nil {
			continue
		}
		_, _ = inverted.AddEdge(destination, source)
	}
	return inverted
}
