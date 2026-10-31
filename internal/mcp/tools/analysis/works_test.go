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

func TestWorksTool(t *testing.T) {
	mockAnaSvc := servicesmocks.NewMockAnalysisService(t)
	mockColSvc := servicesmocks.NewMockCollectionService(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	worksTool := analysis.NewWorks(mockAnaSvc, mockColSvc)
	worksTool.Register(server)

	client := mcptest.Connect(t, server)

	colID := uuid.NewV7()
	testCol := &services.Collection{
		ID:    colID,
		Query: "graph theory",
	}

	year := 2017
	paperTitle := "Attention Is All You Need"
	venue := "NeurIPS"
	abstract := "The dominant sequence transduction models are based on complex recurrent or convolutional neural networks."
	timesCited := 120000
	work := &models.Work{
		Label:      "openalex:W1",
		Title:      &paperTitle,
		Venue:      &venue,
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
		assert.Equal(t, "works", res.Tools[0].Name)
		assert.Equal(t, "Get Works", res.Tools[0].Title)
		assert.False(t, *res.Tools[0].Annotations.OpenWorldHint)
		mcptest.AssertPropertyDescriptions(t, res.Tools[0].InputSchema, "ids", "target")
		mcptest.AssertPropertyDescriptions(t, res.Tools[0].OutputSchema, "works", "not_found")
	})

	t.Run("retrieves works including abstract with target collection", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, colID.String()).
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			GetWorks(mock.Anything, colID, []string{"openalex:W1"}).
			Return([]*models.Work{work}, nil, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "works",
			Arguments: map[string]any{
				"target": colID.String(),
				"ids":    []string{"openalex:W1"},
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.WorksOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		require.Len(t, out.Works, 1)
		assert.Empty(t, out.NotFound)
		assert.Equal(t, "openalex:W1", out.Works[0].Label)
		require.NotNil(t, out.Works[0].Title)
		assert.Equal(t, paperTitle, *out.Works[0].Title)
		require.NotNil(t, out.Works[0].Abstract)
		assert.Equal(t, abstract, *out.Works[0].Abstract)
		assert.Equal(t, []string{"Vaswani, Ashish", "Shazeer, Noam"}, out.Works[0].Authors)
	})

	t.Run("reports not found IDs", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "").
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			GetWorks(mock.Anything, colID, []string{"openalex:W1", "missing-id"}).
			Return([]*models.Work{work}, []string{"missing-id"}, nil).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "works",
			Arguments: map[string]any{
				"ids": []string{"openalex:W1", "missing-id"},
			},
		})
		require.NoError(t, err)
		assert.False(t, res.IsError)

		var out analysis.WorksOutput
		err = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
		require.NoError(t, err)
		require.Len(t, out.Works, 1)
		assert.Equal(t, []string{"missing-id"}, out.NotFound)
	})

	t.Run("empty ids returns error", func(t *testing.T) {
		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "works",
			Arguments: map[string]any{
				"ids": []string{"   "},
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "ids is required")
	})

	t.Run("collection not found returns error", func(t *testing.T) {
		mockColSvc.EXPECT().
			Resolve(mock.Anything, "non-existent").
			Return(nil, services.ErrCollectionNotFound).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "works",
			Arguments: map[string]any{
				"target": "non-existent",
				"ids":    []string{"openalex:W1"},
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
			GetWorks(mock.Anything, colID, []string{"W1"}).
			Return(nil, nil, errors.New("db query timeout")).
			Once()

		res, err := client.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "works",
			Arguments: map[string]any{
				"ids": []string{"W1"},
			},
		})
		require.NoError(t, err)
		assert.True(t, res.IsError)
		assert.Contains(t, res.Content[0].(*mcp.TextContent).Text, "failed to retrieve works: db query timeout")
	})
}
