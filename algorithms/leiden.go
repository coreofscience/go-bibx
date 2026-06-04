package algorithms

import (
	"math"
	"math/rand"

	"github.com/hmdsefi/gograph"
)

const (
	defaultGamma = 1.0
	defaultTheta = 0.01
)

type Leiden[K comparable] struct {
	graph gograph.Graph[K]
	gamma float64
	theta float64
}

type LeidenOption[K comparable] func(*Leiden[K])

func WithGamma[K comparable](gamma float64) LeidenOption[K] {
	return func(l *Leiden[K]) {
		l.gamma = gamma
	}
}

func WithTheta[K comparable](theta float64) LeidenOption[K] {
	return func(l *Leiden[K]) {
		l.theta = theta
	}
}

func NewLeiden[K comparable](graph gograph.Graph[K], opts ...LeidenOption[K]) *Leiden[K] {
	l := &Leiden[K]{
		graph: graph,
		gamma: defaultGamma,
		theta: defaultTheta,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *Leiden[K]) deltaH(evc float64, nodeSize float64, commSize float64) float64 {
	return evc - l.gamma*float64(nodeSize*commSize)
}

type node[T comparable] struct {
	id   T
	size float64
}

type edge[T comparable] struct {
	to     T
	weight float64
}

type runState[T comparable] struct {
	nodes map[T]node[T]
	edges map[T][]edge[T]
}

func (l *Leiden[K]) Run() map[K]int {
	vertices := l.graph.GetAllVertices()
	if len(vertices) == 0 {
		return nil
	}

	totalEdgeWeight := 0.0
	for _, e := range l.graph.AllEdges() {
		w := 1.0
		if l.graph.IsWeighted() {
			w = e.Weight()
		}
		totalEdgeWeight += w
	}

	nodeDegrees := make(map[K]float64)

	state := runState[K]{
		nodes: make(map[K]node[K]),
		edges: make(map[K][]edge[K]),
	}

	adj := make(map[K]map[K]float64)
	for _, e := range l.graph.AllEdges() {
		u := e.Source().Label()
		v := e.Destination().Label()
		w := float64(1)
		if l.graph.IsWeighted() {
			w = e.Weight()
		}
		if adj[u] == nil {
			adj[u] = make(map[K]float64)
		}
		adj[u][v] += w
		nodeDegrees[u] += w

		if adj[v] == nil {
			adj[v] = make(map[K]float64)
		}
		adj[v][u] += w
		nodeDegrees[v] += w
	}

	for _, v := range vertices {
		label := v.Label()
		state.nodes[label] = node[K]{id: label, size: nodeDegrees[label]}
	}

	m2 := totalEdgeWeight * 2
	if m2 == 0 {
		m2 = 1
	}
	l.gamma = l.gamma / m2

	for u, neighbors := range adj {
		for v, w := range neighbors {
			if u != v {
				state.edges[u] = append(state.edges[u], edge[K]{to: v, weight: w})
			}
		}
	}

	// We will track the assignment of the original K vertices to aggregate int nodes
	// For the first iteration, partition is map[K]int
	partition := make(map[K]int)
	globalPartition := make(map[K]int)

	nextCommID := 0
	for _, v := range vertices {
		label := v.Label()
		partition[label] = nextCommID
		globalPartition[label] = nextCommID
		nextCommID++
	}

	// Execute first iteration on K
	newPartition, nextCommID := l.moveNodesFast(state, partition, nextCommID)

	uniqueComms := make(map[int]struct{})
	for _, c := range newPartition {
		uniqueComms[c] = struct{}{}
	}
	if len(uniqueComms) == len(state.nodes) {
		return l.normalizeResult(globalPartition)
	}

	refinedPartition, nextCommID := l.refinePartition(state, newPartition, nextCommID)

	// Create int-based aggregate graph
	intState, intPartition, aggregateMap := l.aggregateGraph(state, refinedPartition, newPartition)

	// Update global partition
	for k := range globalPartition {
		refinedCommID := refinedPartition[k]
		newAggID := aggregateMap[refinedCommID]
		globalPartition[k] = newAggID
	}

	// Subsequent iterations on int
	done := false
	for !done {
		intPartition, nextCommID = l.moveNodesFastInt(intState, intPartition, nextCommID)

		uniqueComms := make(map[int]struct{})
		for _, c := range intPartition {
			uniqueComms[c] = struct{}{}
		}
		if len(uniqueComms) == len(intState.nodes) {
			break
		}

		refinedPartitionInt, _ := l.refinePartitionInt(intState, intPartition, nextCommID)

		newState, newPartitionInt, aggregateMapInt := l.aggregateGraphInt(intState, refinedPartitionInt, intPartition)

		for k := range globalPartition {
			refinedCommID := refinedPartitionInt[globalPartition[k]]
			newAggID := aggregateMapInt[refinedCommID]
			globalPartition[k] = newAggID
		}

		intState = newState
		intPartition = newPartitionInt
	}

	return l.normalizeResult(globalPartition)
}

func (l *Leiden[K]) normalizeResult(globalPartition map[K]int) map[K]int {
	finalCommMap := make(map[int]int)
	normalizedCommCounter := 0
	result := make(map[K]int)
	for k, finalComm := range globalPartition {
		if _, ok := finalCommMap[finalComm]; !ok {
			finalCommMap[finalComm] = normalizedCommCounter
			normalizedCommCounter++
		}
		result[k] = finalCommMap[finalComm]
	}
	return result
}

// ---------------------------------------------------------
// Gen 1: Iterations on map[K]int
// ---------------------------------------------------------

func (l *Leiden[K]) moveNodesFast(state runState[K], partition map[K]int, nextCommID int) (map[K]int, int) {
	commSize := make(map[int]float64)
	for k, comm := range partition {
		commSize[comm] += state.nodes[k].size
	}

	queue := make([]K, 0, len(state.nodes))
	inQueue := make(map[K]bool)
	for k := range state.nodes {
		queue = append(queue, k)
		inQueue[k] = true
	}
	rand.Shuffle(len(queue), func(i, j int) { queue[i], queue[j] = queue[j], queue[i] })

	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		inQueue[v] = false

		vSize := state.nodes[v].size
		currentComm := partition[v]

		commWeight := make(map[int]float64)
		for _, e := range state.edges[v] {
			commWeight[partition[e.to]] += e.weight
		}

		commSize[currentComm] -= vSize

		bestComm := currentComm
		maxDelta := 0.0

		for c, w := range commWeight {
			delta := l.deltaH(w, vSize, commSize[c])
			if delta > maxDelta {
				maxDelta = delta
				bestComm = c
			}
		}

		if 0 > maxDelta {
			bestComm = nextCommID
			nextCommID++
		}

		if bestComm != currentComm {
			partition[v] = bestComm
			commSize[bestComm] += vSize

			for _, e := range state.edges[v] {
				if partition[e.to] != bestComm && !inQueue[e.to] {
					queue = append(queue, e.to)
					inQueue[e.to] = true
				}
			}
		} else {
			commSize[currentComm] += vSize
		}
	}

	return partition, nextCommID
}

