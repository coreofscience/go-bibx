package collection_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/articles"
	"github.com/coreofscience/go-bibx/collection"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollection_Len(t *testing.T) {
	tests := []struct {
		name     string
		articles articles.Articles
		want     int
	}{
		{
			name:     "nil collection",
			articles: nil,
			want:     0,
		},
		{
			name:     "empty collection",
			articles: articles.Articles{},
			want:     0,
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
			want: 1,
		},
		{
			name: "multiple articles",
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
			want: 2,
		},
		{
			name: "duplicate articles",
			articles: articles.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
			want: 1, // Expect deduplication
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := collection.New(tt.articles)
			if tt.articles == nil {
				require.Error(t, err)
				assert.Equal(t, tt.want, c.Len())
				return
			}
			require.NoError(t, err)
			got := c.Len()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCollection_Merge(t *testing.T) {
	tests := []struct {
		name          string
		articles      articles.Articles
		otherArticles articles.Articles
		wantArticles  articles.Articles
		otherNil      bool
	}{
		{
			name:          "merge with nil collection",
			articles:      articles.Articles{},
			otherArticles: nil,
			wantArticles:  articles.Articles{},
			otherNil:      true,
		},
		{
			name:     "merge nil collection with another",
			articles: nil,
			otherArticles: articles.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
			wantArticles: articles.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
		},
		{
			name: "merge two non-empty collections",
			articles: articles.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
			otherArticles: articles.Articles{
				{
					Label:      "article2",
					IDs:        collections.NewSet("id2"),
					References: nil,
				},
			},
			wantArticles: articles.Articles{
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
		},
		{
			name: "merge collections with duplicate articles",
			articles: articles.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
			otherArticles: articles.Articles{
				{
					Label: "article1",
					IDs:   collections.NewSet("id1"),
					References: []*articles.Article{
						{
							Label: "ref1",
							IDs:   collections.NewSet("refid1"),
						},
					},
				},
			},
			wantArticles: articles.Articles{
				{
					Label: "article1",
					IDs:   collections.NewSet("id1"),
					References: []*articles.Article{
						{
							Label: "ref1",
							IDs:   collections.NewSet("refid1"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c *collection.Collection
			var err error
			if tt.articles != nil {
				c, err = collection.New(tt.articles)
				require.NoError(t, err)
			}

			var other *collection.Collection
			if !tt.otherNil && tt.otherArticles != nil {
				other, err = collection.New(tt.otherArticles)
				require.NoError(t, err)
			}

			got, err := c.Merge(other)
			require.NoError(t, err)

			var want *collection.Collection
			if tt.wantArticles != nil {
				want, err = collection.New(tt.wantArticles)
				require.NoError(t, err)
			}
			assert.Equal(t, want, got)
		})
	}
}

func TestCollection_CitationPairs(t *testing.T) {
	tests := []struct {
		name     string
		articles articles.Articles
		want     [][2]*articles.Article
	}{
		{
			name:     "nil collection",
			articles: nil,
			want:     nil,
		},
		{
			name:     "empty collection",
			articles: articles.Articles{},
			want:     [][2]*articles.Article{},
		},
		{
			name: "single article with no references",
			articles: articles.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
			want: [][2]*articles.Article{},
		},
		{
			name: "single article with references",
			articles: articles.Articles{
				{
					Label: "article1",
					IDs:   collections.NewSet("id1"),
					References: []*articles.Article{
						{
							Label: "ref1",
							IDs:   collections.NewSet("refid1"),
						},
					},
				},
			},
			want: [][2]*articles.Article{
				{
					&articles.Article{
						Label: "article1",
						IDs:   collections.NewSet("id1"),
						References: []*articles.Article{
							{
								Label: "ref1",
								IDs:   collections.NewSet("refid1"),
							},
						},
					},
					&articles.Article{
						Label:      "ref1",
						IDs:        collections.NewSet("refid1"),
						References: nil,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := collection.New(tt.articles)
			if tt.articles == nil {
				require.Error(t, err)
				assert.Nil(t, c)
				return
			}
			require.NoError(t, err)
			got := make([][2]*articles.Article, 0, len(tt.articles))
			for a, b := range c.CitationPairs() {
				got = append(got, [2]*articles.Article{a, b})
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
