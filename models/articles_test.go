package models_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
)

func TestArticles_All(t *testing.T) {
	referenced := &models.Article{
		Label:      "referenced",
		IDs:        collections.NewSet("ref1", "ref2"),
		References: nil,
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		articles models.Articles
		want     models.Articles
	}{
		{
			name:     "nil articles",
			articles: nil,
			want:     nil,
		},
		{
			name:     "empty articles",
			articles: models.Articles{},
			want:     models.Articles{},
		},
		{
			name: "single article",
			articles: models.Articles{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			want: models.Articles{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
		},
		{
			name: "article with references",
			articles: models.Articles{
				{
					Label:      "main",
					IDs:        collections.NewSet("main1"),
					References: models.Articles{referenced},
				},
				referenced,
			},
			want: models.Articles{
				{
					Label:      "main",
					IDs:        collections.NewSet("main1"),
					References: []*models.Article{referenced},
				},
				referenced,
			},
		},
		{
			name: "duplicate articles",
			articles: models.Articles{
				referenced,
				referenced, // duplicate
			},
			want: models.Articles{
				referenced,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.articles.All()
			assert.Equal(t, tt.want, got, "allArticles() = %v, want %v", got, tt.want)
		})
	}
}

func TestArticles_UniqueById(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		articles models.Articles
		want     map[string]*models.Article
	}{
		{
			name:     "nil articles",
			articles: nil,
			want:     nil,
		},
		{
			name:     "empty articles",
			articles: models.Articles{},
			want:     map[string]*models.Article{},
		},
		{
			name: "single article with IDs",
			articles: models.Articles{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			want: map[string]*models.Article{
				"single1": {
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
		},
		{
			name: "multiple articles with unique IDs",
			articles: models.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
				{
					Label:      "article2",
					IDs:        collections.NewSet("id2"),
					References: nil,
				},
			},
			want: map[string]*models.Article{
				"id1": {
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
				"id2": {
					Label:      "article2",
					IDs:        collections.NewSet("id2"),
					References: nil,
				},
			},
		},
		{
			name: "articles with overlapping IDs",
			articles: models.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1", "shared"),
					References: nil,
				},
				{
					Label:      "article2",
					IDs:        collections.NewSet("id2", "shared"),
					References: nil,
				},
			},
			want: map[string]*models.Article{
				"id1": {
					Label:      "article1",
					IDs:        collections.NewSet("id1", "id2", "shared"),
					References: nil,
				},
				"id2": {
					Label:      "article1",
					IDs:        collections.NewSet("id1", "id2", "shared"),
					References: nil,
				},
				"shared": {
					Label:      "article1",
					IDs:        collections.NewSet("id1", "id2", "shared"),
					References: nil,
				},
			},
		},
		{
			name: "articles that are also references",
			articles: models.Articles{
				{
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: []*models.Article{
						{
							Label:      "ref1",
							IDs:        collections.NewSet("ref1"),
							References: nil,
						},
						{
							Label:      "ref2",
							IDs:        collections.NewSet("ref2"),
							References: nil,
						},
					},
				},
				{
					Label:      "ref1",
					IDs:        collections.NewSet("ref1"),
					References: nil,
				},
			},
			want: map[string]*models.Article{
				"main1": {
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: []*models.Article{
						{
							Label:      "ref1",
							IDs:        collections.NewSet("ref1"),
							References: nil,
						},
						{
							Label:      "ref2",
							IDs:        collections.NewSet("ref2"),
							References: nil,
						},
					},
				},
				"ref1": {
					Label:      "ref1",
					IDs:        collections.NewSet("ref1"),
					References: nil,
				},
				"ref2": {
					Label:      "ref2",
					IDs:        collections.NewSet("ref2"),
					References: nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.articles.UniqueById()
			assert.Equal(t, tt.want, got, "uniqueArticlesById() = %v, want %v", got, tt.want)
		})
	}
}

func TestArticles_UniqueById_ArticlesWithSharedIdsShareExactlyTheSameMemoryAddress(t *testing.T) {
	articles := models.Articles{
		{
			Label:      "article1",
			IDs:        collections.NewSet("id1", "shared"),
			References: nil,
		},
		{
			Label:      "article2",
			IDs:        collections.NewSet("id2", "shared"),
			References: nil,
		},
	}
	result := articles.UniqueById()
	assert.Len(t, result, 3, "Expected 3 unique articles")
	assert.Same(t, result["id1"], result["shared"], "Expected articles with shared IDs to share the same memory address")
	assert.Same(t, result["id2"], result["shared"], "Expected articles with shared IDs to share the same memory address")
}

func TestArticles_Deduplicate(t *testing.T) {
	tests := []struct {
		name     string // description of this test case
		articles models.Articles
		want     models.Articles
	}{
		{
			name:     "nil articles",
			articles: nil,
			want:     nil,
		},
		{
			name:     "empty articles",
			articles: models.Articles{},
			want:     models.Articles{},
		},
		{
			name: "single article",
			articles: models.Articles{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			want: models.Articles{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
		},
		{
			name: "duplicate articles",
			articles: models.Articles{
				{
					Label:      "dup",
					IDs:        collections.NewSet("id1", "id2"),
					References: nil,
				},
				{
					Label:      "dup",
					IDs:        collections.NewSet("id2", "id1"),
					References: nil,
				},
			},
			want: models.Articles{
				{
					Label:      "dup",
					IDs:        collections.NewSet("id1", "id2"),
					References: nil,
				},
			},
		},
		{
			name: "articles with overlapping IDs",
			articles: models.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1", "shared"),
					References: nil,
				},
				{
					Label:      "article2",
					IDs:        collections.NewSet("id2", "shared"),
					References: nil,
				},
			},
			want: models.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1", "id2", "shared"),
					References: nil,
				},
			},
		},
		{
			name: "articles that are also references",
			articles: models.Articles{
				{
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: models.Articles{
						{
							Label:      "ref1",
							IDs:        collections.NewSet("ref1"),
							References: nil,
						},
					},
				},
				{
					Label: "ref1",
					IDs:   collections.NewSet("ref1"),
					References: []*models.Article{
						{
							Label:      "ref2",
							IDs:        collections.NewSet("ref2"),
							References: nil,
						},
					},
				},
			},
			want: models.Articles{
				{
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: models.Articles{
						{
							Label: "ref1",
							IDs:   collections.NewSet("ref1"),
							References: []*models.Article{
								{
									Label:      "ref2",
									IDs:        collections.NewSet("ref2"),
									References: nil,
								},
							},
						},
					},
				},
				{
					Label: "ref1",
					IDs:   collections.NewSet("ref1"),
					References: []*models.Article{
						{
							Label:      "ref2",
							IDs:        collections.NewSet("ref2"),
							References: nil,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.articles.Deduplicate()
			assert.Equal(t, tt.want, got, "Deduplicate() = %v, want %v", got, tt.want)
		})
	}
}

func TestArticles_Deduplicate_ReferencesShareExactlyTheSameMemoryAddress(t *testing.T) {
	articles := models.Articles{
		{
			Label: "main",
			IDs:   collections.NewSet("main1"),
			References: models.Articles{
				{
					Label:      "ref1",
					IDs:        collections.NewSet("ref1"),
					References: nil,
				},
			},
		},
		{
			Label: "ref1",
			IDs:   collections.NewSet("ref1"),
			References: []*models.Article{
				{
					Label:      "ref2",
					IDs:        collections.NewSet("ref2"),
					References: nil,
				},
			},
		},
	}
	result := articles.Deduplicate()
	assert.Len(t, result, 2, "Expected 2 unique articles")
	assert.Same(t, result[0].References[0], result[1], "Expected references to share the same memory address")
}
