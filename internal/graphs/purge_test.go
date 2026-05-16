package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestPurge(t *testing.T) {
	g := gograph.New[string](gograph.Directed())

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")
	vD := gograph.NewVertex("D")

	_, _ = g.AddEdge(vA, vB)
	_, _ = g.AddEdge(vB, vC)
	_, _ = g.AddEdge(vC, vD)

	toRemove := []string{"B", "D"}
	newGraph, err := graphs.Purge(g, toRemove)
	assert.NoError(t, err)

	// Since we drop B and D:
	// Edges initially: A->B, B->C, C->D
	// B is removed -> A->B and B->C are removed.
	// D is removed -> C->D is removed.
	// So newGraph should have 0 edges.
	// However, we still have A and C left as isolated vertices.
	assert.Equal(t, uint32(2), newGraph.Order())
	assert.Equal(t, 0, len(newGraph.AllEdges()))

	labels := make(map[string]bool)
	for _, v := range newGraph.GetAllVertices() {
		labels[v.Label()] = true
	}
	assert.True(t, labels["A"])
	assert.True(t, labels["C"])
}

func TestPurge_KeepsIntactConnections(t *testing.T) {
	g := gograph.New[string](gograph.Directed())

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")
	vD := gograph.NewVertex("D")

	_, _ = g.AddEdge(vA, vB)
	_, _ = g.AddEdge(vA, vC) // We want to keep A->C
	_, _ = g.AddEdge(vB, vD)

	toRemove := []string{"B", "D"}
	newGraph, err := graphs.Purge(g, toRemove)
	assert.NoError(t, err)

	// A->C remains. Order should be 2.
	assert.Equal(t, uint32(2), newGraph.Order())
	assert.Equal(t, 1, len(newGraph.AllEdges()))

	edge := newGraph.AllEdges()[0]
	assert.Equal(t, "A", edge.Source().Label())
	assert.Equal(t, "C", edge.Destination().Label())
}
