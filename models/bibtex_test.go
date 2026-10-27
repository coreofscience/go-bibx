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
	t.Run("nil article returns fallback", func(t *testing.T) {
		assert.Equal(t, "article", models.GenerateCitationKey(nil))
	})

	t.Run("generates from SimpleId", func(t *testing.T) {
		year := 2017
		art := &models.Article{
			Authors: []string{"Vaswani, Ashish", "Shazeer, Noam"},
			Year:    &year,
		}
		assert.Equal(t, "vaswani2017", models.GenerateCitationKey(art))
	})

	t.Run("generates from DOI when SimpleId is missing", func(t *testing.T) {
		doi := "10.1145/3295222"
		art := &models.Article{
			DOI: &doi,
		}
		assert.Equal(t, "doi_10_1145_3295222", models.GenerateCitationKey(art))
	})

	t.Run("generates from OpenAlex ID when DOI and SimpleId missing", func(t *testing.T) {
		art := &models.Article{
			IDs: collections.NewSet("openalex:https://openalex.org/W2741809807"),
		}
		assert.Equal(t, "w2741809807", models.GenerateCitationKey(art))
	})

	t.Run("generates from Key when others missing", func(t *testing.T) {
		art := &models.Article{
			IDs: collections.NewSet("mag:123456"),
		}
		assert.Equal(t, "mag_123456", models.GenerateCitationKey(art))
	})

	t.Run("generates from Label when IDs missing", func(t *testing.T) {
		art := &models.Article{
			Label: "My Custom-Label/2024",
		}
		assert.Equal(t, "my_custom-label_2024", models.GenerateCitationKey(art))
	})

	t.Run("sanitizes special characters and diacritics", func(t *testing.T) {
		year := 2022
		art := &models.Article{
			Authors: []string{"Müller, Hans", "François, Jean", "Nuñez, Carlos", "Åström, Karl"},
			Year:    &year,
		}
		assert.Equal(t, "muller2022", models.GenerateCitationKey(art))

		art2 := &models.Article{
			Label: "???special!@#characters$$$here",
		}
		assert.Equal(t, "special_characters_here", models.GenerateCitationKey(art2))
	})
}

func TestArticle_ToBibEntry(t *testing.T) {
	t.Run("nil article returns nil", func(t *testing.T) {
		var a *models.Article
		assert.Nil(t, a.ToBibEntry())
	})

	t.Run("full article with journal becomes article type", func(t *testing.T) {
		year := 2021
		title := "Graph Attention Networks in Practice"
		journal := "IEEE Transactions on Pattern Analysis and Machine Intelligence"
		vol := "43"
		issue := "5"
		page := "1500-1512"
		doi := "10.1109/TPAMI.2020.1234567"
		permalink := "https://doi.org/10.1109/TPAMI.2020.1234567"
		abstract := "This paper introduces practical considerations for graph attention networks.\nWith line breaks."

		art := &models.Article{
			Label:     "openalex:W123",
			Authors:   []string{"Veličković, Petar", "Cucurull, Guillem"},
			Year:      &year,
			Title:     &title,
			Journal:   &journal,
			Volume:    &vol,
			Issue:     &issue,
			Page:      &page,
			DOI:       &doi,
			Permalink: &permalink,
			Abstract:  &abstract,
			Keywords:  collections.NewSet("deep learning", "graph neural networks"),
		}

		entry := art.ToBibEntry()
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

	t.Run("article without journal becomes misc type", func(t *testing.T) {
		year := 2023
		title := "ArXiv Preprint on Transformers"
		art := &models.Article{
			Authors: []string{"Smith, John"},
			Year:    &year,
			Title:   &title,
		}

		entry := art.ToBibEntry("customKey2023")
		require.NotNil(t, entry)
		assert.Equal(t, "misc", entry.Type)
		assert.Equal(t, "customKey2023", entry.CiteName)
		_, hasJournal := entry.Fields["journal"]
		assert.False(t, hasJournal)
	})
}

func TestArticle_ToBibTeX(t *testing.T) {
	t.Run("nil article returns empty string", func(t *testing.T) {
		var a *models.Article
		assert.Empty(t, a.ToBibTeX())
	})

	t.Run("formats valid parseable BibTeX string", func(t *testing.T) {
		year := 2017
		title := "Attention Is All You Need"
		journal := "NeurIPS"
		doi := "10.5555/3295222"
		art := &models.Article{
			Authors: []string{"Vaswani, Ashish", "Shazeer, Noam"},
			Year:    &year,
			Title:   &title,
			Journal: &journal,
			DOI:     &doi,
		}

		rawBib := art.ToBibTeX()
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

func TestArticles_ToBibTeX(t *testing.T) {
	t.Run("empty slice returns empty string", func(t *testing.T) {
		var articles models.Articles
		assert.Empty(t, articles.ToBibTeX())
	})

	t.Run("slice with only nil articles returns empty string", func(t *testing.T) {
		articles := models.Articles{nil, nil}
		assert.Empty(t, articles.ToBibTeX())
	})

	t.Run("disambiguates duplicate citation keys beyond 26", func(t *testing.T) {
		year := 2024
		journal := "Science"
		articles := make(models.Articles, 30)
		for i := range 30 {
			title := "Paper"
			articles[i] = &models.Article{
				Authors: []string{"Doe, Jane"},
				Year:    &year,
				Title:   &title,
				Journal: &journal,
			}
		}

		rawBib := articles.ToBibTeX()
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
		journal := "Nature"

		articles := models.Articles{
			{
				Authors: []string{"Smith, John"},
				Year:    &year,
				Title:   &title1,
				Journal: &journal,
			},
			{
				Authors: []string{"Smith, John"},
				Year:    &year,
				Title:   &title2,
				Journal: &journal,
			},
			{
				Authors: []string{"Smith, John"},
				Year:    &year,
				Title:   &title3,
				Journal: &journal,
			},
		}

		rawBib := articles.ToBibTeX()
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
