package analysis_test

import (
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	mcptest "github.com/coreofscience/go-bibx/internal/mcp/test"
	"github.com/coreofscience/go-bibx/internal/mcp/tools/analysis"
	servicesmocks "github.com/coreofscience/go-bibx/internal/mocks/services"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSearchTool(t *testing.T) {
	mockAnaSvc := servicesmocks.NewMockAnalysisService(t)
	mockColSvc := servicesmocks.NewMockCollectionService(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	searchTool := analysis.NewSearch(mockAnaSvc, mockColSvc)
	searchTool.Register(server)

	client := mcptest.Connect(t, server)

	colID := uuid.NewV7()
	testCol := &services.Collection{
		ID:    colID,
		Query: "network science",
	}

	paperTitle := "Shortest Path Algorithms"
	mockAnalysis := &models.Analysis{
		Nodes: []*models.Node{
			{
				ID:        "openalex:W1",
				Category:  models.CategoryRoot,
				Rootness:  0.9,
				Trunkness: 0.1,
				Article: &models.Article{
					Label: "openalex:W1",
					Title: &paperTitle,
				},
			},
		},
		Links: []*models.Link{
			{
				Source: "openalex:W1",
				Target: "openalex:W1",
				Weight: 1.0,
			},
		},
	}

	t.Run("lists tool metadata", func(t *testing.T) {
		res, err := client.ListTools(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tools, 1)
		assert.Equal(t, "search", res.Tools[0].Name)
		assert.Equal(t, "Search collection", res.Tools[0].Title)
	})

	t.Run("successful search with target", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, colID.String()).
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			Search(mock.Anything, colID, "shortest path", 10).
			Return(mockAnalysis, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "search",
			Arguments: map[string]any{
				"query":  "shortest path",
				"target": colID.String(),
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.SearchOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		require.Len(t, out.Nodes, 1)
		assert.Equal(t, "openalex:W1", out.Nodes[0].ID)
		assert.Equal(t, "Shortest Path Algorithms", *out.Nodes[0].Article.Title)
		require.Len(t, out.Links, 1)
		assert.Equal(t, "openalex:W1", out.Links[0].Source)
	})

	t.Run("search empty target defaults to latest", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			Search(mock.Anything, colID, "graph", 20).
			Return(mockAnalysis, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "search",
			Arguments: map[string]any{
				"query": "graph",
				"limit": 20,
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.SearchOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		require.Len(t, out.Nodes, 1)
	})

	t.Run("missing query returns error", func(t *testing.T) {
		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "search",
			Arguments: map[string]any{
				"query": "   ",
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "query is required")
	})

	t.Run("limit out of bounds returns error", func(t *testing.T) {
		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "search",
			Arguments: map[string]any{
				"query": "graph",
				"limit": 1001,
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "limit must be less than or equal to 100")
	})

	t.Run("service failure returns error", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			Search(mock.Anything, colID, "failing query", 10).
			Return(nil, errors.New("search index corrupted")).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "search",
			Arguments: map[string]any{
				"query": "failing query",
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "search index corrupted")
	})
}
