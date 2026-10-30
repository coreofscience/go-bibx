# Recommendations from a literature-review session (2026-10-30)

## Context

The bibx MCP server was used to build a 58-paper reading list and a BibTeX file
for an MSc module on data-intensive scalable systems: MapReduce/HDFS, NoSQL,
CAP, Spark, stream processing, Kafka, Lambda architecture and cloud.

- **Calls:** about 50 (`list`, `inquire`, `tos`, `search`, `articles`,
  `bibtex`). None returned an error.
- **The problems** were in search coverage and in the quality of the exported
  metadata. Every exported entry had to be checked and fixed by hand before it
  could be cited in IEEE style.

Each recommendation below gives the observation, the cause found in the code,
and a suggested change. They are ordered by impact on the user.

## 1. `inquire` silently drops conference papers, books and reports

**What happened:** several foundational works could not be collected at all,
even with targeted queries. "A view of cloud computing Armbrust" returned 0
works. "Kafka distributed messaging system log processing" returned 1 work, and
`tos` on that collection returned nothing. Kreps et al.'s 2011 Kafka paper (a
NetDB workshop paper) never appeared.

**Cause:** `ListRecentArticles` (`clients/openalex/client.go`) hard-codes these
filters:

```go
"type:types/article",
"cited_by_count:>1",
"has_abstract:true",
```

- `type:types/article` excludes OpenAlex types that matter in computer science:
  conference papers (often typed differently, depending on the source), books,
  book chapters and reports.
- `has_abstract:true` excludes many older or magazine-style papers, such as
  CACM viewpoints.

Seminal works only reach a collection indirectly, as references of recent
papers. If no recent paper cites them, they cannot be found.

**Recommendation:**

- Make the work types a parameter of `inquire`, e.g.
  `types: ["article", "proceedings-article", "book", "report"]`. Default it to a
  broader set than `article` alone.
- Make `has_abstract` and the minimum citation count optional parameters.
- When `inquire` returns fewer than about 10 works, include a hint in the
  response that names the active filters, so the agent knows why.

## 2. Multi-concept queries return almost nothing

**What happened:**

| Query                                                           | Works |
| --------------------------------------------------------------- | ----- |
| "NoSQL databases CAP theorem consistency distributed storage"   | 1     |
| "Apache Spark Kafka stream processing big data architecture"    | 3     |
| "NoSQL databases"                                               | 768   |
| "Apache Spark"                                                  | 703   |

**Cause:** the query becomes `title_and_abstract.search:term1+term2+...`, so
OpenAlex requires every term to match.

**Recommendation (any of these):**

- State in the `inquire` tool description that all terms must match, and that
  the query should be one concept of 1–3 words. The `search` tool already gives
  similar advice ("try one-word queries").
- Support OR / quoted phrases (OpenAlex accepts `|` for OR within a filter), or
  accept a list of queries and merge the results into one collection.
- Return a warning when the result count is far below `limit`. The client
  already logs "fetched fewer works than limit" at debug level, so surface it to
  the MCP caller as well.

## 3. `limit` keeps only the most recent works

**What happened:** collections are dominated by the latest papers, including
weak venues. Classic papers appear only as references.

**Cause:** `sort: publication_year:desc` combined with `limit` means a broad
query returns the N newest matches, not the N most relevant or most cited.

**Recommendation:** add a `sort` parameter (`recent` | `relevance` | `cited`),
or document clearly in the tool description that a collection is "the N most
recent matching works plus their references".

## 4. Page ranges are lost (`pages` holds the start page only)

**What happened:** every BibTeX entry has a single page, e.g. `pages = {107}`
for Dean & Ghemawat 2008. IEEE style needs the full range (107–113).

**Cause:** the mapping in `clients/openalex/client.go` sets
`Page: work.Biblio.FirstPage`. `WorkBiblio.LastPage` is fetched but never used.

**Recommendation:** store `first_page–last_page` when both are present. The
BibTeX renderer (`models/bibtex.go`) already converts `-` to `--`. This needs
either a new column or a combined value in `Page`. The second option avoids a
schema change.

## 5. Every entry is `@article` or `@misc`

**What happened:**

- Conference papers (SOSP, OSDI, NSDI, SIGMOD, VLDB) were exported as
  `@article` with the proceedings name in `journal`.
- Books and technical reports came out as `@misc`. For example, Marz & Warren's
  _Big Data_ book got a library-catalogue PDF URL, and NIST SP 800-145 was also
  `@misc`.
