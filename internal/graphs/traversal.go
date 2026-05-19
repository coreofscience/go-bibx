package graphs

import (
	"fmt"

	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/traverse"
)

// TopologicalOrder returns the topological order of the graph.
func TopologicalOrder[V comparable](
	graph gograph.Graph[V],
) ([]*gograph.Vertex[V], error) {
	result := make([]*gograph.Vertex[V], 0, graph.Order())
	iterator, err := traverse.NewTopologicalIterator(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to create topological iterator: %w", err)
	}
	err = iterator.Iterate(func(v *gograph.Vertex[V]) error {
		result = append(result, v)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate topological order: %w", err)
	}
	return result, nil
}

// ReverseTopologicalOrder returns the reverse topological order of the graph.
func ReverseTopologicalOrder[V comparable](
	graph gograph.Graph[V],
) ([]*gograph.Vertex[V], error) {
	result, err := TopologicalOrder(graph)
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}
