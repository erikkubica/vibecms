CREATE TABLE IF NOT EXISTS media_files (
    id SERIAL PRIMARY KEY,
    filename TEXT NOT NULL,
    original_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size BIGINT NOT NULL,
    path TEXT NOT NULL,
    url TEXT NOT NULL,
    width INT,
    height INT,
    alt TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_media_files_mime ON media_files(mime_type);
CREATE INDEX IF NOT EXISTS idx_media_files_name ON media_files(original_name);
