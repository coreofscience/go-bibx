package renderers_test

import (
	"bytes"
	"testing"

	"github.com/coreofscience/go-bibx/internal/cli/renderers"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownRenderer_SimpleFormat(t *testing.T) {
	tests := []struct {
		name     string
		work     *models.Work
		expected string
	}{
		{
			name: "work with permalink",
			work: &models.Work{
				Title:     new("Test Title"),
				Label:     "TestLabel",
				Permalink: new("https://example.com/work"),
				Keywords:  collections.NewSet("A", "B"),
				Abstract:  new("This is an abstract."),
			},
			expected: "# Test Title\n**link**: [https://example.com/work](https://example.com/work)\n\n**keywords**: A, B\n\n## Abstract\n\nThis is an abstract.\n",
		},
		{
			name: "work without permalink but with DOI",
			work: &models.Work{
				Title:    new("Test Title"),
				Label:    "TestLabel",
				DOI:      new("10.1000/xyz123"),
				Keywords: collections.NewSet("A", "B"),
				Abstract: new("This is an abstract."),
			},
			expected: "# Test Title\n**link**: [https://doi.org/10.1000/xyz123](https://doi.org/10.1000/xyz123)\n\n**keywords**: A, B\n\n## Abstract\n\nThis is an abstract.\n",
		},
		{
			name: "work without permalink or DOI",
			work: &models.Work{
				Title:    new("Test Title"),
				Label:    "TestLabel",
				Keywords: collections.NewSet("A", "B"),
				Abstract: new("This is an abstract."),
			},
			expected: "# Test Title\n\n**keywords**: A, B\n\n## Abstract\n\nThis is an abstract.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer, err := renderers.NewMarkdownRenderer("simple")
			require.NoError(t, err)

			var buf bytes.Buffer
			err = renderer.Render(&buf, tt.work, "simple")
			require.NoError(t, err)

			// Remove the separator suffix that render appends
			output := buf.String()
			const suffix = "\n\n---\n\n"
			if assert.Contains(t, output, suffix) {
				output = output[:len(output)-len(suffix)]
			}

			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestMarkdownRenderer_MarkdownFormat(t *testing.T) {
	renderer, err := renderers.NewMarkdownRenderer("markdown")
	require.NoError(t, err)

	work := &models.Work{
		Title:    new("Test Title"),
		Label:    "TestLabel",
		Abstract: new("This is an abstract."),
	}

	var buf bytes.Buffer
	err = renderer.Render(&buf, work, "markdown")
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "keywords: []")

	year := 2024
	refArt := &models.Work{
		Label:   "openalex:W2",
		Title:   new("Referenced Paper"),
		Authors: []string{"Doe, Jane"},
		Venue:   new("Nature"),
		Year:    &year,
		Rich:    true,
	}
	analysis := &models.Analysis{
		Nodes: []*models.Node{
			{ID: "openalex:W1", Work: work},
			{ID: "openalex:W2", Work: refArt},
		},
		Links: []*models.Link{
			{Source: "openalex:W1", Target: "openalex:W2", Weight: 1.0},
		},
	}

	buf.Reset()
	err = renderer.RenderAnalysis(&buf, analysis)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "## References")
	assert.Contains(t, buf.String(), "Referenced Paper")
}

func TestMarkdownRenderer_ReferenceFormat(t *testing.T) {
	year := 2024
	doi := "10.1234/test"
	workWithDOI := &models.Work{
		Rich:    true,
		Authors: []string{"Smith, J."},
		Title:   new("A Study on Citations"),
		Venue:   new("Nature"),
		Year:    &year,
		DOI:     &doi,
	}

	renderer, err := renderers.NewMarkdownRenderer("reference")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = renderer.Render(&buf, workWithDOI, "reference")
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "\n.\n")
	assert.Contains(t, output, "doi: 10.1234/test.")

	// Test without DOI
	workWithoutDOI := &models.Work{
		Rich:    true,
		Authors: []string{"Doe, A."},
		Title:   new("Another Study"),
		Venue:   new("Science"),
		Year:    &year,
	}
	buf.Reset()
	err = renderer.Render(&buf, workWithoutDOI, "reference")
	require.NoError(t, err)

	outputNoDOI := buf.String()
	assert.NotContains(t, outputNoDOI, "\n.\n")
	assert.Contains(t, outputNoDOI, ", 2024.")
}
