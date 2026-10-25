package collections_test

import (
	"encoding/json"
	"testing"
	"time"
	"uuid"

	storagemocks "github.com/coreofscience/go-bibx/internal/mocks/storage"
	mcptest "github.com/coreofscience/go-bibx/mcp/test"
	"github.com/coreofscience/go-bibx/mcp/tools/collections"
	"github.com/coreofscience/go-bibx/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestShowTool(t *testing.T) {
	mockStore := storagemocks.NewMockStore(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	showTool := collections.NewShow(mockStore)
	showTool.Register(server)

	client := mcptest.Connect(t, server)

	colID := uuid.NewV7()
	alias := "transformers-study"
	testCol := &storage.Collection{
		ID:           colID,
		Query:        "attention is all you need",
		ResultLimit:  100,
		Name:         &alias,
		ArticleCount: 20,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	t.Run("lists tool metadata", func(t *testing.T) {
		res, err := client.ListTools(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tools, 1)
		assert.Equal(t, "show", res.Tools[0].Name)
		assert.Equal(t, "Show collection", res.Tools[0].Title)
	})

	t.Run("show by UUID target", func(t *testing.T) {
		mockStore.EXPECT().
			FindCollectionByID(mock.Anything, colID).
			Return(testCol, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "show",
			Arguments: map[string]any{
				"target": colID.String(),
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out collections.ShowOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Equal(t, colID.String(), out.Collection.ID)
		assert.Equal(t, "attention is all you need", out.Collection.Query)
	})

	t.Run("show by name alias", func(t *testing.T) {
		mockStore.EXPECT().
			FindCollectionByName(mock.Anything, "transformers-study").
			Return(testCol, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "show",
			Arguments: map[string]any{
				"target": "transformers-study",
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out collections.ShowOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Equal(t, colID.String(), out.Collection.ID)
	})

	t.Run("show empty target defaults to latest", func(t *testing.T) {
		mockStore.EXPECT().
			ListCollections(mock.Anything).
			Return([]*storage.Collection{testCol}, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "show",
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out collections.ShowOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Equal(t, colID.String(), out.Collection.ID)
	})

	t.Run("show not found returns error", func(t *testing.T) {
		mockStore.EXPECT().
			FindCollectionByName(mock.Anything, "missing-col").
			Return(nil, storage.ErrCollectionNotFound).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "show",
			Arguments: map[string]any{
				"target": "missing-col",
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "collection not found")
	})
}
