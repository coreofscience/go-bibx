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

func TestListTool(t *testing.T) {
	mockColSvc := servicesmocks.NewMockCollectionService(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	listTool := collections.NewList(mockColSvc)
	listTool.Register(server)

	client := mcptest.Connect(t, server)

	t.Run("lists tool metadata", func(t *testing.T) {
		res, err := client.ListTools(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tools, 1)
		assert.Equal(t, "list", res.Tools[0].Name)
		assert.Equal(t, "List collections", res.Tools[0].Title)
	})

	t.Run("returns empty collection list", func(t *testing.T) {
		mockColSvc.EXPECT().
			List(mock.Anything).
			Return([]*services.Collection{}, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "list",
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)
		require.Len(t, res.Content, 1)

		var out collections.ListOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		assert.Empty(t, out.Collections)
	})

	t.Run("returns multiple collections", func(t *testing.T) {
		name1 := "bio-informatics"
		col1 := &services.Collection{
			ID:           uuid.NewV7(),
			Query:        "bioinformatics",
			ResultLimit:  100,
			Name:         &name1,
			ArticleCount: 15,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}
		col2 := &services.Collection{
			ID:           uuid.NewV7(),
			Query:        "crispr",
			ResultLimit:  50,
			ArticleCount: 8,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}

		mockColSvc.EXPECT().
			List(mock.Anything).
			Return([]*services.Collection{col1, col2}, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "list",
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out collections.ListOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		require.Len(t, out.Collections, 2)
		assert.Equal(t, col1.ID.String(), out.Collections[0].ID)
		assert.Equal(t, "bioinformatics", out.Collections[0].Query)
		assert.Equal(t, col2.ID.String(), out.Collections[1].ID)
	})

	t.Run("store failure returns error", func(t *testing.T) {
		mockColSvc.EXPECT().
			List(mock.Anything).
			Return(nil, errors.New("db disk failure")).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "list",
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "failed to list collections")
	})
}
