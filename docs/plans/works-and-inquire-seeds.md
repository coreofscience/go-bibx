# Plan: works, `inquire` seeds and BibTeX export

## Why

A literature-review session built a 58-paper reading list with the bibx MCP
server. Two problems stood out:

- **`inquire` missed works.** Its seed round only fetches OpenAlex works typed
  `article`, so most conference papers never become seeds.
- **BibTeX export needed manual fixes.** Every entry is `@article` or `@misc`:
  21 of 58 had to be retyped by hand. Some venues were repository names instead
  of the real venue.

The article limit dates from when go-bibx followed `python-bibx`, which
aggregates files from several sources. This version only reads OpenAlex, and
Tree of Science (ToS) can use any work with citations as a seed.

## Status

| Phase | Work                                      | State         |
| ----- | ----------------------------------------- | ------------- |
| 1     | Rename `Article` to `Work`                | Done          |
| 2     | Change the `inquire` seed filter          | To do         |
| 3     | BibTeX entry types and venues             | To do         |
| Later | Cut-off titles, off-topic roots, `lookup` | Not scheduled |

Each phase is its own PR, in this order.

## Findings

Checked against the live OpenAlex API on 2026-10-31.

- **Conference papers have their own type.** For "stream processing", works
  with a conference source are 31,381 `conference-paper` and 414 `article`.
- **`review` is also a separate type**, so the current filter excludes reviews.
- **The type limit only applies to seeds.** `ListWorksByIDs` has no type
  filter, so collections already hold books and reports as references.
- **Seeds without references add nothing to the graph.** `Clean` keeps only the
  largest connected component, so such a seed stays only if another seed cites
  it, and then it would enter as a reference anyway. It still uses up `limit`.
- **Kreps' 2011 Kafka paper appears to be missing from OpenAlex.** An exact
  title phrase search returns 0 works, and a broader search did not surface it.
  Nothing in this plan can recover it, including a `lookup` tool.
- **"A view of cloud computing" is not blocked by the filters.** It is typed
  `article`, has an abstract and ranks first for "view cloud computing". The
  original session got 0 works because it added "armbrust" to the query, and
  author names are not in the title or abstract.
- **Collections are cached by query and limit only** (`ComputeQueryHash` in
  `services/collection.go`). A change to what is fetched must change the hash,
  or old queries return old collections.

## Phase 1: rename `Article` to `Work`

**Goal:** use one word for every kind of bibliographic work, with no change in
what gets fetched, analysed or exported.

The structure stays flat. New fields are explicit columns, not a JSON "extras"
column, because BibTeX, the search index and sqlc all work with named fields.

**Changes:**

1. Go names, across about 70 files:
   - `models.Article`, `Articles` and `ArticleOption` become `Work`, `Works` and
     `WorkOption`. Rename the files to match, e.g. `models/article.go` to
     `models/work.go`.
   - `Node.Article` becomes `Node.Work`, and `ArticleCount` becomes `WorkCount`.
   - `ArticleSummary` in `internal/mcp/models` becomes `WorkSummary`.
   - `openalex.Client`: `SearchArticles` becomes `SearchWorks`, and
     `ListArticlesByIDs` becomes `ListWorksByIDs`.
   - `WorkToArticle` and `ReferenceToArticle` become `ToWork` and
     `ReferenceToWork`.
2. `Journal` becomes `Venue` in the model, the database and the MCP and CLI
   output, because it already holds conference and repository names. The BibTeX
   field stays `journal` until phase 3.
3. Add migration `repository/migrations/000003_*`:
   - rename `articles` to `works`, `journal` to `venue`,
     `collections.article_count` to `work_count` and `nodes.article_id` to
     `work_id`;
   - rename the indexes;
   - drop `articles_fts` and its three triggers, recreate them as `works_fts`,
     and run the FTS5 `rebuild` command.

   A plain `ALTER TABLE articles RENAME TO works` is not enough. FTS5 keeps
   `content = 'articles'`, and searches fail with `no such table: main.articles`.
   Use `IF EXISTS` on the drops, because databases created by earlier versions of
   `000001` have no search triggers.
4. Update `queries.sql` and the column overrides in `sqlc.yaml`, then regenerate
   the sqlc code and the mocks (`.mockery.yaml`).
5. User-facing names:
   - the `articles` MCP tool becomes `works`, and its `articles` response field
     becomes `works`;
   - update tool descriptions, `internal/cli/commands/mcp/instructions.txt`, the
     prompts in `internal/mcp/prompts` and the resources in `internal/mcp/resources`.

   Clients that call the `articles` tool break. Say so in the PR.

**Done when:**

- `go test ./...` and golangci-lint pass.
- A migration test takes a database with stored data from `000002` to `000003`:
  rows are kept, and search finds the old rows and newly inserted ones.
- The down migration restores the old schema.
- `git grep -i article` only finds true uses, such as the BibTeX `@article`
  entry and the OpenAlex `article` type.

## Phase 2: change the `inquire` seed filter

**Goal:** let any type of work be a seed, as long as it has references.

For works with more than one citation and an abstract:

| Query           | `article` (today) | All types | All types with references |
| --------------- | ----------------- | --------- | ------------------------- |
| apache kafka    | 233               | 526       | 420                       |
| nosql databases | 957               | 2,265     | 1,894                     |
| cloud computing | 48,914            | 94,824    | 81,733                    |

