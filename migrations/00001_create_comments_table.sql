-- +goose Up
CREATE TABLE IF NOT EXISTS comments (
    id BIGSERIAL PRIMARY KEY,
    parent_id BIGINT REFERENCES comments(id) ON DELETE CASCADE,
    author TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted BOOLEAN NOT NULL DEFAULT false,
    content_tsv tsvector
    );

CREATE OR REPLACE FUNCTION comments_tsv_trigger() RETURNS trigger AS $$
BEGIN
  NEW.content_tsv := to_tsvector('russian', coalesce(NEW.content,'') || ' ' || coalesce(NEW.author,''));
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS tsvectorupdate ON comments;
CREATE TRIGGER tsvectorupdate BEFORE INSERT OR UPDATE
    ON comments FOR EACH ROW EXECUTE PROCEDURE comments_tsv_trigger();

CREATE INDEX IF NOT EXISTS idx_comments_content_tsv ON comments USING GIN (content_tsv);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);
CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at);

-- +goose Down
DROP TRIGGER IF EXISTS tsvectorupdate ON comments;
DROP FUNCTION IF EXISTS comments_tsv_trigger();
DROP TABLE IF EXISTS comments;
