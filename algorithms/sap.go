package algorithms

import (
	"cmp"
	"errors"
	"fmt"
	"log/slog"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/hmdsefi/gograph"
)

// Sap represents the Tree of Science SAP algorithm for citation network analysis.
// It calculates rootness, trunkness, and leafness scores to classify works into
// roots, trunk, and leaves.
type Sap struct {
	graph         gograph.Graph[string]
	roots         int
	trunks        int
	leaves        int
	order         []*gograph.Vertex[string]
	invertedOrder []*gograph.Vertex[string]
}

// Category represents the Tree of Science classification of a work node.
type Category string

const (
	// CategoryNone indicates that the work is not classified into any Tree of Science category.
	CategoryNone Category = "none"
	// CategoryRoot indicates foundational, seminal works that established the research field.
	CategoryRoot Category = "root"
	// CategoryTrunk indicates structural backbone papers that consolidate the research topic.
	CategoryTrunk Category = "trunk"
	// CategoryLeaf indicates recent, specialized developments representing the research frontier.
	CategoryLeaf Category = "leaf"
)

// SapResult contains the outcome of the SAP algorithm, including category assignments
// and metric scores (rootness, trunkness, and leafness) for each vertex in the graph.
type SapResult struct {
	Categories map[string]Category
	Rootness   map[string]float64
	Trunkness  map[string]float64
	Leafness   map[string]float64
}

// SapOption configures a [Sap] algorithm instance.
type SapOption func(*Sap)

// WithRoots sets the maximum number of root nodes to select.
func WithRoots(roots int) SapOption {
	return func(s *Sap) {
		s.roots = roots
	}
}

// WithTrunks sets the maximum number of trunk nodes to select.
func WithTrunks(trunks int) SapOption {
	return func(s *Sap) {
		s.trunks = trunks
	}
}

// WithLeaves sets the maximum number of leaf nodes to select.
func WithLeaves(leaves int) SapOption {
	return func(s *Sap) {
		s.leaves = leaves
	}
}

