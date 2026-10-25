package collections

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreofscience/go-bibx/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ShowInput represents the input parameters for the Show tool.
type ShowInput struct {
	// Target is the collection ID (UUID) or name alias. If omitted, the latest collection is returned.
	Target string `json:"target,omitempty"`
}

// ShowOutput contains the collection details.
type ShowOutput struct {
	Collection *CollectionSummary `json:"collection"`
}

// Show holds the show tool.
type Show struct {
	store storage.Store
}

// NewShow creates a [Show] instance.
func NewShow(store storage.Store) *Show {
	return &Show{
		store: store,
	}
}

// Tool creates the [mcp.Tool] definition for the [Show] tool.
func (s *Show) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(true),
			ReadOnlyHint:    true,
			Title:           "show collection",
		},
		Description: "show details of a bibliographic collection by ID or name (or latest collection)",
		Name:        "show",
		Title:       "Show collection",
	}
}

// Handle handles the incoming tool call requests.
func (s *Show) Handle(
	ctx context.Context,
	request *mcp.CallToolRequest,
	input *ShowInput,
) (*mcp.CallToolResult, *ShowOutput, error) {
	target := ""
	if input != nil {
		target = input.Target
	}

	collection, err := storage.ResolveCollection(ctx, s.store, target)
	if err != nil {
		if errors.Is(err, storage.ErrCollectionNotFound) {
			if target == "" {
				return nil, nil, errors.New("no collections found")
			}
			return nil, nil, fmt.Errorf("collection not found: %s", target)
		}
		return nil, nil, fmt.Errorf("failed to resolve collection: %w", err)
	}

	return &mcp.CallToolResult{IsError: false}, &ShowOutput{
		Collection: NewCollectionSummaryFromModel(collection),
	}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (s *Show) Register(server *mcp.Server) {
	mcp.AddTool(server, s.Tool(), s.Handle)
}
