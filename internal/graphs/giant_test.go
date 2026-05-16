package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestGiant(t *testing.T) {
	t.Run("directed disconnected components", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		vD := gograph.NewVertex("D")
		vE := gograph.NewVertex("E")

		// Component 1 (size 3, cycle)
		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)
		_, _ = g.AddEdge(vC, vA)

		// Component 2 (size 2, cycle)
		_, _ = g.AddEdge(vD, vE)
		_, _ = g.AddEdge(vE, vD)

		newG, err := graphs.Giant(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(3), newG.Order()) // A, B, C should remain

		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.NotNil(t, newG.GetVertexByID("B"))
		assert.NotNil(t, newG.GetVertexByID("C"))
		assert.Nil(t, newG.GetVertexByID("D"))
		assert.Nil(t, newG.GetVertexByID("E"))
	})

	t.Run("undirected disconnected components", func(t *testing.T) {
		g := gograph.New[string]() // Undirected by default
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		vD := gograph.NewVertex("D")
		vE := gograph.NewVertex("E")

		// Component 1 (size 3)
		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		// Component 2 (size 2)
		_, _ = g.AddEdge(vD, vE)

		newG, err := graphs.Giant(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(3), newG.Order()) // A, B, C should remain

		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.NotNil(t, newG.GetVertexByID("B"))
		assert.NotNil(t, newG.GetVertexByID("C"))
		assert.Nil(t, newG.GetVertexByID("D"))
		assert.Nil(t, newG.GetVertexByID("E"))
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		newG, err := graphs.Giant(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), newG.Order())
	})
}
