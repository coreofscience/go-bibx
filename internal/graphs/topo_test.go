package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestReverseTopologicalOrder(t *testing.T) {
	g := gograph.New[string](gograph.Directed())

	vA := gograph.NewVertex("A")
	vB := gograph.NewVertex("B")
	vC := gograph.NewVertex("C")
	vD := gograph.NewVertex("D")

	// Dependencies:
	// A depends on nothing
	// B depends on A
	// C depends on B
	// D depends on C
	_, _ = g.AddEdge(vA, vB)
	_, _ = g.AddEdge(vB, vC)
	_, _ = g.AddEdge(vC, vD)

	order, err := graphs.ReverseTopologicalOrder(g)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(order))

	// In gograph, topological iterator visits from roots to leaves.
	// We expect D, C, B, A in reverse order.
	labels := make([]string, len(order))
	for i, v := range order {
		labels[i] = v.Label()
	}

	// Since there's only one valid topological path A -> B -> C -> D, the reverse must be D, C, B, A
	assert.Equal(t, []string{"D", "C", "B", "A"}, labels)
}
