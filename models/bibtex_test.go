package models_test

import (
	"strings"
	"testing"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/nickng/bibtex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCitationKey(t *testing.T) {
	t.Run("nil work returns fallback", func(t *testing.T) {
		assert.Equal(t, "article", models.GenerateCitationKey(nil))
	})

	t.Run("generates from SimpleId", func(t *testing.T) {
		year := 2017
		work := &models.Work{
			Authors: []string{"Vaswani, Ashish", "Shazeer, Noam"},
			Year:    &year,
		}
		assert.Equal(t, "vaswani2017", models.GenerateCitationKey(work))
	})

	t.Run("generates from DOI when SimpleId is missing", func(t *testing.T) {
		doi := "10.1145/3295222"
		work := &models.Work{
			DOI: &doi,
		}
		assert.Equal(t, "doi_10_1145_3295222", models.GenerateCitationKey(work))
	})

	t.Run("generates from OpenAlex ID when DOI and SimpleId missing", func(t *testing.T) {
		work := &models.Work{
			IDs: collections.NewSet("openalex:https://openalex.org/W2741809807"),
		}
		assert.Equal(t, "w2741809807", models.GenerateCitationKey(work))
	})

	t.Run("generates from Key when others missing", func(t *testing.T) {
		work := &models.Work{
			IDs: collections.NewSet("mag:123456"),
		}
		assert.Equal(t, "mag_123456", models.GenerateCitationKey(work))
	})

	t.Run("generates from Label when IDs missing", func(t *testing.T) {
		work := &models.Work{
			Label: "My Custom-Label/2024",
		}
		assert.Equal(t, "my_custom-label_2024", models.GenerateCitationKey(work))
	})

	t.Run("sanitizes special characters and diacritics", func(t *testing.T) {
		year := 2022
		work := &models.Work{
			Authors: []string{"Müller, Hans", "François, Jean", "Nuñez, Carlos", "Åström, Karl"},
			Year:    &year,
		}
		assert.Equal(t, "muller2022", models.GenerateCitationKey(work))

		art2 := &models.Work{
			Label: "???special!@#characters$$$here",
		}
		assert.Equal(t, "special_characters_here", models.GenerateCitationKey(art2))
	})
}

func TestWork_ToBibEntry(t *testing.T) {
	t.Run("nil work returns nil", func(t *testing.T) {
		var a *models.Work
		assert.Nil(t, a.ToBibEntry())
	})

	t.Run("full work with journal becomes work type", func(t *testing.T) {
		year := 2021
		title := "Graph Attention Networks in Practice"
		venue := "IEEE Transactions on Pattern Analysis and Machine Intelligence"
		vol := "43"
		issue := "5"
		page := "1500-1512"
		doi := "10.1109/TPAMI.2020.1234567"
		permalink := "https://doi.org/10.1109/TPAMI.2020.1234567"
		abstract := "This paper introduces practical considerations for graph attention networks.\nWith line breaks."

		work := &models.Work{
			Label:     "openalex:W123",
			Authors:   []string{"Veličković, Petar", "Cucurull, Guillem"},
			Year:      &year,
			Title:     &title,
			Venue:     &venue,
			Volume:    &vol,
			Issue:     &issue,
			Page:      &page,
			DOI:       &doi,
			Permalink: &permalink,
			Abstract:  &abstract,
			Keywords:  collections.NewSet("deep learning", "graph neural networks"),
		}

		entry := work.ToBibEntry()
		require.NotNil(t, entry)
		assert.Equal(t, "article", entry.Type)
		assert.Equal(t, "velickovic2021", entry.CiteName)

		// Check fields
		assert.Equal(t, "Veličković, Petar and Cucurull, Guillem", entry.Fields["author"].String())
		assert.Equal(t, "Graph Attention Networks in Practice", entry.Fields["title"].String())
		assert.Equal(t, "IEEE Transactions on Pattern Analysis and Machine Intelligence", entry.Fields["journal"].String())
		assert.Equal(t, "2021", entry.Fields["year"].String())
		assert.Equal(t, "43", entry.Fields["volume"].String())
		assert.Equal(t, "5", entry.Fields["number"].String())
		assert.Equal(t, "1500--1512", entry.Fields["pages"].String())
		assert.Equal(t, "10.1109/TPAMI.2020.1234567", entry.Fields["doi"].String())
		assert.Equal(t, "https://doi.org/10.1109/TPAMI.2020.1234567", entry.Fields["url"].String())
		assert.Equal(t, "This paper introduces practical considerations for graph attention networks. With line breaks.", entry.Fields["abstract"].String())
	})

	t.Run("work without journal becomes misc type", func(t *testing.T) {
		year := 2023
		title := "ArXiv Preprint on Transformers"
		work := &models.Work{
			Authors: []string{"Smith, John"},
			Year:    &year,
			Title:   &title,
		}

		entry := work.ToBibEntry(models.WithCiteKey("customKey2023"))
		require.NotNil(t, entry)
		assert.Equal(t, "misc", entry.Type)
		assert.Equal(t, "customKey2023", entry.CiteName)
		_, hasVenue := entry.Fields["journal"]
		assert.False(t, hasVenue)
	})

	t.Run("omits abstract when WithOmitAbstracts is true", func(t *testing.T) {
		year := 2024
		title := "Test Paper"
		abstract := "Very long abstract text."
		work := &models.Work{
			Authors:  []string{"Doe, Jane"},
			Year:     &year,
			Title:    &title,
			Abstract: &abstract,
		}

		entry := work.ToBibEntry(models.WithOmitAbstracts(true))
		require.NotNil(t, entry)
		_, hasAbstract := entry.Fields["abstract"]
		assert.False(t, hasAbstract)
	})
}

func TestWork_ToBibEntry_Types(t *testing.T) {
	tests := []struct {
		name       string
		work       *models.Work
		wantType   string
		wantFields map[string]string
		wantAbsent []string
	}{
		{
			name:       "conference paper",
			work:       &models.Work{Type: new("conference-paper"), Venue: new("SOSP"), VenueType: new("conference")},
			wantType:   "inproceedings",
			wantFields: map[string]string{"booktitle": "SOSP"},
			wantAbsent: []string{"journal"},
		},
		{
			name:       "article at a conference venue",
			work:       &models.Work{Type: new("article"), Venue: new("PVLDB"), VenueType: new("conference")},
			wantType:   "inproceedings",
			wantFields: map[string]string{"booktitle": "PVLDB"},
		},
		{
			name:       "book on an ebook platform",
			work:       &models.Work{Type: new("book"), Venue: new("Apress eBooks"), VenueType: new("ebook platform"), Publisher: new("Apress")},
			wantType:   "book",
			wantFields: map[string]string{"publisher": "Apress"},
			wantAbsent: []string{"journal", "series"},
		},
		{
			name:       "book in a series",
			work:       &models.Work{Type: new("book"), Venue: new("LNCS"), VenueType: new("book series"), Publisher: new("Springer")},
			wantType:   "book",
			wantFields: map[string]string{"publisher": "Springer", "series": "LNCS"},
		},
		{
			name:       "book without a publisher",
			work:       &models.Work{Type: new("book")},
			wantType:   "book",
			wantAbsent: []string{"publisher"},
		},
		{
			name:       "report prefers the author institution",
			work:       &models.Work{Type: new("report"), Institution: new("NIST"), Publisher: new("GPO")},
			wantType:   "techreport",
			wantFields: map[string]string{"institution": "NIST"},
		},
		{
			name:       "report falls back to the publisher",
			work:       &models.Work{Type: new("report"), Publisher: new("GPO")},
			wantType:   "techreport",
			wantFields: map[string]string{"institution": "GPO"},
		},
		{
			name:       "book chapter",
			work:       &models.Work{Type: new("book-chapter"), Venue: new("Internet of Things"), VenueType: new("book series"), Publisher: new("Springer")},
			wantType:   "incollection",
			wantFields: map[string]string{"booktitle": "Internet of Things", "publisher": "Springer"},
		},
		{
			name:       "book chapter on an ebook platform has no book title",
			work:       &models.Work{Type: new("book-chapter"), Venue: new("Elsevier eBooks"), VenueType: new("ebook platform"), Publisher: new("Elsevier")},
			wantType:   "incollection",
			wantFields: map[string]string{"publisher": "Elsevier"},
			wantAbsent: []string{"booktitle"},
		},
		{
			name:       "dissertation in a repository",
			work:       &models.Work{Type: new("dissertation"), Venue: new("DSpace@MIT"), VenueType: new("repository"), Publisher: new("MIT")},
			wantType:   "phdthesis",
			wantFields: map[string]string{"school": "MIT"},
			wantAbsent: []string{"note", "journal"},
		},
		{
			name:       "preprint in a repository is flagged",
			work:       &models.Work{Type: new("preprint"), Venue: new("arXiv"), VenueType: new("repository")},
			wantType:   "misc",
			wantFields: map[string]string{"howpublished": "arXiv", "note": "venue from repository copy"},
			wantAbsent: []string{"journal"},
		},
		{
			name:       "conference paper in a repository is flagged without a book title",
			work:       &models.Work{Type: new("conference-paper"), Venue: new("Zenodo"), VenueType: new("repository")},
			wantType:   "inproceedings",
			wantFields: map[string]string{"note": "venue from repository copy"},
			wantAbsent: []string{"booktitle", "howpublished"},
		},
		{
			name:       "report in a repository is not flagged",
			work:       &models.Work{Type: new("report"), Venue: new("NBER"), VenueType: new("repository"), Institution: new("NBER")},
			wantType:   "techreport",
			wantFields: map[string]string{"institution": "NBER"},
			wantAbsent: []string{"note", "howpublished"},
		},
		{
			name:       "review in a journal",
			work:       &models.Work{Type: new("review"), Venue: new("ACM CSUR"), VenueType: new("journal")},
			wantType:   "article",
			wantFields: map[string]string{"journal": "ACM CSUR"},
		},
		{
			name:       "article on an ebook platform",
			work:       &models.Work{Type: new("article"), Venue: new("Springer eBooks"), VenueType: new("ebook platform")},
			wantType:   "misc",
			wantAbsent: []string{"journal", "howpublished"},
		},
		{
			name:       "unknown type with a venue",
			work:       &models.Work{Type: new("editorial"), Venue: new("Nature"), VenueType: new("journal")},
			wantType:   "article",
			wantFields: map[string]string{"journal": "Nature"},
		},
		{
			name:       "work stored without types keeps its export",
			work:       &models.Work{Venue: new("KTH DiVA")},
			wantType:   "article",
			wantFields: map[string]string{"journal": "KTH DiVA"},
			wantAbsent: []string{"note"},
		},
		{
			name:       "work stored without types or venue",
			work:       &models.Work{},
			wantType:   "misc",
			wantAbsent: []string{"journal"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := tt.work.ToBibEntry()
			require.NotNil(t, entry)
			assert.Equal(t, tt.wantType, entry.Type)
			for field, want := range tt.wantFields {
				require.Contains(t, entry.Fields, field)
				assert.Equal(t, want, entry.Fields[field].String())
			}
			for _, field := range tt.wantAbsent {
				assert.NotContains(t, entry.Fields, field)
			}
		})
	}
}

func TestWork_ToBibTeX(t *testing.T) {
	t.Run("nil work returns empty string", func(t *testing.T) {
		var a *models.Work
		assert.Empty(t, a.ToBibTeX())
	})

	t.Run("formats valid parseable BibTeX string", func(t *testing.T) {
		year := 2017
		title := "Attention Is All You Need"
		venue := "NeurIPS"
		doi := "10.5555/3295222"
		work := &models.Work{
			Authors: []string{"Vaswani, Ashish", "Shazeer, Noam"},
			Year:    &year,
			Title:   &title,
			Venue:   &venue,
			DOI:     &doi,
		}

		rawBib := work.ToBibTeX()
		require.NotEmpty(t, rawBib)
		assert.Contains(t, rawBib, "@article{vaswani2017,")
		assert.Contains(t, rawBib, "author")
		assert.Contains(t, rawBib, "Vaswani, Ashish and Shazeer, Noam")
		assert.Contains(t, rawBib, "title")
		assert.Contains(t, rawBib, "Attention Is All You Need")
		assert.Contains(t, rawBib, "journal")
		assert.Contains(t, rawBib, "NeurIPS")
		assert.Contains(t, rawBib, "year")
		assert.Contains(t, rawBib, "2017")

		// Validate that nickng/bibtex parser can parse the generated BibTeX output
		parsed, err := bibtex.Parse(strings.NewReader(rawBib))
		require.NoError(t, err)
		require.Len(t, parsed.Entries, 1)
		assert.Equal(t, "article", parsed.Entries[0].Type)
		assert.Equal(t, "vaswani2017", parsed.Entries[0].CiteName)
	})
}

func TestWorks_ToBibTeX(t *testing.T) {
	t.Run("empty slice returns empty string", func(t *testing.T) {
		var works models.Works
		assert.Empty(t, works.ToBibTeX())
	})

	t.Run("slice with only nil works returns empty string", func(t *testing.T) {
		works := models.Works{nil, nil}
		assert.Empty(t, works.ToBibTeX())
	})

	t.Run("disambiguates duplicate citation keys beyond 26", func(t *testing.T) {
		year := 2024
		venue := "Science"
		works := make(models.Works, 30)
		for i := range 30 {
			title := "Paper"
			works[i] = &models.Work{
				Authors: []string{"Doe, Jane"},
				Year:    &year,
				Title:   &title,
				Venue:   &venue,
			}
		}

		rawBib := works.ToBibTeX()
		require.NotEmpty(t, rawBib)

		parsed, err := bibtex.Parse(strings.NewReader(rawBib))
		require.NoError(t, err)
		require.Len(t, parsed.Entries, 30)

		assert.Equal(t, "doe2024", parsed.Entries[0].CiteName)
		assert.Equal(t, "doe2024a", parsed.Entries[1].CiteName)
		assert.Equal(t, "doe2024z", parsed.Entries[26].CiteName)
		assert.Equal(t, "doe2024aa", parsed.Entries[27].CiteName)
		assert.Equal(t, "doe2024ab", parsed.Entries[28].CiteName)
		assert.Equal(t, "doe2024ac", parsed.Entries[29].CiteName)
	})

	t.Run("disambiguates duplicate citation keys", func(t *testing.T) {
		year := 2020
		title1 := "First Paper by Smith"
		title2 := "Second Paper by Smith"
		title3 := "Third Paper by Smith"
		venue := "Nature"

		works := models.Works{
			{
				Authors: []string{"Smith, John"},
				Year:    &year,
				Title:   &title1,
				Venue:   &venue,
			},
			{
				Authors: []string{"Smith, John"},
				Year:    &year,
				Title:   &title2,
				Venue:   &venue,
			},
			{
				Authors: []string{"Smith, John"},
				Year:    &year,
				Title:   &title3,
				Venue:   &venue,
			},
		}

		rawBib := works.ToBibTeX()
		require.NotEmpty(t, rawBib)

		assert.Contains(t, rawBib, "@article{smith2020,")
		assert.Contains(t, rawBib, "@article{smith2020a,")
		assert.Contains(t, rawBib, "@article{smith2020b,")

		// Parse back with bibtex.Parse
		parsed, err := bibtex.Parse(strings.NewReader(rawBib))
		require.NoError(t, err)
		require.Len(t, parsed.Entries, 3)

		citeKeys := []string{
			parsed.Entries[0].CiteName,
			parsed.Entries[1].CiteName,
			parsed.Entries[2].CiteName,
		}
		assert.Equal(t, []string{"smith2020", "smith2020a", "smith2020b"}, citeKeys)
	})
}

func TestSanitizeLaTeX(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escapes percent signs",
			input:    "fluctuations smaller than 8% of the bit period",
			expected: `fluctuations smaller than 8\% of the bit period`,
		},
		{
			name:     "escapes ampersands",
			input:    "AT&T Bell Labs & IBM",
			expected: `AT\&T Bell Labs \& IBM`,
		},
		{
			name:     "escapes underscores",
			input:    "deep_learning_models_in_practice",
			expected: `deep\_learning\_models\_in\_practice`,
		},
		{
			name:     "escapes hash symbols",
			input:    "issue #42 and rule #1",
			expected: `issue \#42 and rule \#1`,
		},
		{
			name:     "does not double escape already escaped characters",
			input:    `already escaped 8\% and \& symbol`,
			expected: `already escaped 8\% and \& symbol`,
		},
		{
			name:     "plain string without special characters unchanged",
			input:    "Attention Is All You Need",
			expected: "Attention Is All You Need",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, models.SanitizeLaTeX(tc.input))
		})
	}
}

