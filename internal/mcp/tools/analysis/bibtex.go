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
	// IDs is the list of article IDs, DOIs, or labels to export. If omitted, exports all articles in the collection.
	IDs []string `json:"ids,omitempty"`
	// Target is the collection ID (UUID) or name alias. If omitted, the latest collection is queried.
	Target string `json:"target,omitempty"`
	// Preset selects articles by category ("tos", "roots", "trunks", "leaves", "top_cited").
	Preset string `json:"preset,omitempty"`
	// Limit is the maximum number of articles to export when using a preset (default: 10 per category).
	Limit int `json:"limit,omitempty"`
	// OmitAbstracts excludes abstract fields to minimize payload size and avoid LaTeX collisions (default: true).
	OmitAbstracts *bool `json:"omit_abstracts,omitempty"`
}

// BibtexOutput contains the results of the BibTeX export.
type BibtexOutput struct {
	// Bibtex contains the formatted BibTeX entries string.
	Bibtex string `json:"bibtex"`
	// Count is the number of articles successfully exported.
	Count int `json:"count"`
	// Keys maps each exported article to its generated citation key.
	Keys []models.CitationKeyInfo `json:"keys"`
	// ResourceURI informs the client where the complete collection BibTeX can be read as a resource.
	ResourceURI string `json:"resource_uri,omitempty"`
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
		Description: "Output selected articles from a collection in BibTeX format. " +
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

	var articles []*models.Article
	var notFound []string

	if preset != "" {
		var err error
		articles, err = b.articlesForPreset(ctx, col.ID, preset, limit)
		if err != nil {
			return nil, nil, err
		}
	} else {
		var err error
		articles, notFound, err = b.analysisService.GetArticles(ctx, col.ID, ids)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to retrieve articles: %w", err)
		}
	}

	bibtexStr, keysInfo := models.Articles(articles).ToBibTeXWithKeys(models.WithOmitAbstracts(omitAbstracts))
	if keysInfo == nil {
		keysInfo = []models.CitationKeyInfo{}
	}

	return &mcp.CallToolResult{IsError: false}, &BibtexOutput{
		Bibtex:      bibtexStr,
		Count:       len(articles),
		Keys:        keysInfo,
		ResourceURI: fmt.Sprintf("bibx:/collections/%s/bibtex", col.ID.String()),
		NotFound:    notFound,
	}, nil
}

func (b *Bibtex) articlesForCategory(ctx context.Context, colID uuid.UUID, cat string, limit int) ([]*models.Article, error) {
	res, err := b.analysisService.QueryCategory(ctx, colID, cat, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s category: %w", cat, err)
	}
	articles := make([]*models.Article, 0, len(res))
	for _, r := range res {
		if r != nil && r.Article != nil {
			articles = append(articles, r.Article)
		}
	}
	return articles, nil
}

func (b *Bibtex) articlesForToS(ctx context.Context, colID uuid.UUID, limit int) ([]*models.Article, error) {
	categories := []string{"root", "trunk", "leaf"}
	seen := make(map[string]bool)
	var articles []*models.Article
	for _, cat := range categories {
		arts, err := b.articlesForCategory(ctx, colID, cat, limit)
		if err != nil {
			return nil, err
		}
		for _, art := range arts {
			key := art.Label
			if k := art.Key(); k != nil && *k != "" {
				key = *k
			}
			if !seen[key] {
				seen[key] = true
				articles = append(articles, art)
			}
		}
	}
	return articles, nil
}

func (b *Bibtex) articlesTopCited(ctx context.Context, colID uuid.UUID, limit int) ([]*models.Article, error) {
	allArticles, _, err := b.analysisService.GetArticles(ctx, colID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve articles for top_cited preset: %w", err)
	}
	sort.SliceStable(allArticles, func(i, j int) bool {
		tcI, tcJ := 0, 0
		if allArticles[i] != nil && allArticles[i].TimesCited != nil {
			tcI = *allArticles[i].TimesCited
		}
		if allArticles[j] != nil && allArticles[j].TimesCited != nil {
			tcJ = *allArticles[j].TimesCited
		}
		return tcI > tcJ
	})
	if len(allArticles) > limit {
		return allArticles[:limit], nil
	}
	return allArticles, nil
}

func (b *Bibtex) articlesForPreset(ctx context.Context, colID uuid.UUID, preset string, limit int) ([]*models.Article, error) {
	switch preset {
	case "tos":
		return b.articlesForToS(ctx, colID, limit)
	case "roots", "root":
		return b.articlesForCategory(ctx, colID, "root", limit)
	case "trunks", "trunk":
		return b.articlesForCategory(ctx, colID, "trunk", limit)
	case "leaves", "leaf":
		return b.articlesForCategory(ctx, colID, "leaf", limit)
	case "top_cited":
		return b.articlesTopCited(ctx, colID, limit)
	default:
		return nil, fmt.Errorf("unknown preset: %q (supported: tos, roots, trunks, leaves, top_cited)", preset)
	}
}

// Register registers the current tool to the given [mcp.Server].
func (b *Bibtex) Register(server *mcp.Server) {
	mcp.AddTool(server, b.Tool(), b.Handle)
}
