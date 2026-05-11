# go-bibx

`go-bibx` is a command-line tool for managing and analyzing bibliographic
collections using Go. It allows you to fetch data from OpenAlex, perform
bibliometric analysis, and visualize the resulting citation networks.

## Features

- **Inquire**: Fetch bibliographic data from OpenAlex based on a search query.
- **Analyze**: Automatically process collections (remove cycles, find giant
  components, and enrich data).
- **Visualize**: Start a local web server to explore the collection graph
  interactively.

## Features to be implemented

- **Graph semantic search** Find information in your collection via a combined
  semantic search and graph approach.
- **Queries** query your collection by relevance in tree main categories:
  - *Rootness*: how seminal the article is.
  - *Trunkness*: how important is this article within the structure of the
    topic.
  - *Leafness*: how complete and recent the article is.

## Installation

Ensure you have Go installed, then clone the repository and build the project:

```bash
git clone https://github.com/coreofscience/go-bibx.git
cd go-bibx
go build -o bibx ./cmd/bibx
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

TBD...

### 4. Query

TBD...

## Development

- **Run with Air**: For live-reloading during development, use [Air](https://github.com/air-verse/air).
- **Testing**: Run tests with `go test ./...`.
- **Linting**: Use `golangci-lint run`.

## License

This project is licensed under the MIT License.
