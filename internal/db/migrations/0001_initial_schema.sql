-- VibeCMS Phase 1 Initial Schema

-- users
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'editor',
    full_name VARCHAR(100),
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- sessions
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- content_nodes
CREATE TABLE IF NOT EXISTS content_nodes (
    id SERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    parent_id INT REFERENCES content_nodes(id) ON DELETE SET NULL,
    node_type VARCHAR(50) NOT NULL DEFAULT 'page',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    language_code VARCHAR(10) NOT NULL DEFAULT 'en',
    slug VARCHAR(255) NOT NULL,
    full_url TEXT NOT NULL UNIQUE,
    title VARCHAR(255) NOT NULL,
    blocks_data JSONB NOT NULL DEFAULT '[]',
    seo_settings JSONB NOT NULL DEFAULT '{}',
    translation_group_id UUID,
    version INT NOT NULL DEFAULT 1,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_nodes_status_lang ON content_nodes(status, language_code);
CREATE INDEX IF NOT EXISTS idx_nodes_blocks ON content_nodes USING GIN (blocks_data);

-- content_node_revisions
CREATE TABLE IF NOT EXISTS content_node_revisions (
    id BIGSERIAL PRIMARY KEY,
    node_id INT NOT NULL REFERENCES content_nodes(id) ON DELETE CASCADE,
    blocks_snapshot JSONB NOT NULL,
    seo_snapshot JSONB NOT NULL,
    created_by INT REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- redirects
CREATE TABLE IF NOT EXISTS redirects (
    id SERIAL PRIMARY KEY,
    old_url TEXT NOT NULL UNIQUE,
    new_url TEXT NOT NULL,
    http_code INT NOT NULL DEFAULT 301,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- site_settings
CREATE TABLE IF NOT EXISTS site_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT,
    is_encrypted BOOLEAN DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
