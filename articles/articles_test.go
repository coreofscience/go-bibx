package articles_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/articles"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArticles_All(t *testing.T) {
	referenced := &articles.Article{
		Label:      "referenced",
		IDs:        collections.NewSet("ref1", "ref2"),
		References: nil,
	}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		articles articles.Articles
		want     articles.Articles
	}{
		{
			name:     "nil articles",
			articles: nil,
			want:     nil,
		},
		{
			name:     "empty articles",
			articles: articles.Articles{},
			want:     articles.Articles{},
		},
		{
			name: "single article",
			articles: articles.Articles{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			want: articles.Articles{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
		},
		{
			name: "article with references",
			articles: articles.Articles{
				{
					Label:      "main",
					IDs:        collections.NewSet("main1"),
					References: articles.References{referenced},
				},
				referenced,
			},
			want: articles.Articles{
				{
					Label:      "main",
					IDs:        collections.NewSet("main1"),
					References: []*articles.Article{referenced},
				},
				referenced,
			},
		},
		{
			name: "duplicate articles",
			articles: articles.Articles{
				referenced,
				referenced, // duplicate
			},
			want: articles.Articles{
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
		articles articles.Articles
		want     map[string]*articles.Article
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
			articles: articles.Articles{},
			want:     map[string]*articles.Article{},
			wantErr:  false,
		},
		{
			name: "single article with IDs",
			articles: articles.Articles{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			want: map[string]*articles.Article{
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
			articles: articles.Articles{
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
			want: map[string]*articles.Article{
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
			articles: articles.Articles{
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
			want: map[string]*articles.Article{
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
			articles: articles.Articles{
				{
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: []*articles.Article{
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
			want: map[string]*articles.Article{
				"main1": {
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: []*articles.Article{
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
	arts := articles.Articles{
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
		articles articles.Articles
		want     articles.Articles
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
			articles: articles.Articles{},
			want:     articles.Articles{},
			wantErr:  false,
		},
		{
			name: "single article",
			articles: articles.Articles{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			want: articles.Articles{
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
			articles: articles.Articles{
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
			want: articles.Articles{
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
			articles: articles.Articles{
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
			want: articles.Articles{
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
			articles: articles.Articles{
				{
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: articles.References{
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
					References: []*articles.Article{
						{
							Label:      "ref2",
							IDs:        collections.NewSet("ref2"),
							References: nil,
						},
					},
				},
			},
			want: articles.Articles{
				{
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: articles.References{
						{
							Label: "ref1",
							IDs:   collections.NewSet("ref1"),
							References: []*articles.Article{
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
					References: []*articles.Article{
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
	arts := articles.Articles{
		{
			Label: "main",
			IDs:   collections.NewSet("main1"),
			References: articles.References{
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
			References: []*articles.Article{
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
