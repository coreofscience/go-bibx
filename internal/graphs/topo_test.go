package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestReverseTopologicalOrder(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		// A -> B -> C
		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		order, err := graphs.ReverseTopologicalOrder(g)
		assert.NoError(t, err)
		assert.Len(t, order, 3)

		// The topological order should be A, B, C.
		// Reverse should be C, B, A.
		assert.Equal(t, "C", order[0].Label())
		assert.Equal(t, "B", order[1].Label())
		assert.Equal(t, "A", order[2].Label())
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		order, err := graphs.ReverseTopologicalOrder(g)
		assert.NoError(t, err)
		assert.Empty(t, order)
	})

	t.Run("undirected graph error", func(t *testing.T) {
		g := gograph.New[string]() // Undirected by default
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		_, _ = g.AddEdge(vA, vB) // undirected edge is like a cycle

		order, err := graphs.ReverseTopologicalOrder(g)
		assert.Error(t, err)
		assert.Nil(t, order)
	})

	t.Run("cyclic graph error", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vA)

		order, err := graphs.ReverseTopologicalOrder(g)
		assert.Error(t, err)
		assert.Nil(t, order)
	})
}
