package models

import (
	"log/slog"

	"github.com/hmdsefi/gograph"
	"github.com/hmdsefi/gograph/connectivity"
)

type Collection struct {
	Articles []*Article
}

// allArticles returns a slice of all articles including their references
func allArticles(articles []*Article) []*Article {
	if articles == nil {
		return nil
	}
	seen := make(map[*Article]bool)
	result := make([]*Article, 0, len(articles))
	for _, article := range articles {
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

// uniqueArticlesById returns a map of unique articles by their IDs.
func uniqueArticlesById(articles []*Article) map[string]*Article {
	graph := gograph.New[string]()
	idToArticle := make(map[string][]*Article)
	for _, article := range allArticles(articles) {
		if article == nil || article.IDs == nil || article.IDs.Len() == 0 {
			continue
		}
		allIDs := article.IDs.Items()
		firstID := allIDs[0]
		remainingIDs := allIDs[1:]
		if firstID == nil {
			continue
		}
		firstVertex := gograph.NewVertex(*firstID)
		_, _ = graph.AddEdge(firstVertex, firstVertex)
		idToArticle[*firstID] = append(idToArticle[*firstID], article)
		for _, id := range remainingIDs {
			if id == nil {
				continue
			}
			vertex := gograph.NewVertex(*id)
			_, _ = graph.AddEdge(firstVertex, vertex)
			idToArticle[*id] = append(idToArticle[*id], article)
		}
	}
	unique := make(map[string]*Article, 0)
	scss := connectivity.Tarjan(graph)
	if len(scss) == 0 {
		return unique
	}
	smallest, biggest := len(articles), 0
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
	return unique
}
