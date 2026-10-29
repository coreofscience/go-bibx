package collections

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Bibtex exposes an MCP resource for a full collection's BibTeX.
type Bibtex struct {
	collectionService services.CollectionService
	analysisService   services.AnalysisService
}

// NewBibtex builds a [Bibtex] resource instance.
func NewBibtex(collectionService services.CollectionService, analysisService services.AnalysisService) *Bibtex {
	return &Bibtex{
		collectionService: collectionService,
		analysisService:   analysisService,
	}
}

// ResourceTemplate returns the [mcp.ResourceTemplate] for collection bibtex.
func (b *Bibtex) ResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Annotations: &mcp.Annotations{
			Audience: []mcp.Role{"user", "assistant"},
			Priority: 0.5,
		},
		Description: "Complete BibTeX bibliography (including abstracts) for all articles in a collection by UUID. For filtered exports or presets, prefer the 'bibtex' tool.",
		MIMEType:    "text/x-bibtex",
		Name:        "collection bibtex",
		Title:       "Collection BibTeX",
		URITemplate: "bibx:/collections/{id}/bibtex",
	}
}

// Provide fetches the collection and renders the full BibTeX text.
func (b *Bibtex) Provide(ctx context.Context, rrr *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	u, err := url.Parse(rrr.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource URI: %w", err)
	}
	if u.Scheme != "bibx" {
		return nil, fmt.Errorf("URI scheme should be bibx, got: %q", u.Scheme)
	}

	raw := strings.TrimPrefix(rrr.Params.URI, "bibx:")
	raw = strings.TrimPrefix(raw, "/collections/")
	raw = strings.TrimSuffix(raw, "/bibtex")

	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("provided id could not be parsed as UUID: %q", raw)
	}

	if _, err := b.collectionService.FindByID(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to find collection: %w", err)
	}

	articles, _, err := b.analysisService.GetArticles(ctx, id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve articles for collection: %w", err)
	}

	bibtexStr := models.Articles(articles).ToBibTeX()

	return &mcp.ReadResourceResult{
		TTLMs:      int((60 * time.Minute).Milliseconds()),
		CacheScope: "private",
		Contents: []*mcp.ResourceContents{{
			URI:      rrr.Params.URI,
			MIMEType: "text/x-bibtex",
			Text:     bibtexStr,
		}},
	}, nil
}

// Register registers the resource template to the given [mcp.Server].
func (b *Bibtex) Register(server *mcp.Server) {
	server.AddResourceTemplate(b.ResourceTemplate(), b.Provide)
}
