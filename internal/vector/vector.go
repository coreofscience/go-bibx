package vector

import (
	"cmp"
	"math"
	"sort"
)

type Node[K cmp.Ordered] struct {
	ID     K         `json:"id"`
	Vector []float32 `json:"vector"`
}

type Vectors[K cmp.Ordered] interface {
	Add(node *Node[K]) error
	Search(vector []float32, limit int) ([]*Node[K], error)
}

func NewNode[K cmp.Ordered](id K, vector []float32) *Node[K] {
	return &Node[K]{
		ID:     id,
		Vector: vector,
	}
}

// DumbVectors is a simple in-memory implementation of the Vectors interface
// that performs a linear search over all nodes. This is not optimized for
// performance and should only be used for testing or small datasets.
type DumbVectors[K cmp.Ordered] struct {
	Nodes []*Node[K] `json:"nodes"`
}

func NewDumbVectors[K cmp.Ordered]() *DumbVectors[K] {
	return &DumbVectors[K]{
		Nodes: make([]*Node[K], 0),
	}
}

func (v *DumbVectors[K]) Add(node *Node[K]) error {
	v.Nodes = append(v.Nodes, node)
	return nil
}

func (v *DumbVectors[K]) Search(vector []float32, limit int) ([]*Node[K], error) {
	type result struct {
		node     *Node[K]
		distance float32
	}
	results := make([]*result, 0, len(v.Nodes))

	for _, node := range v.Nodes {
		distance := cosineDistance(node.Vector, vector)
		results = append(results, &result{
			node:     node,
			distance: distance,
		})
	}

	// Sort results by distance
	sort.Slice(results, func(i, j int) bool {
		return results[i].distance < results[j].distance
	})

	// Return the top N results
	limit = min(limit, len(results))
	topResults := make([]*Node[K], limit)
	for i := 0; i < limit; i++ {
		topResults[i] = results[i].node
	}
	return topResults, nil
}

func cosineDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		panic("vectors must have the same length")
	}
	var dotProduct float32
	var normA float32
	var normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	return 1 - (dotProduct / (sqrt(normA) * sqrt(normB)))
}

func sqrt(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}
