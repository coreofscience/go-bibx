package models_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollection_Len(t *testing.T) {
	tests := []struct {
		name     string
		articles models.Articles
		want     int
	}{
		{
			name:     "nil collection",
			articles: nil,
			want:     0,
		},
		{
			name:     "empty collection",
			articles: models.Articles{},
			want:     0,
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
			want: 1,
		},
		{
			name: "multiple articles",
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
			want: 2,
		},
		{
			name: "duplicate articles",
			articles: models.Articles{
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
			c, err := models.NewCollection(tt.articles)
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
		articles      models.Articles
		otherArticles models.Articles
		wantArticles  models.Articles
		otherNil      bool
	}{
		{
			name:          "merge with nil collection",
			articles:      models.Articles{},
			otherArticles: nil,
			wantArticles:  models.Articles{},
			otherNil:      true,
		},
		{
			name:     "merge nil collection with another",
			articles: nil,
			otherArticles: models.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
			wantArticles: models.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
		},
		{
			name: "merge two non-empty collections",
			articles: models.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
			otherArticles: models.Articles{
				{
					Label:      "article2",
					IDs:        collections.NewSet("id2"),
					References: nil,
				},
			},
			wantArticles: models.Articles{
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
			articles: models.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
			otherArticles: models.Articles{
				{
					Label: "article1",
					IDs:   collections.NewSet("id1"),
					References: []*models.Article{
						{
							Label: "ref1",
							IDs:   collections.NewSet("refid1"),
						},
					},
				},
			},
			wantArticles: models.Articles{
				{
					Label: "article1",
					IDs:   collections.NewSet("id1"),
					References: []*models.Article{
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
			var c *models.Collection
			var err error
			if tt.articles != nil {
				c, err = models.NewCollection(tt.articles)
				require.NoError(t, err)
			}

			var other *models.Collection
			if !tt.otherNil && tt.otherArticles != nil {
				other, err = models.NewCollection(tt.otherArticles)
				require.NoError(t, err)
			}

			got, err := c.Merge(other)
			require.NoError(t, err)

			var want *models.Collection
			if tt.wantArticles != nil {
				want, err = models.NewCollection(tt.wantArticles)
				require.NoError(t, err)
			}
			assert.Equal(t, want, got)
		})
	}
}
