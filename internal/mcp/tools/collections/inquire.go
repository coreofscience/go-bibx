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

// DefaultLimit is the default number of works to fetch if not specified.
const DefaultLimit = 200

// InquireInput represents the input parameters for the Inquire tool.
type InquireInput struct {
	// Query is the search query for the collection.
	Query string `json:"query" jsonschema:"A single research concept of 1-3 words, e.g. 'stream processing'. Every term must appear in a work's title or abstract, so long multi-concept queries return almost nothing; inquire separate concepts into separate collections instead."`
	// Limit is the number of initial results to fetch (default: 200, max: 1000).
	Limit int `json:"limit,omitempty" jsonschema:"Maximum number of initial seed works to fetch from OpenAlex (1-1000, default: 200). For best results, try between 200 and 500."`
	// Force forces re-fetching and recalculation of existing query results.
	Force bool `json:"force,omitempty" jsonschema:"Re-fetch from OpenAlex and recompute the citation graph even if a cached collection exists for this query and limit"`
	// Name is an optional name alias for the collection.
	Name string `json:"name,omitempty" jsonschema:"Optional human-readable alias for referencing this collection in other tools"`
}

// InquireOutput contains the collection details and whether it was newly queried.
type InquireOutput struct {
	Collection *models.CollectionSummary `json:"collection" jsonschema:"Metadata of the created or cached bibliographic collection"`
	IsNew      bool                      `json:"is_new" jsonschema:"True if newly fetched and analyzed; false if an existing cached collection was returned"`
	Warnings   []string                  `json:"warnings,omitempty" jsonschema:"Problems with the result that may need a different query, such as far fewer matching works than requested"`
}

// lowSeedRatio is the fraction of the limit below which a newly fetched
// collection is reported as suspiciously small.
const lowSeedRatio = 0.5

// seedFilters describes the fixed OpenAlex filters applied by inquire.
const seedFilters = "works that have an abstract, at least one reference and more than one citation"

// inquireDescription tells the agent how queries are matched and what gets fetched.
const inquireDescription = "Fetch the OpenAlex works most relevant to a research topic, add the works they cite, " +
	"run Tree of Science (SAP) network analysis, and store the collection locally. " +
	"The query is matched against titles and abstracts and every term must match, so use one concept of 1-3 words per call. " +
	"Only " + seedFilters + " are fetched directly, whatever their type (journal articles, conference papers, reviews and so on). " +
	"Works without references, such as many books, enter the collection only when a fetched work cites them. " +
	"Results for the same query and limit are cached; pass force to re-fetch."

// Inquire holds the inquire tool.
type Inquire struct {
	service services.AnalysisService
}

// NewInquire creates an [Inquire] instance.
func NewInquire(service services.AnalysisService) *Inquire {
	return &Inquire{
		service: service,
	}
}

// Tool creates the [mcp.Tool] definition for the [Inquire] tool.
func (i *Inquire) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(true),
			ReadOnlyHint:    false,
			Title:           "inquire collection",
		},
		Description: inquireDescription,
		Name:        "inquire",
		Title:       "Inquire collection",
	}
}

// Handle handles the incoming tool call requests.
func (i *Inquire) Handle(
	ctx context.Context,
	request *mcp.CallToolRequest,
	input *InquireInput,
) (*mcp.CallToolResult, *InquireOutput, error) {
	if input == nil {
		return nil, nil, errors.New("query is required")
	}

	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, nil, errors.New("query is required")
	}

	limit := DefaultLimit
	if input.Limit != 0 {
		if input.Limit < 0 {
			return nil, nil, errors.New("limit must be greater than 0")
		}
		if input.Limit > 1000 {
			return nil, nil, errors.New("limit must be less than or equal to 1000")
		}
		limit = input.Limit
	}

	res, err := i.service.Inquire(ctx, query, limit, services.InquireOptions{
		Force: input.Force,
		Name:  input.Name,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to inquire collection: %w", err)
	}

	return &mcp.CallToolResult{IsError: false}, &InquireOutput{
		Collection: models.NewCollectionSummaryFromModel(res.Collection),
		IsNew:      res.IsNew,
		Warnings:   inquireWarnings(res, limit),
	}, nil
}

// inquireWarnings explains a newly fetched collection that is much smaller than requested.
func inquireWarnings(res *services.InquireResult, limit int) []string {
	if !res.IsNew || float64(res.SeedCount) >= float64(limit)*lowSeedRatio {
		return nil
	}
	return []string{fmt.Sprintf(
		"OpenAlex returned only %d of the %d requested works. Every query term must appear in the title or abstract, "+
			"and only %s are fetched. Try a shorter query with a single concept.",
		res.SeedCount, limit, seedFilters,
	)}
}

// Register registers the current tool to the given [mcp.Server].
func (i *Inquire) Register(server *mcp.Server) {
	mcp.AddTool(server, i.Tool(), i.Handle)
}
