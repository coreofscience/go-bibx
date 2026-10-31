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
		name  string
		works models.Works
		want  int
	}{
		{
			name:  "nil collection",
			works: nil,
			want:  0,
		},
		{
			name:  "empty collection",
			works: models.Works{},
			want:  0,
		},
		{
			name: "single work",
			works: models.Works{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			want: 1,
		},
		{
			name: "multiple works",
			works: models.Works{
				{
					Label:      "work1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
				{
					Label:      "work2",
					IDs:        collections.NewSet("id2"),
					References: nil,
				},
			},
			want: 2,
		},
		{
			name: "duplicate works",
			works: models.Works{
				{
					Label:      "work1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
				{
					Label:      "work1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
			},
			want: 1, // Expect deduplication
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := models.NewCollection(tt.works)
			if tt.works == nil {
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

func TestCollection_Keep_NilKey(t *testing.T) {
	art1 := &models.Work{
		Label: "with-id",
		IDs:   collections.NewSet("id1"),
	}
	art2 := &models.Work{
		Label: "without-id",
		IDs:   collections.NewSet("id2"),
	}
	works := models.Works{art1, art2}

	c, err := models.NewCollection(works)
	require.NoError(t, err)
	require.Equal(t, 2, c.Len())

	// Clear the second work's IDs after collection construction so its Key() returns nil
	art2.IDs = collections.NewSet[string]()
	require.Nil(t, art2.Key())

	// Keep should not panic when encountering a work with nil Key()
	kept, err := c.Keep("id1")
	require.NoError(t, err)
	require.NotNil(t, kept)
	assert.Equal(t, 1, kept.Len())
}

func TestCollection_CitationGraph_IncludesWorksWithoutReferences(t *testing.T) {
	work := &models.Work{
		Label: "isolated",
		IDs:   collections.NewSet("openalex:W1"),
	}
	c, err := models.NewCollection(models.Works{work})
	require.NoError(t, err)

	graph, err := c.CitationGraph()
	require.NoError(t, err)
	assert.Equal(t, uint32(1), graph.Order())
	assert.NotNil(t, graph.GetVertexByID("openalex:W1"))
}
