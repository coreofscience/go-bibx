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

// ArticlesInput represents the input parameters for the Articles tool.
type ArticlesInput struct {
	// IDs is the list of article IDs, DOIs, or labels to retrieve.
	IDs []string `json:"ids" jsonschema:"Article identifiers to retrieve (OpenAlex IDs like 'W...' or 'openalex:W...', DOIs like '10.1109/...', or labels like 'vaswani2017')"`
	// Target is the collection ID (UUID) or name alias. If omitted, the latest collection is queried.
	Target string `json:"target,omitempty" jsonschema:"Collection UUID or name alias. Omit to query the most recently updated collection."`
}

// ArticlesOutput contains the retrieved article metadata and any unresolved IDs.
type ArticlesOutput struct {
	// Articles contains the full bibliographic metadata (including abstracts) for the matched articles.
	Articles []*mcpmodels.ArticleSummary `json:"articles" jsonschema:"Bibliographic metadata including abstracts for the matched articles"`
	// NotFound contains any requested input IDs that could not be resolved in the collection.
	NotFound []string `json:"not_found,omitempty" jsonschema:"Requested input IDs that could not be found in the collection"`
}

// Articles holds the articles metadata retrieval tool.
type Articles struct {
	analysisService   services.AnalysisService
	collectionService services.CollectionService
}

// NewArticles creates an [Articles] instance.
func NewArticles(analysisService services.AnalysisService, collectionService services.CollectionService) *Articles {
	return &Articles{
		analysisService:   analysisService,
		collectionService: collectionService,
	}
}

// Tool creates the [mcp.Tool] definition for the [Articles] tool.
func (a *Articles) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
			ReadOnlyHint:    true,
			Title:           "get article metadata",
		},
		Description: "Retrieve full bibliographic metadata (including abstracts) for specific articles in a collection by their OpenAlex IDs ('W...' or 'openalex:W...'), DOIs ('10.1109/...'), or labels ('vaswani2017').",
		Name:        "articles",
		Title:       "Get Articles",
	}
}

// Handle handles incoming tool call requests.
func (a *Articles) Handle(
	ctx context.Context,
	request *mcp.CallToolRequest,
	input *ArticlesInput,
) (*mcp.CallToolResult, *ArticlesOutput, error) {
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
	col, err := a.collectionService.Resolve(ctx, target)
	if err != nil {
		if errors.Is(err, services.ErrCollectionNotFound) {
			if target == "" {
				return nil, nil, errors.New("no collections found")
			}
			return nil, nil, fmt.Errorf("collection not found: %s", target)
		}
		return nil, nil, fmt.Errorf("failed to resolve collection: %w", err)
	}

	rawArticles, notFound, err := a.analysisService.GetArticles(ctx, col.ID, cleanedIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve articles: %w", err)
	}

	articles := make([]*mcpmodels.ArticleSummary, len(rawArticles))
	for i, art := range rawArticles {
		articles[i] = mcpmodels.NewArticleSummaryFromModel(art)
	}

	return &mcp.CallToolResult{IsError: false}, &ArticlesOutput{
		Articles: articles,
		NotFound: notFound,
	}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (a *Articles) Register(server *mcp.Server) {
	mcp.AddTool(server, a.Tool(), a.Handle)
}
