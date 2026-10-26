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

func TestTosTool(t *testing.T) {
	mockAnaSvc := servicesmocks.NewMockAnalysisService(t)
	mockColSvc := servicesmocks.NewMockCollectionService(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	tosTool := analysis.NewTos(mockAnaSvc, mockColSvc)
	tosTool.Register(server)

	client := mcptest.Connect(t, server)

	colID := uuid.NewV7()
	testCol := &services.Collection{
		ID:    colID,
		Query: "graph theory",
	}

	paperTitle := "Foundational Network Paper"
	mockResults := []*models.Result{
		{
			Score: 0.95,
			Article: &models.Article{
				Label: "openalex:W1",
				Title: &paperTitle,
			},
		},
	}

	t.Run("lists tool metadata", func(t *testing.T) {
		res, err := client.ListTools(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tools, 1)
		assert.Equal(t, "tos", res.Tools[0].Name)
		assert.Equal(t, "Tree of Science", res.Tools[0].Title)
	})

	t.Run("query root category with target collection", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, colID.String()).
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			QueryCategory(mock.Anything, colID, "root", 10).
			Return(mockResults, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "tos",
			Arguments: map[string]any{
				"category": "root",
				"target":   colID.String(),
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.TosOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		require.Len(t, out.Results, 1)
		assert.Equal(t, 0.95, out.Results[0].Score)
		assert.Equal(t, "Foundational Network Paper", *out.Results[0].Article.Title)
	})

	t.Run("query trunk category with custom limit", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			QueryCategory(mock.Anything, colID, "trunk", 25).
			Return(mockResults, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "tos",
			Arguments: map[string]any{
				"category": "TRUNK",
				"limit":    25,
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.TosOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		require.Len(t, out.Results, 1)
	})

	t.Run("invalid category returns error", func(t *testing.T) {
		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "tos",
			Arguments: map[string]any{
				"category": "invalid-category",
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "invalid category")
	})

	t.Run("missing category returns error", func(t *testing.T) {
		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "tos",
			Arguments: map[string]any{
				"category": "   ",
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "category is required")
	})

	t.Run("service failure returns error", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			QueryCategory(mock.Anything, colID, "leaf", 10).
			Return(nil, errors.New("db query timeout")).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "tos",
			Arguments: map[string]any{
				"category": "leaf",
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "db query timeout")
	})
}
