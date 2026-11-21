-- Async Inquire Jobs Table
CREATE TABLE inquire_jobs (
    id            BLOB(16) PRIMARY KEY,
    query_hash    BLOB(32) NOT NULL,
    query         TEXT NOT NULL,
    result_limit  INTEGER NOT NULL,
    name          TEXT,
    force         BOOLEAN NOT NULL DEFAULT 0,
    status        TEXT NOT NULL,
    stage         TEXT NOT NULL DEFAULT '',
    collection_id BLOB(16),
    is_new        BOOLEAN NOT NULL DEFAULT 0,
    seed_count    INTEGER NOT NULL DEFAULT 0,
    error         TEXT,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE SET NULL
);

-- At most one active (pending or running) job per query_hash.
CREATE UNIQUE INDEX idx_inquire_jobs_active_hash
    ON inquire_jobs(query_hash)
    WHERE status IN ('pending', 'running');

CREATE INDEX idx_inquire_jobs_status_updated
    ON inquire_jobs(status, updated_at);
