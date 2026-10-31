package collections_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
	"uuid"

	mcptest "github.com/coreofscience/go-bibx/internal/mcp/test"
	"github.com/coreofscience/go-bibx/internal/mcp/tools/collections"
	servicesmocks "github.com/coreofscience/go-bibx/internal/mocks/services"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInquireTool(t *testing.T) {
	mockAnaSvc := servicesmocks.NewMockAnalysisService(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	inquireTool := collections.NewInquire(mockAnaSvc)
	inquireTool.Register(server)

	client := mcptest.Connect(t, server)

	colID := uuid.NewV7()
	testCol := &services.Collection{
		ID:          colID,
		Query:       "graph algorithms",
		ResultLimit: 200,
		WorkCount:   30,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	t.Run("lists tool metadata", func(t *testing.T) {
		res, err := client.ListTools(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tools, 1)
		assert.Equal(t, "inquire", res.Tools[0].Name)
		assert.Equal(t, "Inquire collection", res.Tools[0].Title)
		assert.True(t, *res.Tools[0].Annotations.OpenWorldHint)
		mcptest.AssertPropertyDescriptions(t, res.Tools[0].InputSchema, "query", "limit", "force", "name")
		mcptest.AssertPropertyDescriptions(t, res.Tools[0].OutputSchema, "collection", "is_new", "warnings")
	})

	t.Run("successful inquiry", func(t *testing.T) {
		mockAnaSvc.EXPECT().
			Inquire(mock.Anything, "graph algorithms", 200, mock.Anything).
			Return(&services.InquireResult{
				Collection: testCol,
				IsNew:      true,
				SeedCount:  200,
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
		assert.Empty(t, out.Warnings)
	})

	t.Run("few matching works returns a warning", func(t *testing.T) {
		mockAnaSvc.EXPECT().
			Inquire(mock.Anything, "nosql cap theorem consistency", 200, mock.Anything).
			Return(&services.InquireResult{
				Collection: testCol,
				IsNew:      true,
				SeedCount:  1,
			}, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "inquire",
			Arguments: map[string]any{
				"query": "nosql cap theorem consistency",
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out collections.InquireOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		require.Len(t, out.Warnings, 1)
		assert.Contains(t, out.Warnings[0], "only 1 of the 200 requested works")
	})

	t.Run("cached collection returns no warning", func(t *testing.T) {
		mockAnaSvc.EXPECT().
			Inquire(mock.Anything, "cached topic", 200, mock.Anything).
			Return(&services.InquireResult{
				Collection: testCol,
				IsNew:      false,
			}, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "inquire",
			Arguments: map[string]any{
				"query": "cached topic",
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out collections.InquireOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Empty(t, out.Warnings)
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

		resNeg, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "inquire",
			Arguments: map[string]any{
				"query": "graph algorithms",
				"limit": -1,
			},
		})
		require.NoError(t, err)
		assert.True(t, resNeg.IsError)
		assert.Contains(t, resNeg.Content[0].(*mcp.TextContent).Text, "limit must be greater than 0")
	})

	t.Run("service failure returns error", func(t *testing.T) {
		mockAnaSvc.EXPECT().
			Inquire(mock.Anything, "failing topic", 200, mock.Anything).
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
