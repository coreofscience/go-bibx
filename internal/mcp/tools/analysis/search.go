package analysis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	mcpmodels "github.com/coreofscience/go-bibx/internal/mcp/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchInput represents the input parameters for the Search tool.
type SearchInput struct {
	// Query is the search query for the collection articles.
	Query string `json:"query"`
	// Limit is the number of initial nodes to start the graph algorithm with (default: 10, max: 1000).
	Limit int `json:"limit,omitempty"`
	// Target is the collection ID (UUID) or name alias. If omitted, the latest collection is queried.
	Target string `json:"target,omitempty"`
}

// SearchOutput contains the graph results of the search query.
type SearchOutput struct {
	Nodes []*mcpmodels.NodeSummary `json:"nodes"`
	Links []*mcpmodels.LinkSummary `json:"links"`
}

// Search holds the search tool.
type Search struct {
	analysisService   services.AnalysisService
	collectionService services.CollectionService
}

// NewSearch creates a [Search] instance.
func NewSearch(analysisService services.AnalysisService, collectionService services.CollectionService) *Search {
	return &Search{
		analysisService:   analysisService,
		collectionService: collectionService,
	}
}

// Tool creates the [mcp.Tool] definition for the [Search] tool.
func (s *Search) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(true),
			ReadOnlyHint:    true,
			Title:           "search collection",
		},
		Description: "search articles in a bibliographic collection using keyword matching and graph Steiner tree reduction",
		Name:        "search",
		Title:       "Search collection",
	}
}

// Handle handles the incoming tool call requests.
func (s *Search) Handle(
	ctx context.Context,
	request *mcp.CallToolRequest,
	input *SearchInput,
) (*mcp.CallToolResult, *SearchOutput, error) {
	if input == nil {
		return nil, nil, errors.New("query is required")
	}

	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, nil, errors.New("query is required")
	}

	limit := DefaultLimit
	if input.Limit != 0 {
		if input.Limit < 1 {
			return nil, nil, errors.New("limit must be greater than 0")
		}
		if input.Limit > 1000 {
			return nil, nil, errors.New("limit must be less than or equal to 1000")
		}
		limit = input.Limit
	}

	target := strings.TrimSpace(input.Target)
	col, err := s.collectionService.Resolve(ctx, target)
	if err != nil {
		if errors.Is(err, services.ErrCollectionNotFound) {
			if target == "" {
				return nil, nil, errors.New("no collections found")
			}
			return nil, nil, fmt.Errorf("collection not found: %s", target)
		}
		return nil, nil, fmt.Errorf("failed to resolve collection: %w", err)
	}

	rawAnalysis, err := s.analysisService.Search(ctx, col.ID, query, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search analysis: %w", err)
	}

	var nodes []*mcpmodels.NodeSummary
	var links []*mcpmodels.LinkSummary
	if rawAnalysis != nil {
		nodes = make([]*mcpmodels.NodeSummary, len(rawAnalysis.Nodes))
		for i, n := range rawAnalysis.Nodes {
			nodes[i] = mcpmodels.NewNodeSummaryFromModel(n)
		}
		links = make([]*mcpmodels.LinkSummary, len(rawAnalysis.Links))
		for i, l := range rawAnalysis.Links {
			links[i] = mcpmodels.NewLinkSummaryFromModel(l)
		}
	}

	return &mcp.CallToolResult{IsError: false}, &SearchOutput{
		Nodes: nodes,
		Links: links,
	}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (s *Search) Register(server *mcp.Server) {
	mcp.AddTool(server, s.Tool(), s.Handle)
}
