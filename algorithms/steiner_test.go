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

	t.Run("consecutive exceeded patience finds subsequent descendant", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Acyclic(), gograph.Weighted())

		vT1 := gograph.NewVertex("T1")
		vT2 := gograph.NewVertex("T2")
		vD1 := gograph.NewVertex("D1")
		vD2 := gograph.NewVertex("D2")

		// Edge D1 -> D2 forces D1 to appear before D2 in topological order
		_, _ = g.AddEdge(vD1, vD2, gograph.WithEdgeWeight(1.0))

		// D1 is high cost: 5.0 + 5.0 = 10.0 > maxEffort(3.0)
		_, _ = g.AddEdge(vT1, vD1, gograph.WithEdgeWeight(5.0))
		_, _ = g.AddEdge(vT2, vD1, gograph.WithEdgeWeight(5.0))

		// D2 is low cost: 1.0 + 1.0 = 2.0 <= maxEffort(3.0)
		_, _ = g.AddEdge(vT1, vD2, gograph.WithEdgeWeight(1.0))
		_, _ = g.AddEdge(vT2, vD2, gograph.WithEdgeWeight(1.0))

		pseudoSteiner, err := algorithms.NewPseudoSteiner(
			g,
			algorithms.WithMaxEffort[string](3.0),
			algorithms.WithMaxEffortExceeded[string](3),
		)
		require.NoError(t, err)
		assert.NotNil(t, pseudoSteiner)

		newGraph, err := pseudoSteiner.Run([]string{"T1", "T2"})
		require.NoError(t, err)
		assert.NotNil(t, newGraph)
		// Should successfully bridge using D2
		assert.Equal(t, uint32(3), newGraph.Order())
		assert.Equal(t, uint32(2), newGraph.Size())
	})

	t.Run("consecutive exceeded patience aborts when limit reached", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Acyclic(), gograph.Weighted())

		vT1 := gograph.NewVertex("T1")
		vT2 := gograph.NewVertex("T2")
		vD1 := gograph.NewVertex("D1")
		vD2 := gograph.NewVertex("D2")
		vD3 := gograph.NewVertex("D3")

		// Force topological order D1 -> D2 -> D3
		_, _ = g.AddEdge(vD1, vD2, gograph.WithEdgeWeight(1.0))
		_, _ = g.AddEdge(vD2, vD3, gograph.WithEdgeWeight(1.0))

		// D1 and D2 have cost 10.0 > maxEffort(3.0)
		_, _ = g.AddEdge(vT1, vD1, gograph.WithEdgeWeight(5.0))
		_, _ = g.AddEdge(vT2, vD1, gograph.WithEdgeWeight(5.0))
		_, _ = g.AddEdge(vT1, vD2, gograph.WithEdgeWeight(5.0))
		_, _ = g.AddEdge(vT2, vD2, gograph.WithEdgeWeight(5.0))

		// D3 has low cost 2.0 <= maxEffort(3.0)
		_, _ = g.AddEdge(vT1, vD3, gograph.WithEdgeWeight(1.0))
		_, _ = g.AddEdge(vT2, vD3, gograph.WithEdgeWeight(1.0))

		// With maxEffortExceeded = 2, encountering D1 and D2 (2 consecutive > maxEffort) aborts before D3
		pseudoSteiner, err := algorithms.NewPseudoSteiner(
			g,
			algorithms.WithMaxEffort[string](3.0),
			algorithms.WithMaxEffortExceeded[string](2),
		)
		require.NoError(t, err)
		assert.NotNil(t, pseudoSteiner)

		newGraph, err := pseudoSteiner.Run([]string{"T1", "T2"})
		require.NoError(t, err)
		assert.NotNil(t, newGraph)
		// Fails to bridge, returns disjoint terminals (order 2, size 0)
		assert.Equal(t, uint32(2), newGraph.Order())
		assert.Equal(t, uint32(0), newGraph.Size())
	})

	t.Run("consecutive exceeded patience finds subsequent ancestor", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Acyclic(), gograph.Weighted())

		vA1 := gograph.NewVertex("A1")
		vA2 := gograph.NewVertex("A2")
		vT1 := gograph.NewVertex("T1")
		vT2 := gograph.NewVertex("T2")

		// Force topological order A1 -> A2 -> T1/T2
		// When searching backward from terminals, A2 is checked first, then A1
		_, _ = g.AddEdge(vA1, vA2, gograph.WithEdgeWeight(1.0))

		// A2 (checked first) is high cost: 5.0 + 5.0 = 10.0 > maxEffort(3.0)
		_, _ = g.AddEdge(vA2, vT1, gograph.WithEdgeWeight(5.0))
		_, _ = g.AddEdge(vA2, vT2, gograph.WithEdgeWeight(5.0))

		// A1 (checked second) is low cost: 1.0 + 1.0 = 2.0 <= maxEffort(3.0)
		_, _ = g.AddEdge(vA1, vT1, gograph.WithEdgeWeight(1.0))
		_, _ = g.AddEdge(vA1, vT2, gograph.WithEdgeWeight(1.0))

		pseudoSteiner, err := algorithms.NewPseudoSteiner(
			g,
			algorithms.WithMaxEffort[string](3.0),
			algorithms.WithMaxEffortExceeded[string](3),
		)
		require.NoError(t, err)
		assert.NotNil(t, pseudoSteiner)

		newGraph, err := pseudoSteiner.Run([]string{"T1", "T2"})
		require.NoError(t, err)
		assert.NotNil(t, newGraph)
		// Should successfully bridge using A1
		assert.Equal(t, uint32(3), newGraph.Order())
		assert.Equal(t, uint32(2), newGraph.Size())
	})

	t.Run("duplicate terminals in input are handled cleanly", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Acyclic())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		_, _ = g.AddEdge(vA, vB)

		pseudoSteiner, err := algorithms.NewPseudoSteiner(g)
		require.NoError(t, err)
		assert.NotNil(t, pseudoSteiner)

		newGraph, err := pseudoSteiner.Run([]string{"A", "A", "B", "B"})
		require.NoError(t, err)
		assert.NotNil(t, newGraph)
		assert.Equal(t, uint32(2), newGraph.Order())
	})

	t.Run("existing terminals are excluded as bridging candidates to avoid infinite loop", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed(), gograph.Acyclic())

		vT1 := gograph.NewVertex("T1")
		vT2 := gograph.NewVertex("T2")
		vT3 := gograph.NewVertex("T3")

		// T1 -> T3 and T2 -> T3
		// T3 is a common descendant of T1 and T2, but T3 is already a terminal!
		_, _ = g.AddEdge(vT1, vT3)
		_, _ = g.AddEdge(vT2, vT3)

		pseudoSteiner, err := algorithms.NewPseudoSteiner(g)
		require.NoError(t, err)
		assert.NotNil(t, pseudoSteiner)

		// Terminals include T1, T2, T3
		newGraph, err := pseudoSteiner.Run([]string{"T1", "T2", "T3"})
		require.NoError(t, err)
		assert.NotNil(t, newGraph)
		// Should contain exactly the 3 terminals without infinite recursion
		assert.Equal(t, uint32(3), newGraph.Order())
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
