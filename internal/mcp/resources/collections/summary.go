package collections

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/internal/mcp/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Summary exposes a resource for a collection's summary.
type Summary struct {
	service services.CollectionService
}

// NewSummary builds a [Summary] instance.
func NewSummary(service services.CollectionService) *Summary {
	return &Summary{service: service}
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
	raw := strings.TrimPrefix(rrr.Params.URI, "bibx:")
	raw = strings.TrimPrefix(raw, "/collections/")
	raw = strings.TrimSuffix(raw, "/summary")
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("the provided id could not be parsed as UUID, got: %q", raw)
	}
	collection, err := s.service.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find collection: %w", err)
	}
	bytes, err := json.Marshal(models.NewCollectionSummaryFromModel(collection))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection summary: %w", err)
	}
	return &mcp.ReadResourceResult{
		TTLMs:      int((60 * time.Minute).Milliseconds()),
		CacheScope: "private",
		Contents: []*mcp.ResourceContents{{
			URI:      rrr.Params.URI,
			MIMEType: "application/json",
			Text:     string(bytes),
		}},
	}, nil
}

// Register registers the resource template to the given [mcp.Server].
func (s *Summary) Register(server *mcp.Server) {
	server.AddResourceTemplate(s.ResourceTemplate(), s.Provide)
}
