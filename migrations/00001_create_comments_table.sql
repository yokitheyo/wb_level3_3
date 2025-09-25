-- +goose Up
CREATE TABLE IF NOT EXISTS comments (
    id BIGSERIAL PRIMARY KEY,
    parent_id BIGINT REFERENCES comments(id) ON DELETE CASCADE,
    author TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted BOOLEAN NOT NULL DEFAULT false,
    content_tsv tsvector
    );

CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);
CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at);
CREATE INDEX IF NOT EXISTS idx_comments_deleted ON comments(deleted);
CREATE INDEX IF NOT EXISTS idx_comments_content_tsv ON comments USING gin(content_tsv);

-- +goose Down
DROP TABLE IF EXISTS comments;