- Of 58 entries, 21 had to be retyped by hand: 16 `@inproceedings`, 3 `@book`
  and 2 `@techreport`.

**Cause:** `models/bibtex.go` sets `entryType = "article"` when a journal exists
and `"misc"` otherwise. The OpenAlex work `type` and `source.type` (journal,
conference, repository, book-series) are not stored.

**Recommendation:** store the work's type and the source's type, then map them:

| OpenAlex type                                  | BibTeX entry     |
| ---------------------------------------------- | ---------------- |
| `source.type == conference`                    | `@inproceedings` |
| `book`                                         | `@book`          |
| `report`                                       | `@techreport`    |
| `book-chapter`                                 | `@incollection`  |

Use `booktitle` instead of `journal` for `@inproceedings`.

## 6. Titles lose their subtitles

**What happened:** 18 exported titles were cut at the colon:

- "MapReduce" (Simplified data processing on large clusters)
- "Bigtable" (A distributed storage system for structured data)
- "Dynamo", "Pregel", "Apache Spark", "Spark SQL", "Structured Streaming", "The
  dataflow model", "Apache Hadoop YARN", "Kafka versus RabbitMQ", "A break in
  the clouds", and others

**Cause:** this appears to come from OpenAlex's `title` for ACM records. bibx
passes it through unchanged.

**Recommendation:**

- Flag likely-truncated titles in `bibtex` and `articles` output: titles of one
  to three words that have a DOI from a publisher known to split subtitles
  (e.g. `10.1145/`).
- Optionally, fill in the full title from Crossref (`/works/{doi}`), which
  stores `title` and `subtitle` separately.

## 7. Venue comes from a repository copy instead of the publisher

**What happened:** Carbone et al. 2015 ("Apache Flink: Stream and batch
processing in a single engine") was exported with
`journal = {KTH Publication Database DiVA ...}` and no DOI. The real venue is
_IEEE Data Engineering Bulletin_.

**Cause:** `journal` is taken from `primary_location.source.display_name`. For
some works this is a repository rather than the publisher.

**Recommendation:**

- When `primary_location.source.type == "repository"`, look through `locations`
  for a journal or conference source.
- If none is found, mark the entry as needing verification. For example, add a
  `note = {venue from repository copy}` field, or a `warnings` array in the MCP
  response.

## 8. Root ranking includes generic, highly cited papers

**What happened:** the top `tos` root for the "cloud computing" collection was
the PRISMA 2020 statement (a guideline for reporting systematic reviews). It
ranked there because many recent papers in the collection cite it, not because
it is about cloud computing.

**Recommendation:** add an optional filter that drops roots whose title and
keywords share no terms with the collection query. Alternatively, return a
relevance score next to each root so the agent can filter.

## 9. `search` cannot find a paper by author

**What happened:** `search` with query "armbrust" returned an empty subgraph,
in a collection of 71 works about cloud computing.

**Recommendation:**

- If author matching is in scope, include author names in the keyword index
  used to pick Steiner seed nodes.
- If it is not, say so in the tool description ("matches titles, abstracts and
  keywords, not authors").
- Separately, a small `lookup` tool that fetches one work by DOI or OpenAlex ID
  into a collection would make it much easier to verify a known paper. Today the
  only way is a narrow `inquire`, which creates a throwaway collection each
  time.

## 10. No way to delete collections

**What happened:** the session left several collections behind:

- two from failed multi-concept queries (`diss-consistency-nosql`,
  `diss-spark-streaming`)
- five from verification queries (`diss-verify-*`)

The MCP server cannot remove them, so `list` keeps growing.

**Recommendation:** add a `delete` collection tool. Mark it destructive, using
the MCP `destructiveHint` annotation.

## Summary

| #   | Area           | Effort | Impact                                             |
| --- | -------------- | ------ | -------------------------------------------------- |
| 1   | `inquire`      | S      | High: seminal non-journal works missing            |
| 2   | `inquire`      | S      | High: silent near-empty collections                |
| 3   | `inquire`      | S      | Medium                                             |
| 4   | client/BibTeX  | S      | High: every entry needs manual fixing for IEEE     |
| 5   | BibTeX         | M      | High: 21/58 entries retyped by hand                |
| 6   | BibTeX         | M      | Medium: 18/58 titles fixed by hand                 |
| 7   | client/BibTeX  | M      | Medium                                             |
| 8   | `tos`          | M      | Low                                                |
| 9   | `search`       | S–M    | Medium                                             |
| 10  | MCP tools      | S      | Low                                                |
