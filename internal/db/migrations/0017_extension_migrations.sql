CREATE TABLE IF NOT EXISTS extension_migrations (
    id SERIAL PRIMARY KEY,
    extension_slug TEXT NOT NULL,
    filename TEXT NOT NULL,
    applied_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(extension_slug, filename)
);
