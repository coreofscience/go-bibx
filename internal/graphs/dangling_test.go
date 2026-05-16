package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestRemoveDangling(t *testing.T) {
	t.Run("with dangling vertices", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC) // C is a dangling vertex (outDegree=0, inDegree=1)

		newG, err := graphs.RemoveDangling(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(2), newG.Order())
		assert.Equal(t, uint32(1), newG.Size())

		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.NotNil(t, newG.GetVertexByID("B"))
		assert.Nil(t, newG.GetVertexByID("C"))
	})

	t.Run("without dangling vertices", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)
		_, _ = g.AddEdge(vC, vA) // Cycle, no dangling vertices

		newG, err := graphs.RemoveDangling(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(3), newG.Order())
		assert.Equal(t, uint32(3), newG.Size())
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		newG, err := graphs.RemoveDangling(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), newG.Order())
	})
}
