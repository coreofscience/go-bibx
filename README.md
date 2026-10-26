# go-bibx

[![Go](https://github.com/coreofscience/go-bibx/actions/workflows/go.yml/badge.svg)](https://github.com/coreofscience/go-bibx/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/coreofscience/go-bibx.svg)](https://pkg.go.dev/github.com/coreofscience/go-bibx)

`go-bibx` is a high-performance command-line tool and Go library designed for researchers to
perform deep, graph-based bibliographic analysis. Maintained by the
[Core of Science](https://coreofscience.org/) organization, it serves as a Go
implementation of the **Tree of Science (ToS)** methodology.

The project is built to be both a standalone CLI tool for academic inquiry and
a robust library for larger data pipelines, featuring a pure-Go embedded SQLite storage engine
with zero CGO requirements.

## The Tree of Science & SAP Algorithm

At its core, `go-bibx` implements the **SAP algorithm** (`algorithms/sap.go`), which analyzes citation graph topology to map the evolutionary structure of a research field. Using the "Tree of Science" metaphor, it categorizes articles into:

- **Roots**: Foundational, seminal works that established the field.
- **Trunk**: Core papers that provide the structural backbone and consolidate the topic.
- **Leaves**: Recent, specialized developments representing the current frontier of research.

## Features

- **Multi-Collection SQLite Storage**: Persistent storage in `.bibx/bibx.db` powered by a pure-Go driver (`modernc.org/sqlite`) with embedded database migrations. Multiple inquiries can coexist in a single database.
- **Query Idempotency**: Deterministic query hashing prevents redundant OpenAlex API calls and recalculations when re-running identical inquiries.
- **Clean 3-Tier Architecture**: Built with clear architectural boundaries separating public data persistence (`repository`), domain services (`services`), and internal presentation/delivery (`internal/cli`, `internal/mcp`).
- **Inquire**: Fetch literature and citation networks directly from the OpenAlex API and run SAP and community detection algorithms.
- **Collection Management**: Inspect, list, and delete inquiry collections by UUIDv7 or human-readable name aliases.
- **Visualize**: Interactive, browser-based graph exploration of any collection.
- **Keyword Search**: Graph search combining SQLite FTS5 (BM25 ranking) with pseudo-Steiner graph reduction to isolate subnetworks related to specific keywords.
- **ToS Category Queries**: Filter and rank articles by relevance across Tree of Science categories (*Root*, *Trunk*, *Leaf*).
- **Community Detection**: High-performance Leiden algorithm for detecting thematic clusters in citation networks.
- **Pure Go & Cross-Platform**: Fully builds and runs with `CGO_ENABLED=0` on Linux, macOS, and Windows.

## Installation

### Via `go install`

```bash
go install github.com/coreofscience/go-bibx/cmd/bibx@latest
```

### Pre-built Binaries

You can download pre-built binaries for your operating system and architecture from the [GitHub Releases](https://github.com/coreofscience/go-bibx/releases) page.

### Build from Source

Ensure you have Go 1.27+ installed, then clone the repository and build:

```bash
git clone https://github.com/coreofscience/go-bibx.git
cd go-bibx
CGO_ENABLED=0 go build -o bibx ./cmd/bibx
```

## CLI Usage

### Global Options

`bibx` supports global flags across all subcommands:

- `--db string`: Custom path to the SQLite database file (defaults to `.bibx/bibx.db` relative to root).
- `--root, -r string`: Root working directory (defaults to current directory `.`).
- `--verbose`: Enable debug logging.

### 1. Inquire a research topic

Fetch papers related to a specific topic, compute Tree of Science metrics and Leiden communities, and save them as a collection:

```bash
# Basic inquiry (defaults to limit 200)
bibx inquire "quantum machine learning" --limit 100

# Name the collection for easy reference
bibx inquire "deep learning transformers" --limit 100 --name "transformers"

# Force re-fetching an existing query
bibx inquire "quantum machine learning" --limit 100 --force
```

If an inquiry with the same normalized query and limit has already been executed, `bibx` detects the existing collection and skips remote network requests unless `--force` (`-f`) is passed.

### 2. Manage collections

List, inspect, and delete collections within your database:

```bash
# List all collections
bibx collection list

# Show collection details (by name alias or UUIDv7, defaults to latest)
bibx collection show transformers
bibx collection show 019a1234-5678-7000-8000-123456789abc

# Delete a collection (cascades to associated nodes and links)
bibx collection delete transformers
```

### 3. Query by Tree of Science category

Query top articles in a collection by category score (`root`, `trunk`, or `leaf`):

```bash
# Query the latest collection
bibx query --category root --limit 10 --format reference

# Query a specific collection by name or ID
bibx query --category trunk --limit 5 --collection transformers --format markdown
```

Supported output formats: `json` (default), `markdown`, `reference`, `simple`.

### 4. Graph Search

Perform a keyword search using full-text search (SQLite FTS5) combined with pseudo-Steiner graph reduction:

```bash
# Search and output simple summary
bibx search "attention mechanism" --limit 10 --collection transformers

# Search and launch interactive browser visualization
bibx search "attention mechanism" --limit 10 --view --port 8081
```

Supported output formats: `simple` (default), `json`, `markdown`, `reference`, `filename`.

### 5. View the collection

Launch an interactive browser-based visualization for any collection:

```bash
# View the latest collection
bibx view --port 8080

# View a specific collection
bibx view --collection transformers --port 8080
```

Open [http://localhost:8080](http://localhost:8080) to explore the interactive citation graph.

## Library Usage

`go-bibx` can be imported as a Go library. The public `repository` and `services` packages provide easy access to the database lifecycle and domain services:

```go
package main

import (
	"context"
	"fmt"

	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
)

func main() {
	ctx := context.Background()

	// Open or create SQLite database with embedded migrations
	db, err := repository.OpenAndMigrateSQLite(".bibx/bibx.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Initialize collection service
	collectionService := services.NewCollectionService(db)

	// List collections
	collections, err := collectionService.List(ctx)
	if err != nil {
		panic(err)
	}

	for _, col := range collections {
		fmt.Printf("Collection: %s, Articles: %d\n", col.ID, col.ArticleCount)
	}
}
```

## Development

- **Run with Air**: For live-reloading during development, use [Air][air].
- **Testing**: Run tests with `CGO_ENABLED=0 go test ./...`.
- **Linting**: Run `golangci-lint run`.
- **SQL Queries (sqlc)**: Regenerate typed database queries with `sqlc generate`.

## License

This project is licensed under the MIT License.

[air]: https://github.com/air-verse/air
