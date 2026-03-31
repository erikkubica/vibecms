-- Add content_hash column for change detection on theme/extension load.
-- Resources are only updated when hash differs, preventing unnecessary writes.
-- Detached (custom) resources are never overwritten.
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS content_hash VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE layouts ADD COLUMN IF NOT EXISTS content_hash VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE layout_blocks ADD COLUMN IF NOT EXISTS content_hash VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE templates ADD COLUMN IF NOT EXISTS content_hash VARCHAR(64) NOT NULL DEFAULT '';
