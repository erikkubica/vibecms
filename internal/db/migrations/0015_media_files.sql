-- Migration 0015: Create media_files table for media management.

CREATE TABLE IF NOT EXISTS media_files (
    id            SERIAL PRIMARY KEY,
    filename      VARCHAR(500) NOT NULL,
    original_name VARCHAR(500) NOT NULL,
    mime_type     VARCHAR(200) NOT NULL,
    size          BIGINT NOT NULL DEFAULT 0,
    path          VARCHAR(1000) NOT NULL,
    url           VARCHAR(1000) NOT NULL,
    width         INT,
    height        INT,
    alt           TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_media_files_mime_type ON media_files (mime_type);
CREATE INDEX IF NOT EXISTS idx_media_files_original_name ON media_files (original_name);
