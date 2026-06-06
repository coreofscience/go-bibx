package algorithms

import (
	"math"
	"math/rand"

	"github.com/coreofscience/go-bibx/internal/collections"
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

	isWeighted := l.graph.IsWeighted()
	totalEdgeWeight := 0.0
	nodeDegrees := make(map[K]float64)
	for _, e := range l.graph.AllEdges() {
		w := 1.0
		if isWeighted {
			w = e.Weight()
		}
		totalEdgeWeight += w
		nodeDegrees[e.Source().Label()] += w
		nodeDegrees[e.Destination().Label()] += w
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
		newAggregateID := aggregateMap[refinedCommID]
		globalPartition[k] = newAggregateID
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
			newAggregateID := aggregateMapInt[refinedCommID]
			globalPartition[k] = newAggregateID
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

	queue := collections.NewUniqueQueue[K]()
	for k := range l.sizes {
		queue.Push(k)
	}
	queue.Shuffle()

	isWeighted := l.graph.IsWeighted()
	commWeight := make(map[int]float64)

	for !queue.Empty() {
		vLabel, ok := queue.Pop()
		if !ok {
			// What the queue was empty
			break
		}

		vSize := l.sizes[vLabel]
		currentComm := partition[vLabel]

		clear(commWeight)
		vVertex := l.graph.GetVertexByID(vLabel)
		if vVertex != nil {
			for _, e := range l.graph.EdgesOf(vVertex) {
				neighbor := e.OtherVertex(vLabel)
				nLabel := neighbor.Label()
				if nLabel == vLabel {
					continue
				}
				w := 1.0
				if isWeighted {
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
					if partition[nLabel] != bestComm {
						queue.Push(nLabel)
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
	refinedCommNodeCount := make(map[int]int)
	for k := range l.sizes {
		refinedPartition[k] = nextCommID
		refinedCommNodeCount[nextCommID] = 1
		nextCommID++
	}

	commNodes := make(map[int][]K)
	for k, c := range partition {
		commNodes[c] = append(commNodes[c], k)
	}

	isWeighted := l.graph.IsWeighted()
	vEdgesToC := make(map[int]float64)

	for _, S := range commNodes {
		S_set := make(map[K]struct{})
		S_size := 0.0
		for _, v := range S {
			S_set[v] = struct{}{}
			S_size += l.sizes[v]
		}

		refinedCommSize := make(map[int]float64)
		commEdgesToS := make(map[int]float64)

		for _, u := range S {
			c_u := refinedPartition[u]
			refinedCommSize[c_u] = l.sizes[u]

			evS := 0.0
			uVertex := l.graph.GetVertexByID(u)
			if uVertex != nil {
				for _, e := range l.graph.EdgesOf(uVertex) {
					neighbor := e.OtherVertex(u)
					nLabel := neighbor.Label()
					if nLabel == u {
						continue
					}
					if _, ok := S_set[nLabel]; ok {
						w := 1.0
						if isWeighted {
							w = e.Weight()
						}
						evS += w
					}
				}
			}
			commEdgesToS[c_u] = evS
		}

		R := make([]K, 0)
		for _, vLabel := range S {
			c_v := refinedPartition[vLabel]
			evS := commEdgesToS[c_v]
			vSize := l.sizes[vLabel]
			if evS >= l.gamma*float64(vSize*(S_size-vSize)) {
				R = append(R, vLabel)
			}
		}

		rand.Shuffle(len(R), func(i, j int) { R[i], R[j] = R[j], R[i] })

		for _, v := range R {
			l.refineNode(
				refinedPartition,
				S,
				S_set,
				S_size,
				v,
				refinedCommNodeCount,
				refinedCommSize,
				commEdgesToS,
				vEdgesToC,
			)
		}
	}

	return refinedPartition, nextCommID
}

func (l *Leiden[K]) refineNode(
	refinedPartition map[K]int,
	S []K,
	S_set map[K]struct{},
	S_size float64,
	v K,
	refinedCommNodeCount map[int]int,
	refinedCommSize map[int]float64,
	commEdgesToS map[int]float64,
	vEdgesToC map[int]float64,
) {
	currentRefinedComm := refinedPartition[v]

	isSingleton := refinedCommNodeCount[currentRefinedComm] == 1
	if !isSingleton {
		return
	}

	vSize := l.sizes[v]

	T_list := make([]int, 0)
	for c, cSize := range refinedCommSize {
		if c == currentRefinedComm {
			continue
		}
		ecS := commEdgesToS[c]
		if ecS >= l.gamma*float64(cSize*(S_size-cSize)) {
			T_list = append(T_list, c)
		}
	}

	if len(T_list) > 0 {
		probabilities := make([]float64, len(T_list))
		sumProbabilities := 0.0

		clear(vEdgesToC)
		vVertex := l.graph.GetVertexByID(v)
		isWeighted := l.graph.IsWeighted()
		if vVertex != nil {
			for _, e := range l.graph.EdgesOf(vVertex) {
				neighbor := e.OtherVertex(v)
				nLabel := neighbor.Label()
				if nLabel == v {
					continue
				}
				if _, ok := S_set[nLabel]; ok {
					w := 1.0
					if isWeighted {
						w = e.Weight()
					}
					vEdgesToC[refinedPartition[nLabel]] += w
				}
			}
		}

		for i, c := range T_list {
			delta := l.deltaH(vEdgesToC[c], vSize, refinedCommSize[c])
			if delta >= 0 {
				probabilities[i] = math.Exp(delta / l.theta)
				sumProbabilities += probabilities[i]
			} else {
				probabilities[i] = 0
			}
		}

		if sumProbabilities > 0 {
			randVal := rand.Float64() * sumProbabilities
			cumSum := 0.0
			for i, c := range T_list {
				cumSum += probabilities[i]
				if randVal <= cumSum {
					refinedPartition[v] = c

					refinedCommNodeCount[currentRefinedComm]--
					refinedCommNodeCount[c]++

					refinedCommSize[c] += vSize
					delete(refinedCommSize, currentRefinedComm)

					weight_v_c := vEdgesToC[c]
					commEdgesToS[c] = commEdgesToS[c] + commEdgesToS[currentRefinedComm] - 2*weight_v_c
					delete(commEdgesToS, currentRefinedComm)

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
		if aggregateID, ok := aggregateMap[c]; ok {
			newSizes[aggregateID] += l.sizes[k]
		} else {
			aggregateID = len(aggregateMap)
			aggregateMap[c] = aggregateID
			newSizes[aggregateID] = l.sizes[k]
		}
	}

	// Add vertices to the new graph with their aggregated sizes as weights
	for aggregateID, size := range newSizes {
		newGraph.AddVertexByLabel(aggregateID, gograph.WithVertexWeight(size))
	}

	isWeighted := l.graph.IsWeighted()
	edgeMap := make(map[int]map[int]float64)
	for _, e := range l.graph.AllEdges() {
		u := e.Source().Label()
		v := e.Destination().Label()
		aggregateU := aggregateMap[refinedPartition[u]]
		aggregateV := aggregateMap[refinedPartition[v]]
		if aggregateU != aggregateV {
			w := 1.0
			if isWeighted {
				w = e.Weight()
			}
			if aggregateU > aggregateV {
				aggregateU, aggregateV = aggregateV, aggregateU
			}
			if edgeMap[aggregateU] == nil {
				edgeMap[aggregateU] = make(map[int]float64)
			}
			edgeMap[aggregateU][aggregateV] += w
		}
	}

	// Add edges to the new graph
	for aggregateU, neighbors := range edgeMap {
		for aggregateV, weight := range neighbors {
			_, _ = newGraph.AddEdge(
				newGraph.GetVertexByID(aggregateU),
				newGraph.GetVertexByID(aggregateV),
				gograph.WithEdgeWeight(weight),
			)
		}
	}

	newPartition := make(map[int]int)
	for k, c := range refinedPartition {
		aggregateID := aggregateMap[c]
		newPartition[aggregateID] = partition[k]
	}

	return newGraph, newSizes, newPartition, aggregateMap
}
