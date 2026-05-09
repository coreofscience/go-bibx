package analysis

import (
	"github.com/coreofscience/go-bibx/articles"
	"github.com/coreofscience/go-bibx/collection"
)

type Category string

const (
	CategoryRoot  Category = "root"
	CategoryTrunk Category = "trunk"
	CategoryLeaf  Category = "leaf"
)

type Node struct {
	ID       string            `json:"id"`
	Category Category          `json:"category"`
	Root     float64           `json:"root"`
	Trunk    float64           `json:"trunk"`
	Leaf     float64           `json:"leaf"`
	Article  *articles.Article `json:"article"`
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
	for article := range c.All() {
		articleKey := article.Key()
		if articleKey == nil {
			continue
		}
		nodes = append(nodes, &Node{
			ID:      *articleKey,
			Article: article,
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
