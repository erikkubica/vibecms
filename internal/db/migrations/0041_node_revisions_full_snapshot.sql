-- Expand content_node_revisions to capture a full snapshot of the node so a
-- restore can recreate every editable field, not just blocks + SEO. Pre-0041
-- rows keep their existing two columns and pick up sensible defaults for the
-- new ones — they're snapshots of an older write model and nothing in the
-- restore path treats missing columns as a hard error.

ALTER TABLE content_node_revisions
    ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS slug TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'draft',
    ADD COLUMN IF NOT EXISTS language_code VARCHAR(10) NOT NULL DEFAULT 'en',
    ADD COLUMN IF NOT EXISTS layout_slug TEXT,
    ADD COLUMN IF NOT EXISTS excerpt TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS featured_image JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS fields_snapshot JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS taxonomies_snapshot JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS version_number INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_node_revisions_node_created
    ON content_node_revisions (node_id, created_at DESC);