func TestWorks_ToBibTeXWithKeys(t *testing.T) {
	year := 2021
	title1 := "Research on 8% efficiency in BPM"
	title2 := "R&D in deep_learning #1"
	venue := "Science & Technology"

	works := models.Works{
		{
			Label:   "openalex:W1",
			Authors: []string{"Albrecht, Manfred"},
			Year:    &year,
			Title:   &title1,
			Venue:   &venue,
		},
		{
			Label:   "openalex:W2",
			Authors: []string{"Albrecht, Manfred"},
			Year:    &year,
			Title:   &title2,
			Venue:   &venue,
		},
	}

	bibStr, keys := works.ToBibTeXWithKeys()
	require.NotEmpty(t, bibStr)
	require.Len(t, keys, 2)

	assert.Equal(t, "openalex:W1", keys[0].ID)
	assert.Equal(t, "albrecht2021", keys[0].CiteKey)
	assert.Equal(t, "Research on 8% efficiency in BPM", keys[0].Title)
	assert.Equal(t, "Albrecht, Manfred", keys[0].FirstAuthor)
	assert.Equal(t, &year, keys[0].Year)

	assert.Equal(t, "openalex:W2", keys[1].ID)
	assert.Equal(t, "albrecht2021a", keys[1].CiteKey)

	// Verify that escaped characters in bibStr parse without errors
	assert.Contains(t, bibStr, `8\%`)
	assert.Contains(t, bibStr, `R\&D`)
	assert.Contains(t, bibStr, `deep\_learning`)
	assert.Contains(t, bibStr, `\#1`)
	assert.Contains(t, bibStr, `Science \& Technology`)

	parsed, err := bibtex.Parse(strings.NewReader(bibStr))
	require.NoError(t, err)
	require.Len(t, parsed.Entries, 2)
}
