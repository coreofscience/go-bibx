# Implementation Plan: Remove AI Setup

## Executive Summary

The purpose of this plan is to design the complete removal of the existing AI
setup from the `go-bibx` codebase.

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

By removing the scaffolding mechanism, `go-bibx` eliminates workspace pollution
and maintenance overhead, keeping the CLI focused on its core bibliographic
graph analysis capabilities.

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
   - `cli/services` will retain only `analysis.go` (`AnalysisService` and
     `OpenAlexAnalysisService`) and `analysis_test.go`.

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
   - Verify `README.md` and `CONTRIBUTING.md` (confirm no references to `bibx
     init` or AI scaffolding exist).

---

## Verification & Testing Plan

### Automated Verification

After executing the removal:

1. **Compilation Check**:
   ```bash
   go build -v ./cmd/bibx
   ```
   Ensure the binary builds without errors and contains no references to `init`
   or `scaffold`.

2. **Test Suite**:
   ```bash
   go test -race -timeout 5s -v ./...
   ```
   All remaining unit tests across `models`, `algorithms`, `internal`, and `cli`
   must pass.

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
   Confirm that the `COMMANDS` list includes `inquire`, `query`, `search`,
   `view`, and **not** `init`.

2. **Regression Check on Core Commands**:
   - Run a test search / query on mock or sample data to guarantee no unintended
     side-effects on `cli/services` or `cli/repos`.
