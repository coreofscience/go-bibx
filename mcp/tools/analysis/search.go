package analysis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/clients/openalex"
	mcpmodels "github.com/coreofscience/go-bibx/mcp/models"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/storage"
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

// SearchService defines the interface needed by the Search tool.
type SearchService interface {
	Search(
		ctx context.Context,
		query string,
		limit int,
		collectionID ...uuid.UUID,
	) (*models.Analysis, error)
}

// Search holds the search tool.
type Search struct {
	service SearchService
	store   storage.Store
}

// NewSearch creates a [Search] instance.
func NewSearch(store storage.Store) *Search {
	return NewSearchWithService(
		services.NewOpenAlexAnalysisService(openalex.NewRestyClient(), store),
		store,
	)
}

// NewSearchWithService creates a [Search] instance with a custom [SearchService].
func NewSearchWithService(service SearchService, store ...storage.Store) *Search {
	var s storage.Store
	if len(store) > 0 {
		s = store[0]
	} else if backend, ok := service.(interface{ StoreBackend() storage.Store }); ok {
		s = backend.StoreBackend()
	}
	return &Search{
		service: service,
		store:   s,
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
	var colID uuid.UUID
	if s.store != nil {
		col, err := storage.ResolveCollection(ctx, s.store, target)
		if err != nil {
			if errors.Is(err, storage.ErrCollectionNotFound) {
				if target == "" {
					return nil, nil, errors.New("no collections found")
				}
				return nil, nil, fmt.Errorf("collection not found: %s", target)
			}
			return nil, nil, fmt.Errorf("failed to resolve collection: %w", err)
		}
		colID = col.ID
	} else if target != "" {
		return nil, nil, errors.New("storage backend required to resolve collection target")
	}

	rawAnalysis, err := s.service.Search(ctx, query, limit, colID)
	if err != nil {
		if errors.Is(err, storage.ErrCollectionNotFound) {
			if target == "" {
				return nil, nil, errors.New("no collections found")
			}
			return nil, nil, fmt.Errorf("collection not found: %s", target)
		}
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