func (l *Leiden[K]) refinePartition(state runState[K], partition map[K]int, nextCommID int) (map[K]int, int) {
	refinedPartition := make(map[K]int)
	for k := range state.nodes {
		refinedPartition[k] = nextCommID
		nextCommID++
	}

	commNodes := make(map[int][]K)
	for k, c := range partition {
		commNodes[c] = append(commNodes[c], k)
	}

	for _, S := range commNodes {
		S_set := make(map[K]struct{})
		S_size := 0.0
		for _, v := range S {
			S_set[v] = struct{}{}
			S_size += state.nodes[v].size
		}

		R := make([]K, 0)
		for _, v := range S {
			vSize := state.nodes[v].size
			evS := 0.0
			for _, e := range state.edges[v] {
				if _, ok := S_set[e.to]; ok {
					evS += e.weight
				}
			}
			if evS >= l.gamma*float64(vSize*(S_size-vSize)) {
				R = append(R, v)
			}
		}

		rand.Shuffle(len(R), func(i, j int) { R[i], R[j] = R[j], R[i] })

		for _, v := range R {
			l.refineNode(state, refinedPartition, S, S_set, S_size, v)
		}
	}

	return refinedPartition, nextCommID
}

func (l *Leiden[K]) refineNode(state runState[K], refinedPartition map[K]int, S []K, S_set map[K]struct{}, S_size float64, v K) {
	currentRefinedComm := refinedPartition[v]

	isSingleton := true
	for k := range state.nodes {
		if k != v && refinedPartition[k] == currentRefinedComm {
			isSingleton = false
			break
		}
	}

	if !isSingleton {
		return
	}

	vSize := state.nodes[v].size

	refinedCommSize := make(map[int]float64)
	for _, u := range S {
		refinedCommSize[refinedPartition[u]] += state.nodes[u].size
	}

	T := make([]int, 0)
	for c := range refinedCommSize {
		if c == currentRefinedComm {
			continue
		}
		ecS := 0.0
		cSize := refinedCommSize[c]
		for _, u := range S {
			if refinedPartition[u] == c {
				for _, e := range state.edges[u] {
					if _, ok := S_set[e.to]; ok && refinedPartition[e.to] != c {
						ecS += e.weight
					}
				}
			}
		}

		if ecS >= l.gamma*float64(cSize*(S_size-cSize)) {
			T = append(T, c)
		}
	}

	if len(T) > 0 {
		probs := make([]float64, len(T))
		sumProbs := 0.0

		vEdgesToC := make(map[int]float64)
		for _, e := range state.edges[v] {
			if _, ok := S_set[e.to]; ok {
				vEdgesToC[refinedPartition[e.to]] += e.weight
			}
		}

		for i, c := range T {
			delta := l.deltaH(vEdgesToC[c], vSize, refinedCommSize[c])
			if delta >= 0 {
				probs[i] = math.Exp(delta / l.theta)
				sumProbs += probs[i]
			} else {
				probs[i] = 0
			}
		}

		if sumProbs > 0 {
			r := rand.Float64() * sumProbs
			cumSum := 0.0
			for i, c := range T {
				cumSum += probs[i]
				if r <= cumSum {
					refinedPartition[v] = c
					break
				}
			}
		}
	}
}

