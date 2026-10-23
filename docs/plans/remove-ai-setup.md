# Implementation Plan: Remove AI Setup & Transition to Model Context Protocol (MCP)

## Executive Summary

The purpose of this plan is to design the complete removal of the existing AI
setup from the `go-bibx` codebase, and outline the architectural roadmap to
replace it with a native **Model Context Protocol (MCP)** server in a subsequent
release.

Currently, `go-bibx` provides an AI integration via the `bibx init` command,
which unpacks markdown instructions (`AGENTS.md`) and agent skills
(`.agents/skills/*`) into the target workspace using Go's `embed.FS`. While this
enabled file-based skill discovery for early coding agents, it introduces
significant drawbacks:

1. **Workspace Pollution**: It litters user workspaces with static documentation
   and markdown files that can quickly become stale when CLI flags or outputs
   change.
2. **Brittle Agent Interactions**: File-based skills rely on unstructured text
   prompts and shell execution rather than strongly typed, schema-validated tool
   invocations.
3. **High Maintenance Overhead**: Maintaining prompt templates and file
   scaffolding code within the core CLI repository distracts from core
   bibliographic graph analysis.

By removing the scaffolding mechanism and planning a future native MCP server,
`go-bibx` will provide modern LLMs and agent harnesses (e.g., OpenCode, Claude
Desktop, Cursor) with direct, schema-driven tool calling (`bibx_inquire`,
`bibx_query`, `bibx_search`, `bibx_view`) and dynamic resource inspection over
standard JSON-RPC 2.0 (stdio / SSE).

---

## Current Architecture of the AI Setup

The AI setup currently touches three layers of the CLI architecture:

```
cli/
├── commands/
│   ├── commands.go              # Registers initcmd.New()
│   └── init/
│       └── init.go              # CLI subcommand: 'bibx init'
├── services/
│   └── scaffold.go              # ScaffoldService interface & DefaultScaffoldService
└── repos/
    ├── scaffold.go              # ScaffoldRepo & TemplateScaffoldRepo (embed.FS)
    ├── scaffold_test.go         # Unit test validating AGENTS.md & skill creation
    └── templates/               # Embedded static assets:
        ├── AGENTS.md            # Agent roles (Primary, Research Subagent, Self Subagent)
        └── .agents/
            └── skills/
                ├── bibx-inquire/SKILL.md  # Skill prompt for topic inquiry
                ├── bibx-query/SKILL.md    # Skill prompt for ToS querying
                └── bibx-search/SKILL.md   # Skill prompt for BM25 graph search
```

### Component Details

1. **Command (`cli/commands/init/init.go`)**:
   - Defines the `init` command registered in `cli/commands/commands.go`.
   - Resolves the root directory from context (`utils.GetRootDir(ctx)`).
   - Instantiates `services.DefaultScaffoldService` and executes
     `Scaffold(ctx)`.

2. **Service (`cli/services/scaffold.go`)**:
   - Declares `ScaffoldService` interface and its default implementation
     `DefaultScaffoldService`.
   - Delegates the unpacking action to `repos.ScaffoldRepo`.

3. **Repository & Embedded FS (`cli/repos/scaffold.go`,
   `cli/repos/scaffold_test.go`)**:
   - Uses `//go:embed all:templates` to bundle `templates/` into the compiled
     binary.
   - Walks the embedded filesystem and recursively writes directories and files
     to disk.
   - Tested by `scaffold_test.go`, which verifies that `AGENTS.md` and relative
     skill paths are scaffolded cleanly into a temporary directory.

4. **Templates (`cli/repos/templates/`)**:
   - `AGENTS.md`: Defines instructions for primary and subagent roles.
   - `.agents/skills/bibx-*`: Markdown prompts explaining CLI arguments, flags,
     and workflow examples for `bibx inquire`, `bibx query`, and `bibx search`.

---

## Confirmed Design Decisions

1. **Complete Removal of `bibx init`**:
   - The `bibx init` command will be removed entirely from the CLI.
   - The associated service (`cli/services/scaffold.go`) and repository
     (`cli/repos/scaffold.go`, `cli/repos/scaffold_test.go`) will be deleted.
   - The embedded templates directory (`cli/repos/templates/`) will be removed.
   - Note: Template files used by renderers (`cli/renderers/templates/` for
     markdown, simple, reference formats) and the viewer
     (`cli/servers/viewer/templates/index.html`) are core
     visualization/rendering components and are **not** part of the AI setup;
     they remain untouched.

