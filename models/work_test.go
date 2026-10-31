package models_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
)

func TestWork_Merge(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		work  *models.Work // the receiver type
		other *models.Work
		want  *models.Work
	}{
		{
			name:  "Merge keeps the longest label",
			work:  &models.Work{Label: "A1"},
			other: &models.Work{Label: "AAAAA2"},
			want:  &models.Work{Label: "AAAAA2"},
		},
		{
			name:  "Merge keeps the lowest label",
			work:  &models.Work{Label: "A1"},
			other: &models.Work{Label: "A2"},
			want:  &models.Work{Label: "A1"},
		},
		{
			name:  "Merge keeps the lowest label",
			work:  &models.Work{Label: "A2"},
			other: &models.Work{Label: "A1"},
			want:  &models.Work{Label: "A1"},
		},
		{
			name:  "Merge keeps the longest label when longest is in the receiver",
			work:  &models.Work{Label: "AAAAA2"},
			other: &models.Work{Label: "A1"},
			want:  &models.Work{Label: "AAAAA2"},
		},
		{
			name:  "Merge makes an union of IDs",
			work:  &models.Work{IDs: collections.NewSet("id1", "id2")},
			other: &models.Work{IDs: collections.NewSet("id2", "id3")},
			want:  &models.Work{IDs: collections.NewSet("id1", "id2", "id3")},
		},
		{
			name:  "Merge keeps the longest list of authors",
			work:  &models.Work{Authors: []string{"Alice", "Bob"}},
			other: &models.Work{Authors: []string{"Alice"}},
			want:  &models.Work{Authors: []string{"Alice", "Bob"}},
		},
		{
			name:  "Merge keeps the longest list of authors when longest is in the other",
			work:  &models.Work{Authors: []string{"Alice"}},
			other: &models.Work{Authors: []string{"Alice", "Bob"}},
			want:  &models.Work{Authors: []string{"Alice", "Bob"}},
		},
		{
			name:  "Merge keeps the year if present",
			work:  &models.Work{},
			other: &models.Work{Year: new(2021)},
			want:  &models.Work{Year: new(2021)},
		},
		{
			name:  "Merge keeps the title if present",
			work:  &models.Work{},
			other: &models.Work{Title: new("Title B")},
			want:  &models.Work{Title: new("Title B")},
		},
		{
			name:  "Merge keeps the venue if present",
			work:  &models.Work{},
			other: &models.Work{Venue: new("Journal B")},
			want:  &models.Work{Venue: new("Journal B")},
		},
		{
			name:  "Merge keeps the volume if present",
			work:  &models.Work{},
			other: &models.Work{Volume: new("Volume B")},
			want:  &models.Work{Volume: new("Volume B")},
		},
		{
			name:  "Merge keeps the issue if present",
			work:  &models.Work{},
			other: &models.Work{Issue: new("Issue B")},
			want:  &models.Work{Issue: new("Issue B")},
		},
		{
			name:  "Merge keeps the page if present",
			work:  &models.Work{},
			other: &models.Work{Page: new("Page B")},
			want:  &models.Work{Page: new("Page B")},
		},
		{
			name:  "Merge keeps the DOI if present",
			work:  &models.Work{},
			other: &models.Work{DOI: new("DOI B")},
			want:  &models.Work{DOI: new("DOI B")},
		},
		{
			name:  "Merge keeps the permalink if present",
			work:  &models.Work{},
			other: &models.Work{Permalink: new("Permalink B")},
			want:  &models.Work{Permalink: new("Permalink B")},
		},
		{
			name:  "Merge merges keywords",
			work:  &models.Work{Keywords: collections.NewSet("keyword1", "keyword2")},
			other: &models.Work{Keywords: collections.NewSet("keyword1")},
			want:  &models.Work{Keywords: collections.NewSet("keyword1", "keyword2")},
		},
		{
			name:  "Merge merges all keywords",
			work:  &models.Work{Keywords: collections.NewSet("keyword1", "keyword2")},
			other: &models.Work{Keywords: collections.NewSet("keyword3", "keyword4")},
			want:  &models.Work{Keywords: collections.NewSet("keyword1", "keyword2", "keyword3", "keyword4")},
		},
		{
			name:  "Merge keeps the abstract if present in receiver",
			work:  &models.Work{Abstract: new("Abstract A")},
			other: &models.Work{},
			want:  &models.Work{Abstract: new("Abstract A")},
		},
		{
			name:  "Merge keeps the abstract if present in other",
			work:  &models.Work{},
			other: &models.Work{Abstract: new("Abstract B")},
			want:  &models.Work{Abstract: new("Abstract B")},
		},
		{
			name:  "Keeps the longest list of references",
			work:  &models.Work{References: []*models.Work{{Label: "Ref1"}, {Label: "Ref2"}}},
			other: &models.Work{References: []*models.Work{{Label: "Ref1"}}},
			want:  &models.Work{References: []*models.Work{{Label: "Ref1"}, {Label: "Ref2"}}},
		},
		{
			name:  "Merge prefers rich work fields over unenriched stub",
			work:  &models.Work{Permalink: new("https://openalex.org/W1"), Rich: false},
			other: &models.Work{Permalink: new("https://publisher.example.com/1"), Rich: true},
			want:  &models.Work{Permalink: new("https://publisher.example.com/1"), Rich: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.work.Merge(tt.other)
			assert.Equal(t, tt.want, got, "Merge() result mismatch")
		})
	}
}

