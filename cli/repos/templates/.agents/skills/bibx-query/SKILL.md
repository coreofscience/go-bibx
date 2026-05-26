---
name: bibx-query
description: >-
  Use the `bibx query` command to query a bibliographic collection and
  filter/render results based on Tree of Science categories (root, trunk,
  leaf).
---

# bibx-query

This skill enables the agent to query an existing bibliographic collection
using the `bibx query` command. Results are filtered by their Tree of Science
structural category (roots, trunk, leaves) and rendered in various formats.

## When to Use

Use this skill when the user asks to:

- Query, list, or retrieve articles from a `bibx` collection.
- Filter research papers by Tree of Science categories: `root`, `trunk`, or
  `leaf`.
- Get seminal/foundational works (`root`), structural/backbone works (`trunk`),
  or recent developments (`leaf`) of a field.
- Render paper lists in formatted JSON, plain text (`simple`), markdown, or
  academic `reference` style.

## Prerequisites

- The `bibx` command-line package must be installed and available in the
  system's `PATH`.
- A valid bibliographic collection (by default `.bibx/collection.json.gz`) must
  have already been created using `bibx inquire`.

## Command Syntax

```bash
bibx query --category <category> [flags]
```

### Required Flags

- `--category <string>`: The Tree of Science category to filter by. Must be one of:
  - `root`: Seminal, foundational works that established the field.
  - `trunk`: Core papers providing the structural backbone and consolidating the topic.
  - `leaf`: Recent, specialized developments representing the research frontier.

### Optional Flags

- `--limit <int>`: The maximum number of results to return. (Default: `5`).
- `--format <string>`: The output format for rendering results. (Default: `json`). Supported values:
  - `json`: Standard JSON-serialized array of articles.
  - `simple`: A lightweight, plain-text list of titles and authors.
  - `markdown`: A structured Markdown list/table suitable for reports.
  - `reference`: Academic reference style.

### Global Flags

- `--root <path>`: The path to the root directory where the `.bibx` folder is located. (Default: `.`).
- `--verbose`: Enable verbose logging to troubleshoot query and parsing issues.

## Expected Outputs

The results are printed directly to the standard output (`stdout`) in the requested format.

## Examples

### 1. Retrieve the top 5 seminal works (roots) in JSON format

```bash
bibx query --category root
```

### 2. Retrieve the top 10 structural papers (trunk) in Markdown format

```bash
bibx query --category trunk --limit 10 --format markdown
```

### 3. Retrieve the top 3 recent frontiers (leaves) in academic reference style

```bash
bibx query --category leaf --limit 3 --format reference
```

### 4. Query a collection located in a custom root directory

```bash
bibx query --category root --root ./data --format simple
```

## Troubleshooting

- **Error: `analysis path not set` or `file does not exist`**: This occurs if
  the collection has not been built yet. Make sure you run `bibx inquire` on
  your topic first to generate `.bibx/collection.json.gz`.
- **Error: `invalid category`**: Double-check that `--category` is spelled
  correctly and is one of `root`, `trunk`, or `leaf`.
- **Error: `invalid format`**: Double-check that `--format` is one of `json`,
  `simple`, `markdown`, or `reference`.
