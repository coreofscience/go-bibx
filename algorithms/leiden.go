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
	sizes map[K]float64
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
	for _, e := range l.graph.AllEdges() {
		u := e.Source().Label()
		v := e.Destination().Label()
		w := 1.0
		if l.graph.IsWeighted() {
			w = e.Weight()
		}
		nodeDegrees[u] += w
		nodeDegrees[v] += w
	}

	m2 := totalEdgeWeight * 2
	if m2 == 0 {
		m2 = 1
	}
	l.gamma = l.gamma / m2
	l.sizes = nodeDegrees

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
	newPartition, nextCommID := l.moveNodesFast(partition, nextCommID)

	uniqueComms := make(map[int]struct{})
	for _, c := range newPartition {
		uniqueComms[c] = struct{}{}
	}
	if len(uniqueComms) == len(l.sizes) {
		return l.normalizeResult(globalPartition)
	}

	refinedPartition, nextCommID := l.refinePartition(newPartition, nextCommID)

	// Create int-based aggregate graph
	intGraph, intSizes, intPartition, aggregateMap := l.aggregateGraph(refinedPartition, newPartition)

	// Update global partition
	for k := range globalPartition {
		refinedCommID := refinedPartition[k]
		newAggID := aggregateMap[refinedCommID]
		globalPartition[k] = newAggID
	}

	// Subsequent iterations on int
	done := false
	for !done {
		intLeiden := &Leiden[int]{
			graph: intGraph,
			gamma: l.gamma,
			theta: l.theta,
			sizes: intSizes,
		}

		intPartition, nextCommID = intLeiden.moveNodesFast(intPartition, nextCommID)

		uniqueComms := make(map[int]struct{})
		for _, c := range intPartition {
			uniqueComms[c] = struct{}{}
		}
		if len(uniqueComms) == len(intSizes) {
			break
		}

		refinedPartitionInt, _ := intLeiden.refinePartition(intPartition, nextCommID)

		newStateGraph, newStateSizes, newPartitionInt, aggregateMapInt := intLeiden.aggregateGraph(refinedPartitionInt, intPartition)

		for k := range globalPartition {
			refinedCommID := refinedPartitionInt[globalPartition[k]]
			newAggID := aggregateMapInt[refinedCommID]
			globalPartition[k] = newAggID
		}

		intGraph = newStateGraph
		intSizes = newStateSizes
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
// Leiden methods for managing partition steps
// ---------------------------------------------------------

func (l *Leiden[K]) moveNodesFast(partition map[K]int, nextCommID int) (map[K]int, int) {
	commSize := make(map[int]float64)
	for k, comm := range partition {
		commSize[comm] += l.sizes[k]
	}

	queue := make([]K, 0, len(l.sizes))
	inQueue := make(map[K]bool)
	for k := range l.sizes {
		queue = append(queue, k)
		inQueue[k] = true
	}
	rand.Shuffle(len(queue), func(i, j int) { queue[i], queue[j] = queue[j], queue[i] })

	for len(queue) > 0 {
		vLabel := queue[0]
		queue = queue[1:]
		inQueue[vLabel] = false

		vSize := l.sizes[vLabel]
		currentComm := partition[vLabel]

		commWeight := make(map[int]float64)
		vVertex := l.graph.GetVertexByID(vLabel)
		if vVertex != nil {
			for _, e := range l.graph.EdgesOf(vVertex) {
				neighbor := e.OtherVertex(vLabel)
				nLabel := neighbor.Label()
				if nLabel == vLabel {
					continue
				}
				w := 1.0
				if l.graph.IsWeighted() {
					w = e.Weight()
				}
				commWeight[partition[nLabel]] += w
			}
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
			partition[vLabel] = bestComm
			commSize[bestComm] += vSize

			if vVertex != nil {
				for _, e := range l.graph.EdgesOf(vVertex) {
					neighbor := e.OtherVertex(vLabel)
					nLabel := neighbor.Label()
					if nLabel == vLabel {
						continue
					}
					if partition[nLabel] != bestComm && !inQueue[nLabel] {
						queue = append(queue, nLabel)
						inQueue[nLabel] = true
					}
				}
			}
		} else {
			commSize[currentComm] += vSize
		}
	}

	return partition, nextCommID
}

func (l *Leiden[K]) refinePartition(partition map[K]int, nextCommID int) (map[K]int, int) {
	refinedPartition := make(map[K]int)
	for k := range l.sizes {
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
			S_size += l.sizes[v]
		}

		R := make([]K, 0)
		for _, vLabel := range S {
			vSize := l.sizes[vLabel]
			evS := 0.0
			vVertex := l.graph.GetVertexByID(vLabel)
			if vVertex != nil {
				for _, e := range l.graph.EdgesOf(vVertex) {
					neighbor := e.OtherVertex(vLabel)
					nLabel := neighbor.Label()
					if nLabel == vLabel {
						continue
					}
					if _, ok := S_set[nLabel]; ok {
						w := 1.0
						if l.graph.IsWeighted() {
							w = e.Weight()
						}
						evS += w
					}
				}
			}
			if evS >= l.gamma*float64(vSize*(S_size-vSize)) {
				R = append(R, vLabel)
			}
		}

		rand.Shuffle(len(R), func(i, j int) { R[i], R[j] = R[j], R[i] })

		for _, v := range R {
			l.refineNode(refinedPartition, S, S_set, S_size, v)
		}
	}

	return refinedPartition, nextCommID
}

