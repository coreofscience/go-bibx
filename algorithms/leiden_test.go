package algorithms_test

import (
	"github.com/coreofscience/go-bibx/algorithms"
)

import (
	"testing"

	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestLeiden_UnweightedGraph(t *testing.T) {
	// Create two triangles connected by a single edge
	g := gograph.New[string]()

	nodes := []string{"A", "B", "C", "D", "E", "F"}
	for _, n := range nodes {
		g.AddVertexByLabel(n)
	}

	// Triangle 1
	_ , _ = g.AddEdge(g.GetVertexByID("A"), g.GetVertexByID("B"))
	_ , _ = g.AddEdge(g.GetVertexByID("B"), g.GetVertexByID("C"))
	_ , _ = g.AddEdge(g.GetVertexByID("C"), g.GetVertexByID("A"))

	// Triangle 2
	_ , _ = g.AddEdge(g.GetVertexByID("D"), g.GetVertexByID("E"))
	_ , _ = g.AddEdge(g.GetVertexByID("E"), g.GetVertexByID("F"))
	_ , _ = g.AddEdge(g.GetVertexByID("F"), g.GetVertexByID("D"))

	// Bridge
	_ , _ = g.AddEdge(g.GetVertexByID("C"), g.GetVertexByID("D"))

	leiden := algorithms.NewLeiden(g)
	partition := leiden.Run()

	assert.NotNil(t, partition)

	// A, B, C should be in one community
	// D, E, F should be in another community
	assert.Equal(t, partition["A"], partition["B"])
	assert.Equal(t, partition["B"], partition["C"])

	assert.Equal(t, partition["D"], partition["E"])
	assert.Equal(t, partition["E"], partition["F"])

	assert.NotEqual(t, partition["A"], partition["D"], "The two triangles should be in different communities")
}

func TestLeiden_WeightedGraph(t *testing.T) {
	// Create a graph where weights heavily favor a certain split
	g := gograph.New[string]( gograph.Weighted())

	nodes := []string{"1", "2", "3", "4"}
	for _, n := range nodes {
		g.AddVertexByLabel(n)
	}

	// Strong connection 1-2
	_ , _ = g.AddEdge(g.GetVertexByID("1"), g.GetVertexByID("2"), gograph.WithEdgeWeight(10.0))
	// Strong connection 3-4
	_ , _ = g.AddEdge(g.GetVertexByID("3"), g.GetVertexByID("4"), gograph.WithEdgeWeight(10.0))

	// Weak connections
	_ , _ = g.AddEdge(g.GetVertexByID("2"), g.GetVertexByID("3"), gograph.WithEdgeWeight(1.0))
	_ , _ = g.AddEdge(g.GetVertexByID("1"), g.GetVertexByID("4"), gograph.WithEdgeWeight(1.0))

	leiden := algorithms.NewLeiden(g)
	partition := leiden.Run()

	assert.NotNil(t, partition)

	// 1 and 2 should be together
	assert.Equal(t, partition["1"], partition["2"])

	// 3 and 4 should be together
	assert.Equal(t, partition["3"], partition["4"])

	// They should be different communities
	assert.NotEqual(t, partition["1"], partition["3"])
}

func TestLeiden_EmptyGraph(t *testing.T) {
	g := gograph.New[string]()
	leiden := algorithms.NewLeiden(g)
	partition := leiden.Run()
	assert.Nil(t, partition)
}

func TestLeiden_SingleNode(t *testing.T) {
	g := gograph.New[string]()
	g.AddVertexByLabel("A")
	leiden := algorithms.NewLeiden(g)
	partition := leiden.Run()
	assert.NotNil(t, partition)
	assert.Equal(t, 1, len(partition))
	assert.Equal(t, 0, partition["A"])
}
