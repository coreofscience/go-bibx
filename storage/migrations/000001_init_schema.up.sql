-- Collections Table
CREATE TABLE collections (
    id            BLOB(16) PRIMARY KEY,
    query_hash    BLOB(32) UNIQUE NOT NULL,
    query         TEXT NOT NULL,
    result_limit  INTEGER NOT NULL,
    parameters    JSON,
    name          TEXT,
    article_count INTEGER DEFAULT 0,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_collections_created_at ON collections(created_at DESC);
CREATE INDEX idx_collections_name ON collections(name);

-- Articles Table (Normalized Bibliographic Data)
CREATE TABLE articles (
    id          BLOB(16) PRIMARY KEY,
    label       TEXT UNIQUE NOT NULL,
    title       TEXT,
    journal     TEXT,
    year        INTEGER,
    volume      TEXT,
    issue       TEXT,
    page        TEXT,
    doi         TEXT,
    permalink   TEXT,
    times_cited INTEGER,
    abstract    TEXT,
    rich        BOOLEAN DEFAULT 0,
    authors     JSON,
    keywords    JSON,
    ids         JSON,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_articles_label ON articles(label);
CREATE INDEX idx_articles_doi ON articles(doi);
CREATE INDEX idx_articles_year ON articles(year);

-- Scored Nodes Table
CREATE TABLE nodes (
    id            BLOB(16) PRIMARY KEY,
    collection_id BLOB(16) NOT NULL,
    article_id    BLOB(16) NOT NULL,
    category      TEXT NOT NULL,
    rootness      REAL NOT NULL,
    trunkness     REAL NOT NULL,
    leafness      REAL NOT NULL,
    community     INTEGER NOT NULL,
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
    UNIQUE(collection_id, article_id)
);

CREATE INDEX idx_nodes_collection_category ON nodes(collection_id, category);
CREATE INDEX idx_nodes_collection_rootness ON nodes(collection_id, rootness DESC);
CREATE INDEX idx_nodes_collection_trunkness ON nodes(collection_id, trunkness DESC);
CREATE INDEX idx_nodes_collection_leafness ON nodes(collection_id, leafness DESC);

-- Citation Graph Links Table
CREATE TABLE links (
    collection_id BLOB(16) NOT NULL,
    source_id     BLOB(16) NOT NULL,
    target_id     BLOB(16) NOT NULL,
    weight        REAL DEFAULT 1.0,
    PRIMARY KEY (collection_id, source_id, target_id),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
    FOREIGN KEY (source_id) REFERENCES nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (target_id) REFERENCES nodes(id) ON DELETE CASCADE
);

CREATE INDEX idx_links_collection ON links(collection_id);

-- FTS5 Full-Text Search Virtual Table
CREATE VIRTUAL TABLE articles_fts USING fts5(
    article_id UNINDEXED,
    title,
    abstract,
    keywords,
    authors,
    tokenize = 'porter unicode61'
);
