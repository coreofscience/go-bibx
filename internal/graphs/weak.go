package graphs

import (
	"fmt"

	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/traverse"
)

// WeaklyConnectedComponents returns the weakly connected components of the graph.
func WeaklyConnectedComponents[T comparable](g gograph.Graph[T]) ([][]*gograph.Vertex[T], error) {
	if g.IsDirected() {
		return nil, fmt.Errorf("graph is directed")
	}
	vertices := g.GetAllVertices()
	labels := make(map[T]int)
	currentLabel := 0
	for _, v := range vertices {
		if _, ok := labels[v.Label()]; ok {
			continue
		}
		iterator, err := traverse.NewDepthFirstIterator(g, v.Label())
		if err != nil {
			return nil, fmt.Errorf("failed to create iterator: %w", err)
		}
		err = iterator.Iterate(func(v *gograph.Vertex[T]) error {
			labels[v.Label()] = currentLabel
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to iterate: %w", err)
		}
		currentLabel++
	}
	components := make([][]*gograph.Vertex[T], currentLabel)
	for _, v := range vertices {
		components[labels[v.Label()]] = append(components[labels[v.Label()]], v)
	}
	return components, nil
}