With the new filter, the top 200 for "cloud computing" are 100 articles, 85
conference papers, 4 book chapters, 3 reviews, 3 `other`, 2 preprints, 2 reports
and 1 conference abstract.

**Changes:**

1. In `SearchWorks` (`clients/openalex/client.go`), replace `type:types/article`
   with `referenced_works_count:>0`. Keep `cited_by_count:>1` and
   `has_abstract:true`.
2. Add a version to `queryFingerprint`, so that old queries fetch again. Old
   collections stop matching their queries and can be removed with `delete`. Say
   so in the PR.
3. Update `seedFilters` and the `inquire` description
   (`internal/mcp/tools/collections/inquire.go`). Both currently say that only
   journal articles are fetched directly.
4. Store the work type and venue type now, so that collections fetched before
   phase 3 do not need `force` later:
   - add `Type string` with the tag `json:"type"` to `openalex.Work`.
     `WorkLocationSource.Type` already exists, and the `select` fields come from
     the struct tags (`jsonFields`);
   - add migration `000004_*` with nullable `type` and `venue_type` columns on
     `works`;
   - add `Type` and `VenueType` to `models.Work`, including `Clone` and `Merge`,
     and map them through the queries and the service.
5. Add tests for the filter parameters (`clients/openalex/client_test.go`), the
   cache key (`services/collection_test.go`) and the new fields.

No new `inquire` parameters. There is no type list left to choose from, and
relevance sort already fixed the complaint that collections only held recent
works.

**Done when:**

- `go test ./...` and golangci-lint pass.
- A live `inquire` returns a work that today's filter excludes, such as the
  `conference-paper` "Parallel visualization on large clusters using MapReduce"
  (W2116018104), which has 32 references.
- `tos` on a live collection, e.g. "apache kafka", still has sensible roots,
  trunks and leaves. This has not been run through bibx yet.

## Phase 3: BibTeX entry types and venues

**Goal:** export the right entry type and venue field for each work, and stop
using repository names as venues.

**Changes:**

1. Add `Locations []WorkLocation` with the tag `json:"locations"` to
   `openalex.Work`.
2. Venue selection in `ToWork`: when `primary_location.source.type` is
   `repository`, use the first location whose source is a journal or a
   conference. Carbone et al.'s Flink paper confirms this finds the right venue:
   its locations include both the KTH repository and IEEE Data Engineering
   Bulletin. If there is none, keep the repository name and flag the entry, using
   a BibTeX `note` field, a `warnings` field in the MCP response, or both.
3. Entry type mapping in `models/bibtex.go`. The first matching row wins, and
   every type needs a result because seeds are no longer limited to a list:

   | Condition                                          | Entry            | Venue field   |
   | -------------------------------------------------- | ---------------- | ------------- |
   | work type `conference-paper` or venue `conference` | `@inproceedings` | `booktitle`   |
   | work type `book`                                   | `@book`          | `publisher`   |
   | work type `report`                                 | `@techreport`    | `institution` |
   | work type `book-chapter`                           | `@incollection`  | `booktitle`   |
   | work type `dissertation`                           | `@phdthesis`     | `school`      |
   | venue type `repository`                            | `@misc`          | none          |
   | has a venue (e.g. `article` or `review`)           | `@article`       | `journal`     |
   | otherwise                                          | `@misc`          | none          |

4. Check where `publisher` comes from before building the `@book` row. In a small
   sample of books, the source was "Apress eBooks" or missing, and
   `host_organization_name` was empty. Check `institution` and `school` the same
   way. If a value is not available, leave the field out.
5. Add tests in `models/bibtex_test.go` and the client tests. Include a work
   stored before phase 2, with null types, to show that its export does not
   change.

**Done when:** these works export with the right entry type and venue:

- Dean & Ghemawat (`10.1145/1327452.1327492`, the CACM version) as `@article`;
- Carbone et al. (W2566979091) as `@article` with IEEE Data Engineering Bulletin
  as the journal;
- "Parallel visualization on large clusters using MapReduce" as `@inproceedings`
  with its venue in `booktitle`.

## Later

These come from the same session and are not scheduled. The first two are the
most useful.

- **`lookup` tool.** A narrow `inquire` is the only way to verify a known paper
  today, and it leaves a throwaway collection behind. Fetch one work by DOI or
  OpenAlex ID into an existing collection. The client can already fetch works by
  ID. The new part is adding a work to a stored collection, including re-running
  SAP or deliberately not re-running it.
- **Cut-off titles.** 18 of 58 exported titles lost their subtitles, e.g.
  "MapReduce" and "Bigtable". OpenAlex stores them this way: the record for
  `10.1145/1327452.1327492` has the title "MapReduce". Start by flagging one- to
  three-word titles that have a `10.1145/` DOI in the `bibtex` and `works` output.
  Filling in the full title from Crossref adds a second outside service and needs
  its own decision.
- **Off-topic roots.** The top `tos` root for a "cloud computing" collection was
  the PRISMA 2020 reporting guideline, because many papers in the collection cite
  it. Either add an optional filter on `tos` that drops roots sharing no terms
  with the collection query, or return a relevance score with each root. Decide
  this before building it.
