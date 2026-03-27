CREATE TABLE IF NOT EXISTS extensions (
    id          SERIAL PRIMARY KEY,
    slug        VARCHAR(100) NOT NULL,
    name        VARCHAR(150) NOT NULL,
    version     VARCHAR(50)  NOT NULL DEFAULT '1.0.0',
    description TEXT         NOT NULL DEFAULT '',
    author      VARCHAR(150) NOT NULL DEFAULT '',
    path        TEXT         NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT false,
    priority    INTEGER      NOT NULL DEFAULT 50,
    settings    JSONB        NOT NULL DEFAULT '{}',
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_extensions_slug ON extensions (slug);
CREATE INDEX IF NOT EXISTS idx_extensions_is_active ON extensions (is_active);
