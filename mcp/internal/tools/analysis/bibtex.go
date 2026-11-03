package analysis

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"uuid"

	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// BibtexInput represents the input parameters for the Bibtex tool.
type BibtexInput struct {
	// IDs is the list of work IDs, DOIs, or labels to export. If omitted, exports all works in the collection.
	IDs []string `json:"ids,omitempty" jsonschema:"Work identifiers to export (OpenAlex IDs like 'W...', DOIs like '10.1109/...', or node labels). Mutually exclusive with 'preset'. Omit both 'ids' and 'preset' to export all works."`
	// Target is the collection ID (UUID) or name alias. If omitted, the latest collection is queried.
	Target string `json:"target,omitempty" jsonschema:"Collection UUID or name alias. Omit to export from the most recently updated collection."`
	// Preset selects works by category ("tos", "roots", "trunks", "leaves", "top_cited").
	Preset string `json:"preset,omitempty" jsonschema:"Category preset to export: 'tos' (top roots, trunks, and leaves combined), 'roots', 'trunks', 'leaves', or 'top_cited'. Mutually exclusive with 'ids'."`
	// Limit is the maximum number of works to export when using a preset (default: 10 per category).
	Limit int `json:"limit,omitempty" jsonschema:"Maximum number of works per category when using 'preset' (1-1000, default: 10)"`
	// OmitAbstracts excludes abstract fields to minimize payload size (default: true).
	OmitAbstracts *bool `json:"omit_abstracts,omitempty" jsonschema:"Omit abstract fields from BibTeX entries to reduce payload size (default: true)"`
}

// BibtexOutput contains the results of the BibTeX export.
type BibtexOutput struct {
	// Bibtex contains the formatted BibTeX entries string.
	Bibtex string `json:"bibtex" jsonschema:"Formatted BibTeX entries string"`
	// Count is the number of works successfully exported.
	Count int `json:"count" jsonschema:"Number of works exported"`
	// Keys maps each exported work to its generated citation key.
	Keys []models.CitationKeyInfo `json:"keys" jsonschema:"Mapping of exported works to their generated BibTeX citation keys"`
	// ResourceURI informs the client where the complete collection BibTeX can be read as a resource.
	ResourceURI string `json:"resource_uri,omitempty" jsonschema:"MCP resource URI (bibx:/collections/{id}/bibtex) for the full collection bibliography"`
	// NotFound contains any input IDs that could not be resolved.
	NotFound []string `json:"not_found,omitempty" jsonschema:"Requested input IDs that could not be found in the collection"`
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
			OpenWorldHint:   new(false),
			ReadOnlyHint:    true,
			Title:           "export works as BibTeX",
		},
		Description: "Output selected works from a collection in BibTeX format. " +
			"Accepts DOIs (e.g., '10.1109/...'), OpenAlex IDs ('W...'), or labels in 'ids'. " +
			"Use 'preset': 'tos' for Tree of Science synthesis ('tos', 'roots', 'trunks', 'leaves', 'top_cited'). " +
			"For the full collection bibliography, access the resource at bibx:/collections/{id}/bibtex.",
		Name:  "bibtex",
		Title: "Export BibTeX",
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
	preset := ""
	limit := 10
	omitAbstracts := true

	if input != nil {
		target = strings.TrimSpace(input.Target)
		ids = input.IDs
		preset = strings.ToLower(strings.TrimSpace(input.Preset))
		if input.Limit != 0 {
			if input.Limit < 1 {
				return nil, nil, errors.New("limit must be greater than 0")
			}
			if input.Limit > 1000 {
				return nil, nil, errors.New("limit must be less than or equal to 1000")
			}
			limit = input.Limit
		}
		if input.OmitAbstracts != nil {
			omitAbstracts = *input.OmitAbstracts
		}
	}

	if preset != "" && len(ids) > 0 {
		return nil, nil, errors.New("cannot specify both 'preset' and 'ids'")
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

	var works []*models.Work
	var notFound []string

	if preset != "" {
		var err error
		works, err = b.worksForPreset(ctx, col.ID, preset, limit)
		if err != nil {
			return nil, nil, err
		}
	} else {
		var err error
		works, notFound, err = b.analysisService.GetWorks(ctx, col.ID, ids)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to retrieve works: %w", err)
		}
	}

	bibtexStr, keysInfo := models.Works(works).ToBibTeXWithKeys(models.WithOmitAbstracts(omitAbstracts))
	if keysInfo == nil {
		keysInfo = []models.CitationKeyInfo{}
	}

	return &mcp.CallToolResult{IsError: false}, &BibtexOutput{
		Bibtex:      bibtexStr,
		Count:       len(works),
		Keys:        keysInfo,
		ResourceURI: fmt.Sprintf("bibx:/collections/%s/bibtex", col.ID.String()),
		NotFound:    notFound,
	}, nil
}

func (b *Bibtex) worksForCategory(ctx context.Context, colID uuid.UUID, cat string, limit int) ([]*models.Work, error) {
	res, err := b.analysisService.QueryCategory(ctx, colID, cat, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s category: %w", cat, err)
	}
	works := make([]*models.Work, 0, len(res))
	for _, r := range res {
		if r != nil && r.Work != nil {
			works = append(works, r.Work)
		}
	}
	return works, nil
}

func (b *Bibtex) worksForToS(ctx context.Context, colID uuid.UUID, limit int) ([]*models.Work, error) {
	categories := []string{"root", "trunk", "leaf"}
	seen := make(map[string]bool)
	var works []*models.Work
	for _, cat := range categories {
		arts, err := b.worksForCategory(ctx, colID, cat, limit)
		if err != nil {
			return nil, err
		}
		for _, work := range arts {
			key := work.Label
			if k := work.Key(); k != nil && *k != "" {
				key = *k
			}
			if !seen[key] {
				seen[key] = true
				works = append(works, work)
			}
		}
	}
	return works, nil
}

func (b *Bibtex) worksTopCited(ctx context.Context, colID uuid.UUID, limit int) ([]*models.Work, error) {
	allWorks, _, err := b.analysisService.GetWorks(ctx, colID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve works for top_cited preset: %w", err)
	}
	sort.SliceStable(allWorks, func(i, j int) bool {
		tcI, tcJ := 0, 0
		if allWorks[i] != nil && allWorks[i].TimesCited != nil {
			tcI = *allWorks[i].TimesCited
		}
		if allWorks[j] != nil && allWorks[j].TimesCited != nil {
			tcJ = *allWorks[j].TimesCited
		}
		return tcI > tcJ
	})
	if len(allWorks) > limit {
		return allWorks[:limit], nil
	}
	return allWorks, nil
}

func (b *Bibtex) worksForPreset(ctx context.Context, colID uuid.UUID, preset string, limit int) ([]*models.Work, error) {
	switch preset {
	case "tos":
		return b.worksForToS(ctx, colID, limit)
	case "roots", "root":
		return b.worksForCategory(ctx, colID, "root", limit)
	case "trunks", "trunk":
		return b.worksForCategory(ctx, colID, "trunk", limit)
	case "leaves", "leaf":
		return b.worksForCategory(ctx, colID, "leaf", limit)
	case "top_cited":
		return b.worksTopCited(ctx, colID, limit)
	default:
		return nil, fmt.Errorf("unknown preset: %q (supported: tos, roots, trunks, leaves, top_cited)", preset)
	}
}

// Register registers the current tool to the given [mcp.Server].
func (b *Bibtex) Register(server *mcp.Server) {
	mcp.AddTool(server, b.Tool(), b.Handle)
}
