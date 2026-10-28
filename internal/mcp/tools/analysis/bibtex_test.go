package analysis_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"uuid"

	mcptest "github.com/coreofscience/go-bibx/internal/mcp/test"
	"github.com/coreofscience/go-bibx/internal/mcp/tools/analysis"
	servicesmocks "github.com/coreofscience/go-bibx/internal/mocks/services"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nickng/bibtex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBibtexTool(t *testing.T) {
	mockAnaSvc := servicesmocks.NewMockAnalysisService(t)
	mockColSvc := servicesmocks.NewMockCollectionService(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	bibtexTool := analysis.NewBibtex(mockAnaSvc, mockColSvc)
	bibtexTool.Register(server)

	client := mcptest.Connect(t, server)

	colID := uuid.NewV7()
	testCol := &services.Collection{
		ID:    colID,
		Query: "graph theory",
	}

	year := 2017
	paperTitle := "Foundational Network Paper"
	journal := "Network Science"
	art := &models.Article{
		Label:   "openalex:W1",
		Title:   &paperTitle,
		Journal: &journal,
		Authors: []string{"Vaswani, Ashish"},
		Year:    &year,
	}

	t.Run("lists tool metadata", func(t *testing.T) {
		res, err := client.ListTools(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tools, 1)
		assert.Equal(t, "bibtex", res.Tools[0].Name)
		assert.Equal(t, "Export BibTeX", res.Tools[0].Title)
	})

	t.Run("exports selected articles with target collection", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, colID.String()).
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			GetArticles(mock.Anything, colID, []string{"openalex:W1"}).
			Return([]*models.Article{art}, nil, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"target": colID.String(),
				"ids":    []string{"openalex:W1"},
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.BibtexOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Equal(t, 1, out.Count)
		assert.Empty(t, out.NotFound)
		assert.Contains(t, out.Bibtex, "@article{vaswani2017,")
		assert.Contains(t, out.Bibtex, "Foundational Network Paper")

		parsed, err := bibtex.Parse(strings.NewReader(out.Bibtex))
		require.NoError(t, err)
		require.Len(t, parsed.Entries, 1)
	})

	t.Run("exports all articles when ids omitted", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			GetArticles(mock.Anything, colID, []string(nil)).
			Return([]*models.Article{art}, nil, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"target": "",
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.BibtexOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Equal(t, 1, out.Count)
		assert.Empty(t, out.NotFound)
		assert.Contains(t, out.Bibtex, "@article{vaswani2017,")
	})

	t.Run("reports not found IDs", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			GetArticles(mock.Anything, colID, []string{"openalex:W1", "unknown-xyz"}).
			Return([]*models.Article{art}, []string{"unknown-xyz"}, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"ids": []string{"openalex:W1", "unknown-xyz"},
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.BibtexOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, []string{"unknown-xyz"}, out.NotFound)
		assert.Contains(t, out.Bibtex, "@article{vaswani2017,")
	})

	t.Run("collection not found returns error", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "non-existent").
			Return(nil, services.ErrCollectionNotFound).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"target": "non-existent",
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "collection not found: non-existent")
	})

	t.Run("service failure returns error", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			GetArticles(mock.Anything, colID, []string{"W1"}).
			Return(nil, nil, errors.New("db query timeout")).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"ids": []string{"W1"},
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "failed to retrieve articles: db query timeout")
	})
}
