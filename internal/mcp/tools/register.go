package tools

import (
	"github.com/coreofscience/go-bibx/internal/mcp/tools/analysis"
	"github.com/coreofscience/go-bibx/internal/mcp/tools/collections"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register all the tools in the given mcp server
func Register(server *mcp.Server, collectionService services.CollectionService, analysisService services.AnalysisService) {
	collections.NewList(collectionService).Register(server)
	collections.NewShow(collectionService).Register(server)
	collections.NewInquire(analysisService).Register(server)
	collections.NewDelete(collectionService).Register(server)
	analysis.NewTos(analysisService, collectionService).Register(server)
	analysis.NewSearch(analysisService, collectionService).Register(server)
	analysis.NewWorks(analysisService, collectionService).Register(server)
	analysis.NewBibtex(analysisService, collectionService).Register(server)
}
