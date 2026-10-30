# Plan: literature-review recommendations (2026-10-30)

This plan turns
[the literature-review recommendations](../recommendations/2026-10-30-literature-review-session.md)
into batches of work. Numbers such as #4 refer to the sections of that file.

## Status

| Batch | Recommendations      | State                                                                       |
| ----- | -------------------- | --------------------------------------------------------------------------- |
| 1     | #2, #3, #4, #9, #10  | Done in [PR #50](https://github.com/coreofscience/go-bibx/pull/50) (draft) |
| 2     | #1, #3               | To do                                                                       |
| 3     | #5, #7               | To do                                                                       |
| Later | #6, #8, #9 (`lookup`) | Not scheduled                                                               |

## Findings that change the recommendations

- **#9 is partly wrong.** The FTS index (`articles_fts` in
  `repository/migrations/000001_init_schema.up.sql`) already covers `title`,
  `abstract`, `keywords` and `authors`. The empty "armbrust" search happened
  because Armbrust's paper was never in the collection, which is #1. Batch 1
  changed the `search` description to say so. Do not add authors to the index.
- **Collections are cached by query and limit only** (`ComputeQueryHash` in
  `services/collection.go`). Re-running an old query returns the old
  collection unless `force` is set. Any new option that changes what is
  fetched must become part of the cache key.

## Batch 1: done

- **Relevance sort.** `inquire` sorts by `relevance_score:desc` instead of
  `publication_year:desc`. `ListRecentArticles` was renamed to `SearchArticles`
  (`clients/openalex/client.go`). The ToS categories come from the citation
  graph, not from the year, so `tos` was not affected.
- **Page ranges.** `pageRange` in `clients/openalex/client.go` stores
  `first-last`. The BibTeX renderer turns this into `first--last`.
- **`delete` MCP tool** in `internal/mcp/tools/collections/delete.go`. It is
  marked destructive and rejects a blank target.
- **Guidance.**
  - The `inquire` description says that every term must match and names the
    fixed seed filters (`seedFilters` in `inquire.go`).
  - `inquire` returns `warnings` when fewer than half of the requested works
    match. It uses `services.InquireResult.SeedCount` for this.

## Batch 2: `inquire` options (#1, #3)

**Goal:** let the caller choose which works are fetched and how they are
ranked, with defaults that also include conference papers and books.

**Changes:**

1. Add an options struct to `openalex.Client.SearchArticles`, e.g.
   `SearchOptions{Types []string; RequireAbstract bool; MinCitations int; Sort string}`.
   Build the filter parts in `clients/openalex/client.go` from this struct
   instead of hard-coding them.
2. Defaults:
   - Types: `article`, `book`, `book-chapter`, `report`. Check against the
     OpenAlex API how conference papers are typed today (`types/article` with
     a `conference` source, or a separate type) before you pick the list.
   - `RequireAbstract`: keep `true`, since the abstracts feed search and
     review prompts.
   - `MinCitations`: keep `1` (the current filter is `cited_by_count:>1`).
   - Sort: `relevance`.
3. Sort values: `relevance` maps to `relevance_score:desc`, `cited` to
   `cited_by_count:desc` and `recent` to `publication_year:desc`.
4. Pass the options through `services.InquireOptions` and add them to
   `queryFingerprint` and `ComputeQueryHash`, so that different options do not
   return the same cached collection.
   - The hash format changes, so old collections stop matching their queries.
     Either accept that and mention it in the PR, or keep the current hash when
     every option is at its default.
5. Expose the options:
   - in the MCP tool (`internal/mcp/tools/collections/inquire.go`), with a
     `jsonschema` description on every field;
   - as CLI flags (`internal/cli/commands/inquire/inquire.go`).
6. Build `seedFilters` and the low-count warning text from the options that
   were actually used.
7. Regenerate the mocks (`.mockery.yaml`).
8. Add tests:
   - the filter and sort query parameters for each option
     (`clients/openalex/client_test.go`);
   - the cache key (`services/collection_test.go`);
   - the tool schema (`inquire_test.go`).

**Done when:** `go test ./...` and golangci-lint pass. A live
`inquire("A view of cloud computing", ...)`, or a similar query for a
non-journal work, returns the work.

## Batch 3: BibTeX entry types and venues (#5, #7)

**Goal:** export `@inproceedings`, `@book`, `@techreport` and `@incollection`
entries with the right venue field, and stop using repository names as
venues.

**Changes:**

1. In `clients/openalex/models.go`:
   - Add `Type string` with the tag `json:"type"` to `Work`.
   - Add `Locations []WorkLocation` with the tag `json:"locations"` to `Work`.
   - `WorkLocationSource.Type` already exists.
   - The `select` fields come from the struct tags (`jsonFields`), so the new
     fields are fetched automatically.
2. Add a migration, `repository/migrations/000003_*`, with nullable columns on
   `articles` for the work type and the venue type. Update the sqlc queries
   (`sqlc.yaml`) and regenerate the code. Map the new values through
   `models.Article`, including `Merge` in `models/article.go`.
3. Venue selection in `WorkToArticle`: when `primary_location.source.type` is
   `repository`, use the first location whose source is a journal or a
   conference. If there is none, keep the repository name and flag the entry.
   Use a BibTeX `note` field, a `warnings` field in the MCP response, or both.
4. Entry type mapping in `models/bibtex.go` (the current fallback is at
   about line 186):

   | Condition                        | Entry            | Venue field |
   | -------------------------------- | ---------------- | ----------- |
   | venue type `conference`          | `@inproceedings` | `booktitle` |
   | work type `book`                 | `@book`          | `publisher` |
   | work type `report`               | `@techreport`    | `institution` |
   | work type `book-chapter`         | `@incollection`  | `booktitle` |
   | has a journal                    | `@article`       | `journal`   |
   | otherwise                        | `@misc`          | none        |

5. Add tests in `models/bibtex_test.go` and the client tests. Include an
   article stored before the migration, with null types, to show that its
   export does not change.

**Done when:** the Dean & Ghemawat (SOSP/CACM), Kreps (NetDB) and Carbone
(IEEE Data Eng. Bull.) works export with the right entry type and venue.

## Later

- **#6 Cut-off titles.** Start with the cheap part: flag one- to three-word
  titles that have a `10.1145/` DOI in the `bibtex` and `articles` output.
  Filling in the full title from Crossref adds a second outside service and
  needs its own decision.
- **#8 Off-topic roots.** Either add an optional filter on `tos` that drops
  roots sharing no terms with the collection query, or return a relevance
  score with each root. This is a design choice with low impact, so decide it
  before you build it.
- **#9 `lookup` tool.** Fetch one work by DOI or OpenAlex ID into an existing
  collection. `ListArticlesByIDs` in the client already fetches works by ID.
  The new part is adding a work to a stored collection, including re-running
  SAP or deliberately not re-running it.
