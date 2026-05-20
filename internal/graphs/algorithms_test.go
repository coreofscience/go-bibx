package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestMST(t *testing.T) {
	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Weighted())
		mst, uf := graphs.MST(g)
		assert.Equal(t, uint32(0), mst.Order())
		assert.Equal(t, uint32(0), mst.Size())
		assert.NotNil(t, uf)
	})

	t.Run("single vertex", func(t *testing.T) {
		g := gograph.New[string](gograph.Weighted())
		g.AddVertex(gograph.NewVertex("A"))
		mst, uf := graphs.MST(g)
		assert.Equal(t, uint32(1), mst.Order())
		assert.Equal(t, uint32(0), mst.Size())
		assert.NotNil(t, uf)
		assert.NotNil(t, mst.GetVertexByID("A"))
	})

	t.Run("simple connected graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Weighted())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB, gograph.WithEdgeWeight(1.0))
		_, _ = g.AddEdge(vB, vC, gograph.WithEdgeWeight(2.0))
		_, _ = g.AddEdge(vA, vC, gograph.WithEdgeWeight(3.0))

		mst, _ := graphs.MST(g)
		assert.Equal(t, uint32(3), mst.Order())
		assert.Equal(t, uint32(4), mst.Size())

		// MST should contain A-B (1.0) and B-C (2.0)
		assert.NotNil(t, mst.GetEdge(mst.GetVertexByID("A"), mst.GetVertexByID("B")))
		assert.NotNil(t, mst.GetEdge(mst.GetVertexByID("B"), mst.GetVertexByID("C")))
		assert.Nil(t, mst.GetEdge(mst.GetVertexByID("A"), mst.GetVertexByID("C")))

		// Verify weights
		edgeAB := mst.GetEdge(mst.GetVertexByID("A"), mst.GetVertexByID("B"))
		edgeBC := mst.GetEdge(mst.GetVertexByID("B"), mst.GetVertexByID("C"))
		assert.Equal(t, 1.0, edgeAB.Weight())
		assert.Equal(t, 2.0, edgeBC.Weight())
	})

	t.Run("disconnected graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Weighted())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")
		vD := gograph.NewVertex("D")

		_, _ = g.AddEdge(vA, vB, gograph.WithEdgeWeight(1.0))
		_, _ = g.AddEdge(vC, vD, gograph.WithEdgeWeight(1.0))

		mst, uf := graphs.MST(g)
		assert.Equal(t, uint32(4), mst.Order())
		assert.Equal(t, uint32(4), mst.Size())

		assert.True(t, uf.Connected("A", "B"))
		assert.True(t, uf.Connected("C", "D"))
		assert.False(t, uf.Connected("A", "C"))
	})

	t.Run("multiple MSTs", func(t *testing.T) {
		g := gograph.New[string](gograph.Weighted())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		// All weights are same, any 2 edges form an MST
		_, _ = g.AddEdge(vA, vB, gograph.WithEdgeWeight(1.0))
		_, _ = g.AddEdge(vB, vC, gograph.WithEdgeWeight(1.0))
		_, _ = g.AddEdge(vA, vC, gograph.WithEdgeWeight(1.0))

		mst, _ := graphs.MST(g)
		assert.Equal(t, uint32(3), mst.Order())
		assert.Equal(t, uint32(4), mst.Size())

		// The sum of weights should be 4.0 (2 edges * 2 directions)
		totalWeight := 0.0
		for _, edge := range mst.AllEdges() {
			totalWeight += edge.Weight()
		}
		assert.Equal(t, 4.0, totalWeight)
	})

	t.Run("directed graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Weighted())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB, gograph.WithEdgeWeight(1.0))
		_, _ = g.AddEdge(vB, vC, gograph.WithEdgeWeight(2.0))
		_, _ = g.AddEdge(vA, vC, gograph.WithEdgeWeight(3.0))

		mst, _ := graphs.MST(g)
		assert.True(t, mst.IsDirected())
		assert.Equal(t, uint32(3), mst.Order())
		assert.Equal(t, uint32(2), mst.Size()) // Directed edges are counted once

		assert.NotNil(t, mst.GetEdge(mst.GetVertexByID("A"), mst.GetVertexByID("B")))
		assert.NotNil(t, mst.GetEdge(mst.GetVertexByID("B"), mst.GetVertexByID("C")))
		assert.Nil(t, mst.GetEdge(mst.GetVertexByID("A"), mst.GetVertexByID("C")))
	})
}
