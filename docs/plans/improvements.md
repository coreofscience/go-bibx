# Codebase Evaluation & Recommended Improvements

**Project:** `go-bibx`  
**Date:** October 2026  
**Document Status:** Analysis & Recommendations (Pending Implementation)

---

## Executive Summary

An evaluation of the `go-bibx` codebase was conducted, reviewing correctness,
performance, concurrency, architecture, test coverage, and documentation. While
the codebase has a clean package layout, modern Go idiomatic practices (Go 1.26
iterators, slices package, structured logging with `slog`), and well-scoped
modules, several critical bugs, subtle algorithmic flaws, and operational gaps
were identified that should be resolved before public release or broader
adoption.

The most critical findings include:

1. **Catastrophic Embedding Slice Misallocation**: Articles receive the
   embedding vector of an empty string `""` due to `append()` on a pre-allocated
   slice, discarding real embeddings.
2. **Semantic Search Discards Exact Matches**: `models.Analysis.Search`
   explicitly skips results with `score == 0` (which corresponds to an identical
   match in cosine distance).
3. **Nil Pointer Dereference in `Collection.Keep`**: Dereferencing
   `*article.Key()` without nil checks when article IDs are empty.
4. **Union-by-Rank Invariant Violation in `UnionFind`**: Increments the rank of
   the merged child instead of the new root.
5. **Premature Steiner Tree Search Abort**: Inappropriately breaking topological
   iteration when distance exceeds `maxEffort` under a false monotonicity
   assumption.
6. **Zero Test Coverage across 14 Packages**: Core packages including
   `cli/services`, `models/analysis.go`, `cli/repos`, `internal/texter`,
   `internal/vector`, and `internal/utils` lack unit tests.

Below is the detailed catalog of findings, categorized by impact and accompanied
by suggested remediations.

---

## 1. Critical Logic Errors & Algorithmic Bugs

### 1.1 Catastrophic Embedding Slice Allocation Bug

* **Location:** `cli/services/analysis.go:107-120`
* **Severity:** Critical
* **Status:** Resolved
* **Problem:** 
  In `OpenAlexAnalysisService.Store`:
  ```go
  texts := make([]string, collection.Len())
  collectionArticles := slices.Collect(collection.All())
  for _, article := range collectionArticles {
      texts = append(texts, s.texter.ExtractText(article))
  }
  vectors, err := s.embeddingClient.EmbedMany(ctx, texts)
  ```
  `make([]string, collection.Len())` initializes `texts` with $N$ empty strings
  `""`. The subsequent `append` calls place the actual article texts at indices
  $N$ through $2N-1$. 
  
  When `utils.Zip(collectionArticles, vectors)` runs:
  ```go
  for article, embedding := range utils.Zip(collectionArticles, vectors)
  ```
  `utils.Zip` stops at length $N$ (the length of `collectionArticles`). As a result:
  - Every single article is assigned the embedding of the empty string `""`.
  - Ollama computes twice as many embeddings as needed.
  - The actual embeddings for the articles are computed and immediately discarded.
* **Suggested Fix:**
  Change the allocation to specify capacity rather than length:
  ```go
  texts := make([]string, 0, collection.Len())
  ```

---

### 1.2 Perfect Semantic Search Matches Are Skipped (`score == 0`)

* **Location:** `models/analysis.go:259-262`
* **Severity:** High
* **Status:** Resolved
* **Problem:**
  In `Analysis.Search`:
  ```go
  for _, node := range a.Nodes {
      score := vector.CosineDistance(node.Embedding, vec)
      if score == 0 {
          continue
      }
      results = append(results, result{node: node, score: score})
  }
  ```
  Cosine distance is defined as $1 - \text{CosineSimilarity} \in [0, 2]$. A
  score of `0` denotes an **exact match**. By continuing when `score == 0`,
  articles whose content or title matches the query perfectly (or rounds to 0
  distance) are discarded from the search results.
* **Suggested Fix:**
  Remove the `score == 0` check. Only filter out non-finite floats (e.g. `math.IsNaN(float64(score))`).

---

### 1.3 Nil Pointer Dereference in `Collection.Keep`

