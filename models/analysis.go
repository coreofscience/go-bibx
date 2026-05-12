package models

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/coreofscience/go-bibx/algorithms"
)

type Node struct {
	ID        string              `json:"id"`
	Category  algorithms.Category `json:"category"`
	Rootness  float64             `json:"rootness"`
	Trunkness float64             `json:"trunkness"`
	Leafness  float64             `json:"leafness"`
	Article   *Article            `json:"article"`
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

func NewAnalysis(c *Collection) *Analysis {
	nodes := make([]*Node, 0, c.Len())
	graph, err := c.CitationGraph()
	if err != nil {
		return nil
	}
	sap := algorithms.NewSap(graph)
	result := sap.Run()
	for article := range c.All() {
		articleKey := article.Key()
		if articleKey == nil {
			slog.Warn("article without key, skipping", "label", article.Label)
			continue
		}
		key := *articleKey
		nodes = append(nodes, &Node{
			ID:        key,
			Article:   article,
			Category:  result.Categories[key],
			Rootness:  result.Rootness[key],
			Trunkness: result.Trunkness[key],
			Leafness:  result.Leafness[key],
		})
	}
	links := make([]*Link, 0, c.Len())
	for _, edge := range graph.AllEdges() {
		links = append(links, &Link{
			Source: edge.Source().Label(),
			Target: edge.Destination().Label(),
		})
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
		switch algorithms.Category(category) {
		case algorithms.CategoryRoot:
			score = node.Rootness
		case algorithms.CategoryTrunk:
			score = node.Trunkness
		case algorithms.CategoryLeaf:
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
	for _, node := range scored[:top] {
		if node.Score == 0 {
			break
		}
		results = append(results, node)
	}
	return results, nil
}
