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
	"github.com/coreofscience/go-bibx/internal/search"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/hmdsefi/gograph"
)

// Category represents the Tree of Science classification category of an analysis node.
type Category string

const (
	// CategoryRoot represents foundational, seminal articles.
	CategoryRoot Category = "root"
	// CategoryTrunk represents core structural articles connecting the field.
	CategoryTrunk Category = "trunk"
	// CategoryLeaf represents recent and specialized articles at the frontier.
	CategoryLeaf Category = "leaf"
)

var (
	nonAlphanumericReg = regexp.MustCompile(`[^a-z0-9\s_-]`)
	spaceReg           = regexp.MustCompile(`[\s_-]+`)
)

// Node represents an article entity within an analysis, containing its metadata,
// Tree of Science category and scores, and community assignment.
type Node struct {
	ID        string   `json:"id"`
	Category  Category `json:"category"`
	Rootness  float64  `json:"rootness"`
	Trunkness float64  `json:"trunkness"`
	Leafness  float64  `json:"leafness"`
	Community int      `json:"community"`
	Article   *Article `json:"article"`
}

// NodeMetadata represents the YAML-serializable metadata of a [Node], commonly
// used for generating frontmatter in markdown exports.
type NodeMetadata struct {
	ID         string              `yaml:"id"`
	Label      string              `yaml:"label"`
	Category   string              `yaml:"category"`
	Rootness   float64             `yaml:"rootness"`
	Trunkness  float64             `yaml:"trunkness"`
	Leafness   float64             `yaml:"leafness"`
	Community  int                 `yaml:"community"`
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

// Metadata extracts and constructs the [NodeMetadata] for this node.
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
			Community: n.Community,
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
		Community:  n.Community,
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

// Filename generates a sanitized, slugified filename with a content hash suffix for exporting the node as markdown.
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
	slug = nonAlphanumericReg.ReplaceAllString(slug, "")

	// Replace whitespace/dashes/underscores with a single dash
	slug = spaceReg.ReplaceAllString(slug, "-")

	// Trim leading/trailing dashes
	slug = strings.Trim(slug, "-")

	// Cap at 60 characters
	if len(slug) > 60 {
		slug = slug[:60]
		slug = strings.TrimRight(slug, "-")
	}

	if slug == "" {
		slug = "article"
	}

	return fmt.Sprintf("%s-%s.md", slug, hashStr)
}

// Result represents a scored search or query result pointing to an [Article].
type Result struct {
	Score   float64  `json:"score"`
	Article *Article `json:"article"`
}

// Link represents a directed, weighted citation edge between two nodes in an [Analysis].
type Link struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Weight float64 `json:"weight"`
}

// Analysis represents the complete bibliographic analysis dataset, containing
// all analyzed nodes and citation links.
type Analysis struct {
	Nodes []*Node `json:"nodes"`
	Links []*Link `json:"links"`
}

// CitationGraph constructs and returns a directed, acyclic, weighted citation graph from the analysis links and nodes.
func (a *Analysis) CitationGraph() (gograph.Graph[string], error) {
	graph := gograph.New[string](
		gograph.Directed(),
		gograph.Acyclic(),
		gograph.Weighted(),
	)
	for _, node := range a.Nodes {
		graph.AddVertexByLabel(node.ID)
	}
	for _, link := range a.Links {
		_, err := graph.AddEdge(
			gograph.NewVertex(link.Source),
			gograph.NewVertex(link.Target),
			gograph.WithEdgeWeight(link.Weight),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to add edge: %w", err)
		}
	}
	return graph, nil
}

// Keep filters the analysis, returning a new [Analysis] containing only nodes with the specified IDs and links between them.
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

// Query returns the top scored results for the given Tree of Science category ("root", "trunk", or "leaf"), up to top items.
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

// Search performs a keyword search using BM25 ranking over the analysis nodes.
func (a *Analysis) Search(query string, limit int) ([]*Node, error) {
	if len(a.Nodes) == 0 || limit <= 0 {
		return nil, nil
	}
	docs := make([]search.Document, len(a.Nodes))
	nodeMap := make(map[string]*Node, len(a.Nodes))
	for i, node := range a.Nodes {
		nodeMap[node.ID] = node
		doc := search.Document{ID: node.ID}
		if node.Article != nil {
			if node.Article.Title != nil {
				doc.Title = *node.Article.Title
			}
			if node.Article.Keywords != nil {
				doc.Keywords = node.Article.Keywords.Items()
			}
			if node.Article.Abstract != nil {
				doc.Abstract = *node.Article.Abstract
			}
			doc.Authors = node.Article.Authors
		}
		docs[i] = doc
	}

	searchResults := search.BM25Search(docs, query, limit)
	if len(searchResults) == 0 {
		return nil, nil
	}

	topResults := make([]*Node, 0, len(searchResults))
	for _, result := range searchResults {
		if node, ok := nodeMap[result.ID]; ok {
			topResults = append(topResults, node)
		}
	}
	return topResults, nil
}
