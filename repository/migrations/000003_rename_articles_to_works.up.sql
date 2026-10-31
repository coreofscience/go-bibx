-- FTS5 keeps the name of its content table, so the index cannot survive a
-- rename of the table it reads from. Drop it here and rebuild it below.
-- Databases created by earlier versions of 000001 may not have the triggers.
DROP TRIGGER IF EXISTS articles_ai;
DROP TRIGGER IF EXISTS articles_ad;
DROP TRIGGER IF EXISTS articles_au;
DROP TABLE IF EXISTS articles_fts;

DROP INDEX IF EXISTS idx_articles_label;
DROP INDEX IF EXISTS idx_articles_doi;
DROP INDEX IF EXISTS idx_articles_year;

ALTER TABLE articles RENAME TO works;
ALTER TABLE works RENAME COLUMN journal TO venue;
ALTER TABLE collections RENAME COLUMN article_count TO work_count;
ALTER TABLE nodes RENAME COLUMN article_id TO work_id;

CREATE INDEX idx_works_label ON works(label);
CREATE INDEX idx_works_doi ON works(doi);
CREATE INDEX idx_works_year ON works(year);

-- FTS5 Full-Text Search Virtual Table
CREATE VIRTUAL TABLE works_fts USING fts5(
    title,
    abstract,
    keywords,
    authors,
    content = 'works',
    content_rowid = 'rowid',
    tokenize = 'porter unicode61'
);

-- Keep FTS index updated on INSERT
CREATE TRIGGER works_ai AFTER INSERT ON works BEGIN
    INSERT INTO works_fts(rowid, title, abstract, keywords, authors)
    VALUES (new.rowid, new.title, new.abstract, new.keywords, new.authors);
END;

-- Keep FTS index updated on DELETE
CREATE TRIGGER works_ad AFTER DELETE ON works BEGIN
    INSERT INTO works_fts(works_fts, rowid, title, abstract, keywords, authors)
    VALUES ('delete', old.rowid, old.title, old.abstract, old.keywords, old.authors);
END;

-- Keep FTS index updated on UPDATE
CREATE TRIGGER works_au AFTER UPDATE ON works BEGIN
    INSERT INTO works_fts(works_fts, rowid, title, abstract, keywords, authors)
    VALUES ('delete', old.rowid, old.title, old.abstract, old.keywords, old.authors);
    INSERT INTO works_fts(rowid, title, abstract, keywords, authors)
    VALUES (new.rowid, new.title, new.abstract, new.keywords, new.authors);
END;

-- Index the rows that already exist
INSERT INTO works_fts(works_fts) VALUES ('rebuild');
