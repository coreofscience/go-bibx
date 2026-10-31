-- name: GetCollectionByHash :one
SELECT *
FROM collections
WHERE query_hash = ?;

-- name: GetCollectionByID :one
SELECT *
FROM collections
WHERE id = ?;

-- name: GetCollectionByName :one
SELECT *
FROM collections
WHERE name = ?;

-- name: GetCollectionByQuery :one
SELECT *
FROM collections
WHERE query = ? AND result_limit = ?
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: UpdateCollectionHash :exec
UPDATE collections
SET query_hash = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ListCollections :many
SELECT *
FROM collections
ORDER BY created_at DESC, id DESC
LIMIT ?;

-- name: CreateCollection :exec
INSERT INTO collections (
    id,
    query_hash,
    query,
    result_limit,
    parameters,
    name,
    work_count
) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: DeleteCollectionById :exec
DELETE FROM collections
WHERE id = ?;

-- name: UpdateCollectionName :exec
UPDATE collections
SET name = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteNodesByCollectionID :exec
DELETE FROM nodes
WHERE collection_id = ?;

-- name: DeleteLinksByCollectionID :exec
DELETE FROM links
WHERE collection_id = ?;

-- name: UpsertWork :one
INSERT INTO works (
    id,
    label,
    title,
    venue,
    type,
    venue_type,
    year,
    volume,
    issue,
    page,
    doi,
    permalink,
    times_cited,
    abstract,
    rich,
    authors,
    keywords,
    ids
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(label) DO UPDATE SET
    title=COALESCE(excluded.title, works.title),
    venue=COALESCE(excluded.venue, works.venue),
    type=COALESCE(excluded.type, works.type),
    venue_type=COALESCE(excluded.venue_type, works.venue_type),
    year=COALESCE(excluded.year, works.year),
    volume=COALESCE(excluded.volume, works.volume),
    issue=COALESCE(excluded.issue, works.issue),
    page=COALESCE(excluded.page, works.page),
    doi=COALESCE(excluded.doi, works.doi),
    permalink=COALESCE(excluded.permalink, works.permalink),
    times_cited=COALESCE(excluded.times_cited, works.times_cited),
    abstract=COALESCE(excluded.abstract, works.abstract),
    rich=excluded.rich OR works.rich,
    authors=COALESCE(excluded.authors, works.authors),
    keywords=COALESCE(excluded.keywords, works.keywords),
    ids=COALESCE(excluded.ids, works.ids)
RETURNING id;

-- name: UpsertNode :one
INSERT INTO nodes (
    id,
    collection_id,
    work_id,
    category,
    rootness,
    trunkness,
    leafness
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(collection_id, work_id) DO UPDATE SET
    category=excluded.category,
    rootness=excluded.rootness,
    trunkness=excluded.trunkness,
    leafness=excluded.leafness
RETURNING id;

-- name: UpdateWorkCount :exec
UPDATE collections
SET work_count = (
        SELECT count(*)
        FROM nodes n
        WHERE n.collection_id = ?
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE collections.id = ?;

-- name: UpsertLink :exec
INSERT OR REPLACE INTO links (
    collection_id,
    source_id,
    target_id,
    weight
) VALUES (?, ?, ?, ?);

-- name: ListNodes :many
SELECT
	n.id, n.category, n.rootness, n.trunkness, n.leafness,
	w.id, w.label, w.title, w.venue, w.type, w.venue_type, w.year, w.volume, w.issue, w.page,
	w.doi, w.permalink, w.times_cited, w.abstract, w.rich,
	w.authors, w.keywords, w.ids
FROM nodes n
JOIN works w ON w.id = n.work_id
WHERE n.collection_id = ?;

-- name: ListLinks :many
SELECT *
FROM links
WHERE collection_id = ?;

-- name: ListRoots :many
SELECT
    n.rootness,
    w.id, w.label, w.title, w.venue, w.type, w.venue_type, w.year, w.volume, w.issue, w.page,
    w.doi, w.permalink, w.times_cited, w.abstract, w.rich,
    w.authors, w.keywords, w.ids
FROM nodes n
JOIN works w ON w.id = n.work_id
WHERE n.collection_id = ? AND n.rootness > 0
ORDER BY n.rootness DESC
LIMIT ?;

-- name: ListTrunks :many
SELECT
    n.trunkness,
    w.id, w.label, w.title, w.venue, w.type, w.venue_type, w.year, w.volume, w.issue, w.page,
    w.doi, w.permalink, w.times_cited, w.abstract, w.rich,
    w.authors, w.keywords, w.ids
FROM nodes n
JOIN works w ON w.id = n.work_id
WHERE n.collection_id = ? AND n.trunkness > 0
ORDER BY n.trunkness DESC
LIMIT ?;

-- name: ListLeaves :many
SELECT
    n.leafness,
    w.id, w.label, w.title, w.venue, w.type, w.venue_type, w.year, w.volume, w.issue, w.page,
    w.doi, w.permalink, w.times_cited, w.abstract, w.rich,
    w.authors, w.keywords, w.ids
FROM nodes n
JOIN works w ON w.id = n.work_id
WHERE n.collection_id = ? AND n.leafness > 0
ORDER BY n.leafness DESC
LIMIT ?;

-- name: Search :many
SELECT
    n.id, n.category, n.rootness, n.trunkness, n.leafness,
    w.id, w.label, w.title, w.venue, w.type, w.venue_type, w.year, w.volume, w.issue, w.page,
    w.doi, w.permalink, w.times_cited, w.abstract, w.rich,
    w.authors, w.keywords, w.ids
FROM works_fts(?) fts
JOIN works w ON w.rowid = fts.rowid
JOIN nodes n ON n.work_id = w.id
WHERE n.collection_id = ?
ORDER BY bm25(works_fts)
LIMIT ?;
