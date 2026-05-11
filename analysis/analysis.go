package analysis

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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

// Load loads an analysis from a gzip-compressed JSON file at the given path.
func Load(path string) (*Analysis, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer func() {
		err = file.Close()
		if err != nil {
			slog.Error("error closing file", "error", err)
		}
	}()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("error creating gzip reader: %w", err)
	}
	defer func() {
		err = gz.Close()
		if err != nil {
			slog.Error("error closing gzip writer", "error", err)
		}
	}()
	var analysis Analysis
	if err := json.NewDecoder(gz).Decode(&analysis); err != nil {
		return nil, fmt.Errorf("error decoding file: %w", err)
	}
	return &analysis, nil
}

// Store stores the analysis to a gzip-compressed JSON file at the given path.
func (a *Analysis) Store(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer func() {
		err = file.Close()
		if err != nil {
			slog.Error("error closing file", "error", err)
		}
	}()
	gz := gzip.NewWriter(file)
	defer func() {
		err = gz.Close()
		if err != nil {
			slog.Error("error closing gzip writer", "error", err)
		}
	}()
	if err := json.NewEncoder(gz).Encode(a); err != nil {
		return fmt.Errorf("error encoding analysis: %w", err)
	}
	return nil
}
