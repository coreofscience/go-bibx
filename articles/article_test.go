package articles_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/articles"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/stretchr/testify/assert"
)

func TestArticle_Merge(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		article *articles.Article // the receiver type
		other   *articles.Article
		want    *articles.Article
	}{
		{
			name:    "Merge keeps the longest label",
			article: &articles.Article{Label: "A1"},
			other:   &articles.Article{Label: "AAAAA2"},
			want:    &articles.Article{Label: "AAAAA2"},
		},
		{
			name:    "Merge keeps the lowest label",
			article: &articles.Article{Label: "A1"},
			other:   &articles.Article{Label: "A2"},
			want:    &articles.Article{Label: "A1"},
		},
		{
			name:    "Merge keeps the lowest label",
			article: &articles.Article{Label: "A2"},
			other:   &articles.Article{Label: "A1"},
			want:    &articles.Article{Label: "A1"},
		},
		{
			name:    "Merge keeps the longest label when longest is in the receiver",
			article: &articles.Article{Label: "AAAAA2"},
			other:   &articles.Article{Label: "A1"},
			want:    &articles.Article{Label: "AAAAA2"},
		},
		{
			name:    "Merge makes an union of IDs",
			article: &articles.Article{IDs: collections.NewSet("id1", "id2")},
			other:   &articles.Article{IDs: collections.NewSet("id2", "id3")},
			want:    &articles.Article{IDs: collections.NewSet("id1", "id2", "id3")},
		},
		{
			name:    "Merge keeps the longers list of authors",
			article: &articles.Article{Authors: []string{"Alice", "Bob"}},
			other:   &articles.Article{Authors: []string{"Alice"}},
			want:    &articles.Article{Authors: []string{"Alice", "Bob"}},
		},
		{
			name:    "Merge keeps the longers list of authors when longest is in the other",
			article: &articles.Article{Authors: []string{"Alice"}},
			other:   &articles.Article{Authors: []string{"Alice", "Bob"}},
			want:    &articles.Article{Authors: []string{"Alice", "Bob"}},
		},
		{
			name:    "Merge keeps the year if present",
			article: &articles.Article{},
			other:   &articles.Article{Year: new(2021)},
			want:    &articles.Article{Year: new(2021)},
		},
		{
			name:    "Merge keeps the title if present",
			article: &articles.Article{},
			other:   &articles.Article{Title: new("Title B")},
			want:    &articles.Article{Title: new("Title B")},
		},
		{
			name:    "Merge keeps the journal if present",
			article: &articles.Article{},
			other:   &articles.Article{Journal: new("Journal B")},
			want:    &articles.Article{Journal: new("Journal B")},
		},
		{
			name:    "Merge keeps the volume if present",
			article: &articles.Article{},
			other:   &articles.Article{Volume: new("Volume B")},
			want:    &articles.Article{Volume: new("Volume B")},
		},
		{
			name:    "Merge keeps the issue if present",
			article: &articles.Article{},
			other:   &articles.Article{Issue: new("Issue B")},
			want:    &articles.Article{Issue: new("Issue B")},
		},
		{
			name:    "Merge keeps the page if present",
			article: &articles.Article{},
			other:   &articles.Article{Page: new("Page B")},
			want:    &articles.Article{Page: new("Page B")},
		},
		{
			name:    "Merge keeps the DOI if present",
			article: &articles.Article{},
			other:   &articles.Article{DOI: new("DOI B")},
			want:    &articles.Article{DOI: new("DOI B")},
		},
		{
			name:    "Merge keeps the permalink if present",
			article: &articles.Article{},
			other:   &articles.Article{Permalink: new("Permalink B")},
			want:    &articles.Article{Permalink: new("Permalink B")},
		},
		{
			name:    "Keeps the longest list of keywords",
			article: &articles.Article{Keywords: []string{"keyword1", "keyword2"}},
			other:   &articles.Article{Keywords: []string{"keyword1"}},
			want:    &articles.Article{Keywords: []string{"keyword1", "keyword2"}},
		},
		{
			name:    "Keeps the longest list of keywords when longest is in the other",
			article: &articles.Article{Keywords: []string{"keyword1"}},
			other:   &articles.Article{Keywords: []string{"keyword1", "keyword2"}},
			want:    &articles.Article{Keywords: []string{"keyword1", "keyword2"}},
		},
		{
			name:    "Keeps the longest list of references",
			article: &articles.Article{References: []*articles.Article{{Label: "Ref1"}, {Label: "Ref2"}}},
			other:   &articles.Article{References: []*articles.Article{{Label: "Ref1"}}},
			want:    &articles.Article{References: []*articles.Article{{Label: "Ref1"}, {Label: "Ref2"}}},
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
		name    string            // description of this test case
		article *articles.Article // the receiver type
		want    *string
	}{
		{
			name:    "Key returns nil when IDs is nil",
			article: &articles.Article{},
			want:    nil,
		},
		{
			name:    "Key returns nil when no IDs are present",
			article: &articles.Article{IDs: collections.NewSet[string]()},
			want:    nil,
		},
		{
			name:    "Key returns the first ID when IDs are present",
			article: &articles.Article{IDs: collections.NewSet("id1", "id2")},
			want:    new("id1"),
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
		name    string            // description of this test case
		article *articles.Article // the receiver type
		want    *string
	}{
		{
			name:    "SimpleLabel returns nil when article is nil",
			article: nil,
			want:    nil,
		},
		{
			name:    "SimpleLabel returns nil when no parts are available",
			article: &articles.Article{Label: ""},
			want:    nil,
		},
		{
			name:    "SimpleLabel returns the first author when available",
			article: &articles.Article{Authors: []string{"Alice"}},
			want:    new("Alice"),
		},
		{
			name:    "SimpleLabel returns the first author and year when available",
			article: &articles.Article{Authors: []string{"Alice"}, Year: new(2021)},
			want:    new("Alice, 2021"),
		},
		{
			name:    "SimpleLabel returns the first author and year with multiple authors",
			article: &articles.Article{Authors: []string{"Alice", "Bob"}, Year: new(2021)},
			want:    new("Alice, 2021"),
		},
		{
			name: "SimpleLabel returns the first author, year and journal when available",
			article: &articles.Article{
				Authors: []string{"Alice"},
				Year:    new(2021),
				Journal: new("Journal A"),
			},
			want: new("Alice, 2021, Journal A"),
		},
		{
			name: "SimpleLabel returns the first author, year, journal and volume when available",
			article: &articles.Article{
				Authors: []string{"Alice"},
				Year:    new(2021),
				Journal: new("Journal A"),
				Volume:  new("1"),
			},
			want: new("Alice, 2021, Journal A, V1"),
		},
		{
			name: "SimpleLabel returns the first author, year, journal, volume and page when available",
			article: &articles.Article{
				Authors: []string{"Alice"},
				Year:    new(2021),
				Journal: new("Journal A"),
				Volume:  new("1"),
				Page:    new("10"),
			},
			want: new("Alice, 2021, Journal A, V1, P10"),
		},
		{
			name: "SimpleLabel returns the first author, year, journal, volume, page and DOI when available",
			article: &articles.Article{
				Authors: []string{"Alice"},
				Year:    new(2021),
				Journal: new("Journal A"),
				Volume:  new("1"),
				Page:    new("10"),
				DOI:     new("10.1000/xyz123"),
			},
			want: new("Alice, 2021, Journal A, V1, P10, DOI 10.1000/xyz123"),
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
		name    string            // description of this test case
		article *articles.Article // the receiver type
		want    *string
	}{
		{
			name:    "SimpleId returns nil when article is nil",
			article: nil,
			want:    nil,
		},
		{
			name:    "SimpleId returns nil when no authors are present",
			article: &articles.Article{Authors: []string{}, Year: new(2021)},
			want:    nil,
		},
		{
			name:    "SimpleId returns nil when year is nil",
			article: &articles.Article{Authors: []string{"Alice"}, Year: nil},
			want:    nil,
		},
		{
			name:    "SimpleId returns first author's name and year",
			article: &articles.Article{Authors: []string{"Alice Smith"}, Year: new(2021)},
			want:    new("alice2021"),
		},
		{
			name:    "SimpleId returns first author's name and year with multiple authors",
			article: &articles.Article{Authors: []string{"Alice Smith", "Bob Johnson"}, Year: new(2021)},
			want:    new("alice2021"),
		},
		{
			name:    "SimpleId returns first author's name and year with comma in name",
			article: &articles.Article{Authors: []string{"Smith, Alice"}, Year: new(2021)},
			want:    new("smith2021"),
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
		name    string            // description of this test case
		article *articles.Article // the receiver type
		want    *string
	}{
		{
			name:    "GetPermalink returns nil when article is nil",
			article: nil,
			want:    nil,
		},
		{
			name: "GetPermalink returns the permalink when permalink is set regardless of DOI",
			article: &articles.Article{
				Permalink: new("https://example.com/article"),
				DOI:       new("10.1000/xyz123"),
			},
			want: new("https://example.com/article"),
		},
		{
			name:    "GetPermalink returns the DOI as permalink when permalink is not set",
			article: &articles.Article{DOI: new("10.1000/xyz123")},
			want:    new("https://doi.org/10.1000/xyz123"),
		},
		{
			name:    "GetPermalink returns nil when neither permalink nor DOI is set",
			article: &articles.Article{Label: "article"},
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
		name    string            // description of this test case
		article *articles.Article // the receiver type
		want    *articles.Article
	}{
		{
			name: "AddSimpleId adds simple ID when SimpleId is available",
			article: &articles.Article{
				Authors: []string{"Alice Smith"},
				Year:    new(2021),
				IDs:     collections.NewSet[string](),
			},
			want: &articles.Article{
				Authors: []string{"Alice Smith"},
				Year:    new(2021),
				IDs:     collections.NewSet("simple:alice2021"),
			},
		},
		{
			name: "AddSimpleId does not add simple ID when SimpleId is nil",
			article: &articles.Article{
				Authors: []string{"Alice Smith"},
				Year:    nil,
				IDs:     collections.NewSet[string](),
			},
			want: &articles.Article{
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
		name    string            // description of this test case
		article *articles.Article // the receiver type
		want    *articles.Article
	}{
		{
			name: "SetSimpleLabel sets simple label when SimpleLabel is available",
			article: &articles.Article{
				Authors: []string{"Alice Smith"},
				Year:    new(2021),
			},
			want: &articles.Article{
				Authors: []string{"Alice Smith"},
				Year:    new(2021),
				Label:   "Alice Smith, 2021",
			},
		},
		{
			name: "SetSimpleLabel does not change label when SimpleLabel is nil",
			article: &articles.Article{
				Authors: []string{},
				Label:   "Existing Label",
			},
			want: &articles.Article{
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
