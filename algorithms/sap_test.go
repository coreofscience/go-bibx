package algorithms_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/algorithms"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestSAP(t *testing.T) {
	t.Run("line shape", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())

		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		sap := algorithms.NewSap(g)
		result := sap.Run()

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

		sap := algorithms.NewSap(g)
		result := sap.Run()

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
}
