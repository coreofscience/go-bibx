package vector

import (
	"cmp"
	"fmt"
	"math"
	"sort"

	"github.com/hmdsefi/gograph"
)

type Node[K cmp.Ordered] struct {
	Key    K         `json:"id"`
	Vector []float32 `json:"vector"`
}

type Link[K cmp.Ordered] struct {
	Source K       `json:"source"`
	Target K       `json:"target"`
	Weight float32 `json:"weight"`
}

type Vectors[K cmp.Ordered] interface {
	Add(node *Node[K]) error
	Search(vector []float32, limit int) ([]*Node[K], error)
}

func NewNode[K cmp.Ordered](key K, vector []float32) *Node[K] {
	return &Node[K]{
		Key:    key,
		Vector: vector,
	}
}

func NewLink[K cmp.Ordered](source, target K, weight float32) *Link[K] {
	return &Link[K]{
		Source: source,
		Target: target,
		Weight: weight,
	}
}

// DumbVectors is a simple in-memory implementation of the Vectors interface
// that performs a linear search over all nodes. This is not optimized for
// performance and should only be used for testing or small datasets.
type DumbVectors[K cmp.Ordered] struct {
	Nodes []*Node[K] `json:"nodes"`
	Links []*Link[K] `json:"links"`
}

func NewDumbVectors[K cmp.Ordered]() *DumbVectors[K] {
	return &DumbVectors[K]{
		Nodes: make([]*Node[K], 0),
	}
}

func (v *DumbVectors[K]) Add(node *Node[K]) error {
	if len(v.Nodes) > 0 && len(node.Vector) != len(v.Nodes[0].Vector) {
		return fmt.Errorf("vector length mismatch: expected %d, got %d", len(v.Nodes[0].Vector), len(node.Vector))
	}
	v.Nodes = append(v.Nodes, node)
	return nil
}

func (v *DumbVectors[K]) Link(link *Link[K]) {
	v.Links = append(v.Links, link)
}

func (v *DumbVectors[K]) Search(vector []float32, limit int) ([]*Node[K], error) {
	if len(v.Nodes) == 0 {
		return nil, nil
	}
	if len(vector) != len(v.Nodes[0].Vector) {
		return nil, fmt.Errorf("vector length mismatch: expected %d, got %d", len(v.Nodes[0].Vector), len(vector))
	}
	type result struct {
		node     *Node[K]
		distance float32
	}
	results := make([]*result, 0, len(v.Nodes))
	for _, node := range v.Nodes {
		distance := CosineDistance(node.Vector, vector)
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

func (v *DumbVectors[K]) Graph() (gograph.Graph[K], error) {
	graph := gograph.New[K](
		gograph.Directed(),
		gograph.Acyclic(),
		gograph.Weighted(),
	)
	for _, node := range v.Nodes {
		graph.AddVertex(gograph.NewVertex(node.Key))
	}
	for _, link := range v.Links {
		_, err := graph.AddEdge(
			gograph.NewVertex(link.Source),
			gograph.NewVertex(link.Target),
			gograph.WithEdgeWeight(float64(link.Weight)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to add edge: %w", err)
		}
	}
	return graph, nil
}

// CosineDistance calculates the cosine distance between two vectors. The
// cosine distance is defined as 1 - (dot product of a and b) / (magnitude of a
// * magnitude of b). The result is in the range [0, 2], where 0 means the
// vectors are identical and 2 means they are opposite.
func CosineDistance(a, b []float32) float32 {
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
