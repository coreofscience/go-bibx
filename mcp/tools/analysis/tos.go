package analysis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/clients/openalex"
	mcpmodels "github.com/coreofscience/go-bibx/mcp/models"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/storage"
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
	Category string `json:"category"`
	// Limit is the number of results to return (default: 10, max: 1000).
	Limit int `json:"limit,omitempty"`
	// Target is the collection ID (UUID) or name alias. If omitted, the latest collection is queried.
	Target string `json:"target,omitempty"`
}

// TosOutput contains the results of the Tree of Science query.
type TosOutput struct {
	Results []*mcpmodels.Result `json:"results"`
}

// TosService defines the interface needed by the Tos tool.
type TosService interface {
	Query(
		ctx context.Context,
		category string,
		limit int,
		collectionID ...uuid.UUID,
	) ([]*models.Result, error)
}

// Tos holds the tos tool.
type Tos struct {
	service TosService
	store   storage.Store
}

// NewTos creates a [Tos] instance.
func NewTos(store storage.Store) *Tos {
	return NewTosWithService(
		services.NewOpenAlexAnalysisService(openalex.NewRestyClient(), store),
		store,
	)
}

// NewTosWithService creates a [Tos] instance with a custom [TosService].
func NewTosWithService(service TosService, store ...storage.Store) *Tos {
	var s storage.Store
	if len(store) > 0 {
		s = store[0]
	} else if backend, ok := service.(interface{ StoreBackend() storage.Store }); ok {
		s = backend.StoreBackend()
	}
	return &Tos{
		service: service,
		store:   s,
	}
}

// Tool creates the [mcp.Tool] definition for the [Tos] tool.
func (t *Tos) Tool() *mcp.Tool {
	return &mcp.Tool{
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(true),
			ReadOnlyHint:    true,
			Title:           "query Tree of Science categories",
		},
		Description: "query articles in a bibliographic collection by Tree of Science (ToS) category (root, trunk, or leaf)",
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
	var colID uuid.UUID
	if t.store != nil {
		col, err := storage.ResolveCollection(ctx, t.store, target)
		if err != nil {
			if errors.Is(err, storage.ErrCollectionNotFound) {
				if target == "" {
					return nil, nil, errors.New("no collections found")
				}
				return nil, nil, fmt.Errorf("collection not found: %s", target)
			}
			return nil, nil, fmt.Errorf("failed to resolve collection: %w", err)
		}
		colID = col.ID
	} else if target != "" {
		return nil, nil, errors.New("storage backend required to resolve collection target")
	}

	rawResults, err := t.service.Query(ctx, category, limit, colID)
	if err != nil {
		if errors.Is(err, storage.ErrCollectionNotFound) {
			if target == "" {
				return nil, nil, errors.New("no collections found")
			}
			return nil, nil, fmt.Errorf("collection not found: %s", target)
		}
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
