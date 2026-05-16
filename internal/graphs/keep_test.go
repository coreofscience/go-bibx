package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestKeep(t *testing.T) {
	g := gograph.New[string](gograph.Directed())

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")
	vD := gograph.NewVertex("D")

	_, _ = g.AddEdge(vA, vB)
	_, _ = g.AddEdge(vB, vC)
	_, _ = g.AddEdge(vC, vD)

	toKeep := []string{"B", "C"}
	newGraph, err := graphs.Keep(g, toKeep)
	assert.NoError(t, err)

	// Keep returns a graph with ONLY the toKeep vertices AND their interconnecting edges.

	// Edges should only be B -> C
	assert.Equal(t, 1, len(newGraph.AllEdges()))
	assert.Equal(t, uint32(2), newGraph.Order())

	edge := newGraph.AllEdges()[0]
	assert.Equal(t, "B", edge.Source().Label())
	assert.Equal(t, "C", edge.Destination().Label())
}

func TestKeep_Isolated(t *testing.T) {
	g := gograph.New[string](gograph.Directed())

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	_, _ = g.AddEdge(vA, vB)

	// Keep "A". Since A->B can't be added (B is not in toKeep), A won't have edges.
	newGraph, err := graphs.Keep(g, []string{"A"})
	assert.NoError(t, err)

	assert.Equal(t, uint32(1), newGraph.Order())
	assert.Equal(t, 0, len(newGraph.AllEdges()))
	assert.Equal(t, "A", newGraph.GetAllVertices()[0].Label())
}
