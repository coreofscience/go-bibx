package collections

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreofscience/go-bibx/internal/mcp/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ShowInput represents the input parameters for the Show tool.
type ShowInput struct {
	// Target is the collection ID (UUID) or name alias. If omitted, the latest collection is returned.
	Target string `json:"target,omitempty" jsonschema:"Collection UUID or name alias. Omit to inspect the most recently updated collection."`
}

// ShowOutput contains the collection details.
type ShowOutput struct {
	Collection *models.CollectionSummary `json:"collection" jsonschema:"Metadata of the resolved bibliographic collection"`
}

// Show holds the show tool.
type Show struct {
	service services.CollectionService
}

// NewShow creates a [Show] instance.
func NewShow(service services.CollectionService) *Show {
	return &Show{
		service: service,
	}
}

// Tool creates the [mcp.Tool] definition for the [Show] tool.
func (s *Show) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
			ReadOnlyHint:    true,
			Title:           "show collection",
		},
		Description: "Show metadata for a bibliographic collection by UUID or name alias (defaults to the latest collection if target is omitted).",
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

	collection, err := s.service.Resolve(ctx, target)
	if err != nil {
		if errors.Is(err, services.ErrCollectionNotFound) {
			if target == "" {
				return nil, nil, errors.New("no collections found")
			}
			return nil, nil, fmt.Errorf("collection not found: %s", target)
		}
		return nil, nil, fmt.Errorf("failed to resolve collection: %w", err)
	}

	return &mcp.CallToolResult{IsError: false}, &ShowOutput{
		Collection: models.NewCollectionSummaryFromModel(collection),
	}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (s *Show) Register(server *mcp.Server) {
	mcp.AddTool(server, s.Tool(), s.Handle)
}
