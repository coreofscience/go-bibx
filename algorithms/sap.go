package algorithms

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
)

type Sap struct {
	graph  gograph.Graph[string]
	roots  int
	trunks int
	leaves int
}

type Category string

const (
	CategoryNone  Category = "none"
	CategoryRoot  Category = "root"
	CategoryTrunk Category = "trunk"
	CategoryLeaf  Category = "leaf"
)

type SapResult struct {
	Categories map[string]Category
	Rootness   map[string]float64
	Trunkness  map[string]float64
	Leafness   map[string]float64
}

type SapOption func(*Sap)

func WithRoots(roots int) SapOption {
	return func(s *Sap) {
		s.roots = roots
	}
}

func WithTrunks(trunks int) SapOption {
	return func(s *Sap) {
		s.trunks = trunks
	}
}

func WithLeaves(leaves int) SapOption {
	return func(s *Sap) {
		s.leaves = leaves
	}
}

func NewSap(graph gograph.Graph[string], opts ...SapOption) *Sap {
	s := &Sap{graph: graph, roots: 20, trunks: 20, leaves: 50}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Sap) Run() *SapResult {
	rootness := s.computeRootness()
	leafness := s.computeLeafness(rootness)
	trunkness := s.computeTrunkness(rootness, leafness)
	categories := s.computeCategories(rootness, trunkness, leafness)
	return &SapResult{
		Categories: categories,
		Rootness:   rootness,
		Trunkness:  trunkness,
		Leafness:   leafness,
	}
}

func (s *Sap) computeRootness() map[string]float64 {
	potentialRoots := make([]property, 0, s.roots)
	for _, vertex := range s.graph.GetAllVertices() {
		if vertex.OutDegree() == 0 {
			potentialRoots = append(potentialRoots, property{
				label: vertex.Label(),
				prop:  float64(vertex.InDegree()),
			})
		}
	}
	roots := limit(potentialRoots, s.roots)
	rootness := make(map[string]float64)
	for _, root := range roots {
		rootness[root.label] = root.prop
	}
	return rootness
}

func (s *Sap) computeLeafness(rootness map[string]float64) map[string]float64 {
	rootConnections := s.computeRootConnections(rootness)
	potentialLeaves := make([]property, 0, s.leaves)
	for _, vertex := range s.graph.GetAllVertices() {
		if vertex.InDegree() > 0 {
			continue
		}
		if connections, ok := rootConnections[vertex.Label()]; ok {
			potentialLeaves = append(potentialLeaves, property{
				label: vertex.Label(),
				prop:  float64(connections),
			})
		}
	}
	leaves := limit(potentialLeaves, s.leaves)
	leafness := make(map[string]float64)
	for _, leaf := range leaves {
		leafness[leaf.label] = leaf.prop
	}
	slog.Debug("found leaf nodes", "count", len(leafness))
	return leafness
}

func (s *Sap) computeRootConnections(rootness map[string]float64) map[string]int64 {
	if len(rootness) == 0 {
		return nil
	}
	rootConnections := make(map[string]int64)
	for label := range rootness {
		if value, ok := rootness[label]; ok && value > 0 {
			rootConnections[label] = 1
		}
	}
	order, err := graphs.ReverseTopologicalOrder(s.graph)
	if err != nil {
		return nil
	}
	for _, vertex := range order {
		connections := int64(0)
		neighbors := vertex.Neighbors()
		if len(neighbors) == 0 {
			continue
		}
		for _, neighbor := range neighbors {
			if existing, ok := rootConnections[neighbor.Label()]; ok {
				connections += existing
			}
		}
		rootConnections[vertex.Label()] = connections
	}
	return rootConnections
}

func (s *Sap) computeLeafConnections(leafness map[string]float64) map[string]int64 {
	if len(leafness) == 0 {
		slog.Debug("no leafness to compute leaf connections")
		return nil
	}
	invertedGraph := graphs.Invert(s.graph)
	slog.Debug("inverted graph", "order", invertedGraph.Order(), "size", invertedGraph.Size())
	leafConnections := make(map[string]int64)
	for label := range leafness {
		if value, ok := leafness[label]; ok && value > 0 {
			leafConnections[label] = 1
		}
	}
	order, err := graphs.ReverseTopologicalOrder(invertedGraph)
	if err != nil {
		slog.Error("failed to compute reverse topological order", "error", err)
		return nil
	}
	for _, vertex := range order {
		connections := int64(0)
		neighbors := vertex.Neighbors()
		if len(neighbors) == 0 {
			continue
		}
		for _, neighbor := range neighbors {
			if existing, ok := leafConnections[neighbor.Label()]; ok {
				connections += existing
			}
		}
		leafConnections[vertex.Label()] = connections
	}
	return leafConnections
}

