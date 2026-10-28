package analysis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// BibtexInput represents the input parameters for the Bibtex tool.
type BibtexInput struct {
	// IDs is the list of article IDs, DOIs, or labels to export. If omitted, exports all articles in the collection.
	IDs []string `json:"ids,omitempty"`
	// Target is the collection ID (UUID) or name alias. If omitted, the latest collection is queried.
	Target string `json:"target,omitempty"`
}

// BibtexOutput contains the results of the BibTeX export.
type BibtexOutput struct {
	// Bibtex contains the formatted BibTeX entries string.
	Bibtex string `json:"bibtex"`
	// Count is the number of articles successfully exported.
	Count int `json:"count"`
	// NotFound contains any input IDs that could not be resolved.
	NotFound []string `json:"not_found,omitempty"`
}

// Bibtex holds the bibtex export tool.
type Bibtex struct {
	analysisService   services.AnalysisService
	collectionService services.CollectionService
}

// NewBibtex creates a [Bibtex] instance.
func NewBibtex(analysisService services.AnalysisService, collectionService services.CollectionService) *Bibtex {
	return &Bibtex{
		analysisService:   analysisService,
		collectionService: collectionService,
	}
}

// Tool creates the [mcp.Tool] definition for the [Bibtex] tool.
func (b *Bibtex) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(true),
			ReadOnlyHint:    true,
			Title:           "export articles as BibTeX",
		},
		Description: "output selected articles from a collection in BibTeX format by their article IDs (or all articles if IDs are omitted)",
		Name:        "bibtex",
		Title:       "Export BibTeX",
	}
}

// Handle handles incoming tool call requests.
func (b *Bibtex) Handle(
	ctx context.Context,
	request *mcp.CallToolRequest,
	input *BibtexInput,
) (*mcp.CallToolResult, *BibtexOutput, error) {
	target := ""
	var ids []string
	if input != nil {
		target = strings.TrimSpace(input.Target)
		ids = input.IDs
	}

	col, err := b.collectionService.Resolve(ctx, target)
	if err != nil {
		if errors.Is(err, services.ErrCollectionNotFound) {
			if target == "" {
				return nil, nil, errors.New("no collections found")
			}
			return nil, nil, fmt.Errorf("collection not found: %s", target)
		}
		return nil, nil, fmt.Errorf("failed to resolve collection: %w", err)
	}

	articles, notFound, err := b.analysisService.GetArticles(ctx, col.ID, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve articles: %w", err)
	}

	bibtexStr := models.Articles(articles).ToBibTeX()

	return &mcp.CallToolResult{IsError: false}, &BibtexOutput{
		Bibtex:   bibtexStr,
		Count:    len(articles),
		NotFound: notFound,
	}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (b *Bibtex) Register(server *mcp.Server) {
	mcp.AddTool(server, b.Tool(), b.Handle)
}
