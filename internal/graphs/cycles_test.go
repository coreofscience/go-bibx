package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestRemoveCycles(t *testing.T) {
	t.Run("DAG", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		newG, err := graphs.RemoveCycles(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(3), newG.Order())
		assert.Equal(t, uint32(2), newG.Size())
	})

	t.Run("graph with cycles", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")
		vD := gograph.NewVertex("D")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)
		_, _ = g.AddEdge(vC, vA) // Cycle A -> B -> C -> A
		_, _ = g.AddEdge(vD, vA)

		newG, err := graphs.RemoveCycles(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(1), newG.Order()) // A, B, C are removed, only D remains
		assert.Equal(t, uint32(0), newG.Size())

		assert.NotNil(t, newG.GetVertexByID("D"))
	})

	t.Run("self loop", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vB) // Self loop on B

		newG, err := graphs.RemoveCycles(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(1), newG.Order())
		assert.Equal(t, uint32(0), newG.Size())
		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.Nil(t, newG.GetVertexByID("B"))
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		newG, err := graphs.RemoveCycles(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), newG.Order())
	})

	t.Run("undirected graph error", func(t *testing.T) {
		g := gograph.New[string]()
		vA := gograph.NewVertex("A")
		g.AddVertex(vA)

		newG, err := graphs.RemoveCycles(g)
		assert.Error(t, err)
		assert.Nil(t, newG)
		assert.Contains(t, err.Error(), "must be directed")
	})
}
