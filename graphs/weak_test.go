package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestWeaklyConnectedComponents_Undirected(t *testing.T) {
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

	components, err := graphs.WeaklyConnectedComponents(g)
	assert.NoError(t, err)
	assert.Len(t, components, 2)

	// Check if components are correct
	foundABC := false
	foundDE := false

	for _, comp := range components {
		if len(comp) == 3 {
			assert.Subset(t, []*gograph.Vertex[string]{vA, vB, vC}, comp)
			foundABC = true
		} else if len(comp) == 2 {
			assert.Subset(t, []*gograph.Vertex[string]{vD, vE}, comp)
			foundDE = true
		}
	}

	assert.True(t, foundABC, "Should have found component {A, B, C}")
	assert.True(t, foundDE, "Should have found component {D, E}")
}

func TestWeaklyConnectedComponents_DirectedError(t *testing.T) {
	g := gograph.New[string](gograph.Directed())
	v1 := gograph.NewVertex("A")
	v2 := gograph.NewVertex("B")
	_, _ = g.AddEdge(v1, v2)

	components, err := graphs.WeaklyConnectedComponents(g)
	assert.Error(t, err)
	assert.Nil(t, components)
	assert.Contains(t, err.Error(), "graph is directed")
}

func TestWeaklyConnectedComponents_Empty(t *testing.T) {
	g := gograph.New[string]()
	components, err := graphs.WeaklyConnectedComponents(g)
	assert.NoError(t, err)
	assert.Empty(t, components)
}

func TestWeaklyConnectedComponents_SingleNode(t *testing.T) {
	g := gograph.New[string]()
	vA := gograph.NewVertex("A")
	g.AddVertex(vA)

	components, err := graphs.WeaklyConnectedComponents(g)
	assert.NoError(t, err)
	assert.Len(t, components, 1)
	assert.ElementsMatch(t, []*gograph.Vertex[string]{vA}, components[0])
}