2. **No Deprecation Period Needed**:
   - `bibx init` only scaffolded markdown files and did not produce persistent
     machine state or configuration required by other subcommands.
   - Existing user repositories that already scaffolded `AGENTS.md` and
     `.agents/` will continue to function without disruption.

3. **Separation of Removal and MCP Implementation**:
   - **Step 1 (Scope of this plan)**: Cleanly excise the legacy file-scaffolding
     AI setup, leaving the codebase streamlined and tests passing.
   - **Step 2 (Future phase)**: Introduce an MCP server package (e.g.,
     `cli/servers/mcp` or `internal/mcp`) exposing `go-bibx` as a standard Model
     Context Protocol server.

---

## Comparison: Legacy File-Scaffolding vs. Future MCP Server

| Aspect | Legacy AI Setup (`bibx init`) | Future MCP Server (`bibx mcp`) |
| :--- | :--- | :--- |
| **Integration Model** | File-based prompt scaffolding (`AGENTS.md`, `SKILL.md`) | Protocol-driven RPC (Model Context Protocol over stdio / SSE) |
| **Agent Interaction** | Agent reads markdown files and constructs shell commands | Agent invokes structured tools with JSON Schema validation |
| **Workspace Footprint** | Pollutes user repository with `.agents/` and `AGENTS.md` | Zero files created in user project; purely runtime communication |
| **Error Handling** | Agent must parse stdout/stderr and exit codes | Structured JSON-RPC error responses with typed diagnostics |
| **State Inspection** | Static markdown documentation | Dynamic resource endpoints (e.g. `bibx://collection/summary`) |
| **Client Support** | Specific to tools supporting `.agents` / `AGENTS.md` | Universal standard supported by Claude, OpenCode, Cursor, etc. |

---

## Proposed Changes: Step-by-Step Removal Plan

### Phase 1: Remove Template Assets and Repository Layer

1. **Delete Embedded Templates**:
   - Remove `cli/repos/templates/AGENTS.md`
   - Remove `cli/repos/templates/.agents/skills/bibx-inquire/SKILL.md`
   - Remove `cli/repos/templates/.agents/skills/bibx-query/SKILL.md`
   - Remove `cli/repos/templates/.agents/skills/bibx-search/SKILL.md`
   - Remove directory `cli/repos/templates/`

2. **Delete Scaffold Repository**:
   - Remove `cli/repos/scaffold.go` (removes `//go:embed all:templates`,
     `ScaffoldRepo`, `TemplateScaffoldRepo`, and file copying logic).
   - Remove `cli/repos/scaffold_test.go`.

3. **Verify `cli/repos` Package**:
   - `cli/repos` will retain only `analysis.go` (`AnalysisRepo` and
     `FileAnalysisRepo`).

---

### Phase 2: Remove Scaffold Service Layer

1. **Delete Scaffold Service**:
   - Remove `cli/services/scaffold.go` (`ScaffoldService`,
     `DefaultScaffoldService`, `DefaultScaffoldServiceConfig`).

2. **Verify `cli/services` Package**:
   - `cli/services` will retain only `analysis.go` (`AnalysisService` and `OpenAlexAnalysisService`) and `analysis_test.go`.

---

### Phase 3: Remove CLI Subcommand Layer

1. **Delete `init` Command Package**:
   - Remove `cli/commands/init/init.go`.
   - Remove directory `cli/commands/init/`.

2. **Update Command Registry (`cli/commands/commands.go`)**:
   - Remove the `initcmd` import:
     ```go
     - initcmd "github.com/coreofscience/go-bibx/cli/commands/init"
     ```
   - Remove `initcmd.New()` from `commands.New()`:
     ```go
     func New() []*cli.Command {
         return []*cli.Command{
     -       initcmd.New(),
             inquire.New(),
             query.New(),
             search.New(),
             view.New(),
         }
     }
     ```

---

### Phase 4: Project & Documentation Clean-up

1. **Dependency & Module Clean-up**:
   - Run `go mod tidy` to verify no lingering module dependencies.
   - Run `gofmt -s -w .` to ensure proper formatting.

2. **Verify Documentation**:
   - Verify `README.md` and `CONTRIBUTING.md` (confirm no references to `bibx init` or AI scaffolding exist).

---

## Future MCP Integration Architecture (Roadmap Preview)

Once the legacy setup is removed, the transition to MCP can proceed cleanly. Below is the proposed design for the future MCP implementation.

### Architecture Overview

