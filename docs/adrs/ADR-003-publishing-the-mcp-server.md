# ADR-003: Publishing the MCP Server

## Status

Accepted. Supersedes the part of ADR-001 that placed the MCP server under
`internal/`.

## Context

ADR-001 kept both delivery channels, the CLI and the MCP server, under
`internal/`, so that external code could not couple to them. The only way to
run the MCP server was `bibx mcp`, over stdio, against a single SQLite file.

Other applications need to serve the same tools themselves, for example over
HTTP with a database per user. They could only do that by copying the server
setup and every tool, because none of it could be imported.

## Decision

The MCP server becomes a public package, `mcp`, with a single entry point:

```go
func NewServer(
	collectionService services.CollectionService,
	analysisService services.AnalysisService,
	opts ...Option,
) *mcp.Server
```

### 1. One constructor, not the tools

`NewServer` registers every tool, resource and prompt and returns the SDK's
`*mcp.Server`. The tools, resources, prompts and their response types move to
`mcp/internal/`, so only the `mcp` package can import them.

The tool schemas are already a contract with MCP clients. Exposing the Go types
behind them would add a second contract, which every change to a tool would
have to respect.

### 2. Services in, transport out

The server takes the public service interfaces, so the caller decides how they
are built: which database, which OpenAlex client, and any wrapper around them.
Building one server per request or per user is how an application gives each
user their own data.

The server does not choose a transport. `bibx mcp` runs it over stdio, and other
applications can serve it with the SDK's streamable HTTP handler.

### 3. Options

`WithLogger` and `WithVersion` cover what the CLI used to hard-code. The
reported version defaults to the version of the `go-bibx` module in the build,
instead of a fixed string.

## Alternatives Considered

- **Publish the whole `internal/mcp` tree**: lets callers pick single tools, but
  makes every tool struct and response type public API.
- **Take a `*sql.DB` instead of services**: would let the services become
  internal, but they are the useful part of `go-bibx` as a library, and callers
  could no longer wrap them.

## Consequences

### Positive

- **Reusable server**: other applications can serve the bibx tools with their
  own transport, authentication and databases.
- **One definition**: the instructions, capabilities and registration live in
  one place that the CLI also uses.

### Negative / Trade-offs

- **No selection of tools**: callers get every tool, including `delete`.
  Choosing tools would need a new option.
- **Name clash**: the package shares its name with the SDK's `mcp`, so code that
  imports both must alias one of them.
