package resources

import (
	"github.com/coreofscience/go-bibx/mcp/resources/collections"
	"github.com/coreofscience/go-bibx/mcp/resources/theory"
	"github.com/coreofscience/go-bibx/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register all the resources in the given mcp server
func Register(server *mcp.Server, store storage.Store) {
	collections.NewSummary(store).Register(server)
	theory.NewSummary().Register(server)
}
