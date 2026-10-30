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

func TestDeleteTool(t *testing.T) {
	mockColSvc := servicesmocks.NewMockCollectionService(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	deleteTool := collections.NewDelete(mockColSvc)
	deleteTool.Register(server)

	client := mcptest.Connect(t, server)

	colID := uuid.NewV7()
	alias := "diss-verify-kafka"
	testCol := &services.Collection{
		ID:           colID,
		Query:        "kafka log processing",
		ResultLimit:  50,
		Name:         &alias,
		ArticleCount: 3,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	t.Run("lists tool metadata", func(t *testing.T) {
		res, err := client.ListTools(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tools, 1)
		assert.Equal(t, "delete", res.Tools[0].Name)
		assert.Equal(t, "Delete collection", res.Tools[0].Title)
		assert.True(t, *res.Tools[0].Annotations.DestructiveHint)
		assert.False(t, res.Tools[0].Annotations.ReadOnlyHint)
		assert.False(t, *res.Tools[0].Annotations.OpenWorldHint)
		mcptest.AssertPropertyDescriptions(t, res.Tools[0].InputSchema, "target")
		mcptest.AssertPropertyDescriptions(t, res.Tools[0].OutputSchema, "collection")
	})

	t.Run("delete by name alias", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, alias).
			Return(testCol, nil).
			Once()
		mockColSvc.EXPECT().
			Delete(mock.Anything, colID).
			Return(nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "delete",
			Arguments: map[string]any{
				"target": alias,
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out collections.DeleteOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Equal(t, colID.String(), out.Collection.ID)
	})

	t.Run("empty target returns error without deleting", func(t *testing.T) {
		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "delete",
			Arguments: map[string]any{
				"target": "  ",
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "target is required")
	})

	t.Run("not found returns error", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "missing-col").
			Return(nil, services.ErrCollectionNotFound).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "delete",
			Arguments: map[string]any{
				"target": "missing-col",
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "collection not found")
	})

	t.Run("service delete failure returns error", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, alias).
			Return(testCol, nil).
			Once()
		mockColSvc.EXPECT().
			Delete(mock.Anything, colID).
			Return(errors.New("db locked")).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "delete",
			Arguments: map[string]any{
				"target": alias,
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "failed to delete collection")
	})
}
