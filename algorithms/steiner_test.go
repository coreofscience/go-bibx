package algorithms_test

import (
	"slices"
	"testing"

	"github.com/coreofscience/go-bibx/algorithms"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPseudoSteiner(t *testing.T) {
	t.Run("sources not in graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Acyclic())
		g.AddVertex(gograph.NewVertex("A"))
		g.AddVertex(gograph.NewVertex("B"))
		g.AddVertex(gograph.NewVertex("C"))
		pseudoSteiner, err := algorithms.NewPseudoSteiner(
			g,
			algorithms.WithMaxEffort[string](10),
		)
		require.NoError(t, err)
		assert.NotNil(t, pseudoSteiner)
		newGraph, err := pseudoSteiner.Run([]string{"D"})
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

		pseudoSteiner, err := algorithms.NewPseudoSteiner(
			g,
			algorithms.WithMaxEffort[string](10),
		)
		require.NoError(t, err)
		assert.NotNil(t, pseudoSteiner)
		newGraph, err := pseudoSteiner.Run([]string{"B", "C"})
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

		pseudoSteiner, err := algorithms.NewPseudoSteiner(
			g,
			algorithms.WithMaxEffort[string](10),
		)
		require.NoError(t, err)
		assert.NotNil(t, pseudoSteiner)
		newGraph, err := pseudoSteiner.Run([]string{"B", "C"})
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

		pseudoSteiner, err := algorithms.NewPseudoSteiner(
			g,
			algorithms.WithMaxEffort[string](10),
		)
		require.NoError(t, err)
		assert.NotNil(t, pseudoSteiner)
		newGraph, err := pseudoSteiner.Run([]string{"A", "C"})
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

		pseudoSteiner, err := algorithms.NewPseudoSteiner(
			g,
			algorithms.WithMaxEffort[string](10),
		)
		require.NoError(t, err)
		assert.NotNil(t, pseudoSteiner)
		newGraph, err := pseudoSteiner.Run([]string{"A", "C"})
		require.NoError(t, err)
		assert.NotNil(t, newGraph)
		assert.Equal(t, uint32(2), newGraph.Order())
		assert.Equal(t, uint32(0), newGraph.Size())
	})
}

func TestReverseSlices(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	b := a[:3]
	slices.Reverse(b)
	assert.Equal(t, []int{3, 2, 1, 4, 5}, a)
}

func TestReverseSlicesPreservingOriginalOrdering(t *testing.T) {
	a := []int{1, 2, 3, 4, 5}
	b := make([]int, 3)
	copy(b, a[:3])
	slices.Reverse(b)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, a)
	assert.Equal(t, []int{3, 2, 1}, b)
}
