package models

import (
	"cmp"
	"slices"
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
