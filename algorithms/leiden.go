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

func (l *Leiden[K]) deltaH(evc float64, nodeSize int, commSize int) float64 {
	return evc - l.gamma*float64(nodeSize*commSize)
}

type node struct {
	id   int
	size int
}

type edge struct {
	to     int
	weight float64
}

type runState struct {
	nodes []node
	edges map[int][]edge
}

func (l *Leiden[K]) Run() map[K]int {
	vertices := l.graph.GetAllVertices()
	if len(vertices) == 0 {
		return nil
	}

	labelToID := make(map[K]int)
	idToLabel := make(map[int]K)
	state := runState{
		nodes: make([]node, len(vertices)),
		edges: make(map[int][]edge),
	}

	for i, v := range vertices {
		labelToID[v.Label()] = i
		idToLabel[i] = v.Label()
		state.nodes[i] = node{id: i, size: 1}
	}

	adj := make(map[int]map[int]float64)
	for _, e := range l.graph.AllEdges() {
		u := labelToID[e.Source().Label()]
		v := labelToID[e.Destination().Label()]
		w := float64(1)
		if l.graph.IsWeighted() {
			w = e.Weight()
		}
		if adj[u] == nil {
			adj[u] = make(map[int]float64)
		}
		adj[u][v] += w
		if adj[v] == nil {
			adj[v] = make(map[int]float64)
		}
		adj[v][u] += w
	}

	for u, neighbors := range adj {
		for v, w := range neighbors {
			if u != v {
				state.edges[u] = append(state.edges[u], edge{to: v, weight: w})
			}
		}
	}

	partition := make([]int, len(state.nodes))
	for i := range partition {
		partition[i] = i
	}

	globalPartition := make([]int, len(state.nodes))
	for i := range globalPartition {
		globalPartition[i] = i
	}

	nextCommID := len(state.nodes)

	done := false
	for !done {
		partition, nextCommID = l.moveNodesFast(&state, partition, nextCommID)

		uniqueComms := make(map[int]struct{})
		for _, c := range partition {
			uniqueComms[c] = struct{}{}
		}
		if len(uniqueComms) == len(state.nodes) {
			break
		}

		refinedPartition, _ := l.refinePartition(&state, partition, nextCommID)

		newState, newPartition, aggregateMap := l.aggregateGraph(&state, refinedPartition, partition)

		for i, currentAggID := range globalPartition {
			refinedCommID := refinedPartition[currentAggID]
			newAggID := aggregateMap[refinedCommID]
			globalPartition[i] = newAggID
		}

		state = newState
		partition = newPartition
	}

	// Normalize community IDs to start from 0
	finalCommMap := make(map[int]int)
	normalizedCommCounter := 0
	result := make(map[K]int)
	for i, finalComm := range globalPartition {
		if _, ok := finalCommMap[finalComm]; !ok {
			finalCommMap[finalComm] = normalizedCommCounter
			normalizedCommCounter++
		}
		result[idToLabel[i]] = finalCommMap[finalComm]
	}

	return result
}

func (l *Leiden[K]) moveNodesFast(state *runState, partition []int, nextCommID int) ([]int, int) {
	commSize := make(map[int]int)
	for i, comm := range partition {
		commSize[comm] += state.nodes[i].size
	}

	queue := make([]int, len(state.nodes))
	inQueue := make([]bool, len(state.nodes))
	for i := range state.nodes {
		queue[i] = i
		inQueue[i] = true
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

func (l *Leiden[K]) refinePartition(state *runState, partition []int, nextCommID int) ([]int, int) {
	refinedPartition := make([]int, len(state.nodes))
	for i := range refinedPartition {
		refinedPartition[i] = nextCommID
		nextCommID++
	}

	commNodes := make(map[int][]int)
	for i, c := range partition {
		commNodes[c] = append(commNodes[c], i)
	}

	for _, S := range commNodes {
		S_set := make(map[int]struct{})
		S_size := 0
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
			l.refineNode(state, refinedPartition, S, S_set, S_size, v)
		}
	}

	return refinedPartition, nextCommID
}

func (l *Leiden[K]) refineNode(state *runState, refinedPartition []int, S []int, S_set map[int]struct{}, S_size int, v int) {
	currentRefinedComm := refinedPartition[v]

	// Check if v is in a singleton community in refinedPartition
	isSingleton := true
	for _, u := range state.nodes {
		if u.id != v && refinedPartition[u.id] == currentRefinedComm {
			isSingleton = false
			break
		}
	}

	if !isSingleton {
		return
	}

	vSize := state.nodes[v].size

	// Calculate refined community sizes and internal edges within S
	refinedCommSize := make(map[int]int)
	for _, u := range S {
		refinedCommSize[refinedPartition[u]] += state.nodes[u].size
	}

	T := make([]int, 0)
	for c := range refinedCommSize {
		if c == currentRefinedComm {
			continue // Don't move to own singleton
		}
		// Calculate edges between refined community c and rest of S
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
		// Calculate probabilities
		probs := make([]float64, len(T))
		sumProbs := 0.0

		// Pre-calculate edges from v to each refined community
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

func (l *Leiden[K]) aggregateGraph(state *runState, refinedPartition []int, partition []int) (runState, []int, map[int]int) {
	newState := runState{
		nodes: make([]node, 0),
		edges: make(map[int][]edge),
	}

	// map from refined community ID to new aggregate node ID
	aggregateMap := make(map[int]int)

	// Create nodes
	for i, c := range refinedPartition {
		if aggID, ok := aggregateMap[c]; ok {
			newState.nodes[aggID].size += state.nodes[i].size
		} else {
			aggID = len(newState.nodes)
			aggregateMap[c] = aggID
			newState.nodes = append(newState.nodes, node{id: aggID, size: state.nodes[i].size})
		}
	}

	// Create edges
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
			newState.edges[u] = append(newState.edges[u], edge{to: v, weight: w})
		}
	}

	// The new partition P assigns the new aggregate nodes to the communities in the original `partition`.
	// A community in `refinedPartition` is always a subset of a community in `partition`.
	newPartition := make([]int, len(newState.nodes))
	for i, c := range refinedPartition {
		aggID := aggregateMap[c]
		newPartition[aggID] = partition[i]
	}

	return newState, newPartition, aggregateMap
}
