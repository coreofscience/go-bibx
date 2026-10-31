package collections

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coreofscience/go-bibx/internal/mcp/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteInput represents the input parameters for the Delete tool.
type DeleteInput struct {
	// Target is the collection ID (UUID) or name alias. Unlike other tools it is required.
	Target string `json:"target" jsonschema:"Collection UUID or name alias to delete. Required; there is no default to the latest collection."`
}

// DeleteOutput contains the details of the deleted collection.
type DeleteOutput struct {
	Collection *models.CollectionSummary `json:"collection" jsonschema:"Metadata of the collection that was deleted"`
}

// Delete holds the delete tool.
type Delete struct {
	service services.CollectionService
}

// NewDelete creates a [Delete] instance.
func NewDelete(service services.CollectionService) *Delete {
	return &Delete{
		service: service,
	}
}

// Tool creates the [mcp.Tool] definition for the [Delete] tool.
func (d *Delete) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(true),
			IdempotentHint:  false,
			OpenWorldHint:   new(false),
			ReadOnlyHint:    false,
			Title:           "delete collection",
		},
		Description: "Permanently delete a stored bibliographic collection, with its works and citation graph, by UUID or name alias. Use it to remove throwaway or failed collections; it cannot be undone.",
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
	target := ""
	if input != nil {
		target = strings.TrimSpace(input.Target)
	}
	if target == "" {
		return nil, nil, errors.New("target is required")
	}

	collection, err := d.service.Resolve(ctx, target)
	if err != nil {
		if errors.Is(err, services.ErrCollectionNotFound) {
			return nil, nil, fmt.Errorf("collection not found: %s", target)
		}
		return nil, nil, fmt.Errorf("failed to resolve collection: %w", err)
	}

	if err := d.service.Delete(ctx, collection.ID); err != nil {
		return nil, nil, fmt.Errorf("failed to delete collection: %w", err)
	}

	return &mcp.CallToolResult{IsError: false}, &DeleteOutput{
		Collection: models.NewCollectionSummaryFromModel(collection),
	}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (d *Delete) Register(server *mcp.Server) {
	mcp.AddTool(server, d.Tool(), d.Handle)
}
