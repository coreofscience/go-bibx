package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestPurge(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		newG, err := graphs.Purge(g, []string{"B"})
		assert.NoError(t, err)
		assert.Equal(t, uint32(2), newG.Order())
		assert.Equal(t, uint32(0), newG.Size())

		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.NotNil(t, newG.GetVertexByID("C"))
		assert.Nil(t, newG.GetVertexByID("B"))
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

		newG, err := graphs.Purge(g, []string{"B"})
		assert.NoError(t, err)
		assert.Equal(t, uint32(2), newG.Order())
		assert.Equal(t, uint32(0), newG.Size()) // Edge from A to B should be gone

		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.NotNil(t, newG.GetVertexByID("C")) // Isolated vertex C should remain
		assert.Nil(t, newG.GetVertexByID("B"))
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		newG, err := graphs.Purge(g, []string{"A"})
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), newG.Order())
	})
}
