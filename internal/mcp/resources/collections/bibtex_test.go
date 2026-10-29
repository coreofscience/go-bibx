package collections_test

import (
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/internal/mcp/resources/collections"
	mcptest "github.com/coreofscience/go-bibx/internal/mcp/test"
	servicesmocks "github.com/coreofscience/go-bibx/internal/mocks/services"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCollectionBibtexResource(t *testing.T) {
	mockColSvc := servicesmocks.NewMockCollectionService(t)
	mockAnaSvc := servicesmocks.NewMockAnalysisService(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	bibtexResource := collections.NewBibtex(mockColSvc, mockAnaSvc)
	bibtexResource.Register(server)

	client := mcptest.Connect(t, server)

	colID := uuid.NewV7()
	colName := "bpm-collection"
	testCol := &services.Collection{
		ID:           colID,
		QueryHash:    services.ComputeQueryHash("bit-patterned media", 100),
		Query:        "bit-patterned media",
		ResultLimit:  100,
		Name:         &colName,
		ArticleCount: 1,
		CreatedAt:    time.Now().UTC().Truncate(time.Second),
		UpdatedAt:    time.Now().UTC().Truncate(time.Second),
	}

	year := 2021
	title := "Advances in Bit-Patterned Media"
	journal := "IEEE Transactions on Magnetics"
	testArticles := []*models.Article{
		{
			Label:   "openalex:W100",
			Authors: []string{"Albrecht, Manfred"},
			Year:    &year,
			Title:   &title,
			Journal: &journal,
		},
	}

	t.Run("list resource template", func(t *testing.T) {
		tmplList, err := client.ListResourceTemplates(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, tmplList)
		require.Len(t, tmplList.ResourceTemplates, 1)
		assert.Equal(t, "bibx:/collections/{id}/bibtex", tmplList.ResourceTemplates[0].URITemplate)
		assert.Equal(t, "text/x-bibtex", tmplList.ResourceTemplates[0].MIMEType)
		assert.Equal(t, "Collection BibTeX", tmplList.ResourceTemplates[0].Title)
		assert.NotEmpty(t, tmplList.ResourceTemplates[0].Description)
		require.NotNil(t, tmplList.ResourceTemplates[0].Annotations)
		assert.Equal(t, 0.5, tmplList.ResourceTemplates[0].Annotations.Priority)
	})

	t.Run("read collection bibtex resource successfully", func(t *testing.T) {
		mockColSvc.EXPECT().
			FindByID(mock.Anything, colID).
			Return(testCol, nil).
			Once()

		mockAnaSvc.EXPECT().
			GetArticles(mock.Anything, colID, ([]string)(nil)).
			Return(testArticles, nil, nil).
			Once()

		res, err := client.ReadResource(t.Context(), &mcp.ReadResourceParams{
			URI: fmt.Sprintf("bibx:/collections/%s/bibtex", colID),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Contents, 1)

		content := res.Contents[0]
		assert.Equal(t, fmt.Sprintf("bibx:/collections/%s/bibtex", colID), content.URI)
		assert.Equal(t, "text/x-bibtex", content.MIMEType)
		assert.Contains(t, content.Text, "@article{albrecht2021,")
		assert.Contains(t, content.Text, "Advances in Bit-Patterned Media")
	})

	t.Run("collection not found returns error", func(t *testing.T) {
		missingID := uuid.NewV7()
		mockColSvc.EXPECT().
			FindByID(mock.Anything, missingID).
			Return(nil, services.ErrCollectionNotFound).
			Once()

		_, err := client.ReadResource(t.Context(), &mcp.ReadResourceParams{
			URI: fmt.Sprintf("bibx:/collections/%s/bibtex", missingID),
		})
		require.Error(t, err)
	})

	t.Run("invalid UUID in URI returns error", func(t *testing.T) {
		_, err := client.ReadResource(t.Context(), &mcp.ReadResourceParams{
			URI: "bibx:/collections/not-a-valid-uuid/bibtex",
		})
		require.Error(t, err)
	})
}
