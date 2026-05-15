package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestUndirected_Directed(t *testing.T) {
	g := gograph.New[string](gograph.Directed())

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")

	g.AddEdge(vA, vB)
	g.AddEdge(vB, vC)

	undirectedGraph, err := graphs.Undirected(g)
	assert.NoError(t, err)

	assert.False(t, undirectedGraph.IsDirected())

	// Edges will be added to the undirected graph, making 2 "pairs" of edges but it's an undirected graph instance.
	// `gograph` undirected graphs store two edges per connection internally.
	assert.Equal(t, uint32(3), undirectedGraph.Order())

	// A undirected graph in gograph typically has len(AllEdges) as 2 * number_of_undirected_connections.
	assert.Equal(t, 4, len(undirectedGraph.AllEdges()))

	vAUn := gograph.NewVertex("A")
	vBUn := gograph.NewVertex("B")
	vCUn := gograph.NewVertex("C")

	// Both directions should be traversable/exist
	assert.NotNil(t, undirectedGraph.GetEdge(vAUn, vBUn))
	assert.NotNil(t, undirectedGraph.GetEdge(vBUn, vAUn))

	assert.NotNil(t, undirectedGraph.GetEdge(vBUn, vCUn))
	assert.NotNil(t, undirectedGraph.GetEdge(vCUn, vBUn))
}

func TestUndirected_AlreadyUndirected(t *testing.T) {
	g := gograph.New[string]()

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")

	g.AddEdge(vA, vB)

	undirectedGraph, err := graphs.Undirected(g)
	assert.NoError(t, err)
	assert.Same(t, g, undirectedGraph)
}
