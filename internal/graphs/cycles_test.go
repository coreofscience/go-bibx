package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestRemoveCycles_Undirected(t *testing.T) {
	g := gograph.New[string]()
	v1 := gograph.NewVertex("A")
	v2 := gograph.NewVertex("B")
	_, _ = g.AddEdge(v1, v2)

	_, err := graphs.RemoveCycles(g)
	assert.ErrorContains(t, err, "input graph must be directed")
}

func TestRemoveCycles_NoCycles(t *testing.T) {
	g := gograph.New[string](gograph.Directed())
	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")

	_, _ = g.AddEdge(vA, vB)
	_, _ = g.AddEdge(vB, vC)

	newGraph, err := graphs.RemoveCycles(g)
	assert.NoError(t, err)
	assert.Equal(t, uint32(3), newGraph.Order())
	assert.Equal(t, 2, len(newGraph.AllEdges()))
}

func TestRemoveCycles_SelfLoop(t *testing.T) {
	g := gograph.New[string](gograph.Directed())
	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")

	_, _ = g.AddEdge(vA, vA) // Self-loop
	_, _ = g.AddEdge(vA, vB)

	newGraph, err := graphs.RemoveCycles(g)
	assert.NoError(t, err)
	// B is left without edges, but Purge now keeps isolated vertices
	assert.Equal(t, uint32(1), newGraph.Order())
	assert.Equal(t, 0, len(newGraph.AllEdges()))
	assert.NotEmpty(t, newGraph.GetAllVertices())
	assert.Equal(t, "B", newGraph.GetAllVertices()[0].Label())
}

func TestRemoveCycles_Cycle(t *testing.T) {
	g := gograph.New[string](gograph.Directed())
	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")
	vD := gograph.NewVertex("D") // Not in cycle
	vE := gograph.NewVertex("E") // To keep D, we can add edge D -> E

	_, _ = g.AddEdge(vA, vB)
	_, _ = g.AddEdge(vB, vC)
	_, _ = g.AddEdge(vC, vA) // Cycle A -> B -> C -> A
	_, _ = g.AddEdge(vC, vD)
	_, _ = g.AddEdge(vD, vE) // D -> E is not in cycle

	newGraph, err := graphs.RemoveCycles(g)
	assert.NoError(t, err)
	assert.Equal(t, uint32(2), newGraph.Order()) // A, B, C removed, D and E kept
	assert.Equal(t, 1, len(newGraph.AllEdges()))
}
