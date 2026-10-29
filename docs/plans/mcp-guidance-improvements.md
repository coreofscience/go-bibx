# Plan: Improve MCP Tools and Resources Guidance

## Context & Problem Statement

The current MCP server implementation in `go-bibx` registers six tools (`list`, `show`, `inquire`, `tos`, `search`, `bibtex`), one static resource (`bibx:/summary`), and two resource templates (`bibx:/collections/{id}/summary`, `bibx:/collections/{id}/bibtex`).

While top-level tool descriptions are intentionally concise to conserve context window tokens, **none of the tool input, tool output, or shared DTO structs define `jsonschema` field tags**. Because [`mcp.AddTool`](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp#AddTool) relies on `github.com/google/jsonschema-go` to infer `inputSchema` and `outputSchema` from Go types, the resulting JSON schemas sent to MCP clients contain bare property names and types with zero field-level documentation. Callers must guess:
- Default values and valid bounds (e.g., `limit` ranges of `1–1000` vs. `1–100`).
- Enum/allowed values (e.g., `category` in `tos`, `preset` in `bibtex`).
- Mutual exclusivity constraints (e.g., `preset` vs. `ids` in `bibtex`).
- Fallback behavior when optional fields are omitted (e.g., omitting `target` resolves the most recently updated collection).
- What output fields represent (e.g., `rootness`, `trunkness`, `leafness`, `is_new`, `keys`).

Additionally, resource descriptions, server-level instructions, and tool annotations (`OpenWorldHint`) can be tightened to give callers clear workflow guidance with minimal token overhead.

---

## Technical Constraints (`jsonschema-go`)

`github.com/modelcontextprotocol/go-sdk` v1.8.0 uses `github.com/google/jsonschema-go` v0.4.3 for schema reflection:
1. **Tag Syntax**: The entire string value of the `jsonschema` struct tag is used directly as the property's `description` in the generated JSON Schema (e.g., `` `json:"limit,omitempty" jsonschema:"Maximum number of results (1-1000, default: 10)."` ``).
2. **No `WORD=` Prefix**: `jsonschema-go` reserves `WORD=` prefixes for future keywords and **returns an error** if a `jsonschema` tag starts with `WORD=` (any non-whitespace sequence followed by `=`). Descriptions must be plain prose and never start with `description=...`.
3. **Required vs. Optional Fields**: Fields without `omitempty` / `omitzero` on their `json` tag are marked `required` in the generated schema; fields with `omitempty` / `omitzero` are optional.

---

## Proposed Changes

### 1. Shared MCP DTOs (`internal/mcp/models/` & `models/`)

Add concise `jsonschema` tags to all fields of the shared output models so every tool returning collections, articles, graph nodes/links, or BibTeX keys provides self-documenting output schemas.

#### [MODIFY] [collection.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/models/collection.go)
Add `jsonschema` tags to [`CollectionSummary`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/models/collection.go#L10-L18):
| Field | JSON Name | Proposed `jsonschema` Description |
| :--- | :--- | :--- |
| `ID` | `id` | `"Unique collection UUID"` |
| `Query` | `query` | `"Search query used to build the collection"` |
| `ResultLimit` | `result_limit` | `"Maximum number of initial seed works requested from OpenAlex"` |
| `Name` | `name,omitempty` | `"Optional human-readable alias for the collection"` |
| `ArticleCount` | `article_count` | `"Total number of articles in the analyzed citation graph"` |
| `CreatedAt` | `created_at` | `"Creation timestamp (RFC 3339)"` |
| `UpdatedAt` | `updated_at` | `"Last update timestamp (RFC 3339)"` |

#### [MODIFY] [article.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/models/article.go)
Add `jsonschema` tags to [`ArticleSummary`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/models/article.go#L7-L22):
| Field | JSON Name | Proposed `jsonschema` Description |
| :--- | :--- | :--- |
| `Label` | `label` | `"Primary article identifier in the collection (e.g., 'openalex:W...')"` |
| `Title` | `title,omitempty` | `"Article title"` |
| `Authors` | `authors,omitempty` | `"Author names"` |
| `Year` | `year,omitempty` | `"Publication year"` |
| `Journal` | `journal,omitempty` | `"Publication venue or journal name"` |
| `Volume` | `volume,omitempty` | `"Journal volume"` |
| `Issue` | `issue,omitempty` | `"Journal issue"` |
| `Page` | `page,omitempty` | `"Page numbers or range"` |
| `DOI` | `doi,omitempty` | `"Digital Object Identifier (DOI)"` |
| `Permalink` | `permalink,omitempty` | `"Canonical URL to the article"` |
| `TimesCited` | `times_cited,omitempty` | `"Global citation count reported by OpenAlex"` |
| `Keywords` | `keywords,omitempty` | `"Associated keywords or concepts"` |
| `Abstract` | `abstract,omitempty` | `"Article abstract text"` |
| `IDs` | `ids,omitempty` | `"All known identifiers for the article (e.g., 'openalex:...', 'doi:...', 'simple:...')"` |

#### [MODIFY] [node.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/models/node.go)
Add `jsonschema` tags to [`NodeSummary`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/models/node.go#L8-L15):
| Field | JSON Name | Proposed `jsonschema` Description |
| :--- | :--- | :--- |
| `ID` | `id` | `"Unique node identifier in the citation graph"` |
| `Category` | `category` | `"Assigned Tree of Science category ('root', 'trunk', 'leaf', or 'none')"` |
| `Rootness` | `rootness` | `"Score measuring foundational/seminal impact (in-degree from collection citation graph)"` |
| `Trunkness` | `trunkness` | `"SAP score measuring structural bridge centrality between roots and leaves"` |
| `Leafness` | `leafness` | `"Score measuring modern state-of-the-art relevance (citation paths reaching roots)"` |
| `Article` | `article,omitempty` | `"Bibliographic metadata for the article"` |

#### [MODIFY] [link.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/models/link.go)
Add `jsonschema` tags to [`LinkSummary`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/models/link.go#L8-L12):
| Field | JSON Name | Proposed `jsonschema` Description |
| :--- | :--- | :--- |
| `Source` | `source` | `"Citing node ID"` |
| `Target` | `target` | `"Cited node ID"` |
| `Weight` | `weight` | `"Edge weight in the citation network"` |

#### [MODIFY] [result.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/models/result.go)
Add `jsonschema` tags to [`Result`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/models/result.go#L6-L9):
| Field | JSON Name | Proposed `jsonschema` Description |
| :--- | :--- | :--- |
| `Score` | `score` | `"Tree of Science category score (unnormalized SAP metric; higher means stronger fit)"` |
| `Article` | `article,omitempty` | `"Bibliographic metadata for the scored article"` |

#### [MODIFY] [bibtex.go](file:///home/oscar/Code/coreofscience/go-bibx/models/bibtex.go)
Add `jsonschema` tags to [`CitationKeyInfo`](file:///home/oscar/Code/coreofscience/go-bibx/models/bibtex.go#L39-L45) (used in `BibtexOutput.Keys`):
| Field | JSON Name | Proposed `jsonschema` Description |
| :--- | :--- | :--- |
| `ID` | `id` | `"Article identifier in the collection"` |
| `CiteKey` | `citekey` | `"Generated BibTeX citation key (disambiguated within the export)"` |
| `Title` | `title,omitempty` | `"Article title"` |
| `Year` | `year,omitempty` | `"Publication year"` |
| `FirstAuthor` | `first_author,omitempty` | `"Primary author name"` |

---

### 2. Collection Tools (`internal/mcp/tools/collections/`)

#### [MODIFY] [list.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/list.go)
- **[`ListOutput`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/list.go#L18-L20)**:
  - `Collections`: `jsonschema:"Stored bibliographic collections ordered by most recently updated"`
- **[`List.Tool`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/list.go#L35-L48)**:
  - `Description`: `"List stored bibliographic collections in the local database, ordered by most recently updated."`
  - `Annotations.OpenWorldHint`: Set to `new(false)` (queries local SQLite DB only).

#### [MODIFY] [show.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/show.go)
- **[`ShowInput`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/show.go#L14-L17)**:
  - `Target`: `jsonschema:"Collection UUID or name alias. Omit to inspect the most recently updated collection."`
- **[`ShowOutput`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/show.go#L20-L22)**:
  - `Collection`: `jsonschema:"Metadata of the resolved bibliographic collection"`
- **[`Show.Tool`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/show.go#L37-L50)**:
  - `Description`: `"Show metadata for a bibliographic collection by UUID or name alias (defaults to the latest collection if target is omitted)."`
  - `Annotations.OpenWorldHint`: Set to `new(false)` (queries local SQLite DB only).

#### [MODIFY] [inquire.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/inquire.go)
- **[`InquireInput`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/inquire.go#L18-L27)**:
  - `Query`: `jsonschema:"Research topic or keywords to search on OpenAlex"`
  - `Limit`: `jsonschema:"Maximum number of initial seed works to fetch from OpenAlex (1-1000, default: 200). For best results, try between 200 and 500."`
  - `Force`: `jsonschema:"Re-fetch from OpenAlex and recompute the citation graph even if a cached collection exists for this query and limit"`
  - `Name`: `jsonschema:"Optional human-readable alias for referencing this collection in other tools"`
- **[`InquireOutput`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/inquire.go#L30-L33)**:
  - `Collection`: `jsonschema:"Metadata of the created or cached bibliographic collection"`
  - `IsNew`: `jsonschema:"True if newly fetched and analyzed; false if an existing cached collection was returned"`
- **[`Inquire.Tool`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/inquire.go#L48-L61)**:
  - `Description`: `"Fetch literature from OpenAlex for a research topic, run Tree of Science (SAP) network analysis, and store the collection locally."`
  - Keep `Annotations.OpenWorldHint: new(true)` (calls external OpenAlex API).

---

### 3. Analysis Tools (`internal/mcp/tools/analysis/`)

#### [MODIFY] [tos.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/tos.go)
- **[`TosInput`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/tos.go#L26-L33)**:
  - `Category`: `jsonschema:"Tree of Science category: 'root' (foundational/seminal papers), 'trunk' (structural bridge papers), or 'leaf' (modern state-of-the-art papers)"`
  - `Limit`: `jsonschema:"Maximum number of top-scored articles to return (1-1000, default: 10). 10 to 20 is usually plenty, except for 'leaf' where up to 40 is also fine."`
  - `Target`: `jsonschema:"Collection UUID or name alias. Omit to query the most recently updated collection."`
- **[`TosOutput`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/tos.go#L36-L38)**:
  - `Results`: `jsonschema:"Articles in the requested category ordered by descending score"`
- **[`Tos.Tool`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/tos.go#L55-L68)**:
  - `Description`: `"Query top articles in a collection by Tree of Science (ToS) category ('root', 'trunk', or 'leaf'). Read resource bibx:/summary for methodology details."`
  - `Annotations.OpenWorldHint`: Set to `new(false)` (queries local SQLite DB only).

#### [MODIFY] [search.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/search.go)
- **[`SearchInput`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/search.go#L15-L22)**:
  - `Query`: `jsonschema:"Keyword to match against article titles, abstracts, and metadata within the collection. For best results, try one-word queries."`
  - `Limit`: `jsonschema:"Number of top keyword-matched seed nodes used to construct the Steiner tree subgraph (1-100, default: 10). For best results, try between 10 and 20."`
  - `Target`: `jsonschema:"Collection UUID or name alias. Omit to search the most recently updated collection."`
- **[`SearchOutput`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/search.go#L25-L28)**:
  - `Nodes`: `jsonschema:"Articles in the connected Steiner tree subgraph (matched seed articles and connecting bridge papers)"`
  - `Links`: `jsonschema:"Directed citation edges connecting nodes in the subgraph"`
- **[`Search.Tool`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/search.go#L45-L58)**:
  - `Description`: `"Search within an existing collection using keyword matching and Steiner tree graph reduction to find relevant articles and connecting citation paths."`
  - `Annotations.OpenWorldHint`: Set to `new(false)` (queries local SQLite DB only).

#### [MODIFY] [bibtex.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/bibtex.go)
- **[`BibtexInput`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/bibtex.go#L17-L28)**:
  - `IDs`: `jsonschema:"Article identifiers to export (OpenAlex IDs like 'W...', DOIs like '10.1109/...', or node labels). Mutually exclusive with 'preset'. Omit both 'ids' and 'preset' to export all articles."`
  - `Target`: `jsonschema:"Collection UUID or name alias. Omit to export from the most recently updated collection."`
  - `Preset`: `jsonschema:"Category preset to export: 'tos' (top roots, trunks, and leaves combined), 'roots', 'trunks', 'leaves', or 'top_cited'. Mutually exclusive with 'ids'."`
  - `Limit`: `jsonschema:"Maximum number of articles per category when using 'preset' (1-1000, default: 10)"`
  - `OmitAbstracts`: `jsonschema:"Omit abstract fields from BibTeX entries to reduce payload size (default: true)"`
- **[`BibtexOutput`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/bibtex.go#L31-L42)**:
  - `Bibtex`: `jsonschema:"Formatted BibTeX entries string"`
  - `Count`: `jsonschema:"Number of articles exported"`
  - `Keys`: `jsonschema:"Mapping of exported articles to their generated BibTeX citation keys"`
  - `ResourceURI`: `jsonschema:"MCP resource URI (bibx:/collections/{id}/bibtex) for the full collection bibliography"`
  - `NotFound`: `jsonschema:"Requested input IDs that could not be found in the collection"`
- **[`Bibtex.Tool`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/bibtex.go#L59-L75)**:
  - Keep the existing concise description and set `Annotations.OpenWorldHint` to `new(false)`.

---

### 4. Resources & Server Instructions

#### [MODIFY] [summary.go (theory)](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/resources/theory/summary.go)
- Fix typo in [`Summary.Resource`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/resources/theory/summary.go#L45-L58) description (`"it's"` -> `"its"`) and clarify purpose:
  - `Description`: `"Overview of the Tree of Science (ToS) bibliometric methodology (root, trunk, and leaf categories) used by bibx."`

#### [MODIFY] [summary.go (collections)](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/resources/collections/summary.go)
- Enhance [`Summary.ResourceTemplate`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/resources/collections/summary.go#L28-L36):
  - `Title`: `"Collection Summary"`
  - `Description`: `"JSON metadata summary for a bibliographic collection by its UUID (query, article count, limits, timestamps)."`
  - Add `Annotations`: `&mcp.Annotations{Audience: []mcp.Role{"user", "assistant"}, Priority: 0.7}`

#### [MODIFY] [bibtex.go (collections)](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/resources/collections/bibtex.go)
- Enhance [`Bibtex.ResourceTemplate`](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/resources/collections/bibtex.go#L31-L39):
  - `Description`: `"Complete BibTeX bibliography (including abstracts) for all articles in a collection by UUID. For filtered exports or presets, prefer the 'bibtex' tool."`
  - Add `Annotations`: `&mcp.Annotations{Audience: []mcp.Role{"user", "assistant"}, Priority: 0.5}`

#### [MODIFY] [mcp.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/cli/commands/mcp/mcp.go)
- Refine [`ServerOptions.Instructions`](file:///home/oscar/Code/coreofscience/go-bibx/internal/cli/commands/mcp/mcp.go#L50-L52) to succinctly guide the agent through the intended workflow:
  - `"Manage and analyze bibliographic collections using the Tree of Science (ToS) methodology (see resource bibx:/summary). Typical workflow: 1) 'list' existing collections or 'inquire' a research topic from OpenAlex; 2) analyze key papers with 'tos' (root/trunk/leaf) or 'search' (Steiner tree subgraph); 3) export citations with 'bibtex'."`

---

## Verification Plan

1. **Schema Property Description Assertions in Tool Tests**:
   Update the `"lists tool metadata"` subtests in:
   - [list_test.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/list_test.go)
   - [show_test.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/show_test.go)
   - [inquire_test.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/collections/inquire_test.go)
   - [tos_test.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/tos_test.go)
   - [search_test.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/search_test.go)
   - [bibtex_test.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/tools/analysis/bibtex_test.go)
   to inspect `res.Tools[0].InputSchema` and `res.Tools[0].OutputSchema` and assert that every expected property has a non-empty `"description"` field in the marshaled JSON schema.
2. **Resource Metadata Assertions**:
   Verify updated `Description`, `Title`, and `Annotations` in [theory/summary_test.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/resources/theory/summary_test.go), [collections/summary_test.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/resources/collections/summary_test.go), and [collections/bibtex_test.go](file:///home/oscar/Code/coreofscience/go-bibx/internal/mcp/resources/collections/bibtex_test.go).
3. **Full Test Suite & Linter**:
   Run `go test ./...` and `golangci-lint run` to ensure all schemas reflect cleanly and pass linting.
