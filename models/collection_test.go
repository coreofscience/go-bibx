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