func TestWork_Key(t *testing.T) {
	tests := []struct {
		name string       // description of this test case
		work *models.Work // the receiver type
		want *string
	}{
		{
			name: "Key returns nil when IDs is nil",
			work: &models.Work{},
			want: nil,
		},
		{
			name: "Key returns nil when no IDs are present",
			work: &models.Work{IDs: collections.NewSet[string]()},
			want: nil,
		},
		{
			name: "Key returns the first ID when IDs are present",
			work: &models.Work{IDs: collections.NewSet("id1", "id2")},
			want: new("id1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.work.Key()
			assert.Equal(t, tt.want, got, "Key() result mismatch")
		})
	}
}

func TestWork_SimpleLabel(t *testing.T) {
	tests := []struct {
		name string       // description of this test case
		work *models.Work // the receiver type
		want *string
	}{
		{
			name: "SimpleLabel returns nil when work is nil",
			work: nil,
			want: nil,
		},
		{
			name: "SimpleLabel returns nil when no parts are available",
			work: &models.Work{Label: ""},
			want: nil,
		},
		{
			name: "SimpleLabel returns the first author when available",
			work: &models.Work{Authors: []string{"Alice"}},
			want: new("Alice"),
		},
		{
			name: "SimpleLabel returns the first author and year when available",
			work: &models.Work{Authors: []string{"Alice"}, Year: new(2021)},
			want: new("Alice, 2021"),
		},
		{
			name: "SimpleLabel returns the first author and year with multiple authors",
			work: &models.Work{Authors: []string{"Alice", "Bob"}, Year: new(2021)},
			want: new("Alice, 2021"),
		},
		{
			name: "SimpleLabel returns the first author, year and venue when available",
			work: &models.Work{
				Authors: []string{"Alice"},
				Year:    new(2021),
				Venue:   new("Journal A"),
			},
			want: new("Alice, 2021, Journal A"),
		},
		{
			name: "SimpleLabel returns the first author, year, venue and volume when available",
			work: &models.Work{
				Authors: []string{"Alice"},
				Year:    new(2021),
				Venue:   new("Journal A"),
				Volume:  new("1"),
			},
			want: new("Alice, 2021, Journal A, V1"),
		},
		{
			name: "SimpleLabel returns the first author, year, venue, volume and page when available",
			work: &models.Work{
				Authors: []string{"Alice"},
				Year:    new(2021),
				Venue:   new("Journal A"),
				Volume:  new("1"),
				Page:    new("10"),
			},
			want: new("Alice, 2021, Journal A, V1, P10"),
		},
		{
			name: "SimpleLabel returns the first author, year, venue, volume, page and DOI when available",
			work: &models.Work{
				Authors: []string{"Alice"},
				Year:    new(2021),
				Venue:   new("Journal A"),
				Volume:  new("1"),
				Page:    new("10"),
				DOI:     new("10.1000/xyz123"),
			},
			want: new("Alice, 2021, Journal A, V1, P10, DOI 10.1000/xyz123"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.work.SimpleLabel()
			assert.Equal(t, tt.want, got, "SimpleLabel() result mismatch")
		})
	}
}

