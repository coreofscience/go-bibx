# Plan: literature-review recommendations (2026-10-30)

This plan turns
[the literature-review recommendations](../recommendations/2026-10-30-literature-review-session.md)
into batches of work. Numbers such as #4 refer to the sections of that file.

## Status

| Batch | Recommendations      | State                                                                       |
| ----- | -------------------- | --------------------------------------------------------------------------- |
| 1     | #2, #3, #4, #9, #10  | Done in [PR #50](https://github.com/coreofscience/go-bibx/pull/50)         |
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
- **The Armbrust example in #1 has the wrong cause.** Checked against the live
  OpenAlex API on 2026-10-31, "A view of cloud computing" is typed `article`,
  has an abstract and passes the current filters. It ranks first for "view
  cloud computing". Adding "armbrust" gives 0 works because author names are
  not in the title or abstract, so this is a #2 problem.
- **Conference papers have their own type.** OpenAlex uses `conference-paper`.
  For "stream processing", works with a conference source are 31,381
  `conference-paper` and 414 `article`.
- **Kreps' Kafka paper appears to be missing from OpenAlex.** An exact title
  phrase search returns 0 works, and a broader search did not surface it. No
  batch here can recover it, including a `lookup` tool.

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

**Goal:** let the caller choose which types of work are fetched and how they
are ranked, with defaults that also include conference papers and books.

Broader defaults are the main fix. For "apache kafka", with the current
citation and abstract filters:

| Types                                  | Works |
| -------------------------------------- | ----- |
| `article` (today)                      | 233   |
| + `conference-paper`                   | 484   |
| + `book`, `book-chapter`, `report`     | 501   |

**Changes:**

1. Add an options struct to `openalex.Client.SearchArticles`, e.g.
   `SearchOptions{Types []string; Sort string}`. Build the filter parts in
   `clients/openalex/client.go` from this struct instead of hard-coding them.
   Join several types with `|`, e.g. `type:types/article|types/conference-paper`.
2. Defaults:
   - Types: `article`, `conference-paper`, `book`, `book-chapter`, `report`.
     Preprints and dissertations stay out of the defaults, but callers can
     request them through `types`.
   - Sort: `relevance`.
3. Keep `has_abstract:true` and `cited_by_count:>1` fixed. The main evidence
   for loosening them was the Armbrust example, which already passes them. Add
   options later if a real miss shows up.
4. Sort values: `relevance` maps to `relevance_score:desc`, `cited` to
   `cited_by_count:desc` and `recent` to `publication_year:desc`.
5. Pass the options through `services.InquireOptions` and add them to
   `queryFingerprint` and `ComputeQueryHash`, so that different options do not
   return the same cached collection.
   - Always change the hash format. The defaults are changing, so keeping the
     current hash for default options would silently return old articles-only
     collections.
   - Old collections stop matching their queries. They can be removed with the
     `delete` tool. Mention this in the PR.
6. Expose the options:
   - in the MCP tool (`internal/mcp/tools/collections/inquire.go`), with a
     `jsonschema` description on every field;
   - as CLI flags (`internal/cli/commands/inquire/inquire.go`).
7. Build `seedFilters`, the `inquire` description and the low-count warning
   text from the options that were actually used.
8. Regenerate the mocks (`.mockery.yaml`).
9. Add tests:
   - the filter and sort query parameters for each option
     (`clients/openalex/client_test.go`);
   - the cache key (`services/collection_test.go`);
   - the tool schema (`inquire_test.go`).

**Done when:** `go test ./...` and golangci-lint pass. A live `inquire` returns
a work that today's filter excludes, such as the `conference-paper`
"Parallel visualization on large clusters using MapReduce". "A view of cloud
computing" is not a valid test, because it is already returned today.

## Batch 3: BibTeX entry types and venues (#5, #7)

**Goal:** export `@inproceedings`, `@book`, `@techreport` and `@incollection`
entries with the right venue field, and stop using repository names as
venues.

Do this straight after Batch 2. Batch 2 brings in many more conference papers,
and they export as `@article` with the proceedings name in `journal` until this
lands. Collections fetched before this batch need `force` to pick up the new
type fields.

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
   The Carbone (Flink) work confirms this finds the right venue: its locations
   include both the KTH repository and IEEE Data Engineering Bulletin.
4. Entry type mapping in `models/bibtex.go` (the current fallback is at
   about line 186):

   | Condition                                          | Entry            | Venue field   |
   | -------------------------------------------------- | ---------------- | ------------- |
   | work type `conference-paper` or venue `conference` | `@inproceedings` | `booktitle`   |
   | work type `book`                                   | `@book`          | `publisher`   |
   | work type `report`                                 | `@techreport`    | `institution` |
   | work type `book-chapter`                           | `@incollection`  | `booktitle`   |
   | has a journal                                      | `@article`       | `journal`     |
   | otherwise                                          | `@misc`          | none          |

5. Add tests in `models/bibtex_test.go` and the client tests. Include an
   article stored before the migration, with null types, to show that its
   export does not change.

**Done when:** these works export with the right entry type and venue:

- Dean & Ghemawat (`10.1145/1327452.1327492`, the CACM version) as `@article`;
- Carbone as `@article` with IEEE Data Engineering Bulletin as the journal;
- a `conference-paper`, such as "Parallel visualization on large clusters using
  MapReduce", as `@inproceedings` with its venue in `booktitle`.

## Later

The `lookup` tool and flagging cut-off titles are the most useful of these. #8
can wait until it causes more trouble.

- **#6 Cut-off titles.** OpenAlex does store these titles cut off: the record
  for `10.1145/1327452.1327492` has the title "MapReduce". Start with the cheap
  part: flag one- to three-word titles that have a `10.1145/` DOI in the
  `bibtex` and `articles` output. Filling in the full title from Crossref adds
  a second outside service and needs its own decision.
- **#8 Off-topic roots.** Either add an optional filter on `tos` that drops
  roots sharing no terms with the collection query, or return a relevance
  score with each root. This is a design choice with low impact, so decide it
  before you build it.
- **#9 `lookup` tool.** Fetch one work by DOI or OpenAlex ID into an existing
  collection. `ListArticlesByIDs` in the client already fetches works by ID.
  The new part is adding a work to a stored collection, including re-running
  SAP or deliberately not re-running it.
