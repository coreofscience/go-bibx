package analysis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	mcpmodels "github.com/coreofscience/go-bibx/internal/mcp/models"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultLimit is the default number of articles to return if not specified.
const DefaultLimit = 10

// ValidCategories contains the supported Tree of Science categories.
var ValidCategories = map[string]struct{}{
	string(models.CategoryRoot):  {},
	string(models.CategoryTrunk): {},
	string(models.CategoryLeaf):  {},
}

// TosInput represents the input parameters for the Tos tool.
type TosInput struct {
	// Category is the Tree of Science category to query: "root", "trunk", or "leaf".
	Category string `json:"category" jsonschema:"Tree of Science category: 'root' (foundational/seminal papers), 'trunk' (structural bridge papers), or 'leaf' (modern state-of-the-art papers)"`
	// Limit is the number of results to return (default: 10, max: 1000).
	Limit int `json:"limit,omitempty" jsonschema:"Maximum number of top-scored articles to return (1-1000, default: 10). 10 to 20 is usually plenty, except for 'leaf' where up to 40 is also fine."`
	// Target is the collection ID (UUID) or name alias. If omitted, the latest collection is queried.
	Target string `json:"target,omitempty" jsonschema:"Collection UUID or name alias. Omit to query the most recently updated collection."`
}

// TosOutput contains the results of the Tree of Science query.
type TosOutput struct {
	Results []*mcpmodels.Result `json:"results" jsonschema:"Articles in the requested category ordered by descending score"`
}

// Tos holds the tos tool.
type Tos struct {
	analysisService   services.AnalysisService
	collectionService services.CollectionService
}

// NewTos creates a [Tos] instance.
func NewTos(analysisService services.AnalysisService, collectionService services.CollectionService) *Tos {
	return &Tos{
		analysisService:   analysisService,
		collectionService: collectionService,
	}
}

// Tool creates the [mcp.Tool] definition for the [Tos] tool.
func (t *Tos) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
			ReadOnlyHint:    true,
			Title:           "query Tree of Science categories",
		},
		Description: "Query top articles in a collection by Tree of Science (ToS) category ('root', 'trunk', or 'leaf'). Read resource bibx:/summary for methodology details.",
		Name:        "tos",
		Title:       "Tree of Science",
	}
}

// Handle handles the incoming tool call requests.
func (t *Tos) Handle(
	ctx context.Context,
	request *mcp.CallToolRequest,
	input *TosInput,
) (*mcp.CallToolResult, *TosOutput, error) {
	if input == nil {
		return nil, nil, errors.New("category is required")
	}

	category := strings.ToLower(strings.TrimSpace(input.Category))
	if category == "" {
		return nil, nil, errors.New("category is required")
	}

	if _, ok := ValidCategories[category]; !ok {
		return nil, nil, fmt.Errorf("invalid category %q, must be one of: root, trunk, leaf", input.Category)
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
	col, err := t.collectionService.Resolve(ctx, target)
	if err != nil {
		if errors.Is(err, services.ErrCollectionNotFound) {
			if target == "" {
				return nil, nil, errors.New("no collections found")
			}
			return nil, nil, fmt.Errorf("collection not found: %s", target)
		}
		return nil, nil, fmt.Errorf("failed to resolve collection: %w", err)
	}

	rawResults, err := t.analysisService.QueryCategory(ctx, col.ID, category, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query tos category: %w", err)
	}

	results := make([]*mcpmodels.Result, len(rawResults))
	for i, r := range rawResults {
		results[i] = mcpmodels.NewResultFromModel(r)
	}

	return &mcp.CallToolResult{IsError: false}, &TosOutput{
		Results: results,
	}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (t *Tos) Register(server *mcp.Server) {
	mcp.AddTool(server, t.Tool(), t.Handle)
}
