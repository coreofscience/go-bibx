package models

import (
	"testing"

	"github.com/coreofscience/go-bibx/collections"
	"github.com/stretchr/testify/assert"
)

func Test_allArticles(t *testing.T) {
	referenced := &Article{
		Label:      "referenced",
		IDs:        collections.NewSet("ref1", "ref2"),
		References: nil,
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		articles []*Article
		want     []*Article
	}{
		{
			name:     "nil articles",
			articles: nil,
			want:     nil,
		},
		{
			name:     "empty articles",
			articles: []*Article{},
			want:     []*Article{},
		},
		{
			name: "single article",
			articles: []*Article{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			want: []*Article{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
		},
		{
			name: "article with references",
			articles: []*Article{
				{
					Label:      "main",
					IDs:        collections.NewSet("main1"),
					References: []*Article{referenced},
				},
				referenced,
			},
			want: []*Article{
				{
					Label:      "main",
					IDs:        collections.NewSet("main1"),
					References: []*Article{referenced},
				},
				referenced,
			},
		},
		{
			name: "duplicate articles",
			articles: []*Article{
				referenced,
				referenced, // duplicate
			},
			want: []*Article{
				referenced,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allArticles(tt.articles)
			assert.Equal(t, tt.want, got, "allArticles() = %v, want %v", got, tt.want)
		})
	}
}

func Test_uniqueArticlesById(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		articles []*Article
		want     map[string]*Article
	}{
		{
			name:     "nil articles",
			articles: nil,
			want:     map[string]*Article{},
		},
		{
			name:     "empty articles",
			articles: []*Article{},
			want:     map[string]*Article{},
		},
		{
			name: "single article with IDs",
			articles: []*Article{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			want: map[string]*Article{
				"single1": {
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
		},
		{
			name: "multiple articles with unique IDs",
			articles: []*Article{
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
			want: map[string]*Article{
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
			articles: []*Article{
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
			want: map[string]*Article{
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
			articles: []*Article{
				{
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: []*Article{
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
			want: map[string]*Article{
				"main1": {
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: []*Article{
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
			got := uniqueArticlesById(tt.articles)
			assert.Equal(t, tt.want, got, "uniqueArticlesById() = %v, want %v", got, tt.want)
		})
	}
}

func Test_uniqueArticlesById_ArticlesWithSharedIdsShareExactlyTheSameMemoryAddress(t *testing.T) {
	articles := []*Article{
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
	result := uniqueArticlesById(articles)
	assert.Len(t, result, 3, "Expected 3 unique articles")
	assert.Same(t, result["id1"], result["shared"], "Expected articles with shared IDs to share the same memory address")
	assert.Same(t, result["id2"], result["shared"], "Expected articles with shared IDs to share the same memory address")
}
