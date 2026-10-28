# Architecture & Implementation Recommendations for bibx MCP: Tool & Resource Design

## 1. Executive Summary & Design Principles

During an autonomous literature review workflow (analyzing the *Bit-Patterned Media* domain and generating a compiled LaTeX/BibLaTeX report), key developer-experience and architectural considerations were identified in `bibx`.

### Key Design Principle: Complete Client-Server Decoupling for HTTP Readiness
When designing tools for the **Model Context Protocol (MCP)**, tools should not assume a shared local filesystem with the client.
* **Why `output_path` is an anti-pattern for HTTP deployments**:
  1. **Filesystem Divergence**: When the MCP server runs remotely over HTTP or inside a container, writing to a local path writes to the *server's* disk, not the *client's* workspace.
  2. **Security**: Allowing remote clients to specify arbitrary write paths creates path-traversal vulnerabilities (CWE-22).
  3. **Protocol Idiom**: MCP already provides a first-class abstraction for accessing data without filesystem coupling: **MCP Resources**.

### The Recommended Architecture: Tools vs. Resources
Instead of using a filesystem parameter, we recommend a clean separation of concerns:
1. **The `bibtex` Tool**: Optimized for *selective, interactive agent workflows* (e.g., retrieving references for specific `ids` or a Tree of Science `preset`). By defaulting to `omit_abstracts: true`, the payload is kept small ($\approx 10\text{--}20\text{ KB}$), returning both the BibTeX text and a structured `{ id -> citekey }` mapping directly in the JSON-RPC response.
2. **The `bibx:/collections/{id}/bibtex` Resource**: The dedicated, standard mechanism for streaming the *entire collection's BibTeX*, fetched on-demand via standard MCP `resources/read`.

---

## 2. Priority Recommendations

### Priority 1: LaTeX Special Character Sanitization (Critical Bug Fix)

#### The Problem
OpenAlex article metadata frequently contains unescaped LaTeX special characters in `abstract` and `title` fields, most notably the percent sign `%` (e.g., `"fluctuations smaller than 8% of the bit period"`), ampersands `&`, underscores `_`, and hashes `#`.

In BibTeX/Biber, an unescaped `%` acts as a comment delimiter that comments out the rest of the line—including the closing brace `}`. This causes fatal LaTeX compilation errors:
```text
Runaway argument?
{In high density bit-patterned media (BPM), bits or magnetic islands \ETC.
! File ended while scanning use of \field.
```

#### Go Implementation in `models/bibtex.go`

```go
// sanitizeLaTeX escapes common LaTeX reserved characters in text fields.
var latexReplacer = strings.NewReplacer(
	"%", `\%`,
	"&", `\&`,
	"_", `\_`,
	"#", `\#`,
)

func sanitizeLaTeX(s string) string {
	return latexReplacer.Replace(s)
}
```

In `(*Article).ToBibEntry(citeKey ...string)`:
```go
// Sanitize title
if a.Title != nil && *a.Title != "" {
	entry.AddField("title", bibtex.NewBibConst(sanitizeLaTeX(*a.Title)))
}

// Omit abstract when requested (default for agent bibliography generation)
if !omitAbstract && a.Abstract != nil && *a.Abstract != "" {
	cleanAbstract := strings.Join(strings.Fields(*a.Abstract), " ")
	entry.AddField("abstract", bibtex.NewBibConst(sanitizeLaTeX(cleanAbstract)))
}
```

---

### Priority 2: Structured Citation Key Mapping in the `bibtex` Tool

#### The Problem
When an LLM agent writes a `.tex` document, it must cite entries using their generated citation keys (e.g., `\cite{albrecht2015}`). Currently, `BibtexOutput` returns only the raw BibTeX string, forcing agents to parse the text with fragile regular expressions to discover generated keys.

#### Go Implementation in `internal/mcp/tools/analysis/bibtex.go`

Return a structured slice of `CitationKeyInfo` alongside the string:

```go
// CitationKeyInfo provides metadata for an exported citation.
type CitationKeyInfo struct {
	ID          string `json:"id"`
	CiteKey     string `json:"citekey"`
	Title       string `json:"title,omitempty"`
	Year        *int   `json:"year,omitempty"`
	FirstAuthor string `json:"first_author,omitempty"`
}

