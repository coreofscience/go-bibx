package algorithms

import (
	"fmt"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/hmdsefi/gograph"
)

type Leiden[K comparable] struct {
	graph            gograph.Graph[K]
	communityDegrees map[int]int
	m                float64
	gamma            float64
}

type LeidenOption[K comparable] func(*Leiden[K])

func WithGamma[K comparable](gamma float64) LeidenOption[K] {
	return func(l *Leiden[K]) {
		l.gamma = gamma
	}
}

func NewLeiden[K comparable](graph gograph.Graph[K], opts ...LeidenOption[K]) *Leiden[K] {
	initialDegrees := make(map[int]int)
	for i, vertex := range graph.GetAllVertices() {
		initialDegrees[i] = vertex.Degree()
	}
	m := 0.0
	for _, edge := range graph.AllEdges() {
		if graph.IsWeighted() {
			m += edge.Weight()
		} else {
			m += 1.0
		}
	}
	l := &Leiden[K]{
		graph:            graph,
		communityDegrees: initialDegrees,
		m:                m,
		gamma:            1.0,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *Leiden[K]) Run() (map[K]int, error) {
	partition := l.singletonPartition()
	newPartition, err := l.localMove(partition)
	if err != nil {
		return nil, fmt.Errorf("error during local move: %w", err)
	}
	return newPartition, nil
}

func (l *Leiden[K]) singletonPartition() map[K]int {
	partition := make(map[K]int)
	for i, vertex := range l.graph.GetAllVertices() {
		partition[vertex.Label()] = i
	}
	return partition
}

func (l *Leiden[K]) localMove(partition map[K]int) (map[K]int, error) {
	done := false
	for !done {
		done = true
		queue := collections.NewQueue[K]()
		for _, vertex := range l.graph.GetAllVertices() {
			queue.Enqueue(vertex.Label())
		}
		queue.Shuffle()
		for !queue.Empty() {
			v, err := queue.Dequeue()
			if err != nil {
				return nil, fmt.Errorf("dequeue error: %w", err)
			}
			currentCommunity := partition[v]

			bestCommunity := currentCommunity
			bestDelta := 0.0

			vertex := l.graph.GetVertexByID(v)
			neighborWeights, err := l.neighborWeights(partition, v)
			if err != nil {
				return nil, fmt.Errorf("error calculating neighbor weights: %w", err)
			}
			currentWeight := neighborWeights[currentCommunity]
			for neighboringCommunity, neighboringWeight := range neighborWeights {
				delta := l.deltaQuality(
					vertex,
					currentCommunity,
					neighboringCommunity,
					currentWeight,
					neighboringWeight,
				)
				if delta > bestDelta {
					bestDelta = delta
					bestCommunity = neighboringCommunity
				}
			}
			if bestCommunity != currentCommunity {
				// Move vertex to the best community
				partition[v] = bestCommunity
				l.communityDegrees[currentCommunity] -= vertex.Degree()
				l.communityDegrees[bestCommunity] += vertex.Degree()

				// Enqueue neighbors for potential moves
				for _, neighbor := range vertex.Neighbors() {
					isSelfLoop := neighbor.Label() == vertex.Label()
					belongsToSameCommunity := partition[neighbor.Label()] == bestCommunity
					if isSelfLoop || belongsToSameCommunity {
						continue
					}
					queue.Enqueue(neighbor.Label())
				}

				// Mark that we made a change, so we need to check for more moves
				done = false
			}
		}
	}
	return partition, nil
}

func (l *Leiden[K]) neighborWeights(partition map[K]int, v K) (map[int]float64, error) {
	vertex := l.graph.GetVertexByID(v)
	if vertex == nil {
		return nil, fmt.Errorf("vertex with label %v not found", v)
	}
	weights := make(map[int]float64)
	for _, edge := range l.graph.EdgesOf(vertex) {
		neighbor := edge.Destination()
		if neighbor.Label() == vertex.Label() {
			continue
		}
		neighborCommunity := partition[neighbor.Label()]
		edgeWeight := float64(1)
		if l.graph.IsWeighted() {
			edgeWeight = edge.Weight()
		}
		weights[neighborCommunity] += edgeWeight
	}
	return weights, nil
}

func (l *Leiden[K]) deltaQuality(
	vertex *gograph.Vertex[K],
	communityFrom int,
	communityTo int,
	weightFrom float64,
	weightTo float64,
) float64 {
	gain := weightTo - l.gamma*float64(vertex.Degree())*float64(l.communityDegrees[communityTo])/(2*l.m)
	loss := weightFrom - l.gamma*float64(vertex.Degree())*float64(l.communityDegrees[communityFrom]-vertex.Degree())/(2*l.m)
	return (gain - loss) / l.m
}

func (l *Leiden[K]) refinePartition(partition map[K]int) (map[K]int, error) {
	refinedPartition := l.singletonPartition()
	communityMap := make(map[int][]K)
	for vertexLabel, community := range partition {
		communityMap[community] = append(communityMap[community], vertexLabel)
	}
	for community, vertexLabels := range communityMap {
		refinedVertexLabels := make([]K, 0, len(vertexLabels))
		for _, vertexLabel := range vertexLabels {
			vertex := l.graph.GetVertexByID(vertexLabel)
			if vertex == nil {
				return nil, fmt.Errorf("vertex with label %v not found", vertexLabel)
			}
			edgeWeights := float64(0)
			for _, edge := range l.graph.EdgesOf(vertex) {
				neighbor := edge.Destination()
				isSameVertex := neighbor.Label() == vertex.Label()
				isDifferentCommunity := partition[neighbor.Label()] != community
				if isSameVertex || isDifferentCommunity {
					continue
				}
				edgeWeight := float64(1)
				if l.graph.IsWeighted() {
					edgeWeight = edge.Weight()
				}
				edgeWeights += edgeWeight
			}
			if edgeWeights > l.gamma*float64(vertex.Degree())*float64(l.communityDegrees[community]-vertex.Degree()) {
				refinedVertexLabels = append(refinedVertexLabels, vertexLabel)
			}
		}
		// for _, vertexLabel := range refinedVertexLabels {
		// 	isSingleton := true

		// }
	}
	return refinedPartition, nil
}
