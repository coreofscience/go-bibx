package collections_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/internal/mcp/resources/collections"
	mcptest "github.com/coreofscience/go-bibx/internal/mcp/test"
	servicesmocks "github.com/coreofscience/go-bibx/internal/mocks/services"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCollectionSummaryResource(t *testing.T) {
	mockColSvc := servicesmocks.NewMockCollectionService(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	summaryResource := collections.NewSummary(mockColSvc)
	summaryResource.Register(server)

	client := mcptest.Connect(t, server)

	colID := uuid.NewV7()
	colName := "nlp-collection"
	testCol := &services.Collection{
		ID:          colID,
		QueryHash:   services.ComputeQueryHash("natural language processing", 100),
		Query:       "natural language processing",
		ResultLimit: 100,
		Name:        &colName,
		WorkCount:   25,
		CreatedAt:   time.Now().UTC().Truncate(time.Second),
		UpdatedAt:   time.Now().UTC().Truncate(time.Second),
	}

	t.Run("list resource template", func(t *testing.T) {
		tmplList, err := client.ListResourceTemplates(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, tmplList)
		require.Len(t, tmplList.ResourceTemplates, 1)
		assert.Equal(t, "bibx:/collections/{id}/summary", tmplList.ResourceTemplates[0].URITemplate)
		assert.Equal(t, "application/json", tmplList.ResourceTemplates[0].MIMEType)
		assert.Equal(t, "Collection Summary", tmplList.ResourceTemplates[0].Title)
		assert.NotEmpty(t, tmplList.ResourceTemplates[0].Description)
		require.NotNil(t, tmplList.ResourceTemplates[0].Annotations)
		assert.Equal(t, 0.7, tmplList.ResourceTemplates[0].Annotations.Priority)
	})

	t.Run("read collection summary resource successfully", func(t *testing.T) {
		mockColSvc.EXPECT().
			FindByID(mock.Anything, colID).
			Return(testCol, nil).
			Once()

		res, err := client.ReadResource(t.Context(), &mcp.ReadResourceParams{
			URI: fmt.Sprintf("bibx:/collections/%s/summary", colID),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Contents, 1)

		content := res.Contents[0]
		assert.Equal(t, fmt.Sprintf("bibx:/collections/%s/summary", colID), content.URI)
		assert.Equal(t, "application/json", content.MIMEType)

		var retrieved services.Collection
		err = json.Unmarshal([]byte(content.Text), &retrieved)
		require.NoError(t, err)
		assert.Equal(t, colID, retrieved.ID)
		assert.Equal(t, "natural language processing", retrieved.Query)
		assert.Equal(t, "nlp-collection", *retrieved.Name)
	})

	t.Run("collection not found returns error", func(t *testing.T) {
		missingID := uuid.NewV7()
		mockColSvc.EXPECT().
			FindByID(mock.Anything, missingID).
			Return(nil, services.ErrCollectionNotFound).
			Once()

		_, err := client.ReadResource(t.Context(), &mcp.ReadResourceParams{
			URI: fmt.Sprintf("bibx:/collections/%s/summary", missingID),
		})
		require.Error(t, err)
	})

	t.Run("invalid UUID in URI returns error", func(t *testing.T) {
		_, err := client.ReadResource(t.Context(), &mcp.ReadResourceParams{
			URI: "bibx:/collections/not-a-valid-uuid/summary",
		})
		require.Error(t, err)
	})
}