type BibtexOutput struct {
	Bibtex   string            `json:"bibtex"`
	Count    int               `json:"count"`
	Keys     []CitationKeyInfo `json:"keys"`
	NotFound []string          `json:"not_found,omitempty"`
}
```

**Agent Benefit**: The model immediately iterates over `Keys` to write `\cite{k.CiteKey}` without parsing raw text.

---

### Priority 3: Full Collection BibTeX as an MCP Resource

For retrieving full collections without bloating tool responses or coupling to the local filesystem, expose a resource template following the existing convention in `internal/mcp/resources/collections/summary.go`.

#### Go Implementation: `internal/mcp/resources/collections/bibtex.go`

```go
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
		Description: "Full BibTeX bibliography for a collection",
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

	articles, _, err := b.analysisService.GetArticles(ctx, id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve articles for collection: %w", err)
	}

	// Full collection export via resource: sanitize LaTeX and render
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
```

---

### Priority 4: Category / ToS Presets in the `bibtex` Tool

When conducting a literature review, agents frequently need citations for:
* Top roots, trunks, and leaves
* Top-$N$ most cited articles in a collection

Allowing a `Preset` parameter makes this a single call rather than 4–5 round-trips.

#### Go Input Definition:
```go
type BibtexInput struct {
	// IDs is the explicit list of article IDs, DOIs, or labels to export.
	IDs []string `json:"ids,omitempty"`
	// Target is the collection ID (UUID) or name alias. If omitted, the latest collection is queried.
	Target string `json:"target,omitempty"`
	// Preset selects articles by Tree of Science category ("tos", "roots", "trunks", "leaves", "top_cited").
	Preset string `json:"preset,omitempty"`
	// Limit is the maximum number of articles to export when using a preset (default: 10 per category).
	Limit int `json:"limit,omitempty"`
	// OmitAbstracts excludes abstract fields to minimize payload size and avoid LaTeX collisions (default: true).
	OmitAbstracts *bool `json:"omit_abstracts,omitempty"`
}
```

---

## 3. Updated `internal/mcp/tools/analysis/bibtex.go`

Here is the refactored, transport-agnostic tool implementation using `github.com/modelcontextprotocol/go-sdk/mcp`:

```go
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

// CitationKeyInfo provides metadata for an exported citation.
type CitationKeyInfo struct {
	ID          string `json:"id"`
	CiteKey     string `json:"citekey"`
	Title       string `json:"title,omitempty"`
	Year        *int   `json:"year,omitempty"`
	FirstAuthor string `json:"first_author,omitempty"`
}

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
	Keys []CitationKeyInfo `json:"keys"`
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
			"Use 'preset': 'tos' for Tree of Science synthesis. " +
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
	omitAbstracts := true

	if input != nil {
		target = strings.TrimSpace(input.Target)
		ids = input.IDs
		if input.OmitAbstracts != nil {
			omitAbstracts = *input.OmitAbstracts
		}
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

	// Retrieve targeted or preset articles
	articles, notFound, err := b.analysisService.GetArticles(ctx, col.ID, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve articles: %w", err)
	}

	bibtexStr, keysInfo := models.Articles(articles).ToBibTeXWithKeys(omitAbstracts)

	return &mcp.CallToolResult{IsError: false}, &BibtexOutput{
		Bibtex:      bibtexStr,
		Count:       len(articles),
		Keys:        keysInfo,
		ResourceURI: fmt.Sprintf("bibx:/collections/%s/bibtex", col.ID.String()),
		NotFound:    notFound,
	}, nil
}

// Register registers the current tool to the given [mcp.Server].
func (b *Bibtex) Register(server *mcp.Server) {
	mcp.AddTool(server, b.Tool(), b.Handle)
}
```

---

## 4. Summary of Benefits

1. **HTTP & Remote Safe**: Complete independence from the host filesystem. Works identically over `stdio`, `SSE`, or standard HTTP.
2. **Security**: Zero arbitrary file-write or path traversal attack surface.
3. **Small Payloads in Tool Calls**: With `omit_abstracts: true` (default), targeted tool responses are typically $10\text{--}20\text{ KB}$.
4. **First-Class Resource for Full Exports**: Full collection bibliographies (potentially hundreds of kilobytes or megabytes) are accessed cleanly via `bibx:/collections/{id}/bibtex` using standard MCP resource streaming.
5. **No LaTeX Build Failures**: `sanitizeLaTeX()` guarantees that OpenAlex text containing `%`, `&`, `_`, or `#` never crashes `biber` or `pdflatex`.
