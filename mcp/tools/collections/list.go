package collections

import (
	"context"
	"fmt"

	"github.com/coreofscience/go-bibx/mcp/models"
	"github.com/coreofscience/go-bibx/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListInput is intentionally empty.
//
// In the future this one might contain cursors and limits.
type ListInput struct{}

// ListOutput contains the collection we want to show to the user.
type ListOutput struct {
	Collections []*models.CollectionSummary `json:"collections"`
}

// List holds the list command.
type List struct {
	store storage.Store
}

// NewList creates a [List] instance.
func NewList(store storage.Store) *List {
	return &List{
		store: store,
	}
}

// Tool creates the [mcp.Tool] definition for the [List] tool.
func (l *List) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(true),
			ReadOnlyHint:    true,
			Title:           "list collections",
		},
		Description: "list collections in your bibliography database",
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
	collections, err := l.store.ListCollections(ctx)
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
