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
	graph             gograph.Graph[K]
	topologicalOrder  []*gograph.Vertex[K]
	topologicalIndex  map[K]int
	ssspDist          map[K]map[K]float64
	ssspParent        map[K]map[K]K
	maxEffort         float64
	maxEffortExceeded int
}

// PseudoSteinerOption configures a [PseudoSteiner] algorithm instance.
type PseudoSteinerOption[K comparable] func(*PseudoSteiner[K])

// WithMaxEffort sets the maximum effort for the pseudo-steier algorithm.
func WithMaxEffort[K comparable](maxEffort float64) PseudoSteinerOption[K] {
	return func(q *PseudoSteiner[K]) {
		q.maxEffort = maxEffort
	}
}

// WithMaxEffortExceeded sets the maximum number of consecutive candidates exceeding maxEffort before aborting search for a pair.
func WithMaxEffortExceeded[K comparable](maxEffortExceeded int) PseudoSteinerOption[K] {
	return func(q *PseudoSteiner[K]) {
		q.maxEffortExceeded = maxEffortExceeded
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
		graph:             graph,
		topologicalOrder:  topologicalOrder,
		topologicalIndex:  topologicalIndex,
		ssspDist:          make(map[K]map[K]float64),
		ssspParent:        make(map[K]map[K]K),
		maxEffort:         3.0,
		maxEffortExceeded: 3,
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
	terminals = slices.Clone(terminals)

	// Make sure all the terminals exists in the graph.
	for _, terminal := range terminals {
		if _, ok := q.topologicalIndex[terminal]; !ok {
			return nil, fmt.Errorf("terminal %v not found in graph", terminal)
		}
	}

	// Sort the terminals by their topological index.
	q.sortTopological(terminals)
	terminals = slices.Compact(terminals)

	terminalSet := make(map[K]struct{}, len(terminals))
	for _, terminal := range terminals {
		terminalSet[terminal] = struct{}{}
	}

	// Create a new weighted graph with all the terminals as vertices and their
	// distances as weights.
	metaGraph := gograph.New[K](
		gograph.Directed(),
		gograph.Weighted(),
	)
	for _, terminal := range terminals {
		metaGraph.AddVertex(gograph.NewVertex(terminal))
	}
	// Compute SSSP for each terminal only once
	for _, a := range terminals {
		q.computeSSSP(a)
	}

	for a, b := range sortedPairs(terminals) {
		dist, path := q.shortestPath(a, b)
		if math.IsInf(dist, 1) {
			continue
		}
		// If the shortest path between terminals a and b contains any intermediate terminal,
		// the sub-path edges between adjacent terminals already cover it with equal or smaller weights.
		containsIntermediateTerminal := false
		if len(path) > 2 {
			for _, node := range path[1 : len(path)-1] {
				if _, ok := terminalSet[node]; ok {
					containsIntermediateTerminal = true
					break
				}
			}
		}
		if containsIntermediateTerminal {
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
	bestDescendant := q.findBestDescendant(components, terminalSet)
	if bestDescendant != nil {
		slog.Debug("found a best descendant", "descendant", *bestDescendant)
		newTerminals := append(terminals, *bestDescendant)
		return q.Run(newTerminals)
	}

	// If no best descendant, try to find a best ancestor.
	bestAncestor := q.findBestAncestor(components, terminalSet)
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

func (q *PseudoSteiner[K]) computeSSSP(a K) {
	if q.ssspDist == nil {
		q.ssspDist = make(map[K]map[K]float64)
		q.ssspParent = make(map[K]map[K]K)
	}
	if _, exists := q.ssspDist[a]; exists {
		return
	}

	indexA, ok := q.topologicalIndex[a]
	if !ok {
		q.ssspDist[a] = make(map[K]float64)
		q.ssspParent[a] = make(map[K]K)
		return
	}

	maxLength := len(q.topologicalOrder) - indexA
	dist := make(map[K]float64, maxLength)
	parent := make(map[K]K, maxLength)
	dist[a] = 0

	for _, u := range q.topologicalOrder[indexA:] {
		labelU := u.Label()
		du, ok := dist[labelU]
		if !ok {
			continue
		}
		for _, edge := range q.graph.EdgesOf(u) {
			v := edge.Destination()
			labelV := v.Label()
			weight := float64(1)
			if q.graph.IsWeighted() {
				weight = edge.Weight()
			}
			dv, ok := dist[labelV]
			if !ok || dv > du+weight {
				dist[labelV] = du + weight
				parent[labelV] = labelU
			}
		}
	}

	q.ssspDist[a] = dist
	q.ssspParent[a] = parent
}

func (q *PseudoSteiner[K]) shortestPath(a, b K) (float64, []K) {
	indexA, okA := q.topologicalIndex[a]
	indexB, okB := q.topologicalIndex[b]
	if !okA || !okB || indexA >= indexB {
		return math.Inf(1), nil
	}

	q.computeSSSP(a)

	distMap := q.ssspDist[a]
	distB, ok := distMap[b]
	if !ok {
		return math.Inf(1), nil
	}

	parentMap := q.ssspParent[a]
	path := make([]K, 0, indexB-indexA+1)
	curr := b
	for curr != a {
		path = append(path, curr)
		curr = parentMap[curr]
	}
	path = append(path, a)
	slices.Reverse(path)

	return distB, path
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

func (q *PseudoSteiner[K]) findBestDescendant(toBridge [][]K, terminalSet map[K]struct{}) *K {
	numComponents := len(toBridge)
	if numComponents < 2 {
		return nil
	}

	// Pre-calculate min distance from each component i to each node w
	distComp := make([]map[K]float64, numComponents)
	for i, comp := range toBridge {
		distComp[i] = make(map[K]float64)
		for _, a := range comp {
			q.computeSSSP(a)
			for w, d := range q.ssspDist[a] {
				if cur, ok := distComp[i][w]; !ok || d < cur {
					distComp[i][w] = d
				}
			}
		}
	}

	bestCost := math.Inf(1)
	var candidate *K

	for i := 0; i < numComponents-1; i++ {
		for j := i + 1; j < numComponents; j++ {
			consecutiveExceeded := 0
			for _, wNode := range q.topologicalOrder {
				w := wNode.Label()
				if _, ok := terminalSet[w]; ok {
					continue
				}
				dA, okA := distComp[i][w]
				dB, okB := distComp[j][w]
				if !okA || !okB || math.IsInf(dA, 1) || math.IsInf(dB, 1) {
					continue
				}
				cost := dA + dB
				if cost > q.maxEffort {
					slog.Debug("descendant cost exceeds max effort", "cost", cost, "maxEffort", q.maxEffort)
					consecutiveExceeded++
					if q.maxEffortExceeded > 0 && consecutiveExceeded >= q.maxEffortExceeded {
						break
					}
					continue
				}
				consecutiveExceeded = 0
				if cost < bestCost {
					bestCost = cost
					cand := w
					candidate = &cand
				}
			}
		}
	}

	if math.IsInf(bestCost, 1) || candidate == nil {
		return nil
	}
	return candidate
}

func (q *PseudoSteiner[K]) findBestAncestor(toBridge [][]K, terminalSet map[K]struct{}) *K {
	numComponents := len(toBridge)
	if numComponents < 2 {
		return nil
	}

	bestCost := math.Inf(1)
	var candidate *K

	for i := 0; i < numComponents-1; i++ {
		for j := i + 1; j < numComponents; j++ {
			consecutiveExceeded := 0
			for k := len(q.topologicalOrder) - 1; k >= 0; k-- {
				wNode := q.topologicalOrder[k]
				w := wNode.Label()
				if _, ok := terminalSet[w]; ok {
					continue
				}
				q.computeSSSP(w)
				distMap := q.ssspDist[w]

				dA := math.Inf(1)
				for _, a := range toBridge[i] {
					if d, ok := distMap[a]; ok && d < dA {
						dA = d
					}
				}
				dB := math.Inf(1)
				for _, b := range toBridge[j] {
					if d, ok := distMap[b]; ok && d < dB {
						dB = d
					}
				}

				if math.IsInf(dA, 1) || math.IsInf(dB, 1) {
					continue
				}
				cost := dA + dB
				if cost > q.maxEffort {
					slog.Debug("ancestor cost exceeds max effort", "cost", cost, "maxEffort", q.maxEffort)
					consecutiveExceeded++
					if q.maxEffortExceeded > 0 && consecutiveExceeded >= q.maxEffortExceeded {
						break
					}
					continue
				}
				consecutiveExceeded = 0
				if cost < bestCost {
					bestCost = cost
					cand := w
					candidate = &cand
				}
			}
		}
	}

	if math.IsInf(bestCost, 1) || candidate == nil {
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
