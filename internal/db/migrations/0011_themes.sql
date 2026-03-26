-- Migration 0011: Create themes table for theme management.

CREATE TABLE IF NOT EXISTS themes (
    id          SERIAL PRIMARY KEY,
    slug        VARCHAR(100) UNIQUE NOT NULL,
    name        VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version     VARCHAR(50) NOT NULL DEFAULT '',
    author      VARCHAR(200) NOT NULL DEFAULT '',
    source      VARCHAR(20) NOT NULL DEFAULT 'upload',
    git_url     TEXT,
    git_branch  VARCHAR(100) NOT NULL DEFAULT 'main',
    git_token   TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT false,
    path        VARCHAR(500) NOT NULL,
    thumbnail   VARCHAR(500),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
