-- Create taxonomies table. Idempotent so existing installs (where this
-- table was seeded by a now-removed earlier migration) are not affected.
-- Later migration 0026 adds hierarchical/show_ui columns.
CREATE TABLE IF NOT EXISTS taxonomies (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(50) NOT NULL UNIQUE,
    label VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    node_types TEXT[] NOT NULL DEFAULT '{}',
    field_schema JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
