package algorithms_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/coreofscience/go-bibx/algorithms"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSAP(t *testing.T) {
	t.Run("line shape", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())

		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		sap, err := algorithms.NewSapAlgorithm(g)
		assert.NoError(t, err)
		result, err := sap.Run()

		assert.NoError(t, err)
		assert.Equal(t, &algorithms.SapResult{
			Categories: map[string]algorithms.Category{
				"A": algorithms.CategoryLeaf,
				"B": algorithms.CategoryTrunk,
				"C": algorithms.CategoryRoot,
			},
			Rootness: map[string]float64{
				"C": 1.0,
			},
			Trunkness: map[string]float64{
				"B": 2.0,
			},
			Leafness: map[string]float64{
				"A": 1.0,
			},
		}, result)
	})
	t.Run("diamond shape", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())

		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")
		vD := gograph.NewVertex("D")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vA, vC)
		_, _ = g.AddEdge(vB, vD)
		_, _ = g.AddEdge(vC, vD)

		sap, err := algorithms.NewSapAlgorithm(g)
		assert.NoError(t, err)
		result, err := sap.Run()

		assert.NoError(t, err)
		assert.Equal(t, &algorithms.SapResult{
			Categories: map[string]algorithms.Category{
				"A": algorithms.CategoryLeaf,
				"B": algorithms.CategoryTrunk,
				"C": algorithms.CategoryTrunk,
				"D": algorithms.CategoryRoot,
			},
			Rootness: map[string]float64{
				"D": 2.0,
			},
			Trunkness: map[string]float64{
				"B": 4.0,
				"C": 4.0,
			},
			Leafness: map[string]float64{
				"A": 2.0,
			},
		}, result)
	})
	t.Run("undirected graph error", func(t *testing.T) {
		g := gograph.New[string]() // Undirected by default

		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		_, _ = g.AddEdge(vA, vB)

		sap, err := algorithms.NewSapAlgorithm(g)
		assert.Error(t, err)
		assert.Nil(t, sap)
		assert.Equal(t, "graph must be directed", err.Error())
	})
	t.Run("cycle error", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())

		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vA)

		sap, err := algorithms.NewSapAlgorithm(g)
		assert.Error(t, err)
		assert.Nil(t, sap)
	})
	t.Run("deep diamond chain does not overflow int64 path counts", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		const depth = 65 // 2^65 paths exceeds MaxInt64

		leaf := gograph.NewVertex("L")
		root := gograph.NewVertex("R")

		prevA := leaf
		prevB := leaf
		for i := 0; i < depth; i++ {
			curA := gograph.NewVertex(fmt.Sprintf("A_%d", i))
			curB := gograph.NewVertex(fmt.Sprintf("B_%d", i))
			_, _ = g.AddEdge(prevA, curA)
			_, _ = g.AddEdge(prevA, curB)
			if prevB != prevA {
				_, _ = g.AddEdge(prevB, curA)
				_, _ = g.AddEdge(prevB, curB)
			}
			prevA = curA
			prevB = curB
		}
		_, _ = g.AddEdge(prevA, root)
		_, _ = g.AddEdge(prevB, root)

		sap, err := algorithms.NewSapAlgorithm(g)
		require.NoError(t, err)
		result, err := sap.Run()
		require.NoError(t, err)

		expectedPaths := math.Pow(2, float64(depth))
		assert.Equal(t, expectedPaths, result.Leafness["L"])
		assert.Equal(t, algorithms.CategoryLeaf, result.Categories["L"])
		assert.Equal(t, algorithms.CategoryRoot, result.Categories["R"])
		for _, score := range result.Trunkness {
			assert.Greater(t, score, 0.0)
		}
	})
	t.Run("isolated vertices and unselected roots do not store zero scores", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())

		// Main component connected to top root R1 (in-degree 2)
		l1 := gograph.NewVertex("L1")
		l2 := gograph.NewVertex("L2")
		t1 := gograph.NewVertex("T1")
		r1 := gograph.NewVertex("R1")
		_, _ = g.AddEdge(l1, t1)
		_, _ = g.AddEdge(l2, r1)
		_, _ = g.AddEdge(t1, r1)

		// Secondary component connected only to unselected root R2 (in-degree 1)
		l3 := gograph.NewVertex("L3")
		r2 := gograph.NewVertex("R2")
		_, _ = g.AddEdge(l3, r2)

		// Isolated vertex (in-degree 0, out-degree 0)
		iso := gograph.NewVertex("ISO")
		g.AddVertex(iso)

		sap, err := algorithms.NewSapAlgorithm(g, algorithms.WithRoots(1))
		require.NoError(t, err)
		result, err := sap.Run()
		require.NoError(t, err)

		assert.NotContains(t, result.Rootness, "ISO")
		assert.NotContains(t, result.Leafness, "L3")
		assert.Equal(t, algorithms.CategoryNone, result.Categories["ISO"])
		assert.Equal(t, algorithms.CategoryNone, result.Categories["L3"])
		assert.Equal(t, algorithms.CategoryRoot, result.Categories["R1"])
		assert.Equal(t, algorithms.CategoryTrunk, result.Categories["T1"])
	})
}
