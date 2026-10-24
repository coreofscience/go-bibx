package tools

import (
	"github.com/coreofscience/go-bibx/cli/mcp/tools/collections"
	"github.com/coreofscience/go-bibx/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register all the tools in the given mcp server
func Register(server *mcp.Server, store storage.Store) {
	collections.NewList(store).Register(server)
	collections.NewShow(store).Register(server)
}
