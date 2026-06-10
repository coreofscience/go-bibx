package graphs_test

import (
	"math"
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestInvertWeight(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Weighted())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		g.AddVertex(vA)
		g.AddVertex(vB)
		g.AddVertex(vC)

		_, _ = g.AddEdge(vA, vB, gograph.WithEdgeWeight(1.0))
		_, _ = g.AddEdge(vB, vC, gograph.WithEdgeWeight(2.0))

		newG := graphs.InvertWeight(g)
		assert.Equal(t, uint32(3), newG.Order())
		assert.Equal(t, uint32(2), newG.Size())

		vA_new := newG.GetVertexByID("A")
		vB_new := newG.GetVertexByID("B")
		vC_new := newG.GetVertexByID("C")

		edge1 := newG.GetEdge(vA_new, vB_new)
		assert.NotNil(t, edge1)
		assert.InDelta(t, math.Exp(-1.0), edge1.Weight(), 1e-9)

		edge2 := newG.GetEdge(vB_new, vC_new)
		assert.NotNil(t, edge2)
		assert.InDelta(t, math.Exp(-2.0), edge2.Weight(), 1e-9)
	})

	t.Run("isolated vertices", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Weighted())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		g.AddVertex(vA)
		g.AddVertex(vB)
		g.AddVertex(vC)
		_, _ = g.AddEdge(vA, vB, gograph.WithEdgeWeight(0.5))

		newG := graphs.InvertWeight(g)
		assert.Equal(t, uint32(3), newG.Order())
		assert.Equal(t, uint32(1), newG.Size())

		vA_new := newG.GetVertexByID("A")
		vB_new := newG.GetVertexByID("B")

		assert.NotNil(t, newG.GetVertexByID("C")) // Isolated vertex C should remain

		edge := newG.GetEdge(vA_new, vB_new)
		assert.NotNil(t, edge)
		assert.InDelta(t, math.Exp(-0.5), edge.Weight(), 1e-9)
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Weighted())
		newG := graphs.InvertWeight(g)
		assert.Equal(t, uint32(0), newG.Order())
	})

	t.Run("undirected graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Weighted())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")

		g.AddVertex(vA)
		g.AddVertex(vB)
		_, _ = g.AddEdge(vA, vB, gograph.WithEdgeWeight(3.0))

		newG := graphs.InvertWeight(g)
		assert.Equal(t, uint32(2), newG.Order())

		vA_new := newG.GetVertexByID("A")
		vB_new := newG.GetVertexByID("B")

		edge := newG.GetEdge(vA_new, vB_new)
		assert.NotNil(t, edge)
		assert.InDelta(t, math.Exp(-3.0), edge.Weight(), 1e-9)
	})
}
