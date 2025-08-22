package models_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
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
			c := models.NewCollection(tt.articles)
			got := c.Len()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCollection_Merge(t *testing.T) {
	tests := []struct {
		name     string
		articles models.Articles
		other    *models.Collection
		want     *models.Collection
	}{
		{
			name:     "merge with nil collection",
			articles: models.Articles{},
			other:    nil,
			want:     models.NewCollection(models.Articles{}),
		},
		{
			name:     "merge nil collection with another",
			articles: nil,
			other: models.NewCollection(models.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			}),
			want: models.NewCollection(models.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			}),
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
			other: models.NewCollection(models.Articles{
				{
					Label:      "article2",
					IDs:        collections.NewSet("id2"),
					References: nil,
				},
			}),
			want: models.NewCollection(models.Articles{
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
			}),
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
			other: models.NewCollection(models.Articles{
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
			}),
			want: models.NewCollection(models.Articles{
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
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := models.NewCollection(tt.articles)
			got := c.Merge(tt.other)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCollection_CitationPairs(t *testing.T) {
	tests := []struct {
		name     string
		articles models.Articles
		want     [][2]*models.Article
	}{
		{
			name:     "nil collection",
			articles: nil,
			want:     nil,
		},
		{
			name:     "empty collection",
			articles: models.Articles{},
			want:     nil,
		},
		{
			name: "single article with no references",
			articles: models.Articles{
				{
					Label:      "article1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
			want: nil,
		},
		{
			name: "single article with references",
			articles: models.Articles{
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
			want: [][2]*models.Article{
				{
					&models.Article{
						Label: "article1",
						IDs:   collections.NewSet("id1"),
						References: []*models.Article{
							{
								Label: "ref1",
								IDs:   collections.NewSet("refid1"),
							},
						},
					},
					&models.Article{
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
			c := models.NewCollection(tt.articles)
			got := c.CitationPairs()
			assert.Equal(t, tt.want, got)
		})
	}
}
