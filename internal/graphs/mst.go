package graphs

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/coreofscience/go-bibx/internal/union"
	"github.com/hmdsefi/gograph"
)

// MST returns the minimum spanning tree of the given graph.
func MST[K comparable](graph gograph.Graph[K]) (gograph.Graph[K], *union.UnionFind[K]) {
	mst := Mimic(graph, gograph.Weighted())
	for _, vertex := range graph.GetAllVertices() {
		mst.AddVertex(gograph.NewVertex(vertex.Label()))
	}
	edges := graph.AllEdges()
	slices.SortFunc(edges, func(a, b *gograph.Edge[K]) int {
		return cmp.Compare(a.Weight(), b.Weight())
	})
	labels := make([]K, 0, graph.Order())
	for _, v := range graph.GetAllVertices() {
		labels = append(labels, v.Label())
	}
	unionFind := union.New(labels)
	for _, edge := range edges {
		a := edge.Source().Label()
		b := edge.Destination().Label()
		if unionFind.Connected(a, b) {
			continue
		}
		_, err := mst.AddEdge(
			gograph.NewVertex(a),
			gograph.NewVertex(b),
			gograph.WithEdgeWeight(edge.Weight()),
		)
		if err != nil {
			slog.Error("error adding edge to the MST, this shouldn't happen", "error", err)
		}
		unionFind.Union(a, b)
	}
	return mst, unionFind
}