// NewSapAlgorithm creates a new instance of the [Sap] algorithm for the given directed citation graph.
// It computes topological orders required for sap flow simulations. Returns an error if the graph is undirected
// or contains cycles.
func NewSapAlgorithm(graph gograph.Graph[string], opts ...SapOption) (*Sap, error) {
	if !graph.IsDirected() {
		return nil, errors.New("graph must be directed")
	}

	order, err := graphs.ReverseTopologicalOrder(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to compute reverse topological order: %w", err)
	}

	invertedGraph := graphs.Invert(graph)
	invertedOrder, err := graphs.ReverseTopologicalOrder(invertedGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to compute reverse topological order for inverted graph: %w", err)
	}

	s := &Sap{
		graph:         graph,
		roots:         20,
		trunks:        20,
		leaves:        50,
		order:         order,
		invertedOrder: invertedOrder,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// Run executes the SAP algorithm and returns the [SapResult] containing work categories and scores.
func (s *Sap) Run() (*SapResult, error) {
	rootness := s.computeRootness()
	leafness := s.computeLeafness(rootness)
	trunkness := s.computeTrunkness(rootness, leafness)
	categories := s.computeCategories(rootness, trunkness, leafness)
	return &SapResult{
		Categories: categories,
		Rootness:   rootness,
		Trunkness:  trunkness,
		Leafness:   leafness,
	}, nil
}

func (s *Sap) computeRootness() map[string]float64 {
	potentialRoots := make([]property, 0, s.roots)
	for _, vertex := range s.graph.GetAllVertices() {
		if vertex.OutDegree() == 0 && vertex.InDegree() > 0 {
			potentialRoots = append(potentialRoots, property{
				label: vertex.Label(),
				prop:  float64(vertex.InDegree()),
			})
		}
	}
	roots := utils.Limit(potentialRoots, s.roots, func(a, b property) int {
		return cmp.Compare(b.prop, a.prop)
	})
	rootness := make(map[string]float64)
	for _, root := range roots {
		rootness[root.label] = root.prop
	}
	slog.Debug("found root nodes", "count", len(rootness))
	return rootness
}

func (s *Sap) computeLeafness(
	rootness map[string]float64,
) map[string]float64 {
	rootConnections := s.computeRootConnections(rootness)
	slog.Debug("found root connections", "count", len(rootConnections))
	potentialLeaves := make([]property, 0, s.leaves)
	for _, vertex := range s.graph.GetAllVertices() {
		if vertex.InDegree() > 0 {
			continue
		}
		if connections := rootConnections[vertex.Label()]; connections > 0 {
			potentialLeaves = append(potentialLeaves, property{
				label: vertex.Label(),
				prop:  connections,
			})
		}
	}
	leaves := utils.Limit(potentialLeaves, s.leaves, func(a, b property) int {
		return cmp.Compare(b.prop, a.prop)
	})
	leafness := make(map[string]float64)
	for _, leaf := range leaves {
		leafness[leaf.label] = leaf.prop
	}
	slog.Debug("found leaf nodes", "count", len(leafness))
	return leafness
}

func (s *Sap) computeRootConnections(
	rootness map[string]float64,
) map[string]float64 {
	if len(rootness) == 0 {
		return nil
	}
	rootConnections := make(map[string]float64)
	for label := range rootness {
		if value, ok := rootness[label]; ok && value > 0 {
			rootConnections[label] = 1
		}
	}
	for _, vertex := range s.order {
		connections := float64(0)
		neighbors := vertex.Neighbors()
		if len(neighbors) == 0 {
			continue
		}
		for _, neighbor := range neighbors {
			if existing, ok := rootConnections[neighbor.Label()]; ok {
				connections += existing
			}
		}
		if connections > 0 {
			rootConnections[vertex.Label()] = connections
		}
	}
	return rootConnections
}

func (s *Sap) computeLeafConnections(
	leafness map[string]float64,
) map[string]float64 {
	if len(leafness) == 0 {
		slog.Debug("no leafness to compute leaf connections")
		return nil
	}
	leafConnections := make(map[string]float64)
	for label := range leafness {
		if value, ok := leafness[label]; ok && value > 0 {
			leafConnections[label] = 1
		}
	}
	for _, vertex := range s.invertedOrder {
		connections := float64(0)
		neighbors := vertex.Neighbors()
		if len(neighbors) == 0 {
			continue
		}
		for _, neighbor := range neighbors {
			if existing, ok := leafConnections[neighbor.Label()]; ok {
				connections += existing
			}
		}
		if connections > 0 {
			leafConnections[vertex.Label()] = connections
		}
	}
	return leafConnections
}

// Raw sap flows from roots to leaves
func (s *Sap) computeRawSap(
	rootness map[string]float64,
) map[string]float64 {
	if len(rootness) == 0 {
		return nil
	}
	rawSap := make(map[string]float64)
	for label := range rootness {
		if value, ok := rootness[label]; ok && value > 0 {
			rawSap[label] = value
		}
	}
	for _, vertex := range s.order {
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
		if vertexRawSap > 0 {
			rawSap[vertex.Label()] = vertexRawSap
		}
	}
	return rawSap
}

// Elaborate sap flows from leaves to roots
func (s *Sap) computeElaborateSap(
	leafness map[string]float64,
) map[string]float64 {
	if len(leafness) == 0 {
		slog.Debug("no leafness to compute elaborate sap")
		return nil
	}
	elaborateSap := make(map[string]float64)
	for label := range leafness {
		if value, ok := leafness[label]; ok && value > 0 {
			elaborateSap[label] = value
		}
	}
	for _, vertex := range s.invertedOrder {
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
		if vertexElaborateSap > 0 {
			elaborateSap[vertex.Label()] = vertexElaborateSap
		}
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
		score := leafConnections[label]*rawSap[label] +
			rootConnections[label]*elaborateSap[label]
		if score > 0 {
			sap[label] = score
		}
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
		if prop <= 0 {
			continue
		}
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
	trunk := utils.Limit(potentialTrunk, s.trunks, func(a, b property) int {
		return cmp.Compare(b.prop, a.prop)
	})
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