func (l *Leiden[K]) aggregateGraph(state runState[K], refinedPartition map[K]int, partition map[K]int) (runState[int], map[int]int, map[int]int) {
	newState := runState[int]{
		nodes: make(map[int]node[int]),
		edges: make(map[int][]edge[int]),
	}

	aggregateMap := make(map[int]int)

	for k, c := range refinedPartition {
		if aggID, ok := aggregateMap[c]; ok {
			n := newState.nodes[aggID]
			n.size += state.nodes[k].size
			newState.nodes[aggID] = n
		} else {
			aggID = len(newState.nodes)
			aggregateMap[c] = aggID
			newState.nodes[aggID] = node[int]{id: aggID, size: state.nodes[k].size}
		}
	}

	edgeMap := make(map[int]map[int]float64)
	for u, neighbors := range state.edges {
		aggU := aggregateMap[refinedPartition[u]]
		if edgeMap[aggU] == nil {
			edgeMap[aggU] = make(map[int]float64)
		}
		for _, e := range neighbors {
			aggV := aggregateMap[refinedPartition[e.to]]
			if aggU != aggV {
				edgeMap[aggU][aggV] += e.weight
			}
		}
	}

	for u, neighbors := range edgeMap {
		for v, w := range neighbors {
			newState.edges[u] = append(newState.edges[u], edge[int]{to: v, weight: w})
		}
	}

	newPartition := make(map[int]int)
	for k, c := range refinedPartition {
		aggID := aggregateMap[c]
		newPartition[aggID] = partition[k]
	}

	return newState, newPartition, aggregateMap
}

// ---------------------------------------------------------
// Gen 2+: Iterations on map[int]int
// ---------------------------------------------------------

func (l *Leiden[K]) moveNodesFastInt(state runState[int], partition map[int]int, nextCommID int) (map[int]int, int) {
	commSize := make(map[int]float64)
	for k, comm := range partition {
		commSize[comm] += state.nodes[k].size
	}

	queue := make([]int, 0, len(state.nodes))
	inQueue := make(map[int]bool)
	for k := range state.nodes {
		queue = append(queue, k)
		inQueue[k] = true
	}
	rand.Shuffle(len(queue), func(i, j int) { queue[i], queue[j] = queue[j], queue[i] })

	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		inQueue[v] = false

		vSize := state.nodes[v].size
		currentComm := partition[v]

		commWeight := make(map[int]float64)
		for _, e := range state.edges[v] {
			commWeight[partition[e.to]] += e.weight
		}

		commSize[currentComm] -= vSize

		bestComm := currentComm
		maxDelta := 0.0

		for c, w := range commWeight {
			delta := l.deltaH(w, vSize, commSize[c])
			if delta > maxDelta {
				maxDelta = delta
				bestComm = c
			}
		}

		if 0 > maxDelta {
			bestComm = nextCommID
			nextCommID++
		}

		if bestComm != currentComm {
			partition[v] = bestComm
			commSize[bestComm] += vSize

			for _, e := range state.edges[v] {
				if partition[e.to] != bestComm && !inQueue[e.to] {
					queue = append(queue, e.to)
					inQueue[e.to] = true
				}
			}
		} else {
			commSize[currentComm] += vSize
		}
	}

	return partition, nextCommID
}

