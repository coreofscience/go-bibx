package collections

import (
	"context"
	"fmt"

	"github.com/coreofscience/go-bibx/internal/mcp/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListInput is intentionally empty.
//
// In the future this one might contain cursors and limits.
type ListInput struct{}

// ListOutput contains the collection we want to show to the user.
type ListOutput struct {
	Collections []*models.CollectionSummary `json:"collections" jsonschema:"Stored bibliographic collections ordered by most recently updated"`
}

// List holds the list command.
type List struct {
	service services.CollectionService
}

// NewList creates a [List] instance.
func NewList(service services.CollectionService) *List {
	return &List{
		service: service,
	}
}

// Tool creates the [mcp.Tool] definition for the [List] tool.
func (l *List) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
			ReadOnlyHint:    true,
			Title:           "list collections",
		},
		Description: "List stored bibliographic collections in the local database, ordered by most recently updated.",
		Name:        "list",
		Title:       "List collections",
	}
}

// Handle handles the incoming tool call requests.
func (l *List) Handle(
	ctx context.Context,
	request *mcp.CallToolRequest,
	input *ListInput,
) (*mcp.CallToolResult, *ListOutput, error) {
	collections, err := l.service.List(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list collections: %w", err)
	}
	summaries := make([]*models.CollectionSummary, len(collections))
	for i, collection := range collections {
		summaries[i] = models.NewCollectionSummaryFromModel(collection)
	}
	return &mcp.CallToolResult{IsError: false}, &ListOutput{Collections: summaries}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (l *List) Register(server *mcp.Server) {
	mcp.AddTool(server, l.Tool(), l.Handle)
}
