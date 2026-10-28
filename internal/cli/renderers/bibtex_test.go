package renderers_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/coreofscience/go-bibx/internal/cli/renderers"
	"github.com/coreofscience/go-bibx/models"
	"github.com/nickng/bibtex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBibtexRenderer_Formats(t *testing.T) {
	assert.Contains(t, renderers.AnalysisFormats(), "bibtex")
	assert.Contains(t, renderers.ResultFormats(), "bibtex")

	r, err := renderers.NewRendererWithFormat("bibtex")
	require.NoError(t, err)
	assert.IsType(t, &renderers.BibtexRenderer{}, r)

	ar, err := renderers.NewAnalysisRendererWithFormat("bibtex")
	require.NoError(t, err)
	assert.IsType(t, &renderers.BibtexRenderer{}, ar)

	rr, err := renderers.NewResultRendererWithFormat("bibtex")
	require.NoError(t, err)
	assert.IsType(t, &renderers.BibtexRenderer{}, rr)
}

func TestBibtexRenderer_RenderResults(t *testing.T) {
	renderer := renderers.NewBibtexRenderer()

	t.Run("renders scored results as bibtex", func(t *testing.T) {
		year := 2020
		title := "Graph Attention Networks"
		journal := "ICLR"
		art := &models.Article{
			Authors: []string{"Veličković, Petar"},
			Year:    &year,
			Title:   &title,
			Journal: &journal,
		}

		results := []*models.Result{
			{Score: 0.95, Article: art},
			{Score: 0.80, Article: nil}, // nil article handled gracefully
		}

		var buf bytes.Buffer
		err := renderer.RenderResults(&buf, results)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "@article{velickovic2020,")
		assert.Contains(t, output, "Graph Attention Networks")

		parsed, err := bibtex.Parse(strings.NewReader(output))
		require.NoError(t, err)
		require.Len(t, parsed.Entries, 1)
	})

	t.Run("empty results produces empty output", func(t *testing.T) {
		var buf bytes.Buffer
		err := renderer.RenderResults(&buf, nil)
		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})
}

func TestBibtexRenderer_RenderAnalysis(t *testing.T) {
	renderer := renderers.NewBibtexRenderer()

	t.Run("renders analysis nodes as bibtex", func(t *testing.T) {
		year1 := 2017
		title1 := "Attention Is All You Need"
		journal1 := "NeurIPS"

		year2 := 2018
		title2 := "BERT"
		journal2 := "NAACL"

		analysis := &models.Analysis{
			Nodes: []*models.Node{
				{
					ID: "w1",
					Article: &models.Article{
						Authors: []string{"Vaswani, Ashish"},
						Year:    &year1,
						Title:   &title1,
						Journal: &journal1,
					},
				},
				{
					ID: "w2",
					Article: &models.Article{
						Authors: []string{"Devlin, Jacob"},
						Year:    &year2,
						Title:   &title2,
						Journal: &journal2,
					},
				},
				{
					ID:      "w3",
					Article: nil, // nil article handled gracefully
				},
			},
		}

		var buf bytes.Buffer
		err := renderer.RenderAnalysis(&buf, analysis)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "@article{vaswani2017,")
		assert.Contains(t, output, "@article{devlin2018,")

		parsed, err := bibtex.Parse(strings.NewReader(output))
		require.NoError(t, err)
		require.Len(t, parsed.Entries, 2)
	})

	t.Run("nil analysis produces empty output", func(t *testing.T) {
		var buf bytes.Buffer
		err := renderer.RenderAnalysis(&buf, nil)
		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})
}
