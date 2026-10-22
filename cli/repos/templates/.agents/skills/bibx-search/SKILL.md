---
name: bibx-search
description: >-
  Use the `bibx search` command to perform keyword searches on bibliographic
  collections, render search results, and visualize them using an interactive
  browser-based graph.
---

# bibx-search

This skill enables the agent to perform a keyword, graph-based search on a
bibliographic collection using the `bibx search` command. Results can be
rendered in various text-based formats or served on an interactive,
browser-based visualization server.

## When to Use

Use this skill when the user asks to:

- Perform a keyword or graph-based search on an existing `bibx` collection.
- Find papers relevant to keywords, author names, or concepts.
- Visualize search results or collections inside a web browser.
- Render search results in formatted JSON, simple text, markdown, or academic
  references.

## Prerequisites

- The `bibx` command-line package must be installed and available in the
  system's `PATH`.
- A valid bibliographic analysis file (`.bibx/analysis.json.gz`) must have
  already been created using `bibx inquire`.

## Command Syntax

```bash
bibx search "<query>" [flags]
```

### Arguments

- `<query>` (String, Required): The keyword search query (e.g.,
  `"applications of transformer models"`).

### Flags

- `--limit <int>`: The maximum number of top results to start the graph algorithm with. (Default: `5`).
- `--format <string>`: The output format for rendering results. (Default: `simple`). Supported values:
  - `simple`: A lightweight, plain-text list of titles and authors.
  - `filename`: Prints the target filenames.
  - `json`: Standard JSON-serialized array of search results with scores and metadata.
  - `markdown`: A structured Markdown list/table suitable for reports.
  - `reference`: Academic reference style.
- `--view`: Start a local HTTP server to host an interactive, browser-based visualization of the search results graph.
- `--port <int>`: The port to serve the visualization on if `--view` is enabled. (Default: `8080`).

### Global Flags

- `--root <path>`: The path to the root directory where the `.bibx` folder is
  located. (Default: `.`).
- `--verbose`: Enable verbose logging to troubleshoot search issues.

## Expected Outputs

- By default, search results are printed directly to the standard output
  (`stdout`) in the requested format.
- If `--view` is passed, the CLI starts a visualization server at
  `http://localhost:<port>` to host an interactive graph.

## Examples

### 1. Simple keyword search on "reinforcement learning"

```bash
bibx search "reinforcement learning"
```

### 2. Keyword search with custom limit and Markdown format

```bash
bibx search "explainable AI in medicine" --limit 10 --format markdown
```

### 3. Launch an interactive browser visualization of the search results

To search and immediately explore the results on a graph locally at `http://localhost:8081`:

```bash
bibx search "quantum cryptography" --view --port 8081
```

### 4. Search within a custom root directory
```bash
bibx search "synthetic biology" --root ./data --format reference
```

## Troubleshooting

- **Error: `analysis path not set` or `file does not exist`**: This occurs if
  the collection and analysis have not been built. Run `bibx inquire` on your
  topic first to generate `.bibx/analysis.json.gz`.
- **Port Collisions**: If you see a "port already in use" error when using
  `--view`, specify a different port with `--port <number>` (e.g. `--port
  8082`).
