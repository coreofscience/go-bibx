---
name: bibx-inquire
description: >-
  Use the `bibx inquire` command to search for research topics, fetch
  bibliographic data from the OpenAlex API, and build semantic search graphs.
---

# bibx-inquire

This skill enables the agent to search for academic and research topics, fetch
their corresponding bibliographic metadata, and generate a citation graph using
the `bibx inquire` command.

## When to Use

Use this skill when the user asks to:

- Fetch papers or bibliographic data on a research topic or field.
- Inquire or search for a research topic using `bibx`.
- Search the OpenAlex API for a specific query.
- Create or download a new bibliographic collection and search database.

## Prerequisites

- The `bibx` command-line package must be installed and available in the
  system's `PATH`.

## Command Syntax

```bash
bibx inquire "<query>" [flags]
```

### Arguments

- `<query>` (String, Required): The search query or topic to inquire (e.g.,
  `"bibliometric analysis"` or `"graph neural networks"`).

### Flags

- `--limit <int>`: The number of initial results to fetch from the API.
  (Default: `200`).
- `--force`: Force overwrite of existing collection and search graph files if
  they already exist in the target directory.

### Global Flags

- `--root <path>`: The path to the root directory where the `.bibx` folder will
  be located. (Default: `.`).
- `--verbose`: Enable verbose logging to troubleshoot performance or execution
  issues.

## Expected Outputs

Running this command generates the following compressed files under the
`.bibx/` subdirectory of the active root directory (usually `.bibx/` in the
workspace root):

1. **`.bibx/collection.json.gz`**: A compressed JSON file containing the
   retrieved academic publications, citation graphs, and associated metadata.
2. **`.bibx/search.json.gz`**: A compressed semantic search index generated
   from the collected papers, enabling subsequent semantic graph searches.

## Examples

### 1. Inquire about a topic with default settings

To fetch papers on "deep learning in agriculture" with the default limit of
200:

```bash
bibx inquire "deep learning in agriculture"
```

### 2. Fetch a specific number of papers

To fetch up to 100 papers on "quantum computing":

```bash
bibx inquire "quantum computing" --limit 100
```

### 3. Force overwrite an existing collection

If a collection already exists and you want to start a new inquiry on the same
or a different topic:

```bash
bibx inquire "large language models" --force
```

### 4. Direct output to a custom root directory

To store the generated `.bibx` files in a custom folder (e.g., `./data`):

```bash
bibx inquire "sustainability" --root ./data
```

## Troubleshooting

- **Error: `file already exists`**: This occurs if `.bibx/collection.json.gz`
  or `.bibx/search.json.gz` is already present. Either back up the old files
  and delete them, or use the `--force` flag to overwrite them.
- **Failed to store search results / embedding errors**: Ensure `ollama` is
  running locally and the semantic search model has been set up via `bibx
  setup` prior to running the inquiry.
