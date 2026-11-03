package mcp_test

import (
	"testing"

	servicesmocks "github.com/coreofscience/go-bibx/internal/mocks/services"
	bibxmcp "github.com/coreofscience/go-bibx/mcp"
	mcptest "github.com/coreofscience/go-bibx/mcp/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	server := bibxmcp.NewServer(
		servicesmocks.NewMockCollectionService(t),
		servicesmocks.NewMockAnalysisService(t),
		bibxmcp.WithVersion("1.2.3"),
	)
	client := mcptest.Connect(t, server)

	init := client.InitializeResult()
	require.NotNil(t, init)
	assert.Equal(t, "bibx", init.ServerInfo.Name)
	assert.Equal(t, "1.2.3", init.ServerInfo.Version)
	assert.Contains(t, init.Instructions, "Tree of Science")

	tools, err := client.ListTools(t.Context(), nil)
	require.NoError(t, err)
	names := make([]string, len(tools.Tools))
	for i, tool := range tools.Tools {
		names[i] = tool.Name
	}
	assert.ElementsMatch(t, []string{
		"list", "show", "inquire", "delete", "tos", "search", "works", "bibtex",
	}, names)

	prompts, err := client.ListPrompts(t.Context(), nil)
	require.NoError(t, err)
	assert.Len(t, prompts.Prompts, 2)
}
