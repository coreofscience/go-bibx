package models_test

import (
	"slices"
	"testing"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorks_All(t *testing.T) {
	referenced := &models.Work{
		Label:      "referenced",
		IDs:        collections.NewSet("ref1", "ref2"),
		References: nil,
	}
	tests := []struct {
		name  string
		works models.Works
		want  []*models.Work
	}{
		{
			name:  "nil works",
			works: nil,
			want:  nil,
		},
		{
			name:  "empty works",
			works: models.Works{},
			want:  nil,
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
			want: models.Works{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
		},
		{
			name: "work with references",
			works: models.Works{
				{
					Label:      "main",
					IDs:        collections.NewSet("main1"),
					References: models.References{referenced},
				},
				referenced,
			},
			want: models.Works{
				{
					Label:      "main",
					IDs:        collections.NewSet("main1"),
					References: models.References{referenced},
				},
				referenced,
			},
		},
		{
			name: "duplicate works",
			works: models.Works{
				referenced,
				referenced, // duplicate
			},
			want: []*models.Work{
				referenced,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slices.Collect(tt.works.All())
			assert.Equal(t, tt.want, got, "allWorks() = %v, want %v", got, tt.want)
		})
	}
}

func TestWorks_UniqueById(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		works   models.Works
		want    map[string]*models.Work
		wantErr bool
	}{
		{
			name:    "nil works",
			works:   nil,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty works",
			works:   models.Works{},
			want:    map[string]*models.Work{},
			wantErr: false,
		},
		{
			name: "single work with IDs",
			works: models.Works{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			want: map[string]*models.Work{
				"single1": {
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "multiple works with unique IDs",
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
			want: map[string]*models.Work{
				"id1": {
					Label:      "work1",
					IDs:        collections.NewSet("id1"),
					References: nil,
				},
				"id2": {
					Label:      "work2",
					IDs:        collections.NewSet("id2"),
					References: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "works with overlapping IDs",
			works: models.Works{
				{
					Label:      "work1",
					IDs:        collections.NewSet("id1", "shared"),
					References: nil,
				},
				{
					Label:      "work2",
					IDs:        collections.NewSet("id2", "shared"),
					References: nil,
				},
			},
			want: map[string]*models.Work{
				"id1": {
					Label:      "work1",
					IDs:        collections.NewSet("id1", "id2", "shared"),
					References: nil,
				},
				"id2": {
					Label:      "work1",
					IDs:        collections.NewSet("id1", "id2", "shared"),
					References: nil,
				},
				"shared": {
					Label:      "work1",
					IDs:        collections.NewSet("id1", "id2", "shared"),
					References: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "works that are also references",
			works: models.Works{
				{
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: []*models.Work{
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
			want: map[string]*models.Work{
				"main1": {
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: []*models.Work{
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
			got, err := tt.works.UniqueById()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "uniqueWorksById() = %v, want %v", got, tt.want)
		})
	}
}

func TestWorks_UniqueById_WorksWithSharedIdsShareExactlyTheSameMemoryAddress(t *testing.T) {
	arts := models.Works{
		{
			Label:      "work1",
			IDs:        collections.NewSet("id1", "shared"),
			References: nil,
		},
		{
			Label:      "work2",
			IDs:        collections.NewSet("id2", "shared"),
			References: nil,
		},
	}
	result, err := arts.UniqueById()
	require.NoError(t, err)
	assert.Len(t, result, 3, "Expected 3 unique works")
	assert.Same(t, result["id1"], result["shared"], "Expected works with shared IDs to share the same memory address")
	assert.Same(t, result["id2"], result["shared"], "Expected works with shared IDs to share the same memory address")
}

func TestWorks_Deduplicate(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		works   models.Works
		want    models.Works
		wantErr bool
	}{
		{
			name:    "nil works",
			works:   nil,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty works",
			works:   models.Works{},
			want:    models.Works{},
			wantErr: false,
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
			want: models.Works{
				{
					Label:      "single",
					IDs:        collections.NewSet("single1"),
					References: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate works",
			works: models.Works{
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
			want: models.Works{
				{
					Label:      "dup",
					IDs:        collections.NewSet("id1", "id2"),
					References: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "works with overlapping IDs",
			works: models.Works{
				{
					Label:      "work1",
					IDs:        collections.NewSet("id1", "shared"),
					References: nil,
				},
				{
					Label:      "work2",
					IDs:        collections.NewSet("id2", "shared"),
					References: nil,
				},
			},
			want: models.Works{
				{
					Label:      "work1",
					IDs:        collections.NewSet("id1", "id2", "shared"),
					References: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "works that are also references",
			works: models.Works{
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
					References: []*models.Work{
						{
							Label:      "ref2",
							IDs:        collections.NewSet("ref2"),
							References: nil,
						},
					},
				},
			},
			want: models.Works{
				{
					Label: "main",
					IDs:   collections.NewSet("main1"),
					References: models.References{
						{
							Label: "ref1",
							IDs:   collections.NewSet("ref1"),
							References: []*models.Work{
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
					References: []*models.Work{
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
			got, err := tt.works.Deduplicate()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "Deduplicate() = %v, want %v", got, tt.want)
		})
	}
}

func TestWorks_Deduplicate_ReferencesShareExactlyTheSameMemoryAddress(t *testing.T) {
	arts := models.Works{
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
			References: []*models.Work{
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
	assert.Len(t, result, 2, "Expected 2 unique works")
	assert.Same(t, result[0].References[0], result[1], "Expected references to share the same memory address")
}