func (l *Leiden[K]) refinePartitionInt(state runState[int], partition map[int]int, nextCommID int) (map[int]int, int) {
	refinedPartition := make(map[int]int)
	for k := range state.nodes {
		refinedPartition[k] = nextCommID
		nextCommID++
	}

	commNodes := make(map[int][]int)
	for k, c := range partition {
		commNodes[c] = append(commNodes[c], k)
	}

	for _, S := range commNodes {
		S_set := make(map[int]struct{})
		S_size := 0.0
		for _, v := range S {
			S_set[v] = struct{}{}
			S_size += state.nodes[v].size
		}

		R := make([]int, 0)
		for _, v := range S {
			vSize := state.nodes[v].size
			evS := 0.0
			for _, e := range state.edges[v] {
				if _, ok := S_set[e.to]; ok {
					evS += e.weight
				}
			}
			if evS >= l.gamma*float64(vSize*(S_size-vSize)) {
				R = append(R, v)
			}
		}

		rand.Shuffle(len(R), func(i, j int) { R[i], R[j] = R[j], R[i] })

		for _, v := range R {
			l.refineNodeInt(state, refinedPartition, S, S_set, S_size, v)
		}
	}

	return refinedPartition, nextCommID
}

func (l *Leiden[K]) refineNodeInt(state runState[int], refinedPartition map[int]int, S []int, S_set map[int]struct{}, S_size float64, v int) {
	currentRefinedComm := refinedPartition[v]

	isSingleton := true
	for k := range state.nodes {
		if k != v && refinedPartition[k] == currentRefinedComm {
			isSingleton = false
			break
		}
	}

	if !isSingleton {
		return
	}

	vSize := state.nodes[v].size

	refinedCommSize := make(map[int]float64)
	for _, u := range S {
		refinedCommSize[refinedPartition[u]] += state.nodes[u].size
	}

	T := make([]int, 0)
	for c := range refinedCommSize {
		if c == currentRefinedComm {
			continue
		}
		ecS := 0.0
		cSize := refinedCommSize[c]
		for _, u := range S {
			if refinedPartition[u] == c {
				for _, e := range state.edges[u] {
					if _, ok := S_set[e.to]; ok && refinedPartition[e.to] != c {
						ecS += e.weight
					}
				}
			}
		}

		if ecS >= l.gamma*float64(cSize*(S_size-cSize)) {
			T = append(T, c)
		}
	}

	if len(T) > 0 {
		probs := make([]float64, len(T))
		sumProbs := 0.0

		vEdgesToC := make(map[int]float64)
		for _, e := range state.edges[v] {
			if _, ok := S_set[e.to]; ok {
				vEdgesToC[refinedPartition[e.to]] += e.weight
			}
		}

		for i, c := range T {
			delta := l.deltaH(vEdgesToC[c], vSize, refinedCommSize[c])
			if delta >= 0 {
				probs[i] = math.Exp(delta / l.theta)
				sumProbs += probs[i]
			} else {
				probs[i] = 0
			}
		}

		if sumProbs > 0 {
			r := rand.Float64() * sumProbs
			cumSum := 0.0
			for i, c := range T {
				cumSum += probs[i]
				if r <= cumSum {
					refinedPartition[v] = c
					break
				}
			}
		}
	}
}

func (l *Leiden[K]) aggregateGraphInt(state runState[int], refinedPartition map[int]int, partition map[int]int) (runState[int], map[int]int, map[int]int) {
	newState := runState[int]{
		nodes: make(map[int]node[int]),
		edges: make(map[int][]edge[int]),
	}

	aggregateMap := make(map[int]int)

	for k, c := range refinedPartition {
		if aggID, ok := aggregateMap[c]; ok {
			n := newState.nodes[aggID]
			n.size += state.nodes[k].size
			newState.nodes[aggID] = n
		} else {
			aggID = len(newState.nodes)
			aggregateMap[c] = aggID
			newState.nodes[aggID] = node[int]{id: aggID, size: state.nodes[k].size}
		}
	}

	edgeMap := make(map[int]map[int]float64)
	for u, neighbors := range state.edges {
		aggU := aggregateMap[refinedPartition[u]]
		if edgeMap[aggU] == nil {
			edgeMap[aggU] = make(map[int]float64)
		}
		for _, e := range neighbors {
			aggV := aggregateMap[refinedPartition[e.to]]
			if aggU != aggV {
				edgeMap[aggU][aggV] += e.weight
			}
		}
	}

	for u, neighbors := range edgeMap {
		for v, w := range neighbors {
			newState.edges[u] = append(newState.edges[u], edge[int]{to: v, weight: w})
		}
	}

	newPartition := make(map[int]int)
	for k, c := range refinedPartition {
		aggID := aggregateMap[c]
		newPartition[aggID] = partition[k]
	}

	return newState, newPartition, aggregateMap
}
