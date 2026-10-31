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

// WorksInput represents the input parameters for the Works tool.
type WorksInput struct {
	// IDs is the list of work IDs, DOIs, or labels to retrieve.
	IDs []string `json:"ids" jsonschema:"Work identifiers to retrieve (OpenAlex IDs like 'W...' or 'openalex:W...', DOIs like '10.1109/...', or labels like 'vaswani2017')"`
	// Target is the collection ID (UUID) or name alias. If omitted, the latest collection is queried.
	Target string `json:"target,omitempty" jsonschema:"Collection UUID or name alias. Omit to query the most recently updated collection."`
}

// WorksOutput contains the retrieved work metadata and any unresolved IDs.
type WorksOutput struct {
	// Works contains the full bibliographic metadata (including abstracts) for the matched works.
	Works []*mcpmodels.WorkSummary `json:"works" jsonschema:"Bibliographic metadata including abstracts for the matched works"`
	// NotFound contains any requested input IDs that could not be resolved in the collection.
	NotFound []string `json:"not_found,omitempty" jsonschema:"Requested input IDs that could not be found in the collection"`
}

// Works holds the works metadata retrieval tool.
type Works struct {
	analysisService   services.AnalysisService
	collectionService services.CollectionService
}

// NewWorks creates an [Works] instance.
func NewWorks(analysisService services.AnalysisService, collectionService services.CollectionService) *Works {
	return &Works{
		analysisService:   analysisService,
		collectionService: collectionService,
	}
}

// Tool creates the [mcp.Tool] definition for the [Works] tool.
func (w *Works) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
			ReadOnlyHint:    true,
			Title:           "get work metadata",
		},
		Description: "Retrieve full bibliographic metadata (including abstracts) for specific works in a collection by their OpenAlex IDs ('W...' or 'openalex:W...'), DOIs ('10.1109/...'), or labels ('vaswani2017').",
		Name:        "works",
		Title:       "Get Works",
	}
}

// Handle handles incoming tool call requests.
func (w *Works) Handle(
	ctx context.Context,
	request *mcp.CallToolRequest,
	input *WorksInput,
) (*mcp.CallToolResult, *WorksOutput, error) {
	if input == nil {
		return nil, nil, errors.New("ids is required")
	}

	cleanedIDs := make([]string, 0, len(input.IDs))
	for _, id := range input.IDs {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			cleanedIDs = append(cleanedIDs, trimmed)
		}
	}
	if len(cleanedIDs) == 0 {
		return nil, nil, errors.New("ids is required")
	}

	target := strings.TrimSpace(input.Target)
	col, err := w.collectionService.Resolve(ctx, target)
	if err != nil {
		if errors.Is(err, services.ErrCollectionNotFound) {
			if target == "" {
				return nil, nil, errors.New("no collections found")
			}
			return nil, nil, fmt.Errorf("collection not found: %s", target)
		}
		return nil, nil, fmt.Errorf("failed to resolve collection: %w", err)
	}

	rawWorks, notFound, err := w.analysisService.GetWorks(ctx, col.ID, cleanedIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve works: %w", err)
	}

	works := make([]*mcpmodels.WorkSummary, len(rawWorks))
	for i, work := range rawWorks {
		works[i] = mcpmodels.NewWorkSummaryFromModel(work)
	}

	return &mcp.CallToolResult{IsError: false}, &WorksOutput{
		Works:    works,
		NotFound: notFound,
	}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (w *Works) Register(server *mcp.Server) {
	mcp.AddTool(server, w.Tool(), w.Handle)
}