* **Location:** `models/collection.go:78-81`
* **Severity:** High
* **Status:** Resolved
* **Problem:**
  ```go
  for _, article := range c.articles {
      if !toKeep.Contains(*article.Key()) {
          continue
      }
  ```
  If an article has an empty ID set (`article.IDs.Len() == 0`), `article.Key()`
  returns `nil`. Direct dereference `*article.Key()` causes a runtime panic. (In
  `Collection.CitationGraph()`, this is guarded by `articleKey := article.Key();
  if articleKey == nil { continue }`, but `Collection.Keep` omitted this check).
* **Suggested Fix:**
  Safely inspect the pointer before dereferencing:
  ```go
  key := article.Key()
  if key == nil || !toKeep.Contains(*key) {
      continue
  }
  ```

---

### 1.4 Broken Union-by-Rank Invariant in `UnionFind`

* **Location:** `internal/union/union.go:49-51`
* **Severity:** Medium
* **Status:** Resolved
* **Problem:**
  In `UnionFind.Union`:
  ```go
  if u.rank[rootA] == u.rank[rootB] {
      u.parent[rootA] = rootB
      u.rank[rootA]++
  }
  ```
  Here, `rootB` is made the new parent of `rootA` (i.e. `rootB` is the root of
  the merged tree). However, the code increments `u.rank[rootA]++` instead of
  `u.rank[rootB]++`. As a consequence, the rank of `rootB` never increases,
  degrading the tree height guarantees of union-find from nearly $O(1)$ to
  unoptimized chains.
* **Suggested Fix:**
  ```go
  u.parent[rootA] = rootB
  u.rank[rootB]++
  ```

---

### 1.5 Premature Search Termination in Steiner Tree Algorithm

* **Location:** `algorithms/steiner.go:270-273` & `algorithms/steiner.go:309-312`
* **Severity:** High
* **Status:** Resolved
* **Problem:**
  Inside `findBestDescendant` and `findBestAncestor`:
  ```go
  for _, w := range q.topologicalOrder[earliestDescendant+1:] {
      ...
      cost := distA + distB
      if cost > q.maxEffort {
          slog.Debug("descendant cost exceeds max effort", "cost", cost, "maxEffort", q.maxEffort)
          break
      }
      ...
  }
  ```
  Topological order ensures that if edge $u \to v$ exists, index($u$) <
  index($v$). It does **not** imply that path distance `distA + distB` is
  monotonic across topological ordering. If an early node in the topological
  order happens to have a high cost, hitting `break` terminates the search
  immediately and misses subsequent vertices in the slice that have costs below
  `maxEffort`.
* **Suggested Fix:**
  Use `continue` instead of `break`, or define effort as the maximum number of
  candidate nodes to evaluate (e.g. `evaluated++ > maxCandidates`).

---

### 1.6 Potential Infinite Recursion & Duplicate Terminals in Steiner Tree

* **Location:** `algorithms/steiner.go:122-135` & `algorithms/steiner.go:252-328`
* **Severity:** Medium
* **Status:** Resolved
* **Problem:**
  `findBestDescendant` and `findBestAncestor` do not exclude vertices that are
  already existing terminals. If the algorithm selects an existing terminal as
  the bridging node, appending it (`newTerminals := append(terminals,
  *bestDescendant)`) introduces duplicate terminals and risks an infinite
  recursion loop in `q.Run(newTerminals)`.
* **Suggested Fix:**
  Pass the terminal set or check `slices.Contains(terminals, labelW)` to ensure
  candidates are strictly non-terminal vertices.

---

### 1.7 Division by Zero & Unclamped Values in `vector.CosineDistance`

* **Location:** `internal/vector/vector.go:9-22`
* **Severity:** Medium
* **Problem:**
  ```go
  return 1 - (dotProduct / (sqrt(normA) * sqrt(normB)))
  ```
  1. If a vector contains all zeros (or is empty), `normA` or `normB` is `0`,
     leading to `0 / 0 = NaN`.
  2. Due to floating-point imprecision, `(dotProduct / (sqrt(normA) *
     sqrt(normB)))` can yield values slightly greater than `1.0` (e.g.
     `1.0000001`), producing negative distance (`-0.0000001`), violating the
     documented $[0, 2]$ range.
