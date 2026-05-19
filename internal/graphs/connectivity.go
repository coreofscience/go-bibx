package graphs

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"

	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/connectivity"
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

// Giant returns the giant component of the graph. The giant component is the largest connected component of the graph. If the graph is directed, it returns the largest strongly connected component.
func Giant[V comparable](g gograph.Graph[V]) (gograph.Graph[V], error) {
	var ccs [][]*gograph.Vertex[V]
	if g.IsDirected() {
		ccs = connectivity.Tarjan(g)
	} else {
		var err error
		ccs, err = WeaklyConnectedComponents(g)
		if err != nil {
			return nil, fmt.Errorf("failed to compute weakly connected components: %w", err)
		}
	}
	if len(ccs) == 0 {
		return gograph.New[V](), nil
	}
	slog.Debug(
		"found connected components",
		"size", len(ccs),
		"largest", len(ccs[0]),
		"smallest", len(ccs[len(ccs)-1]),
	)
	slices.SortFunc(ccs, func(a, b []*gograph.Vertex[V]) int {
		return cmp.Compare(len(b), len(a))
	})
	largestCC := ccs[0]
	toKeep := make([]V, len(largestCC))
	for i, vertex := range largestCC {
		toKeep[i] = vertex.Label()
	}
	newGraph, err := Keep(g, toKeep)
	if err != nil {
		return nil, fmt.Errorf("failed to keep giant component: %w", err)
	}
	return newGraph, nil
}