func (l *Leiden[K]) refineNode(refinedPartition map[K]int, S []K, S_set map[K]struct{}, S_size float64, v K) {
	currentRefinedComm := refinedPartition[v]

	isSingleton := true
	for k := range l.sizes {
		if k != v && refinedPartition[k] == currentRefinedComm {
			isSingleton = false
			break
		}
	}

	if !isSingleton {
		return
	}

	vSize := l.sizes[v]

	refinedCommSize := make(map[int]float64)
	for _, u := range S {
		refinedCommSize[refinedPartition[u]] += l.sizes[u]
	}

	T_list := make([]int, 0)
	for c := range refinedCommSize {
		if c == currentRefinedComm {
			continue
		}
		ecS := 0.0
		cSize := refinedCommSize[c]
		for _, u := range S {
			if refinedPartition[u] == c {
				uVertex := l.graph.GetVertexByID(u)
				if uVertex != nil {
					for _, e := range l.graph.EdgesOf(uVertex) {
						neighbor := e.OtherVertex(u)
						nLabel := neighbor.Label()
						if nLabel == u {
							continue
						}
						if _, ok := S_set[nLabel]; ok && refinedPartition[nLabel] != c {
							w := 1.0
							if l.graph.IsWeighted() {
								w = e.Weight()
							}
							ecS += w
						}
					}
				}
			}
		}

		if ecS >= l.gamma*float64(cSize*(S_size-cSize)) {
			T_list = append(T_list, c)
		}
	}

	if len(T_list) > 0 {
		probs := make([]float64, len(T_list))
		sumProbs := 0.0

		vEdgesToC := make(map[int]float64)
		vVertex := l.graph.GetVertexByID(v)
		if vVertex != nil {
			for _, e := range l.graph.EdgesOf(vVertex) {
				neighbor := e.OtherVertex(v)
				nLabel := neighbor.Label()
				if nLabel == v {
					continue
				}
				if _, ok := S_set[nLabel]; ok {
					w := 1.0
					if l.graph.IsWeighted() {
						w = e.Weight()
					}
					vEdgesToC[refinedPartition[nLabel]] += w
				}
			}
		}

		for i, c := range T_list {
			delta := l.deltaH(vEdgesToC[c], vSize, refinedCommSize[c])
			if delta >= 0 {
				probs[i] = math.Exp(delta / l.theta)
				sumProbs += probs[i]
			} else {
				probs[i] = 0
			}
		}

		if sumProbs > 0 {
			randVal := rand.Float64() * sumProbs
			cumSum := 0.0
			for i, c := range T_list {
				cumSum += probs[i]
				if randVal <= cumSum {
					refinedPartition[v] = c
					break
				}
			}
		}
	}
}

func (l *Leiden[K]) aggregateGraph(refinedPartition map[K]int, partition map[K]int) (gograph.Graph[int], map[int]float64, map[int]int, map[int]int) {
	newGraph := gograph.New[int](
		gograph.Weighted(),
	)
	newSizes := make(map[int]float64)
	aggregateMap := make(map[int]int)

	for k, c := range refinedPartition {
		if aggID, ok := aggregateMap[c]; ok {
			newSizes[aggID] += l.sizes[k]
		} else {
			aggID = len(aggregateMap)
			aggregateMap[c] = aggID
			newSizes[aggID] = l.sizes[k]
		}
	}

	// Add vertices to the new graph with their aggregated sizes as weights
	for aggID, size := range newSizes {
		newGraph.AddVertexByLabel(aggID, gograph.WithVertexWeight(size))
	}

	edgeMap := make(map[int]map[int]float64)
	for _, uVertex := range l.graph.GetAllVertices() {
		u := uVertex.Label()
		aggU := aggregateMap[refinedPartition[u]]
		if edgeMap[aggU] == nil {
			edgeMap[aggU] = make(map[int]float64)
		}
		for _, e := range l.graph.EdgesOf(uVertex) {
			neighbor := e.OtherVertex(u)
			v := neighbor.Label()
			aggV := aggregateMap[refinedPartition[v]]
			if aggU != aggV {
				w := 1.0
				if l.graph.IsWeighted() {
					w = e.Weight()
				}
				edgeMap[aggU][aggV] += w
			}
		}
	}

	// Add edges to the new graph
	for aggU, neighbors := range edgeMap {
		for aggV, weight := range neighbors {
			if aggU < aggV {
				_, _ = newGraph.AddEdge(
					newGraph.GetVertexByID(aggU),
					newGraph.GetVertexByID(aggV),
					gograph.WithEdgeWeight(weight),
				)
			}
		}
	}

	newPartition := make(map[int]int)
	for k, c := range refinedPartition {
		aggID := aggregateMap[c]
		newPartition[aggID] = partition[k]
	}

	return newGraph, newSizes, newPartition, aggregateMap
}
