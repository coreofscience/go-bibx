package prompts

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers all available prompts in the given MCP server.
func Register(server *mcp.Server) {
	NewReview().Register(server)
	NewGaps().Register(server)
}
