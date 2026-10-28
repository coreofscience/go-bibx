package analysis_test

import (
	"encoding/json"
	"errors"
	"fmt"
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
	abstract := "Comprehensive overview of graph algorithms and network structure."
	timesCited := 150
	art := &models.Article{
		Label:      "openalex:W1",
		Title:      &paperTitle,
		Journal:    &journal,
		Authors:    []string{"Vaswani, Ashish", "Shazeer, Noam"},
		Year:       &year,
		Abstract:   &abstract,
		TimesCited: &timesCited,
	}

	t.Run("lists tool metadata", func(t *testing.T) {
		res, err := client.ListTools(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tools, 1)
		assert.Equal(t, "bibtex", res.Tools[0].Name)
		assert.Equal(t, "Export BibTeX", res.Tools[0].Title)
	})

	t.Run("exports selected articles with target collection and keys mapping", func(t *testing.T) {
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
		assert.Equal(t, fmt.Sprintf("bibx:/collections/%s/bibtex", colID), out.ResourceURI)
		assert.Contains(t, out.Bibtex, "@article{vaswani2017,")
		assert.Contains(t, out.Bibtex, "Foundational Network Paper")
		// Defaults to omitting abstract
		assert.NotContains(t, out.Bibtex, "abstract")

		require.Len(t, out.Keys, 1)
		assert.Equal(t, "openalex:W1", out.Keys[0].ID)
		assert.Equal(t, "vaswani2017", out.Keys[0].CiteKey)
		assert.Equal(t, "Foundational Network Paper", out.Keys[0].Title)
		assert.Equal(t, "Vaswani, Ashish", out.Keys[0].FirstAuthor)
		assert.Equal(t, &year, out.Keys[0].Year)

		parsed, err := bibtex.Parse(strings.NewReader(out.Bibtex))
		require.NoError(t, err)
		require.Len(t, parsed.Entries, 1)
	})

	t.Run("includes abstract when omit_abstracts is false", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			GetArticles(mock.Anything, colID, []string{"openalex:W1"}).
			Return([]*models.Article{art}, nil, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"ids":            []string{"openalex:W1"},
				"omit_abstracts": false,
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.BibtexOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Contains(t, out.Bibtex, "abstract")
		assert.Contains(t, out.Bibtex, "Comprehensive overview")
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

	t.Run("preset tos queries roots, trunks, and leaves", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		rootArt := &models.Article{Label: "root1", Authors: []string{"Root, Alan"}, Year: &year}
		trunkArt := &models.Article{Label: "trunk1", Authors: []string{"Trunk, Bob"}, Year: &year}
		leafArt := &models.Article{Label: "leaf1", Authors: []string{"Leaf, Charlie"}, Year: &year}

		mockAnaSvc.EXPECT().
			QueryCategory(mock.Anything, colID, "root", 10).
			Return([]*models.Result{{Score: 0.9, Article: rootArt}}, nil).
			Once()
		mockAnaSvc.EXPECT().
			QueryCategory(mock.Anything, colID, "trunk", 10).
			Return([]*models.Result{{Score: 0.8, Article: trunkArt}}, nil).
			Once()
		mockAnaSvc.EXPECT().
			QueryCategory(mock.Anything, colID, "leaf", 10).
			Return([]*models.Result{{Score: 0.7, Article: leafArt}}, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"preset": "tos",
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.BibtexOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Equal(t, 3, out.Count)
		require.Len(t, out.Keys, 3)
		assert.Equal(t, "root2017", out.Keys[0].CiteKey)
		assert.Equal(t, "trunk2017", out.Keys[1].CiteKey)
		assert.Equal(t, "leaf2017", out.Keys[2].CiteKey)
	})

	t.Run("preset roots queries roots only", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		rootArt := &models.Article{Label: "root1", Authors: []string{"Root, Alan"}, Year: &year}
		mockAnaSvc.EXPECT().
			QueryCategory(mock.Anything, colID, "root", 5).
			Return([]*models.Result{{Score: 0.9, Article: rootArt}}, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"preset": "roots",
				"limit":  5,
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.BibtexOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Equal(t, 1, out.Count)
	})

	t.Run("preset top_cited sorts articles by TimesCited", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		tc10 := 10
		tc50 := 50
		tc30 := 30
		art1 := &models.Article{Label: "art1", Authors: []string{"A1"}, Year: &year, TimesCited: &tc10}
		art2 := &models.Article{Label: "art2", Authors: []string{"A2"}, Year: &year, TimesCited: &tc50}
		art3 := &models.Article{Label: "art3", Authors: []string{"A3"}, Year: &year, TimesCited: &tc30}

		mockAnaSvc.EXPECT().
			GetArticles(mock.Anything, colID, ([]string)(nil)).
			Return([]*models.Article{art1, art2, art3}, nil, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"preset": "top_cited",
				"limit":  2,
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.BibtexOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Equal(t, 2, out.Count)
		// Should be art2 (50 citations) followed by art3 (30 citations)
		assert.Equal(t, "art2", out.Keys[0].ID)
		assert.Equal(t, "art3", out.Keys[1].ID)
	})

	t.Run("cannot specify both preset and ids", func(t *testing.T) {
		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"preset": "tos",
				"ids":    []string{"openalex:W1"},
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "cannot specify both 'preset' and 'ids'")
	})

	t.Run("unknown preset returns error", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"preset": "invalid_preset",
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "unknown preset: \"invalid_preset\"")
	})

	t.Run("limit validation returns error when out of bounds", func(t *testing.T) {
		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"limit": -5,
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "limit must be greater than 0")

		res2, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "bibtex",
			Arguments: map[string]any{
				"limit": 1005,
			},
		})
		require.NoError(t, err)
		assert.True(t, res2.IsError)
		assert.Contains(t, res2.Content[0].(*mcp.TextContent).Text, "limit must be less than or equal to 1000")
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
