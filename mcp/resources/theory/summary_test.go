package theory_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/mcp/resources/theory"
	mcptest "github.com/coreofscience/go-bibx/mcp/test"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummaryResource(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)

	summaryResource := theory.NewSummary()
	summaryResource.Register(server)

	client := mcptest.Connect(t, server)

	t.Run("list resource", func(t *testing.T) {
		resList, err := client.ListResources(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, resList)
		require.Len(t, resList.Resources, 1)
		assert.Equal(t, "bibx:/summary", resList.Resources[0].URI)
		assert.Equal(t, "text/plain", resList.Resources[0].MIMEType)
	})

	t.Run("read resource successfully", func(t *testing.T) {
		res, err := client.ReadResource(t.Context(), &mcp.ReadResourceParams{
			URI: "bibx:/summary",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Contents, 1)

		content := res.Contents[0]
		assert.Equal(t, "bibx:/summary", content.URI)
		assert.Equal(t, "text/plain", content.MIMEType)
		assert.Contains(t, content.Text, "Tree of Science (ToS)")
		assert.Contains(t, content.Text, "root")
		assert.Contains(t, content.Text, "trunk")
		assert.Contains(t, content.Text, "leaf")
	})

	t.Run("invalid URI scheme returns error", func(t *testing.T) {
		_, err := client.ReadResource(t.Context(), &mcp.ReadResourceParams{
			URI: "file:///summary",
		})
		require.Error(t, err)
	})
}