```
               ┌────────────────────────────────────────────────────────┐
               │              AI Agent / Client Harness                 │
               │         (OpenCode / Claude Desktop / Cursor)           │
               └───────────────────────────┬────────────────────────────┘
                                           │
                                           │ JSON-RPC 2.0 (stdio)
                                           ▼
┌───────────────────────────────────────────────────────────────────────────────┐
│                                 go-bibx                                       │
│                                                                               │
│   ┌───────────────────────────────────────────────────────────────────────┐   │
│   │                         MCP Server Handler                            │   │
│   │                      (tools/list, tools/call)                         │   │
│   └──────┬──────────────────────┬───────────────────┬───────────────────┬─┘   │
│          │                      │                   │                   │     │
│          ▼                      ▼                   ▼                   ▼     │
│   ┌──────────────┐       ┌──────────────┐    ┌──────────────┐    ┌──────────┐ │
│   │ bibx_inquire │       │  bibx_query  │    │ bibx_search  │    │bibx_view │ │
│   └──────┬───────┘       └──────┬───────┘    └──────┬───────┘    └─────┬────┘ │
│          │                      │                   │                  │      │
│          └──────────────────────┴─────────┬─────────┴──────────────────┘      │
│                                           │                                   │
│                                           ▼                                   │
│                           ┌───────────────────────────────┐                   │
│                           │      cli/services.Analysis    │                   │
│                           │   (OpenAlex, ToS/SAP, BM25)   │                   │
│                           └───────────────────────────────┘                   │
└───────────────────────────────────────────────────────────────────────────────┘
```

### Proposed MCP Tools

The MCP server will expose the existing domain services directly as typed tools:

1. **`bibx_inquire`**:
   - **Description**: Fetch bibliographic metadata from OpenAlex, build citation networks, compute Tree of Science metrics (SAP), run Leiden clustering, and persist the analysis.
   - **Parameters**:
     - `query` (string, required): Research topic or query string.
     - `limit` (integer, optional, default: 200): Maximum number of root works to retrieve.
     - `force` (boolean, optional, default: false): Overwrite existing `.bibx` analysis files.

2. **`bibx_query`**:
   - **Description**: Filter and extract articles based on their Tree of Science structural category.
   - **Parameters**:
     - `category` (string, enum: `["root", "trunk", "leaf"]`, required): Category to query.
     - `limit` (integer, optional, default: 10): Number of top articles to retrieve.
     - `format` (string, enum: `["reference", "markdown", "simple", "json"]`, optional, default: `"reference"`): Output representation.

3. **`bibx_search`**:
   - **Description**: Perform BM25 keyword search on the bibliographic collection and derive connected subgraphs using pseudo-Steiner tree reduction.
   - **Parameters**:
     - `query` (string, required): Keywords to match across title, keywords, abstract, and authors.
     - `limit` (integer, optional, default: 10): Maximum number of matching articles.
     - `format` (string, enum: `["reference", "markdown", "simple", "json"]`, optional, default: `"markdown"`): Output representation.

4. **`bibx_view`**:
   - **Description**: Launch the interactive local HTTP visualization server for graph exploration.
   - **Parameters**:
     - `port` (integer, optional, default: 8080): Port to bind the viewer server.

### Proposed MCP Resources

- **`bibx://analysis`**: Returns current collection statistics (total nodes, edges, density, cluster count, date generated).
- **`bibx://articles/{id}`**: Returns bibliographic metadata and Tree of Science metrics for an individual paper.

---

## Verification & Testing Plan

### Automated Verification

After executing the removal:

1. **Compilation Check**:
   ```bash
   go build -v ./cmd/bibx
   ```
   Ensure the binary builds without errors and contains no references to `init` or `scaffold`.

2. **Test Suite**:
   ```bash
   go test -race -timeout 5s -v ./...
   ```
   All remaining unit tests across `models`, `algorithms`, `internal`, and `cli` must pass.

3. **Code Quality and Linter**:
   ```bash
   golangci-lint run
   gofmt -d . && test -z "$(gofmt -l .)"
   go mod tidy -diff
   ```

### Manual CLI Verification

1. **Help Output**:
   ```bash
   ./bibx --help
   ```
   Confirm that the `COMMANDS` list includes `inquire`, `query`, `search`, `view`, and **not** `init`.

2. **Regression Check on Core Commands**:
   - Run a test search / query on mock or sample data to guarantee no unintended side-effects on `cli/services` or `cli/repos`.
