package algorithms_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/algorithms"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuasiStainer(t *testing.T) {
	t.Run("sources not in graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Acyclic())
		g.AddVertex(gograph.NewVertex("A"))
		g.AddVertex(gograph.NewVertex("B"))
		g.AddVertex(gograph.NewVertex("C"))
		quasiStainer, err := algorithms.NewPseudoStainer(g)
		require.NoError(t, err)
		assert.NotNil(t, quasiStainer)
		newGraph, err := quasiStainer.Run([]string{"D"})
		require.Error(t, err)
		assert.Nil(t, newGraph)
	})

	t.Run("common descendant", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Acyclic())

		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vB, vA)
		_, _ = g.AddEdge(vC, vA)

		quasiStainer, err := algorithms.NewPseudoStainer(g)
		require.NoError(t, err)
		assert.NotNil(t, quasiStainer)
		newGraph, err := quasiStainer.Run([]string{"B", "C"})
		require.NoError(t, err)
		assert.NotNil(t, newGraph)
		assert.Equal(t, uint32(3), newGraph.Order())
		assert.Equal(t, uint32(2), newGraph.Size())
	})

	t.Run("common ancestor", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Acyclic())

		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vA, vC)

		quasiStainer, err := algorithms.NewPseudoStainer(g)
		require.NoError(t, err)
		assert.NotNil(t, quasiStainer)
		newGraph, err := quasiStainer.Run([]string{"B", "C"})
		require.NoError(t, err)
		assert.NotNil(t, newGraph)
		assert.Equal(t, uint32(3), newGraph.Order())
		assert.Equal(t, uint32(2), newGraph.Size())
	})

	t.Run("intermediate", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Acyclic())

		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		quasiStainer, err := algorithms.NewPseudoStainer(g)
		require.NoError(t, err)
		assert.NotNil(t, quasiStainer)
		newGraph, err := quasiStainer.Run([]string{"A", "C"})
		require.NoError(t, err)
		assert.NotNil(t, newGraph)
		assert.Equal(t, uint32(3), newGraph.Order())
		assert.Equal(t, uint32(2), newGraph.Size())
	})

	t.Run("unrelated", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Acyclic())

		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		g.AddVertex(vC)
		_, _ = g.AddEdge(vA, vB)

		quasiStainer, err := algorithms.NewPseudoStainer(g)
		require.NoError(t, err)
		assert.NotNil(t, quasiStainer)
		newGraph, err := quasiStainer.Run([]string{"A", "C"})
		require.NoError(t, err)
		assert.NotNil(t, newGraph)
		assert.Equal(t, uint32(2), newGraph.Order())
		assert.Equal(t, uint32(0), newGraph.Size())
	})
}
