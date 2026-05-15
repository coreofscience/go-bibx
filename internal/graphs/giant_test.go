package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestGiant_Undirected(t *testing.T) {
	g := gograph.New[string]()

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")

	vD := gograph.NewVertex("D")
	vE := gograph.NewVertex("E")

	// Component 1: A-B-C
	_, _ = g.AddEdge(vA, vB)
	_, _ = g.AddEdge(vB, vC)

	// Component 2: D-E
	_, _ = g.AddEdge(vD, vE)

	newGraph, err := graphs.Giant(g)
	assert.NoError(t, err)

	assert.Equal(t, uint32(3), newGraph.Order())
	assert.Equal(t, 4, len(newGraph.AllEdges())) // Undirected creates 2 directed edges per connection

	labels := make(map[string]bool)
	for _, v := range newGraph.GetAllVertices() {
		labels[v.Label()] = true
	}
	assert.True(t, labels["A"])
	assert.True(t, labels["B"])
	assert.True(t, labels["C"])
	assert.False(t, labels["D"])
}

func TestGiant_Directed(t *testing.T) {
	g := gograph.New[string](gograph.Directed())

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")

	vD := gograph.NewVertex("D")
	vE := gograph.NewVertex("E")

	// SCC 1: A -> B -> C -> A
	_, _ = g.AddEdge(vA, vB)
	_, _ = g.AddEdge(vB, vC)
	_, _ = g.AddEdge(vC, vA)

	// SCC 2: D -> E -> D
	_, _ = g.AddEdge(vD, vE)
	_, _ = g.AddEdge(vE, vD)

	newGraph, err := graphs.Giant(g)
	assert.NoError(t, err)

	assert.Equal(t, uint32(3), newGraph.Order())
	assert.Equal(t, 3, len(newGraph.AllEdges()))

	labels := make(map[string]bool)
	for _, v := range newGraph.GetAllVertices() {
		labels[v.Label()] = true
	}
	assert.True(t, labels["A"])
	assert.True(t, labels["B"])
	assert.True(t, labels["C"])
}

func TestGiant_Empty(t *testing.T) {
	g := gograph.New[string]()
	newGraph, err := graphs.Giant(g)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), newGraph.Order())
}
