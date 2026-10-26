package analysis_test

import (
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	analysismocks "github.com/coreofscience/go-bibx/internal/mocks/mcp/tools/analysis"
	storagemocks "github.com/coreofscience/go-bibx/internal/mocks/storage"
	mcptest "github.com/coreofscience/go-bibx/mcp/test"
	"github.com/coreofscience/go-bibx/mcp/tools/analysis"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTosTool(t *testing.T) {
	mockService := analysismocks.NewMockTosService(t)
	mockStore := storagemocks.NewMockStore(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	tosTool := analysis.NewTosWithService(mockService, mockStore)
	tosTool.Register(server)

	client := mcptest.Connect(t, server)

	colID := uuid.NewV7()
	testCol := &storage.Collection{
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
		mockStore.EXPECT().
			FindCollectionByID(mock.Anything, colID).
			Return(testCol, nil).
			Once()

		mockService.On("Query", mock.Anything, "root", 10, mock.Anything).
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
		mockStore.EXPECT().
			ListCollections(mock.Anything).
			Return([]*storage.Collection{testCol}, nil).
			Once()

		mockService.On("Query", mock.Anything, "trunk", 25, mock.Anything).
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
		mockStore.EXPECT().
			ListCollections(mock.Anything).
			Return([]*storage.Collection{testCol}, nil).
			Once()

		mockService.On("Query", mock.Anything, "leaf", 10, mock.Anything).
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
