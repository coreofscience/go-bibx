package resources

import (
	"github.com/coreofscience/go-bibx/mcp/internal/resources/collections"
	"github.com/coreofscience/go-bibx/mcp/internal/resources/theory"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register all the resources in the given mcp server
func Register(server *mcp.Server, collectionService services.CollectionService, analysisService services.AnalysisService) {
	collections.NewSummary(collectionService).Register(server)
	collections.NewBibtex(collectionService, analysisService).Register(server)
	theory.NewSummary().Register(server)
}
