package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestInvert_Directed(t *testing.T) {
	g := gograph.New[string](gograph.Directed())

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")

	// A -> B
	// B -> C
	_, _ = g.AddEdge(vA, vB)
	_, _ = g.AddEdge(vB, vC)

	inverted := graphs.Invert(g)

	assert.Equal(t, uint32(3), inverted.Order())
	assert.Equal(t, 2, len(inverted.AllEdges()))

	vAInv := gograph.NewVertex("A")
	vBInv := gograph.NewVertex("B")
	vCInv := gograph.NewVertex("C")

	// Check if edges are B -> A and C -> B
	assert.NotNil(t, inverted.GetEdge(vBInv, vAInv)) // from B to A
	assert.NotNil(t, inverted.GetEdge(vCInv, vBInv)) // from C to B

	// Original edges should not exist
	assert.Nil(t, inverted.GetEdge(vAInv, vBInv))
	assert.Nil(t, inverted.GetEdge(vBInv, vCInv))
}

func TestInvert_Undirected(t *testing.T) {
	g := gograph.New[string]()

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")

	_, _ = g.AddEdge(vA, vB)

	inverted := graphs.Invert(g)

	// Should be the exact same graph instance
	assert.Same(t, g, inverted)
}
