# go-bibx

`go-bibx` is a high-performance command-line tool designed for researchers to
perform deep, graph-based bibliographic analysis. Maintained by the [Core of
Science](https://coreofscience.org/) organization, it serves as a Go
implementation of the **Tree of Science (ToS)** methodology.

The project is built to be both a standalone tool for quick academic inquiry and
a robust component for larger data pipelines, offering high interoperability
through its binary distribution.

## The Tree of Science & SAP Algorithm

At its core, `go-bibx` implements the **SAP algorithm** (found in
`algorithms/sap.go`), which analyzes citation graph topology to map the
evolutionary structure of a research field. Using the "Tree of Science"
metaphor, it categorizes articles into:

- **Roots**: Foundational, seminal works that established the field.
- **Trunk**: Core papers that provide the structural backbone and consolidate
  the topic.
- **Leaves**: Recent, specialized developments representing the current frontier
  of research.

## Features

- **Inquire**: Efficiently fetch bibliographic data from the OpenAlex API.
- **SAP Analysis**: Automatically derive the Tree of Science structure from any
  citation network.
- **Visualize**: Interactive, browser-based graph exploration of your
  collections.
- **Graph Semantic Search**: Find information in your collection via a combined
  semantic search and graph approach.
- **Queries**: Query your collection by relevance in three main categories:
  - *Rootness*: how seminal the article is.
  - *Trunkness*: how important is this article within the structure of the
    topic.
  - *Leafness*: how complete and recent the article is.
- **Interoperable**: Designed as a standalone CLI that fits perfectly into
  automated workflows.

## Installation

Ensure you have Go installed, then clone the repository and build the project:

```bash
git clone https://github.com/coreofscience/go-bibx.git
cd go-bibx
go build -o bibx ./cmd/bibx
```

Note: `go-bibx` depends on `ollama` running locally to generate embeddings for semantic search.
Users will need to run the setup command once to download the model:

```bash
./bibx setup
```

## Usage

### 1. Inquire a research topic

Fetch papers related to a specific topic and save them to a compressed JSON
file:

```bash
./bibx inquire "bibliometric analysis" --limit 100
```

This will create a collection file at `.bibx/collection.json.gz`.

### 2. View the collection

Visualize the generated collection in your browser:

```bash
./bibx view --file .bibx/collection.json.gz --port 8080
```

Open [http://localhost:8080](http://localhost:8080) to explore the interactive graph.

### 3. Search

Perform a semantic search to find relevant articles:

```bash
./bibx search "your query"
```

### 4. Query

Query your collection by relevance in three main categories (e.g., root, trunk, leaf):

```bash
./bibx query --category root
```

## Development

- **Run with Air**: For live-reloading during development, use [Air](https://github.com/air-verse/air).
- **Testing**: Run tests with `go test ./...`.
- **Linting**: Use `golangci-lint run`.

## License

This project is licensed under the MIT License.
