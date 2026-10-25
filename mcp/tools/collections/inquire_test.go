package collections_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/cli/services"
	collectionsmocks "github.com/coreofscience/go-bibx/internal/mocks/mcp/tools/collections"
	mcptest "github.com/coreofscience/go-bibx/mcp/test"
	"github.com/coreofscience/go-bibx/mcp/tools/collections"
	"github.com/coreofscience/go-bibx/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInquireTool(t *testing.T) {
	mockService := collectionsmocks.NewMockInquireService(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	inquireTool := collections.NewInquireWithService(mockService)
	inquireTool.Register(server)

	client := mcptest.Connect(t, server)

	colID := uuid.NewV7()
	testCol := &storage.Collection{
		ID:           colID,
		Query:        "graph algorithms",
		ResultLimit:  200,
		ArticleCount: 30,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	t.Run("lists tool metadata", func(t *testing.T) {
		res, err := client.ListTools(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tools, 1)
		assert.Equal(t, "inquire", res.Tools[0].Name)
		assert.Equal(t, "Inquire collection", res.Tools[0].Title)
	})

	t.Run("successful inquiry", func(t *testing.T) {
		mockService.EXPECT().
			StoreWithOptions(mock.Anything, "graph algorithms", 200, mock.Anything).
			Return(&services.StoreResult{
				Collection: testCol,
				IsNew:      true,
			}, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "inquire",
			Arguments: map[string]any{
				"query": "graph algorithms",
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out collections.InquireOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.True(t, out.IsNew)
		require.NotNil(t, out.Collection)
		assert.Equal(t, colID.String(), out.Collection.ID)
		assert.Equal(t, "graph algorithms", out.Collection.Query)
	})

	t.Run("missing query returns error", func(t *testing.T) {
		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "inquire",
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
			Name: "inquire",
			Arguments: map[string]any{
				"query": "graph algorithms",
				"limit": 1001,
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "limit must be less than or equal to 1000")
	})

	t.Run("service failure returns error", func(t *testing.T) {
		mockService.EXPECT().
			StoreWithOptions(mock.Anything, "failing topic", 200, mock.Anything).
			Return(nil, errors.New("upstream API error")).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "inquire",
			Arguments: map[string]any{
				"query": "failing topic",
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "upstream API error")
	})
}