func TestWork_SimpleId(t *testing.T) {
	tests := []struct {
		name string       // description of this test case
		work *models.Work // the receiver type
		want *string
	}{
		{
			name: "SimpleId returns nil when work is nil",
			work: nil,
			want: nil,
		},
		{
			name: "SimpleId returns nil when no authors are present",
			work: &models.Work{Authors: []string{}, Year: new(2021)},
			want: nil,
		},
		{
			name: "SimpleId returns nil when year is nil",
			work: &models.Work{Authors: []string{"Alice"}, Year: nil},
			want: nil,
		},
		{
			name: "SimpleId returns first author's name and year",
			work: &models.Work{Authors: []string{"Alice Smith"}, Year: new(2021)},
			want: new("alice2021"),
		},
		{
			name: "SimpleId returns first author's name and year with multiple authors",
			work: &models.Work{Authors: []string{"Alice Smith", "Bob Johnson"}, Year: new(2021)},
			want: new("alice2021"),
		},
		{
			name: "SimpleId returns first author's name and year with comma in name",
			work: &models.Work{Authors: []string{"Smith, Alice"}, Year: new(2021)},
			want: new("smith2021"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.work.SimpleId()
			assert.Equal(t, tt.want, got, "SimpleId() result mismatch")
		})
	}
}

func TestWork_GetPermalink(t *testing.T) {
	tests := []struct {
		name string       // description of this test case
		work *models.Work // the receiver type
		want *string
	}{
		{
			name: "GetPermalink returns nil when work is nil",
			work: nil,
			want: nil,
		},
		{
			name: "GetPermalink returns the permalink when permalink is set regardless of DOI",
			work: &models.Work{
				Permalink: new("https://example.com/work"),
				DOI:       new("10.1000/xyz123"),
			},
			want: new("https://example.com/work"),
		},
		{
			name: "GetPermalink returns the DOI as permalink when permalink is not set",
			work: &models.Work{DOI: new("10.1000/xyz123")},
			want: new("https://doi.org/10.1000/xyz123"),
		},
		{
			name: "GetPermalink returns nil when neither permalink nor DOI is set",
			work: &models.Work{Label: "work"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.work.GetPermalink()
			assert.Equal(t, tt.want, got, "GetPermalink() result mismatch")
		})
	}
}

func TestWork_AddSimpleId(t *testing.T) {
	tests := []struct {
		name string       // description of this test case
		work *models.Work // the receiver type
		want *models.Work
	}{
		{
			name: "AddSimpleId adds simple ID when SimpleId is available",
			work: &models.Work{
				Authors: []string{"Alice Smith"},
				Year:    new(2021),
				IDs:     collections.NewSet[string](),
			},
			want: &models.Work{
				Authors: []string{"Alice Smith"},
				Year:    new(2021),
				IDs:     collections.NewSet("simple:alice2021"),
			},
		},
		{
			name: "AddSimpleId does not add simple ID when SimpleId is nil",
			work: &models.Work{
				Authors: []string{"Alice Smith"},
				Year:    nil,
				IDs:     collections.NewSet[string](),
			},
			want: &models.Work{
				Authors: []string{"Alice Smith"},
				Year:    nil,
				IDs:     collections.NewSet[string](),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.
			got := tt.work.AddSimpleId()
			assert.Equal(t, tt.want, got, "AddSimpleId() result mismatch")
		})
	}
}

func TestWork_SetSimpleLabel(t *testing.T) {
	tests := []struct {
		name string       // description of this test case
		work *models.Work // the receiver type
		want *models.Work
	}{
		{
			name: "SetSimpleLabel sets simple label when SimpleLabel is available",
			work: &models.Work{
				Authors: []string{"Alice Smith"},
				Year:    new(2021),
			},
			want: &models.Work{
				Authors: []string{"Alice Smith"},
				Year:    new(2021),
				Label:   "Alice Smith, 2021",
			},
		},
		{
			name: "SetSimpleLabel does not change label when SimpleLabel is nil",
			work: &models.Work{
				Authors: []string{},
				Label:   "Existing Label",
			},
			want: &models.Work{
				Authors: []string{},
				Label:   "Existing Label",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.work.SetSimpleLabel()
			assert.Equal(t, tt.want, got, "SetSimpleLabel() result mismatch")
		})
	}
}
