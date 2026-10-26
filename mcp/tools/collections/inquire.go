package collections

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coreofscience/go-bibx/clients/openalex"
	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/mcp/models"
	"github.com/coreofscience/go-bibx/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultLimit is the default number of works to fetch if not specified.
const DefaultLimit = 200

// InquireInput represents the input parameters for the Inquire tool.
type InquireInput struct {
	// Query is the search query for the collection.
	Query string `json:"query"`
	// Limit is the number of initial results to fetch (default: 200, max: 1000).
	Limit int `json:"limit,omitempty"`
	// Force forces re-fetching and recalculation of existing query results.
	Force bool `json:"force,omitempty"`
	// Name is an optional name alias for the collection.
	Name string `json:"name,omitempty"`
}

// InquireOutput contains the collection details and whether it was newly queried.
type InquireOutput struct {
	Collection *models.CollectionSummary `json:"collection"`
	IsNew      bool                      `json:"is_new"`
}

// InquireService defines the interface needed by the Inquire tool.
type InquireService interface {
	StoreWithOptions(
		ctx context.Context,
		query string,
		limit int,
		opts services.StoreOptions,
	) (*services.StoreResult, error)
}

// Inquire holds the inquire tool.
type Inquire struct {
	service InquireService
}

// NewInquire creates an [Inquire] instance.
func NewInquire(store storage.Store) *Inquire {
	return NewInquireWithService(
		services.NewOpenAlexAnalysisService(openalex.NewRestyClient(), store),
	)
}

// NewInquireWithService creates an [Inquire] instance with a custom [InquireService].
func NewInquireWithService(service InquireService) *Inquire {
	return &Inquire{
		service: service,
	}
}

// Tool creates the [mcp.Tool] definition for the [Inquire] tool.
func (i *Inquire) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(true),
			ReadOnlyHint:    false,
			Title:           "inquire collection",
		},
		Description: "fetch literature from OpenAlex for a research topic, run network algorithms (SAP, Leiden), and store the resulting collection",
		Name:        "inquire",
		Title:       "Inquire collection",
	}
}

// Handle handles the incoming tool call requests.
func (i *Inquire) Handle(
	ctx context.Context,
	request *mcp.CallToolRequest,
	input *InquireInput,
) (*mcp.CallToolResult, *InquireOutput, error) {
	if input == nil {
		return nil, nil, errors.New("query is required")
	}

	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, nil, errors.New("query is required")
	}

	limit := DefaultLimit
	if input.Limit != 0 {
		if input.Limit < 0 {
			return nil, nil, errors.New("limit must be greater than 0")
		}
		if input.Limit > 1000 {
			return nil, nil, errors.New("limit must be less than or equal to 1000")
		}
		limit = input.Limit
	}

	res, err := i.service.StoreWithOptions(ctx, query, limit, services.StoreOptions{
		Force: input.Force,
		Name:  input.Name,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to inquire collection: %w", err)
	}

	return &mcp.CallToolResult{IsError: false}, &InquireOutput{
		Collection: models.NewCollectionSummaryFromModel(res.Collection),
		IsNew:      res.IsNew,
	}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (i *Inquire) Register(server *mcp.Server) {
	mcp.AddTool(server, i.Tool(), i.Handle)
}
