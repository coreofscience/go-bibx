package models_test

import (
	"slices"
	"testing"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArticles_All(t *testing.T) {
	referenced := &models.Article{
		Label:      "referenced",
		IDs:        collections.NewSet("ref1", "ref2"),
		References: nil,
	}
	tests := []struct {
		name     string
		articles models.Articles
		want     []*models.Article
	}{
		{
			name:     "nil articles",
			articles: nil,
			want:     nil,
		},
		{
			name:     "empty articles",
			articles: models.Articles{},
			want:     nil,
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
					References: models.References{referenced},
				},
				referenced,
			},
			want: models.Articles{
				{
					Label:      "main",
					IDs:        collections.NewSet("main1"),
					References: models.References{referenced},
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
			want: []*models.Article{
				referenced,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slices.Collect(tt.articles.All())
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
		wantErr  bool
	}{
		{
			name:     "nil articles",
			articles: nil,
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "empty articles",
			articles: models.Articles{},
			want:     map[string]*models.Article{},
			wantErr:  false,
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
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.articles.UniqueById()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "uniqueArticlesById() = %v, want %v", got, tt.want)
		})
	}
}

func TestArticles_UniqueById_ArticlesWithSharedIdsShareExactlyTheSameMemoryAddress(t *testing.T) {
	arts := models.Articles{
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
	result, err := arts.UniqueById()
	require.NoError(t, err)
	assert.Len(t, result, 3, "Expected 3 unique articles")
	assert.Same(t, result["id1"], result["shared"], "Expected articles with shared IDs to share the same memory address")
	assert.Same(t, result["id2"], result["shared"], "Expected articles with shared IDs to share the same memory address")
}

func TestArticles_Deduplicate(t *testing.T) {
	tests := []struct {
		name     string // description of this test case
		articles models.Articles
		want     models.Articles
		wantErr  bool
	}{
		{
			name:     "nil articles",
			articles: nil,
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "empty articles",
			articles: models.Articles{},
			want:     models.Articles{},
			wantErr:  false,
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
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
		},
		{
			name: "articles that are also references",
			articles: models.Articles{
				{
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: models.References{
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
					References: models.References{
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
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.articles.Deduplicate()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "Deduplicate() = %v, want %v", got, tt.want)
		})
	}
}

func TestArticles_Deduplicate_ReferencesShareExactlyTheSameMemoryAddress(t *testing.T) {
	arts := models.Articles{
		{
			Label: "main",
			IDs:   collections.NewSet("main1"),
			References: models.References{
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
	result, err := arts.Deduplicate()
	require.NoError(t, err)
	assert.Len(t, result, 2, "Expected 2 unique articles")
	assert.Same(t, result[0].References[0], result[1], "Expected references to share the same memory address")
}
