package collections

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Summary exposes a resource for a collection's summary.
type Summary struct {
	store storage.Store
}

// NewSummary builds a [Summary] instance.
func NewSummary(store storage.Store) *Summary {
	return &Summary{store: store}
}

// ResourceTemplate exports the [mcp.ResourceTemplate] required for a collection summary.
func (s *Summary) ResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Description: "description of a bibliographic collection",
		MIMEType:    "application/json",
		Name:        "collection summary",
		Title:       "collection summary",
		URITemplate: "bibx:/collections/{id}/summary",
	}
}

// Provide actually fetches and returns the requested resource.
func (s *Summary) Provide(ctx context.Context, rrr *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	u, err := url.Parse(rrr.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse provided resource URI: %w", err)
	}
	if u.Scheme != "bibx" {
		return nil, fmt.Errorf("URI scheme should be bibx, got: %q", u.Scheme)
	}
	id, err := uuid.Parse(u.Opaque)
	if err != nil {
		return nil, fmt.Errorf("the provided id could not be parsed as UUID, got: %q", u.Opaque)
	}
	collection, err := s.store.FindCollectionByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find collection: %w", err)
	}
	bytes, err := json.Marshal(collection)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection summary: %w", err)
	}
	return &mcp.ReadResourceResult{
		Cacheable: mcp.Cacheable{
			TTLMs:      int((60 * time.Minute).Milliseconds()),
			CacheScope: "private",
		},
		Contents: []*mcp.ResourceContents{{
			URI:      rrr.Params.URI,
			MIMEType: "application/json",
			Blob:     bytes,
		}},
	}, nil
}

// Register registers the resource template to the given [mcp.Server].
func (s *Summary) Register(server *mcp.Server) {
	server.AddResourceTemplate(s.ResourceTemplate(), s.Provide)
}
