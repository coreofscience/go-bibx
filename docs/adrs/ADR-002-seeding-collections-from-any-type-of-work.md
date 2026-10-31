# ADR-002: Seeding Collections From Any Type of Work

## Status

Accepted

## Context

`inquire` builds a collection in two steps: a seed round searches OpenAlex for
the query, then the references of those seeds are fetched by ID. The seed round
only accepted works typed `article`, a limit that dates from when `go-bibx`
followed `python-bibx`, which aggregates files from several sources.

That limit no longer had a reason to exist, and it cost real results:

1. **Whole fields were under-represented**: OpenAlex gives conference papers
   their own type. For "stream processing", works with a conference source are
   31,381 `conference-paper` and 414 `article`, so most conference papers could
   never be seeds. A literature-review session built on the MCP server ran into
   this directly. `review` is a separate type too, and was also excluded.
2. **The limit was inconsistent**: it only applied to seeds. `ListWorksByIDs`
   has no type filter, so collections already held books and reports as
   references.
3. **Tree of Science does not need it**: the algorithm works on the citation
   graph, so any work with citations can take part, whatever its type.

## Decision

Any type of work with references can seed a collection. OpenAlex stays the only
data provider; what widens is the kinds of work taken from it.

### 1. Seed filter

`SearchWorks` replaces `type:types/article` with `referenced_works_count:>0`,
and keeps `cited_by_count:>1` and `has_abstract:true`.

The requirement for references is what replaces the type check. `Clean` keeps
only the largest connected component, so a seed without references stays only if
another seed cites it, in which case it would enter as a reference anyway.
Without this filter, such seeds would use up `limit` and add nothing to the
graph.

For works with more than one citation and an abstract, checked against the live
OpenAlex API on 2026-10-31:

| Query           | `article` (before) | All types | All types with references |
| --------------- | ------------------ | --------- | ------------------------- |
| apache kafka    | 233                | 526       | 420                       |
| nosql databases | 957                | 2,265     | 1,894                     |
| cloud computing | 48,914             | 94,824    | 81,733                    |

### 2. Vocabulary and data model

Because a collection is no longer made of articles, `Article` becomes `Work` and
`Journal` becomes `Venue`, in the code, the database and the MCP and CLI output.

The model stays flat. Type-specific data such as `type`, `venue_type`,
`publisher` and `institution` are explicit, nullable columns rather than a JSON
"extras" column, because BibTeX, the search index and `sqlc` all work with named
fields.

### 3. Every type must have an export

Seeds are no longer limited to a known list, so any code that branches on the
type of a work needs a result for every type, including ones it has not seen.
BibTeX export maps types to entry types (`@inproceedings`, `@book`,
`@techreport`, `@incollection`, `@phdthesis`, `@article`) and falls back to
`@misc`.

### 4. Cached collections

Collections are cached by a hash of the query and limit. Any change to what is
fetched must bump `queryFingerprintVersion`, so that old queries fetch again
instead of returning collections built the old way.

## Alternatives Considered

- **Add `conference-paper` and `review` to an allow-list**: fixes the reported
  gap, but the list would need extending for each type found to matter, and it
  would keep excluding book chapters and reports that the graph can use.
- **Let callers choose types through an `inquire` parameter**: adds a choice
  that callers have no basis for making, since the citation graph decides which
  works matter.
- **Allow all types with no reference requirement**: lets works without
  references use up `limit`, for no gain in the graph.

## Consequences

### Positive

- **Better coverage**: conference-driven fields such as distributed systems now
  produce seeds from their main publication venues. The top 200 for "cloud
  computing" are 100 articles, 85 conference papers and 15 works of other types.
- **More accurate exports**: stored types and venues let BibTeX use the right
  entry type and venue field without manual fixes.
- **Simpler rule**: seeds and references no longer follow different type rules,
  and `inquire` needs no new parameters.

### Negative / Trade-offs

- **Less curated seeds**: preprints, reports and works typed `other` can now be
  seeds. Citations and the graph are the only quality signal, as there is no
  longer an implied preference for journal articles.
- **Breaking rename**: clients that call the `articles` MCP tool or read its
  `articles` field must move to `works`.
- **Collections change on refresh**: rerunning a query refreshes its collection
  in place, keeping its ID and name, so the same query can return different
  works than before.
- **Works without references cannot be seeds**: OpenAlex has no references for
  some works, so they can only enter a collection when a seed cites them.
