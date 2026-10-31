# go-bibx

[![Go](https://github.com/coreofscience/go-bibx/actions/workflows/go.yml/badge.svg)](https://github.com/coreofscience/go-bibx/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/coreofscience/go-bibx.svg)](https://pkg.go.dev/github.com/coreofscience/go-bibx)

`go-bibx` builds and analyses citation networks with the **Tree of Science
(ToS)** method. It fetches works from [OpenAlex](https://openalex.org/), stores
them in a local SQLite database, and works as a command-line tool, an
[MCP server](#mcp-server) for AI agents, and a Go library. Maintained by
[Core of Science](https://coreofscience.org/).

## The Tree of Science

The **SAP algorithm** uses the topology of the citation graph to map how a
research field evolved. It sorts works into:

- **Roots**: Seminal works that many citation chains in the collection end at as
  their original source.
- **Trunk**: Structural works where the topic took its current shape, joining
  two or more roots.
- **Leaves**: Current, state-of-the-art works whose citation chains lead back to
  the roots.

Roots can look a bit off topic. Topics seldom start under their conventional
modern names, so the works they grew from may use other terms or come from
neighbouring fields: for "apache kafka", roots include ZooKeeper and MapReduce.
Judge a root by whether the topic grew from it, not by whether it mentions the
query. Generic method citations, such as reporting guidelines, are the
exception: many papers cite them, but the topic did not grow from them.

## Installation

Download a binary from
[GitHub Releases](https://github.com/coreofscience/go-bibx/releases), or install
with Go 1.27+:

```bash
go install github.com/coreofscience/go-bibx/cmd/bibx@latest
```

## CLI Usage

```bash
# Fetch works on a topic, run the analysis, and save a named collection
bibx inquire "quantum machine learning" --limit 300 --name qml

# Top works by category in a collection (defaults to the latest)
bibx query --category root --limit 10 --collection qml --format reference

# Keyword search, returning the matches and the citation paths connecting them
bibx search "entanglement" --collection qml

# Explore the citation graph in the browser at http://localhost:8080
bibx view --collection qml

# Export a collection as BibTeX
bibx collection export qml --output qml.bib

# Manage collections, by name or UUID
bibx collection list
bibx collection show qml
bibx collection delete qml
```

`inquire` matches every term of the query against titles and abstracts, so use
one concept of 1-3 words per call. Results are cached per query and limit; pass
`--force` to fetch them again.

Collections live in `.bibx/bibx.db` under the current directory. Use the global
flags `--root` or `--db`, placed before the subcommand, to change that. Run
`bibx <command> --help` for all the options and output formats.

## MCP Server

`bibx mcp` serves the same collections to AI agents over stdio, using the
[Model Context Protocol](https://modelcontextprotocol.io/). Add it to your
client's configuration:

```json
{
  "mcpServers": {
    "bibx": {
      "command": "bibx",
      "args": ["--db", "/absolute/path/to/bibx.db", "mcp"]
    }
  }
}
```

Pass an absolute `--db` path, since clients may start the server from any
directory.

| Tools                    | Purpose                                                     |
| ------------------------ | ----------------------------------------------------------- |
| `inquire`                | Fetch a topic from OpenAlex and store it as a collection    |
| `list`, `show`, `delete` | Manage collections                                          |
| `tos`                    | Top works by Tree of Science category                       |
| `search`                 | Keyword search returning a connected citation subgraph      |
| `works`                  | Full metadata and abstracts by OpenAlex ID, DOI, or label   |
| `bibtex`                 | Export selected works, or a preset such as `tos`, to BibTeX |

| Resources                        | Purpose                                  |
| -------------------------------- | ---------------------------------------- |
| `bibx:/summary`                  | The Tree of Science method               |
| `bibx:/collections/{id}/summary` | Metadata of a collection                 |
| `bibx:/collections/{id}/bibtex`  | Full BibTeX bibliography of a collection |

| Prompts  | Purpose                            |
| -------- | ---------------------------------- |
| `review` | Run a literature review on a topic |
| `gaps`   | Find research gaps in a field      |

## Library Usage

Import the public packages to build on `go-bibx`: `services` for the domain
operations, `repository` for the database, `clients/openalex` for the OpenAlex
API, `algorithms` for the graph algorithms, and `models` for the shared types.
See the
[package documentation](https://pkg.go.dev/github.com/coreofscience/go-bibx).

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for the workflow and
[docs/adrs](docs/adrs) for the architecture decisions.

## License

This project is licensed under the MIT License.
