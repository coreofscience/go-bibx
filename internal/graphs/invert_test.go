package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestInvert(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		newG := graphs.Invert(g)
		assert.Equal(t, uint32(3), newG.Order())
		assert.Equal(t, uint32(2), newG.Size())

		vA_new := newG.GetVertexByID("A")
		vB_new := newG.GetVertexByID("B")
		vC_new := newG.GetVertexByID("C")

		assert.NotNil(t, newG.GetEdge(vB_new, vA_new))
		assert.NotNil(t, newG.GetEdge(vC_new, vB_new))
		assert.Nil(t, newG.GetEdge(vA_new, vB_new))
		assert.Nil(t, newG.GetEdge(vB_new, vC_new))
	})

	t.Run("isolated vertices", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		g.AddVertex(vA)
		g.AddVertex(vB)
		g.AddVertex(vC)
		_, _ = g.AddEdge(vA, vB)

		newG := graphs.Invert(g)
		assert.Equal(t, uint32(3), newG.Order())
		assert.Equal(t, uint32(1), newG.Size())

		vA_new := newG.GetVertexByID("A")
		vB_new := newG.GetVertexByID("B")

		assert.NotNil(t, newG.GetVertexByID("C")) // Isolated vertex C should remain
		assert.NotNil(t, newG.GetEdge(vB_new, vA_new))
		assert.Nil(t, newG.GetEdge(vA_new, vB_new))
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		newG := graphs.Invert(g)
		assert.Equal(t, uint32(0), newG.Order())
	})

	t.Run("undirected graph", func(t *testing.T) {
		g := gograph.New[string]()
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		_, _ = g.AddEdge(vA, vB)

		newG := graphs.Invert(g)
		// It should just return the same graph pointer
		assert.Equal(t, g, newG)
	})
}