* **Suggested Fix:**
  Check for zero norms and clamp the similarity quotient to `[-1, 1]`:
  ```go
  if normA == 0 || normB == 0 {
      return 2.0
  }
  sim := dotProduct / (sqrt(normA) * sqrt(normB))
  if sim > 1.0 { sim = 1.0 } else if sim < -1.0 { sim = -1.0 }
  return 1.0 - sim
  ```

---

### 1.8 Removal of Self-Looping Articles in `RemoveCycles`

* **Location:** `internal/graphs/transformation.go:174-194`
* **Severity:** Medium
* **Problem:**
  Academic metadata occasionally contains self-citations ($A \to A$). In `RemoveCycles`:
  ```go
  for _, edge := range g.AllEdges() {
      if edge.Source().Label() == edge.Destination().Label() {
          selfLoops = append(selfLoops, edge.Source())
      }
  }
  ...
  newGraph, err := Purge(g, toRemove)
  ```
  `Purge` deletes the **entire vertex** from the graph. If an article cites
  itself, all its legitimate citations and its presence in the collection are
  deleted.
* **Suggested Fix:**
  Remove the self-loop edge from the graph rather than purging the vertex and
  article record.

---

## 2. Concurrency, I/O & Operational Robustness

### 2.1 Lack of Graceful Shutdown & Context Propagation for HTTP Servers

* **Location:** `cli/commands/search/search.go:112`, `cli/commands/view/view.go:40`, `cmd/bibx/main.go:38`
* **Severity:** Medium
* **Problem:**
  `http.ListenAndServe(fmt.Sprintf(":%d", port), mux)` is used directly without
  binding to the command `ctx`. In `main.go`, `cmd.Run(context.Background(),
  os.Args)` does not capture SIGINT / SIGTERM signals via
  `signal.NotifyContext`. When a user runs `bibx view` or `bibx search --view`
  and hits Ctrl+C, the server cannot shut down gracefully.
* **Suggested Fix:**
  1. In `main.go`, wrap the root context with
     `signal.NotifyContext(context.Background(), os.Interrupt,
     syscall.SIGTERM)`.
  2. In `view` and `search`, create an `http.Server` and use a goroutine waiting
     on `<-ctx.Done()` to invoke `server.Shutdown(shutdownCtx)`.

---

### 2.2 Superfluous `http.Error` Call After `WriteHeader`

* **Location:** `cli/servers/viewer/server.go:19-25`
* **Severity:** Low
* **Problem:**
  ```go
  w.WriteHeader(http.StatusOK)
  if _, err := w.Write(indexTemplate); err != nil {
      slog.ErrorContext(r.Context(), "failed to write index template", "error", err)
      http.Error(w, "internal server error", http.StatusInternalServerError)
  }
  ```
  Once `w.WriteHeader()` has been called, calling `http.Error()` produces a Go
  runtime warning (`http: superfluous response.WriteHeader call`) and does not
  change the already-sent HTTP status. Also, the `Content-Type: text/html;
  charset=utf-8` header is not set.

---

### 2.3 Non-Atomic File Writes in `StoreGzipJSON`

* **Location:** `internal/formats/gzipjson.go:40-65`
* **Severity:** Medium
* **Problem:**
  `os.Create(fileName)` creates or immediately truncates the target file. If
  writing or compression fails (disk full, process cancelled via SIGINT,
  serialization failure), the previous valid analysis file is destroyed, leaving
  a corrupted or empty file.
* **Suggested Fix:**
  Write to a temporary file (`fileName + ".tmp"`), flush, close, and then use
  `os.Rename` for atomic replacement.

---

### 2.4 Error Message Bug in `cli/repos/scaffold.go`

* **Location:** `cli/repos/scaffold.go:76-78`
* **Severity:** Low
* **Problem:**
  In the defer block closing `srcFile`:
  ```go
  if cerr != nil {
      slog.Error("failed to close source file", "file", srcPath, "error", cerr)
      err = errors.Join(err, fmt.Errorf("failed to close destination file %s: %w", destPath, cerr))
  }
  ```
  The error string says `"failed to close destination file"` when closing the
  embedded source file.

---

### 2.5 Private Machine Paths & Assistant Artifacts in `AGENTS.md` Template

