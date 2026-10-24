package collections

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coreofscience/go-bibx/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteInput represents the input parameters for the Delete tool.
type DeleteInput struct {
	// Target is the collection ID (UUID) or name alias to delete.
	Target string `json:"target" jsonschema:"required,description=collection ID or name alias to delete"`
}

// DeleteOutput contains the result of the deletion.
type DeleteOutput struct {
	DeletedID   string  `json:"deleted_id"`
	DeletedName *string `json:"deleted_name,omitempty"`
}

// Delete holds the delete tool.
type Delete struct {
	store storage.Store
}

// NewDelete creates a [Delete] instance.
func NewDelete(store storage.Store) *Delete {
	return &Delete{
		store: store,
	}
}

// Tool creates the [mcp.Tool] definition for the [Delete] tool.
func (d *Delete) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(true),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
			ReadOnlyHint:    false,
			Title:           "delete collection",
		},
		Description: "delete a collection and its associated nodes and links by ID or name",
		Name:        "delete",
		Title:       "Delete collection",
	}
}

// Handle handles the incoming tool call requests.
func (d *Delete) Handle(
	ctx context.Context,
	request *mcp.CallToolRequest,
	input *DeleteInput,
) (*mcp.CallToolResult, *DeleteOutput, error) {
	if input == nil || strings.TrimSpace(input.Target) == "" {
		return nil, nil, errors.New("target is required")
	}

	col, err := storage.ResolveCollection(ctx, d.store, input.Target)
	if err != nil {
		if errors.Is(err, storage.ErrCollectionNotFound) {
			return nil, nil, fmt.Errorf("collection not found: %s", input.Target)
		}
		return nil, nil, fmt.Errorf("failed to resolve collection: %w", err)
	}

	if err := d.store.DeleteCollection(ctx, col.ID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete collection: %w", err)
	}

	return &mcp.CallToolResult{IsError: false}, &DeleteOutput{
		DeletedID:   col.ID.String(),
		DeletedName: col.Name,
	}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (d *Delete) Register(server *mcp.Server) {
	mcp.AddTool(server, d.Tool(), d.Handle)
}
