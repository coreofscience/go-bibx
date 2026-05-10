package analysis

import (
	"github.com/coreofscience/go-bibx/algorithms"
	"github.com/coreofscience/go-bibx/articles"
	"github.com/coreofscience/go-bibx/collection"
)

type Node struct {
	ID        string              `json:"id"`
	Category  algorithms.Category `json:"category"`
	Rootness  float64             `json:"rootness"`
	Trunkness float64             `json:"trunkness"`
	Leafness  float64             `json:"leafness"`
	Article   *articles.Article   `json:"article"`
}

type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Analysis struct {
	Nodes []*Node `json:"nodes"`
	Links []*Link `json:"links"`
}

func New(c *collection.Collection) *Analysis {
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
	for article := range c.Main() {
		articleKey := article.Key()
		if articleKey == nil {
			continue
		}
		for _, ref := range article.References {
			refKey := ref.Key()
			if refKey == nil {
				continue
			}
			links = append(links, &Link{
				Source: *articleKey,
				Target: *refKey,
			})
		}
	}
	return &Analysis{
		Nodes: nodes,
		Links: links,
	}
}
