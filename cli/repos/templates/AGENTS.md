# Managing Bibliographic Collections with Agents & Skills

This workspace is equipped with **Agents** and specialized **Skills** to build,
query, search, and visualize academic literature and bibliographic collections
using the `bibx` CLI tool.

## Agents Directory

We have two primary agent profiles available for managing this bibliographic
collection. These agents can be invoked or delegated to for
parallel/specialized tasks.

### 1. Primary Agent

The lead coordinator agent designed for pair programming, planning, and task
execution. Antigravity initiates the implementation plans, coordinates
subagents, and applies skills directly to construct or query collections.

### 2. Research Subagent (`research`)

A specialized, read-only subagent designed to explore the codebase, search the
web, and read literature or documentation.

* **When to Delegate:** Use `research` for high-latency or search-intensive
  background tasks (e.g., performing a web search or summarizing codebase
  files) so the primary agent can continue executing commands.
* **Capabilities:** Equipped with search and view tools.

### 3. Self Subagent (`self`)

A cloned subagent that inherits the primary agent's entire configuration,
tools, system prompt, and LLM.

* **When to Delegate:** Use `self` when you need to execute separate complex
  processes in isolated branches/workspaces without cluttering the primary chat
  history.

## Specialized Skills

The workspace contains three key skills under the
[.agents/skills/](.agents/skills) directory. These skills encapsulate the
command-line usage of the `bibx` tool.

| Skill Name | Description | Command | Link to Skill |
| :--- | :--- | :--- | :--- |
| **`bibx-inquire`** | Searches research topics, fetches OpenAlex API bibliographic data, and builds semantic graphs. | `bibx inquire "<query>"` | [bibx-inquire SKILL.md](file:///home/oscar/Code/experiments/bpm/.agents/skills/bibx-inquire/SKILL.md) |
| **`bibx-query`** | Queries and filters local collections based on Tree of Science categories (roots, trunk, leaves). | `bibx query --category <cat>` | [bibx-query SKILL.md](file:///home/oscar/Code/experiments/bpm/.agents/skills/bibx-query/SKILL.md) |
| **`bibx-search`** | Performs semantic search on a collection and starts a browser-based visualization server. | `bibx search "<query>"` | [bibx-search SKILL.md](file:///home/oscar/Code/experiments/bpm/.agents/skills/bibx-search/SKILL.md) |

---

## How to Use the Skills

Here is a step-by-step workflow of how the agents manage the bibliographic collection using the skills:

### 1. Inquire (Building the Collection)

Use the [bibx-inquire SKILL.md](.agents/skills/bibx-inquire/SKILL.md) to
bootstrap a brand new research topic. This generates `.bibx/collection.json.gz`
and `.bibx/search.json.gz`.

```bash
# Example: Bootstrap a collection for graph neural networks
bibx inquire "graph neural networks" --limit 150
```

> [!TIP]
> Use the `--force` flag if you want to overwrite an existing collection and start fresh.

### 2. Query (Structural Classification)

Use the [bibx-query SKILL.md](.agents/skills/bibx-query/SKILL.md) to parse the
collection based on the **Tree of Science** structure:

- **`root`**: The seminal, foundational papers that established the field.
- **`trunk`**: The backbone papers connecting foundation to modern
  applications.
- **`leaf`**: The modern frontiers, recent developments, and current research
  boundary.

```bash
# Example: Query the top 10 foundational papers in markdown format
bibx query --category root --limit 10 --format markdown
```

### 3. Search & Visualize (Semantic Exploration)

Use the [bibx-search SKILL.md](.agents/skills/bibx-search/SKILL.md) to query
concepts semantically or view the citation graph in a web browser.

```bash
# Example: Launch the web-based visualization server on port 8080
bibx search "attention mechanisms" --view --port 8080
```

> [!NOTE]
> Make sure `ollama` is running locally and the embedding model has been set up
> via `bibx setup` before using semantic search functions.