* **Location:** `cli/repos/templates/AGENTS.md:16, 46-48`
* **Severity:** Medium
* **Problem:**
  The embedded `AGENTS.md` scaffolded into user workspaces contains hardcoded local file links:
  - `[bibx-inquire SKILL.md](file:///home/oscar/Code/experiments/bpm/.agents/skills/bibx-inquire/SKILL.md)`
  - `[bibx-query SKILL.md](file:///home/oscar/Code/experiments/bpm/.agents/skills/bibx-query/SKILL.md)`
  - `[bibx-search SKILL.md](file:///home/oscar/Code/experiments/bpm/.agents/skills/bibx-search/SKILL.md)`
  It also references developer-specific AI names ("Antigravity").
* **Suggested Fix:**
  Replace local machine file URLs with relative repository paths, and remove
  development agent artifacts.

---

### 2.6 Server Defaults to Binding on All Network Interfaces

* **Location:** `cli/commands/search/search.go:112`, `cli/commands/view/view.go:40`
* **Severity:** Low
* **Problem:**
  The server listens on `fmt.Sprintf(":%d", port)`, binding `0.0.0.0` (all
  network interfaces). For local CLI development tools, binding to `127.0.0.1`
  by default (with an optional `--host` flag) prevents unintentional exposure on
  local Wi-Fi / networks.

---

## 3. Design, Architecture & Performance Improvements

### 3.1 Misleading Naming & Contract of `WeaklyConnectedComponents`

* **Location:** `internal/graphs/connectivity.go:14-18`
* **Description:**
  ```go
  func WeaklyConnectedComponents[T comparable](g gograph.Graph[T]) ([][]*gograph.Vertex[T], error) {
      if g.IsDirected() {
          return nil, fmt.Errorf("graph is directed")
      }
  ```
  By definition, weakly connected components apply to **directed** graphs
  (representing connected components when edge direction is ignored). On
  undirected graphs, they are simply connected components. Rejecting directed
  graphs means the function does not actually compute weakly connected
  components.
* **Recommendation:**
  Rename to `ConnectedComponents`, or implement true weakly connected components
  by ignoring edge directions if the input graph is directed.

---

### 3.2 Reference Merging Drops Non-Overlapping References

* **Location:** `models/article.go:89`
* **Description:**
  In `Article.Merge`:
  ```go
  References: utils.KeepLongestSlice(a.References, other.References),
  ```
  When deduplicating articles in `Articles.UniqueById`, if record $A$ has
  references $[R_1, R_2]$ and record $B$ has references $[R_2, R_3, R_4]$,
  `KeepLongestSlice` discards $[R_1, R_2]$ completely. References should be
  merged by identifier rather than keeping only the longer slice.

---

### 3.3 Redundant Root Connection Graph Traversals in `Sap`

* **Location:** `algorithms/sap.go:126`, `algorithms/sap.go:272`
* **Description:**
  `s.computeRootConnections(rootness)` is executed twice during `Sap.Run()`:
  once within `computeLeafness` and again within `computeSap`. Passing the
  computed map or caching it eliminates an extra $O(V+E)$ graph traversal pass.

---

### 3.4 Slicing Memory Retention in `UniqueQueue`

* **Location:** `internal/collections/queue.go:43-47`
* **Description:**
  ```go
  item := q.queue[0]
  q.queue = q.queue[1:]
  ```
  In Go, slicing the front of a slice does not clear the pointer at index 0. If
  `UniqueQueue` holds pointers, popped elements remain referenced in memory
  until the slice is reallocated. Zeroing `q.queue[0] = zero` prior to reslicing
  allows garbage collection.

---

### 3.5 Hardcoded Model ID in Ollama Embeddings Client

* **Location:** `cli/clients/embeddings/client.go:13`
* **Description:**
  `modelID = "hf.co/ggml-org/embeddinggemma-300M-GGUF"` is hardcoded as a
  constant. Providing a CLI flag (e.g. `--model`), configuration parameter, or
  environment variable (`BIBX_EMBEDDING_MODEL`) allows users to utilize
  different embedding models or offline setups.

---

### 3.6 Dangling Period in Markdown Reference Template

* **Location:** `cli/renderers/templates/reference.md:7-8`
* **Description:**
  ```markdown
  {{- if .DOI }}. doi: {{ .DOI }}{{ end -}}
  .
  ```
  Line 8 contains an isolated dot on its own line. In rendered markdown
  references, this leaves an orphan period on the line following the reference.

