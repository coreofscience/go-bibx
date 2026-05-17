package graphs

import (
	"fmt"
	"slices"

	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/traverse"
)

func ReverseTopologicalOrder[V comparable](
	graph gograph.Graph[V],
) ([]*gograph.Vertex[V], error) {
	result, err := TopologicalOrder(graph)
	if err != nil {
		return nil, err
	}
	slices.Reverse(result)
	return result, nil
}

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

