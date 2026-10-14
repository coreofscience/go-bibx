package renderers_test

import (
	"bytes"
	"testing"

	"github.com/coreofscience/go-bibx/cli/renderers"
	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownRenderer_SimpleFormat(t *testing.T) {
	tests := []struct {
		name     string
		article  *models.Article
		expected string
	}{
		{
			name: "article with permalink",
			article: &models.Article{
				Title:     new("Test Title"),
				Label:     "TestLabel",
				Permalink: new("https://example.com/article"),
				Keywords:  collections.NewSet("A", "B"),
				Abstract:  new("This is an abstract."),
			},
			expected: "# Test Title\n**link**: [https://example.com/article](https://example.com/article)\n\n**keywords**: A, B\n\n## Abstract\n\nThis is an abstract.\n",
		},
		{
			name: "article without permalink but with DOI",
			article: &models.Article{
				Title:    new("Test Title"),
				Label:    "TestLabel",
				DOI:      new("10.1000/xyz123"),
				Keywords: collections.NewSet("A", "B"),
				Abstract: new("This is an abstract."),
			},
			expected: "# Test Title\n**link**: [https://doi.org/10.1000/xyz123](https://doi.org/10.1000/xyz123)\n\n**keywords**: A, B\n\n## Abstract\n\nThis is an abstract.\n",
		},
		{
			name: "article without permalink or DOI",
			article: &models.Article{
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
			err = renderer.Render(&buf, tt.article, "simple")
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

	article := &models.Article{
		Title:    new("Test Title"),
		Label:    "TestLabel",
		Abstract: new("This is an abstract."),
	}

	var buf bytes.Buffer
	err = renderer.Render(&buf, article, "markdown")
	assert.NoError(t, err)
}
