DROP TRIGGER works_ai;
DROP TRIGGER works_ad;
DROP TRIGGER works_au;
DROP TABLE works_fts;

DROP INDEX idx_works_label;
DROP INDEX idx_works_doi;
DROP INDEX idx_works_year;

ALTER TABLE nodes RENAME COLUMN work_id TO article_id;
ALTER TABLE collections RENAME COLUMN work_count TO article_count;
ALTER TABLE works RENAME COLUMN venue TO journal;
ALTER TABLE works RENAME TO articles;

CREATE INDEX idx_articles_label ON articles(label);
CREATE INDEX idx_articles_doi ON articles(doi);
CREATE INDEX idx_articles_year ON articles(year);

-- FTS5 Full-Text Search Virtual Table
CREATE VIRTUAL TABLE articles_fts USING fts5(
    title,
    abstract,
    keywords,
    authors,
    content = 'articles',
    content_rowid = 'rowid',
    tokenize = 'porter unicode61'
);

-- Keep FTS index updated on INSERT
CREATE TRIGGER articles_ai AFTER INSERT ON articles BEGIN
    INSERT INTO articles_fts(rowid, title, abstract, keywords, authors)
    VALUES (new.rowid, new.title, new.abstract, new.keywords, new.authors);
END;

-- Keep FTS index updated on DELETE
CREATE TRIGGER articles_ad AFTER DELETE ON articles BEGIN
    INSERT INTO articles_fts(articles_fts, rowid, title, abstract, keywords, authors)
    VALUES ('delete', old.rowid, old.title, old.abstract, old.keywords, old.authors);
END;

-- Keep FTS index updated on UPDATE
CREATE TRIGGER articles_au AFTER UPDATE ON articles BEGIN
    INSERT INTO articles_fts(articles_fts, rowid, title, abstract, keywords, authors)
    VALUES ('delete', old.rowid, old.title, old.abstract, old.keywords, old.authors);
    INSERT INTO articles_fts(rowid, title, abstract, keywords, authors)
    VALUES (new.rowid, new.title, new.abstract, new.keywords, new.authors);
END;

-- Index the rows that already exist
INSERT INTO articles_fts(articles_fts) VALUES ('rebuild');