// Raw sap flows from roots to leaves
func (s *Sap) computeRawSap(rootness map[string]float64) map[string]float64 {
	if len(rootness) == 0 {
		return nil
	}
	rawSap := make(map[string]float64)
	for label := range rootness {
		if value, ok := rootness[label]; ok && value > 0 {
			rawSap[label] = value
		}
	}
	order, err := graphs.ReverseTopologicalOrder(s.graph)
	if err != nil {
		return nil
	}
	for _, vertex := range order {
		vertexRawSap := float64(0)
		neighbors := vertex.Neighbors()
		if len(neighbors) == 0 {
			continue
		}
		for _, neighbor := range neighbors {
			if existing, ok := rawSap[neighbor.Label()]; ok {
				vertexRawSap += existing
			}
		}
		rawSap[vertex.Label()] = vertexRawSap
	}
	return rawSap
}

// Elaborate sap flows from leaves to roots
func (s *Sap) computeElaborateSap(leafness map[string]float64) map[string]float64 {
	if len(leafness) == 0 {
		slog.Debug("no leafness to compute elaborate sap")
		return nil
	}
	invertedGraph := graphs.Invert(s.graph)
	elaborateSap := make(map[string]float64)
	for label := range leafness {
		if value, ok := leafness[label]; ok && value > 0 {
			elaborateSap[label] = value
		}
	}
	order, err := graphs.ReverseTopologicalOrder(invertedGraph)
	if err != nil {
		return nil
	}
	for _, vertex := range order {
		vertexElaborateSap := float64(0)
		neighbors := vertex.Neighbors()
		if len(neighbors) == 0 {
			continue
		}
		for _, neighbor := range neighbors {
			if existing, ok := elaborateSap[neighbor.Label()]; ok {
				vertexElaborateSap += existing
			}
		}
		elaborateSap[vertex.Label()] = vertexElaborateSap
	}
	return elaborateSap
}

// Sap flows from roots to leaves and back
func (s *Sap) computeSap(
	rootness map[string]float64,
	leafness map[string]float64,
) map[string]float64 {
	rootConnections := s.computeRootConnections(rootness)
	slog.Debug("computing root connections", "count", len(rootConnections))
	leafConnections := s.computeLeafConnections(leafness)
	slog.Debug("computing leaf connections", "count", len(leafConnections))
	rawSap := s.computeRawSap(rootness)
	slog.Debug("computing raw sap", "count", len(rawSap))
	elaborateSap := s.computeElaborateSap(leafness)
	slog.Debug("computing elaborate sap", "count", len(elaborateSap))
	sap := make(map[string]float64)
	for label := range rawSap {
		sap[label] = (float64(leafConnections[label])*rawSap[label] +
			float64(rootConnections[label])*elaborateSap[label])
	}
	return sap
}

func (s *Sap) computeTrunkness(
	rootness map[string]float64,
	leafness map[string]float64,
) map[string]float64 {
	sap := s.computeSap(rootness, leafness)
	slog.Debug("computing trunkness", "count", len(sap))
	potentialTrunk := make([]property, 0, len(sap))
	for label, prop := range sap {
		if _, ok := leafness[label]; ok {
			continue
		}
		if _, ok := rootness[label]; ok {
			continue
		}
		potentialTrunk = append(potentialTrunk, property{
			label: label,
			prop:  prop,
		})
	}
	trunk := limit(potentialTrunk, s.trunks)
	result := make(map[string]float64, len(trunk))
	for _, prop := range trunk {
		result[prop.label] = prop.prop
	}
	return result
}

func (s *Sap) computeCategories(
	rootness map[string]float64,
	trunkness map[string]float64,
	leafness map[string]float64,
) map[string]Category {
	categories := make(map[string]Category, len(rootness))
	for _, v := range s.graph.GetAllVertices() {
		categories[v.Label()] = CategoryNone
	}
	slog.Debug("labeling leaf nodes", "count", len(leafness))
	for label, value := range leafness {
		if value > 0 {
			categories[label] = CategoryLeaf
		}
	}
	slog.Debug("labeling trunk nodes", "count", len(trunkness))
	for label, value := range trunkness {
		if value > 0 {
			categories[label] = CategoryTrunk
		}
	}
	slog.Debug("labeling root nodes", "count", len(rootness))
	for label, value := range rootness {
		if value > 0 {
			categories[label] = CategoryRoot
		}
	}
	return categories
}

type property struct {
	label string
	prop  float64
}

func limit(properties []property, n int) []property {
	if len(properties) <= n {
		return properties
	}
	slices.SortFunc(properties, func(a, b property) int {
		return cmp.Compare(b.prop, a.prop)
	})
	return properties[:n]
}