---

## 4. Documentation & CLI Usability Inconsistencies

### 4.1 CLI Documentation Discrepancies in `README.md`

* **Location:** `README.md:94-123`
* **Issues:**
  1. **File paths:** README states `inquire` generates
     `.bibx/collection.json.gz` and `.bibx/search.json.gz`. In reality, it
     generates `.bibx/analysis.json.gz`.
  2. **View flags:** README instructs `./bibx view --file
     .bibx/collection.json.gz`. The `view` command does not have a `--file` flag;
     it relies on `--root`.
  3. **Query flags:** README instructs `./bibx query --top 10`. The actual flag
     is `--limit` / `-l`.

---

### 4.2 Unconditional Overwrite in `bibx init`

* **Location:** `cli/repos/scaffold.go:88-91`
* **Description:**
  `bibx init` creates scaffold files using `os.Create`, which unconditionally
  truncates existing files (such as `AGENTS.md`) without confirmation or a
  `--force` flag.

---

## 5. Test Coverage & CI Gaps

### 5.1 Fourteen Packages Have 0.0% Statement Coverage

Current test coverage across the repository:

* `cli/clients/embeddings`: **0.0%**
* `cli/commands`: **0.0%**
* `cli/commands/init`: **0.0%**
* `cli/commands/inquire`: **0.0%**
* `cli/commands/query`: **0.0%**
* `cli/commands/search`: **0.0%**
* `cli/commands/setup`: **0.0%**
* `cli/commands/view`: **0.0%**
* `cli/repos`: **0.0%**
* `cli/servers/viewer`: **0.0%**
* `cli/services`: **0.0%**
* `cmd/bibx`: **0.0%**
* `internal/texter`: **0.0%**
* `internal/utils`: **0.0%**
* `internal/vector`: **0.0%**
* `models/analysis.go`: **0.0%**

### 5.2 CI Does Not Run the Race Detector

* **Location:** `.github/workflows/go.yml:27`
* **Issue:** The test step runs `go test -timeout 5s -v ./...` without `-race`.
  Since OpenAlex client fetching and graph operations involve concurrency,
  enabling `-race` in CI will catch data races automatically.

---

## Recommended Prioritized Roadmap

| Priority | Task | Affected Files | Effort | Status |
| :--- | :--- | :--- | :--- | :--- |
| **P0** | Fix `make([]string, 0, len)` allocation in embedding service | `cli/services/analysis.go` | Quick | ✅ Resolved |
| **P0** | Remove `score == 0` skip in semantic search | `models/analysis.go` | Quick | ✅ Resolved |
| **P0** | Fix nil pointer dereference in `Collection.Keep` | `models/collection.go` | Quick | ✅ Resolved |
| **P0** | Fix UnionFind rank increment bug (`u.rank[rootB]++`) | `internal/union/union.go` | Quick | ✅ Resolved |
| **P1** | Replace `break` with `continue` / counter in Steiner search | `algorithms/steiner.go` | Moderate | ✅ Resolved |
| **P1** | Exclude existing terminals in Steiner search | `algorithms/steiner.go` | Moderate | ✅ Resolved |
| **P1** | Handle zero norms & clamping in `vector.CosineDistance` | `internal/vector/vector.go` | Quick | Pending |
| **P1** | Break self-loops without deleting whole articles | `internal/graphs/transformation.go` | Moderate | Pending |
| **P1** | Clean up scaffold template `AGENTS.md` private paths | `cli/repos/templates/AGENTS.md` | Quick | Pending |
| **P2** | Add atomic writing to `StoreGzipJSON` | `internal/formats/gzipjson.go` | Moderate | Pending |
| **P2** | Add graceful HTTP server shutdown & signal handling | `main.go`, `view.go`, `search.go` | Moderate | Pending |
| **P2** | Fix CLI flag & output discrepancies in `README.md` | `README.md` | Quick | Pending |
| **P2** | Fix dangling dot in `reference.md` template | `cli/renderers/templates/reference.md` | Quick | Pending |
| **P3** | Add unit tests for `models/analysis.go`, `vector`, `utils`, `services` | `*_test.go` | Significant | Pending |
| **P3** | Add `-race` flag to GitHub Actions workflow | `.github/workflows/go.yml` | Quick | Pending |
