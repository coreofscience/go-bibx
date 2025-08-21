package models_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
)

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func TestArticle_Merge(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		article *models.Article // the receiver type
		other   *models.Article
		want    *models.Article
	}{
		{
			name:    "Merge keeps the longest label",
			article: &models.Article{Label: "A1"},
			other:   &models.Article{Label: "AAAAA2"},
			want:    &models.Article{Label: "AAAAA2"},
		},
		{
			name:    "Merge keeps the lowest label",
			article: &models.Article{Label: "A1"},
			other:   &models.Article{Label: "A2"},
			want:    &models.Article{Label: "A1"},
		},
		{
			name:    "Merge keeps the lowest label",
			article: &models.Article{Label: "A2"},
			other:   &models.Article{Label: "A1"},
			want:    &models.Article{Label: "A1"},
		},
		{
			name:    "Merge keeps the longest label when longest is in the receiver",
			article: &models.Article{Label: "AAAAA2"},
			other:   &models.Article{Label: "A1"},
			want:    &models.Article{Label: "AAAAA2"},
		},
		{
			name:    "Merge makes an union of IDs",
			article: &models.Article{IDs: collections.NewSet("id1", "id2")},
			other:   &models.Article{IDs: collections.NewSet("id2", "id3")},
			want:    &models.Article{IDs: collections.NewSet("id1", "id2", "id3")},
		},
		{
			name:    "Merge keeps the longers list of authors",
			article: &models.Article{Authors: []string{"Alice", "Bob"}},
			other:   &models.Article{Authors: []string{"Alice"}},
			want:    &models.Article{Authors: []string{"Alice", "Bob"}},
		},
		{
			name:    "Merge keeps the longers list of authors when longest is in the other",
			article: &models.Article{Authors: []string{"Alice"}},
			other:   &models.Article{Authors: []string{"Alice", "Bob"}},
			want:    &models.Article{Authors: []string{"Alice", "Bob"}},
		},
		{
			name:    "Merge keeps the year if present",
			article: &models.Article{},
			other:   &models.Article{Year: intPtr(2021)},
			want:    &models.Article{Year: intPtr(2021)},
		},
		{
			name:    "Merge keeps the title if present",
			article: &models.Article{},
			other:   &models.Article{Title: stringPtr("Title B")},
			want:    &models.Article{Title: stringPtr("Title B")},
		},
		{
			name:    "Merge keeps the journal if present",
			article: &models.Article{},
			other:   &models.Article{Journal: stringPtr("Journal B")},
			want:    &models.Article{Journal: stringPtr("Journal B")},
		},
		{
			name:    "Merge keeps the volume if present",
			article: &models.Article{},
			other:   &models.Article{Volume: stringPtr("Volume B")},
			want:    &models.Article{Volume: stringPtr("Volume B")},
		},
		{
			name:    "Merge keeps the issue if present",
			article: &models.Article{},
			other:   &models.Article{Issue: stringPtr("Issue B")},
			want:    &models.Article{Issue: stringPtr("Issue B")},
		},
		{
			name:    "Merge keeps the page if present",
			article: &models.Article{},
			other:   &models.Article{Page: stringPtr("Page B")},
			want:    &models.Article{Page: stringPtr("Page B")},
		},
		{
			name:    "Merge keeps the DOI if present",
			article: &models.Article{},
			other:   &models.Article{DOI: stringPtr("DOI B")},
			want:    &models.Article{DOI: stringPtr("DOI B")},
		},
		{
			name:    "Merge keeps the permalink if present",
			article: &models.Article{},
			other:   &models.Article{Permalink: stringPtr("Permalink B")},
			want:    &models.Article{Permalink: stringPtr("Permalink B")},
		},
		{
			name:    "Keeps the longest list of keywords",
			article: &models.Article{Keywords: []string{"keyword1", "keyword2"}},
			other:   &models.Article{Keywords: []string{"keyword1"}},
			want:    &models.Article{Keywords: []string{"keyword1", "keyword2"}},
		},
		{
			name:    "Keeps the longest list of keywords when longest is in the other",
			article: &models.Article{Keywords: []string{"keyword1"}},
			other:   &models.Article{Keywords: []string{"keyword1", "keyword2"}},
			want:    &models.Article{Keywords: []string{"keyword1", "keyword2"}},
		},
		{
			name:    "Keeps the longest list of references",
			article: &models.Article{References: []*models.Article{{Label: "Ref1"}, {Label: "Ref2"}}},
			other:   &models.Article{References: []*models.Article{{Label: "Ref1"}}},
			want:    &models.Article{References: []*models.Article{{Label: "Ref1"}, {Label: "Ref2"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.article.Merge(tt.other)
			assert.Equal(t, tt.want, got, "Merge() result mismatch")
		})
	}
}

func TestArticle_Key(t *testing.T) {
	tests := []struct {
		name    string          // description of this test case
		article *models.Article // the receiver type
		want    *string
	}{
		{
			name:    "Key returns nil when IDs is nil",
			article: &models.Article{},
			want:    nil,
		},
		{
			name:    "Key returns nil when no IDs are present",
			article: &models.Article{IDs: collections.NewSet[string]()},
			want:    nil,
		},
		{
			name:    "Key returns the first ID when IDs are present",
			article: &models.Article{IDs: collections.NewSet("id1", "id2")},
			want:    stringPtr("id1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.article.Key()
			assert.Equal(t, tt.want, got, "Key() result mismatch")
		})
	}
}

func TestArticle_SimpleLabel(t *testing.T) {
	tests := []struct {
		name    string          // description of this test case
		article *models.Article // the receiver type
		want    *string
	}{
		{
			name:    "SimpleLabel returns nil when article is nil",
			article: nil,
			want:    nil,
		},
		{
			name:    "SimpleLabel returns nil when no parts are available",
			article: &models.Article{Label: ""},
			want:    nil,
		},
		{
			name:    "SimpleLabel returns the first author when available",
			article: &models.Article{Authors: []string{"Alice"}},
			want:    stringPtr("Alice"),
		},
		{
			name:    "SimpleLabel returns the first author and year when available",
			article: &models.Article{Authors: []string{"Alice"}, Year: intPtr(2021)},
			want:    stringPtr("Alice, 2021"),
		},
		{
			name:    "SimpleLabel returns the first author and year with multiple authors",
			article: &models.Article{Authors: []string{"Alice", "Bob"}, Year: intPtr(2021)},
			want:    stringPtr("Alice, 2021"),
		},
		{
			name: "SimpleLabel returns the first author, year and journal when available",
			article: &models.Article{
				Authors: []string{"Alice"},
				Year:    intPtr(2021),
				Journal: stringPtr("Journal A"),
			},
			want: stringPtr("Alice, 2021, Journal A"),
		},
		{
			name: "SimpleLabel returns the first author, year, journal and volume when available",
			article: &models.Article{
				Authors: []string{"Alice"},
				Year:    intPtr(2021),
				Journal: stringPtr("Journal A"),
				Volume:  stringPtr("1"),
			},
			want: stringPtr("Alice, 2021, Journal A, V1"),
		},
		{
			name: "SimpleLabel returns the first author, year, journal, volume and page when available",
			article: &models.Article{
				Authors: []string{"Alice"},
				Year:    intPtr(2021),
				Journal: stringPtr("Journal A"),
				Volume:  stringPtr("1"),
				Page:    stringPtr("10"),
			},
			want: stringPtr("Alice, 2021, Journal A, V1, P10"),
		},
		{
			name: "SimpleLabel returns the first author, year, journal, volume, page and DOI when available",
			article: &models.Article{
				Authors: []string{"Alice"},
				Year:    intPtr(2021),
				Journal: stringPtr("Journal A"),
				Volume:  stringPtr("1"),
				Page:    stringPtr("10"),
				DOI:     stringPtr("10.1000/xyz123"),
			},
			want: stringPtr("Alice, 2021, Journal A, V1, P10, DOI 10.1000/xyz123"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.article.SimpleLabel()
			assert.Equal(t, tt.want, got, "SimpleLabel() result mismatch")
		})
	}
}

func TestArticle_SimpleId(t *testing.T) {
	tests := []struct {
		name    string          // description of this test case
		article *models.Article // the receiver type
		want    *string
	}{
		{
			name:    "SimpleId returns nil when article is nil",
			article: nil,
			want:    nil,
		},
		{
			name:    "SimpleId returns nil when no authors are present",
			article: &models.Article{Authors: []string{}, Year: intPtr(2021)},
			want:    nil,
		},
		{
			name:    "SimpleId returns nil when year is nil",
			article: &models.Article{Authors: []string{"Alice"}, Year: nil},
			want:    nil,
		},
		{
			name:    "SimpleId returns first author's name and year",
			article: &models.Article{Authors: []string{"Alice Smith"}, Year: intPtr(2021)},
			want:    stringPtr("alice2021"),
		},
		{
			name:    "SimpleId returns first author's name and year with multiple authors",
			article: &models.Article{Authors: []string{"Alice Smith", "Bob Johnson"}, Year: intPtr(2021)},
			want:    stringPtr("alice2021"),
		},
		{
			name:    "SimpleId returns first author's name and year with comma in name",
			article: &models.Article{Authors: []string{"Smith, Alice"}, Year: intPtr(2021)},
			want:    stringPtr("smith2021"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.article.SimpleId()
			assert.Equal(t, tt.want, got, "SimpleId() result mismatch")
		})
	}
}

func TestArticle_GetPermalink(t *testing.T) {
	tests := []struct {
		name    string          // description of this test case
		article *models.Article // the receiver type
		want    *string
	}{
		{
			name:    "GetPermalink returns nil when article is nil",
			article: nil,
			want:    nil,
		},
		{
			name: "GetPermalink returns the permalink when permalink is set regardless of DOI",
			article: &models.Article{
				Permalink: stringPtr("https://example.com/article"),
				DOI:       stringPtr("10.1000/xyz123"),
			},
			want: stringPtr("https://example.com/article"),
		},
		{
			name:    "GetPermalink returns the DOI as permalink when permalink is not set",
			article: &models.Article{DOI: stringPtr("10.1000/xyz123")},
			want:    stringPtr("https://doi.org/10.1000/xyz123"),
		},
		{
			name:    "GetPermalink returns nil when neither permalink nor DOI is set",
			article: &models.Article{Label: "article"},
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.article.GetPermalink()
			assert.Equal(t, tt.want, got, "GetPermalink() result mismatch")
		})
	}
}

