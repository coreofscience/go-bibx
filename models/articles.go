package models

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/coreofscience/go-bibx/graphs"
	"github.com/hmdsefi/gograph"
)

type Articles []*Article

// All returns a slice of all articles including their references
func (a *Articles) All() Articles {
	if a == nil || *a == nil {
		return nil
	}
	seen := make(map[*Article]bool)
	result := make(Articles, 0, len(*a))
	for _, article := range *a {
		if article == nil || seen[article] {
			continue
		}
		result = append(result, article)
		seen[article] = true
		for _, reference := range article.References {
			if reference == nil || seen[reference] {
				continue
			}
			result = append(result, reference)
			seen[reference] = true
		}
	}
	return result
}

// UniqueById returns a map of unique articles by their IDs.
func (a *Articles) UniqueById() (map[string]*Article, error) {
	if a == nil || *a == nil {
		return nil, errors.New("articles is nil or empty")
	}
	graph := gograph.New[string]()
	idToArticle := make(map[string][]*Article)
	for _, article := range a.All() {
		if article == nil || article.IDs == nil || article.IDs.Len() == 0 {
			continue
		}
		allIDs := article.IDs.Items()
		firstID := allIDs[0]
		remainingIDs := allIDs[1:]
		firstVertex := gograph.NewVertex(firstID)
		_, _ = graph.AddEdge(firstVertex, firstVertex)
		idToArticle[firstID] = append(idToArticle[firstID], article)
		for _, id := range remainingIDs {
			vertex := gograph.NewVertex(id)
			_, _ = graph.AddEdge(firstVertex, vertex)
			idToArticle[id] = append(idToArticle[id], article)
		}
	}
	unique := make(map[string]*Article, 0)
	scss, err := graphs.WeaklyConnectedComponents(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate weakly connected components: %w", err)
	}
	if len(scss) == 0 {
		return unique, nil
	}
	smallest, biggest := len(*a), 0
	for _, sc := range scss {
		if len(sc) < smallest {
			smallest = len(sc)
		}
		if len(sc) > biggest {
			biggest = len(sc)
		}
	}
	slog.Debug("found strongly connected components", "count", len(scss), "smallest", smallest, "biggest", biggest)
	for _, sc := range scss {
		if len(sc) == 0 {
			continue
		}
		visited := make(map[*Article]bool)
		connectedArticles := make([]*Article, 0, len(sc))
		ids := make([]string, 0, len(sc))
		for _, vertex := range sc {
			id := vertex.Label()
			ids = append(ids, id)
			articles, exists := idToArticle[id]
			if !exists {
				slog.Warn("no articles found for ID", "id", id)
				continue
			}
			for _, article := range articles {
				if article == nil || visited[article] {
					continue
				}
				visited[article] = true
				connectedArticles = append(connectedArticles, article)
			}
		}
		if len(connectedArticles) == 0 {
			continue
		}
		if len(connectedArticles) == 1 {
			for _, id := range ids {
				if id == "" {
					continue
				}
				if _, exists := unique[id]; !exists {
					unique[id] = connectedArticles[0]
				}
			}
			continue
		}
		firstArticle := connectedArticles[0]
		remainingArticles := connectedArticles[1:]
		for _, article := range remainingArticles {
			firstArticle = firstArticle.Merge(article)
		}
		for _, id := range ids {
			if id == "" {
				continue
			}
			if _, exists := unique[id]; !exists {
				unique[id] = firstArticle
			}
		}
	}
	return unique, nil
}

// Deduplicate returns a slice of unique articles, preserving the order of the first occurrence of each article.
func (a *Articles) Deduplicate() (Articles, error) {
	if a == nil || *a == nil {
		return nil, errors.New("articles are empty")
	}
	uniqueMap, err := a.UniqueById()
	if err != nil {
		return nil, fmt.Errorf("failed to get unique articles: %w", err)
	}
	// Create the list of unique articles
	uniqueArticles := make(Articles, 0, len(*a))
	seen := make(map[*Article]bool)
	for _, article := range *a {
		if article == nil || article.IDs.Len() == 0 {
			continue
		}
		firstID := article.IDs.Items()[0]
		uniqueArticle, exists := uniqueMap[firstID]
		if !exists || uniqueArticle == nil || seen[uniqueArticle] {
			continue
		}
		uniqueArticles = append(uniqueArticles, uniqueArticle)
		seen[uniqueArticle] = true
	}

	// Replace the references in each article with their deduplicated versions
	for _, article := range uniqueArticles {
		if article == nil || article.References == nil || len(article.References) == 0 {
			continue
		}
		dedupedReferences := make([]*Article, 0, len(article.References))
		seenRefs := make(map[*Article]bool)
		for _, ref := range article.References {
			if ref == nil || ref.IDs.Len() == 0 {
				continue
			}
			firstRefID := ref.IDs.Items()[0]
			dedupedRef, exists := uniqueMap[firstRefID]
			if !exists || dedupedRef == nil || seenRefs[dedupedRef] {
				continue
			}
			dedupedReferences = append(dedupedReferences, dedupedRef)
			seenRefs[dedupedRef] = true
		}
		article.References = dedupedReferences
	}

	return uniqueArticles, nil
}
