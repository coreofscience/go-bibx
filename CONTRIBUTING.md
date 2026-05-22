# Contributing to go-bibx

First off, thank you for considering contributing to `go-bibx`! It's people like you that make open source tools great.

The following is a set of guidelines for contributing to `go-bibx`, which is hosted in the [Core of Science](https://github.com/coreofscience) Organization on GitHub. These are mostly guidelines, not rules. Use your best judgment, and feel free to propose changes to this document in a pull request.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. We expect all contributors to maintain a welcoming and inclusive environment.

## Getting Started

### Prerequisites

*   **Go**: The project requires Go 1.26 or newer.
*   **Ollama**: `go-bibx` depends on `ollama` running locally to generate embeddings for semantic search.
*   **Make** (optional, but recommended if we add a Makefile later).
*   **Git**: For version control.

### Setting up the Development Environment

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/coreofscience/go-bibx.git
    cd go-bibx
    ```

2.  **Build the application**:
    ```bash
    go build -o bibx ./cmd/bibx
    ```

3.  **Setup the application**:
    Users will need to run the setup command once to download the necessary models:
    ```bash
    ./bibx setup
    ```

## Development Process

### Branches and Pull Requests

1.  **Fork the repository** and create your branch from `main`.
    ```bash
    git checkout -b my-feature-branch
    ```
2.  If you've added code that should be tested, **add tests**.
3.  If you've changed APIs or features, **update the documentation**.
4.  Ensure the test suite passes (`go test ./...`).
5.  Make sure your code passes the linter (`golangci-lint run`).
6.  Issue that pull request!

### Local Development

For a better development experience, we recommend using [Air](https://github.com/air-verse/air) for live-reloading:

```bash
air
```

### Testing

Run the test suite before submitting any changes:

```bash
go test -v ./...
```

### Linting

We use `golangci-lint` to maintain code quality. Run it locally before submitting your PR:

```bash
# If you don't have it installed:
# go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.0

golangci-lint run
```

### Formatting

Go provides an automatic formatter. Ensure your code is formatted before committing:

```bash
go fmt ./...
```

## Reporting Bugs

*   Ensure the bug was not already reported by searching on GitHub under Issues.
*   If you're unable to find an open issue addressing the problem, open a new one. Be sure to include a title and clear description, as much relevant information as possible, and a code sample or an executable test case demonstrating the expected behavior that is not occurring.

## Suggesting Enhancements

*   Open a new Issue and describe the enhancement you would like to see.
*   Provide a clear explanation of why this enhancement would be useful to most users.

## Questions?

If you have questions, feel free to open an issue or reach out to the maintainers.

Thank you for contributing!
