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

-- name: ListCollections :many
SELECT *
FROM collections
WHERE id > ?
ORDER BY id
LIMIT ?;

-- name: CreateCollection :exec
INSERT INTO collections (
    id,
    query_hash,
    query,
    result_limit,
    parameters,
    name,
    article_count
) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: DeleteCollectionById :exec
DELETE FROM collections
WHERE id = ?;

-- name: UpsertArticle :one
INSERT INTO articles (
    id,
    label,
    title,
    journal,
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
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(label) DO UPDATE SET
    title=COALESCE(excluded.title, articles.title),
    journal=COALESCE(excluded.journal, articles.journal),
    year=COALESCE(excluded.year, articles.year),
    volume=COALESCE(excluded.volume, articles.volume),
    issue=COALESCE(excluded.issue, articles.issue),
    page=COALESCE(excluded.page, articles.page),
    doi=COALESCE(excluded.doi, articles.doi),
    permalink=COALESCE(excluded.permalink, articles.permalink),
    times_cited=COALESCE(excluded.times_cited, articles.times_cited),
    abstract=COALESCE(excluded.abstract, articles.abstract),
    rich=excluded.rich OR articles.rich,
    authors=COALESCE(excluded.authors, articles.authors),
    keywords=COALESCE(excluded.keywords, articles.keywords),
    ids=COALESCE(excluded.ids, articles.ids)
RETURNING id;

-- name: DeleteArticleFTS :exec
DELETE FROM articles_fts
WHERE article_id = ?;

-- name: CreateArticleFTS :exec
INSERT INTO articles_fts (
    article_id,
    title,
    abstract,
    keywords,
    authors
) VALUES (?, ?, ?, ?, ?);

-- name: UpsertNode :one
INSERT INTO nodes (
    id,
    collection_id,
    article_id,
    category,
    rootness,
    trunkness,
    leafness,
    community
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(collection_id, article_id) DO UPDATE SET
    category=excluded.category,
    rootness=excluded.rootness,
    trunkness=excluded.trunkness,
    leafness=excluded.leafness,
    community=excluded.community
RETURNING id;

-- name: UpdateArticleCount :exec
UPDATE collections
SET article_count = (
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
	n.id, n.category, n.rootness, n.trunkness, n.leafness, n.community,
	a.id, a.label, a.title, a.journal, a.year, a.volume, a.issue, a.page,
	a.doi, a.permalink, a.times_cited, a.abstract, a.rich,
	a.authors, a.keywords, a.ids
FROM nodes n
JOIN articles a ON a.id = n.article_id
WHERE n.collection_id = ?;

-- name: ListLinks :many
SELECT *
FROM links
WHERE collection_id = ?;

-- name: ListRoots :many
SELECT
    n.rootness,
    a.id, a.label, a.title, a.journal, a.year, a.volume, a.issue, a.page,
    a.doi, a.permalink, a.times_cited, a.abstract, a.rich,
    a.authors, a.keywords, a.ids
FROM nodes n
JOIN articles a ON a.id = n.article_id
WHERE n.collection_id = ? AND n.rootness > 0
ORDER BY n.rootness DESC
LIMIT ?;

-- name: ListTrunks :many
SELECT
    n.trunkness,
    a.id, a.label, a.title, a.journal, a.year, a.volume, a.issue, a.page,
    a.doi, a.permalink, a.times_cited, a.abstract, a.rich,
    a.authors, a.keywords, a.ids
FROM nodes n
JOIN articles a ON a.id = n.article_id
WHERE n.collection_id = ? AND n.trunkness > 0
ORDER BY n.trunkness DESC
LIMIT ?;

-- name: ListLeaves :many
SELECT
    n.leafness,
    a.id, a.label, a.title, a.journal, a.year, a.volume, a.issue, a.page,
    a.doi, a.permalink, a.times_cited, a.abstract, a.rich,
    a.authors, a.keywords, a.ids
FROM nodes n
JOIN articles a ON a.id = n.article_id
WHERE n.collection_id = ? AND n.leafness > 0
ORDER BY n.leafness DESC
LIMIT ?;

-- name: Search :many
SELECT
    n.id, n.category, n.rootness, n.trunkness, n.leafness, n.community,
    a.id, a.label, a.title, a.journal, a.year, a.volume, a.issue, a.page,
    a.doi, a.permalink, a.times_cited, a.abstract, a.rich,
    a.authors, a.keywords, a.ids
FROM articles a
JOIN articles_fts(?) fts ON a.id = fts.article_id
JOIN nodes n ON n.article_id = articles.id
WHERE n.collection_id = ?
ORDER BY bm25(articles_fts)
LIMIT ?;
