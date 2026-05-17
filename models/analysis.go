package models

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/hmdsefi/gograph"
)

type Category string

const (
	CategoryRoot  Category = "root"
	CategoryTrunk Category = "trunk"
	CategoryLeaf  Category = "leaf"
)

type Node struct {
	ID        string   `json:"id"`
	Category  Category `json:"category"`
	Rootness  float64  `json:"rootness"`
	Trunkness float64  `json:"trunkness"`
	Leafness  float64  `json:"leafness"`
	Article   *Article `json:"article"`
}

type Result struct {
	Score   float64  `json:"score"`
	Article *Article `json:"article"`
}

type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Analysis struct {
	Nodes []*Node `json:"nodes"`
	Links []*Link `json:"links"`
}

func (a *Analysis) CitationGraph() (gograph.Graph[string], error) {
	graph := gograph.New[string](gograph.Directed(), gograph.Acyclic())
	for _, link := range a.Links {
		_, err := graph.AddEdge(
			gograph.NewVertex(link.Source),
			gograph.NewVertex(link.Target),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to add edge: %w", err)
		}
	}
	return graph, nil
}

func (a *Analysis) Keep(ids []string) *Analysis {
	toKeep := collections.NewSet(ids...)
	nodes := make([]*Node, 0, len(a.Nodes))
	for _, node := range a.Nodes {
		if toKeep.Contains(node.ID) {
			nodes = append(nodes, node)
		}
	}
	links := make([]*Link, 0, len(a.Links))
	for _, link := range a.Links {
		if toKeep.Contains(link.Source) && toKeep.Contains(link.Target) {
			links = append(links, link)
		}
	}
	return &Analysis{
		Nodes: nodes,
		Links: links,
	}
}

func (a *Analysis) Query(category string, top int) ([]*Result, error) {
	results := make([]*Result, 0, top)
	scored := make([]*Result, 0, len(a.Nodes))
	for _, node := range a.Nodes {
		var score float64
		switch Category(category) {
		case CategoryRoot:
			score = node.Rootness
		case CategoryTrunk:
			score = node.Trunkness
		case CategoryLeaf:
			score = node.Leafness
		default:
			score = node.Rootness
		}
		if score == 0 {
			continue
		}
		scored = append(scored, &Result{
			Score:   score,
			Article: node.Article,
		})
	}
	slices.SortFunc(scored, func(a, b *Result) int {
		return cmp.Compare(b.Score, a.Score)
	})

	limit := min(top, len(scored))
	for _, node := range scored[:limit] {
		if node.Score == 0 {
			break
		}
		results = append(results, node)
	}
	return results, nil
}
