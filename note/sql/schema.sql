CREATE TABLE IF NOT EXISTS articles
(
    article_id INTEGER   NOT NULL PRIMARY KEY,
    name       TEXT      NOT NULL,
    content    TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_blocks_name ON articles (name);

CREATE TABLE IF NOT EXISTS article_tags
(
    article_id INTEGER NOT NULL,
    tag        TEXT    NOT NULL,
    FOREIGN KEY (article_id) REFERENCES articles (article_id),
    PRIMARY KEY (article_id, tag)
);
CREATE INDEX IF NOT EXISTS idx_block_tags_block_id ON article_tags (article_id);
CREATE INDEX IF NOT EXISTS idx_block_tags_tag ON article_tags (tag);