func TestArticle_AddSimpleId(t *testing.T) {
	tests := []struct {
		name    string          // description of this test case
		article *models.Article // the receiver type
		want    *models.Article
	}{
		{
			name: "AddSimpleId adds simple ID when SimpleId is available",
			article: &models.Article{
				Authors: []string{"Alice Smith"},
				Year:    intPtr(2021),
				IDs:     collections.NewSet[string](),
			},
			want: &models.Article{
				Authors: []string{"Alice Smith"},
				Year:    intPtr(2021),
				IDs:     collections.NewSet("simple:alice2021"),
			},
		},
		{
			name: "AddSimpleId does not add simple ID when SimpleId is nil",
			article: &models.Article{
				Authors: []string{"Alice Smith"},
				Year:    nil,
				IDs:     collections.NewSet[string](),
			},
			want: &models.Article{
				Authors: []string{"Alice Smith"},
				Year:    nil,
				IDs:     collections.NewSet[string](),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.
			got := tt.article.AddSimpleId()
			assert.Equal(t, tt.want, got, "AddSimpleId() result mismatch")
		})
	}
}

func TestArticle_SetSimpleLabel(t *testing.T) {
	tests := []struct {
		name    string          // description of this test case
		article *models.Article // the receiver type
		want    *models.Article
	}{
		{
			name: "SetSimpleLabel sets simple label when SimpleLabel is available",
			article: &models.Article{
				Authors: []string{"Alice Smith"},
				Year:    intPtr(2021),
			},
			want: &models.Article{
				Authors: []string{"Alice Smith"},
				Year:    intPtr(2021),
				Label:   "Alice Smith, 2021",
			},
		},
		{
			name: "SetSimpleLabel does not change label when SimpleLabel is nil",
			article: &models.Article{
				Authors: []string{},
				Label:   "Existing Label",
			},
			want: &models.Article{
				Authors: []string{},
				Label:   "Existing Label",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.article.SetSimpleLabel()
			assert.Equal(t, tt.want, got, "SetSimpleLabel() result mismatch")
		})
	}
}
