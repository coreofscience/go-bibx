package models

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
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

type NodeMetadata struct {
	ID         string              `yaml:"id"`
	Label      string              `yaml:"label"`
	Category   string              `yaml:"category"`
	Rootness   float64             `yaml:"rootness"`
	Trunkness  float64             `yaml:"trunkness"`
	Leafness   float64             `yaml:"leafness"`
	Rich       bool                `yaml:"rich"`
	Title      *utils.FoldedString `yaml:"title,omitempty"`
	Year       *int                `yaml:"year,omitempty"`
	Journal    *utils.FoldedString `yaml:"journal,omitempty"`
	Volume     *string             `yaml:"volume,omitempty"`
	Issue      *string             `yaml:"issue,omitempty"`
	Page       *string             `yaml:"page,omitempty"`
	DOI        *string             `yaml:"doi,omitempty"`
	Permalink  *string             `yaml:"permalink,omitempty"`
	TimesCited *int                `yaml:"times_cited,omitempty"`
	Authors    []string            `yaml:"authors,omitempty"`
	IDs        []string            `yaml:"ids,omitempty"`
}

func (n *Node) Metadata() *NodeMetadata {
	if n == nil {
		return nil
	}
	art := n.Article
	if art == nil {
		return &NodeMetadata{
			ID:        n.ID,
			Category:  string(n.Category),
			Rootness:  n.Rootness,
			Trunkness: n.Trunkness,
			Leafness:  n.Leafness,
		}
	}
	var ids []string
	if art.IDs != nil {
		ids = art.IDs.Items()
	}
	return &NodeMetadata{
		ID:         n.ID,
		Label:      art.Label,
		Category:   string(n.Category),
		Rootness:   n.Rootness,
		Trunkness:  n.Trunkness,
		Leafness:   n.Leafness,
		Rich:       art.Rich,
		Title:      utils.NewFoldedString(art.Title),
		Year:       art.Year,
		Journal:    utils.NewFoldedString(art.Journal),
		Volume:     art.Volume,
		Issue:      art.Issue,
		Page:       art.Page,
		DOI:        art.DOI,
		Permalink:  art.Permalink,
		TimesCited: art.TimesCited,
		Authors:    art.Authors,
		IDs:        ids,
	}
}

func (n *Node) Filename() string {
	id := n.ID
	hashBytes := sha256.Sum256([]byte(id))
	hashStr := hex.EncodeToString(hashBytes[:])[:8]

	var title string
	if n.Article != nil && n.Article.Title != nil {
		title = *n.Article.Title
	}
	if title == "" && n.Article != nil {
		title = n.Article.Label
	}
	if title == "" {
		title = "article"
	}

	// Slugify the title
	slug := strings.ToLower(title)

	// Replace non-alphanumeric with spaces
	reg := regexp.MustCompile(`[^a-z0-9\s-_]`)
	slug = reg.ReplaceAllString(slug, "")

	// Replace whitespace/dashes/underscores with a single dash
	regSpace := regexp.MustCompile(`[\s-_]+`)
	slug = regSpace.ReplaceAllString(slug, "-")

	// Trim leading/trailing dashes
	slug = strings.Trim(slug, "-")

	// Cap at 70 characters
	if len(slug) > 70 {
		slug = slug[:70]
		slug = strings.TrimRight(slug, "-")
	}

	if slug == "" {
		slug = "article"
	}

	return fmt.Sprintf("%s-%s.md", slug, hashStr)
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
