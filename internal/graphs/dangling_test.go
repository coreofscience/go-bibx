package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestRemoveDangling(t *testing.T) {
	g := gograph.New[string](gograph.Directed())

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")
	vD := gograph.NewVertex("D")

	// A -> B -> C -> D
	// C is dangling if we only look at A->B->C, but D has out-degree 0 and in-degree 1.
	// Actually dangling: out-degree == 0, in-degree == 1.
	_, _ = g.AddEdge(vA, vB)
	_, _ = g.AddEdge(vB, vC)
	_, _ = g.AddEdge(vC, vD)

	newGraph, err := graphs.RemoveDangling(g)
	assert.NoError(t, err)

	// D is dangling (in-degree 1, out-degree 0), so it should be removed.
	// A -> B -> C remains.
	assert.Equal(t, uint32(3), newGraph.Order())
	assert.Equal(t, 2, len(newGraph.AllEdges()))

	for _, v := range newGraph.GetAllVertices() {
		assert.NotEqual(t, "D", v.Label())
	}
}

func TestRemoveDangling_NoDangling(t *testing.T) {
	g := gograph.New[string](gograph.Directed())

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")

	// A -> B -> C -> A
	_, _ = g.AddEdge(vA, vB)
	_, _ = g.AddEdge(vB, vC)
	_, _ = g.AddEdge(vC, vA)

	newGraph, err := graphs.RemoveDangling(g)
	assert.NoError(t, err)

	assert.Equal(t, uint32(3), newGraph.Order())
	assert.Equal(t, 3, len(newGraph.AllEdges()))
}
