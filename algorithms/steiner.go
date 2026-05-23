package algorithms

import (
	"cmp"
	"fmt"
	"iter"
	"log/slog"
	"math"
	"slices"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/traverse"
)

// PseudoSteiner is a struct that represents a pseudo-steiner algorithm.
type PseudoSteiner[K comparable] struct {
	graph                 gograph.Graph[K]
	topologicalOrder      []*gograph.Vertex[K]
	topologicalIndex      map[K]int
	shortestDistanceCache map[K]map[K]float64
	shortestPathCache     map[K]map[K][]K
	maxEffort             float64
}

type PseudoSteinerOption[K comparable] func(*PseudoSteiner[K])

// WithMaxEffort sets the maximum effort for the pseudo-steier algorithm.
func WithMaxEffort[K comparable](maxEffort float64) PseudoSteinerOption[K] {
	return func(q *PseudoSteiner[K]) {
		q.maxEffort = maxEffort
	}
}

// NewPseudoSteiner creates a new PseudoSteiner instance.
//
// It takes linear or amortized O(V + E) time to prepare the pseudo-steiner
// structure.
func NewPseudoSteiner[K comparable](graph gograph.Graph[K], opts ...PseudoSteinerOption[K]) (*PseudoSteiner[K], error) {
	topologicalIterator, err := traverse.NewTopologicalIterator(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to create topological iterator: %w", err)
	}
	topologicalOrder := make([]*gograph.Vertex[K], 0, graph.Order())
	err = topologicalIterator.Iterate(func(v *gograph.Vertex[K]) error {
		topologicalOrder = append(topologicalOrder, v)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate topological order: %w", err)
	}
	topologicalIndex := make(map[K]int, graph.Order())
	for i, v := range topologicalOrder {
		topologicalIndex[v.Label()] = i
	}
	pseudoSteiner := &PseudoSteiner[K]{
		graph:                 graph,
		topologicalOrder:      topologicalOrder,
		topologicalIndex:      topologicalIndex,
		shortestDistanceCache: make(map[K]map[K]float64),
		shortestPathCache:     make(map[K]map[K][]K),
		maxEffort:             1.5,
	}
	for _, opt := range opts {
		opt(pseudoSteiner)
	}
	return pseudoSteiner, nil
}

// Run runs the pseudo-steiner algorithm on the given terminals and returns the
// resulting graph.
//
// Go figure what the complexity is. But don't call this with a large number of
// terminals.
func (q *PseudoSteiner[K]) Run(terminals []K) (gograph.Graph[K], error) {
	// Make sure all the terminals exists in the graph.
	for _, terminal := range terminals {
		if _, ok := q.topologicalIndex[terminal]; !ok {
			return nil, fmt.Errorf("terminal %v not found in graph", terminal)
		}
	}

	// Sort the terminals by their topological index.
	q.sortTopological(terminals)

	// Create a new weighted graph with all the terminals as vertices and their
	// distances as weights.
	metaGraph := gograph.New[K](
		gograph.Directed(),
		gograph.Acyclic(),
		gograph.Weighted(),
	)
	for _, terminal := range terminals {
		metaGraph.AddVertex(gograph.NewVertex(terminal))
	}
	for a, b := range sortedPairs(terminals) {
		dist, _ := q.shortestPath(a, b)
		if math.IsInf(dist, 1) {
			continue
		}
		_, err := metaGraph.AddEdge(
			gograph.NewVertex(a),
			gograph.NewVertex(b),
			gograph.WithEdgeWeight(dist),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to construct meta graph: %w", err)
		}
	}

	// Now find the minimum spanning tree of the meta-graph.
	mst, unionFind := graphs.MST(metaGraph)
	components := unionFind.Components()
	if len(components) == 1 {
		slog.Debug("graph is connected, no need to bridge")
		return q.materialize(mst), nil
	}

	slog.Debug("graph is disconnected, need to bridge", "components", len(components))

	// First, try to find a best descendant.
	bestDescendant := q.findBestDescendant(components)
	if bestDescendant != nil {
		slog.Debug("found a best descendant", "descendant", *bestDescendant)
		newTerminals := append(terminals, *bestDescendant)
		return q.Run(newTerminals)
	}

	// If no best descendant, try to find a best ancestor.
	bestAncestor := q.findBestAncestor(components)
	if bestAncestor != nil {
		slog.Debug("found a best ancestor", "ancestor", *bestAncestor)
		newTerminals := append(terminals, *bestAncestor)
		return q.Run(newTerminals)
	}

	// If no best ancestor is found, the mst cannot be bridged, return disjoint.
	return q.materialize(mst), nil
}

func (q *PseudoSteiner[K]) sortTopological(items []K) {
	slices.SortFunc(items, func(a, b K) int {
		return cmp.Compare(q.topologicalIndex[a], q.topologicalIndex[b])
	})
}

func (q *PseudoSteiner[K]) shortestPath(a, b K) (float64, []K) {
	indexA := q.topologicalIndex[a]
	indexB := q.topologicalIndex[b]

	// If a is after b in the topological order, there is no path.
	if indexA >= indexB {
		return math.Inf(1), nil
	}

	// Find in cache
	if dist, cachedPath, ok := q.getShortestPathFromCache(a, b); ok {
		return dist, cachedPath
	}

	maxLength := indexB - indexA + 1

	// Initialize the distance map and parent map.
	dist := make(map[K]float64, maxLength)
	dist[a] = 0

	// Initialize the parent map.
	parent := make(map[K]K, maxLength)

	for _, u := range q.topologicalOrder[indexA : indexB+1] {
		for _, edge := range q.graph.EdgesOf(u) {
			v := edge.Destination()
			weight := float64(1)
			if q.graph.IsWeighted() {
				weight = edge.Weight()
			}
			if du, ok := dist[u.Label()]; ok {
				dv, ok := dist[v.Label()]
				if !ok || dv > du+weight {
					dist[v.Label()] = du + weight
					parent[v.Label()] = u.Label()
				}
			}
		}
	}

	// Find out if we found a path from a to b.
	distB, ok := dist[b]
	if !ok {
		// Cache the result as nil.
		q.cacheShortestPath(a, b, math.Inf(1), nil)
		return math.Inf(1), nil
	}

	// Reconstruct the path from b to a using the parent map.
	path := make([]K, 0, maxLength)
	curr := b
	for curr != a {
		path = append(path, curr)
		curr = parent[curr]
	}
	path = append(path, a)
	slices.Reverse(path)

	// Cache the path.
	q.cacheShortestPath(a, b, distB, path)
	return distB, path
}

func (q *PseudoSteiner[K]) getShortestPathFromCache(a, b K) (float64, []K, bool) {
	cachedPath, ok := q.shortestPathCache[a][b]
	if !ok {
		return math.Inf(1), nil, false
	}
	cachedDistance, ok := q.shortestDistanceCache[a][b]
	if !ok {
		return math.Inf(1), nil, false
	}
	return cachedDistance, cachedPath, true
}

func (q *PseudoSteiner[K]) cacheShortestPath(a, b K, d float64, path []K) {
	if q.shortestPathCache[a] == nil {
		q.shortestPathCache[a] = make(map[K][]K)
	}
	q.shortestPathCache[a][b] = path
	if q.shortestDistanceCache[a] == nil {
		q.shortestDistanceCache[a] = make(map[K]float64)
	}
	q.shortestDistanceCache[a][b] = d
}

func (q *PseudoSteiner[K]) materialize(mst gograph.Graph[K]) gograph.Graph[K] {
	toKeep := make([]K, 0, len(mst.AllEdges()))
	for _, vertex := range mst.GetAllVertices() {
		toKeep = append(toKeep, vertex.Label())
	}
	for _, edge := range mst.AllEdges() {
		a := edge.Source().Label()
		b := edge.Destination().Label()
		_, path := q.shortestPath(a, b)
		toKeep = append(toKeep, path...)
	}
	graph, err := graphs.Keep(q.graph, toKeep)
	if err != nil {
		slog.Error("error materializing mst, this should never happen", "error", err)
		return nil
	}
	return graph
}

func (q *PseudoSteiner[K]) findBestDescendant(toBridge [][]K) *K {
	bestCost := math.Inf(1)

	// Iterate over all pairs of components to bridge.
	var candidate *K
	for sources, targets := range allPairs(toBridge) {
		for a, b := range cartesianProduct(sources, targets) {
			indexA := q.topologicalIndex[a]
			indexB := q.topologicalIndex[b]
			earliestDescendant := max(indexA, indexB)
			for _, w := range q.topologicalOrder[earliestDescendant+1:] {
				labelW := w.Label()
				distA, _ := q.shortestPath(a, labelW)
				distB, _ := q.shortestPath(b, labelW)
				if math.IsInf(distA, 1) || math.IsInf(distB, 1) {
					continue
				}
				cost := distA + distB
				if cost > q.maxEffort {
					slog.Debug("descendant cost exceeds max effort", "cost", cost, "maxEffort", q.maxEffort)
					break
				}
				if cost < bestCost {
					bestCost = cost
					candidate = &labelW
				}
			}
		}
	}

	// If no candidate was found, return nil.
	if bestCost == math.MaxInt64 || candidate == nil {
		return nil
	}

	return candidate
}

func (q *PseudoSteiner[K]) findBestAncestor(toBridge [][]K) *K {
	bestCost := math.Inf(1)

	// Iterate over all pairs of components to bridge.
	var candidate *K
	for sources, targets := range allPairs(toBridge) {
		for a, b := range cartesianProduct(sources, targets) {
			indexA := q.topologicalIndex[a]
			indexB := q.topologicalIndex[b]
			oldestAncestor := min(indexA, indexB)
			for i := oldestAncestor - 1; i >= 0; i-- {
				w := q.topologicalOrder[i]
				labelW := w.Label()
				distA, _ := q.shortestPath(labelW, a)
				distB, _ := q.shortestPath(labelW, b)
				if math.IsInf(distA, 1) || math.IsInf(distB, 1) {
					continue
				}
				cost := distA + distB
				if cost > q.maxEffort {
					slog.Debug("ancestor cost exceeds max effort", "cost", cost, "maxEffort", q.maxEffort)
					break
				}
				if cost < bestCost {
					bestCost = cost
					label := w.Label()
					candidate = &label
				}
			}
		}
	}

	// If no candidate was found, return nil.
	if bestCost == math.MaxInt64 || candidate == nil {
		return nil
	}

	return candidate
}

func sortedPairs[K any](items []K) iter.Seq2[K, K] {
	return func(yield func(K, K) bool) {
		for i := 0; i < len(items)-1; i++ {
			for j := i + 1; j < len(items); j++ {
				if !yield(items[i], items[j]) {
					return
				}
			}
		}
	}
}

func allPairs[K any](items []K) iter.Seq2[K, K] {
	return func(yield func(K, K) bool) {
		for i := range items {
			for j := range items {
				if i == j {
					continue
				}
				if !yield(items[i], items[j]) {
					return
				}
			}
		}
	}
}

func cartesianProduct[T any](a, b []T) iter.Seq2[T, T] {
	return func(yield func(T, T) bool) {
		for _, x := range a {
			for _, y := range b {
				if !yield(x, y) {
					return
				}
			}
		}
	}
}
