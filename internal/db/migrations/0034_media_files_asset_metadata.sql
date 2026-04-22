-- Migration 0034: Add asset metadata columns to media_files table.
-- This supports mapping filesystem assets from themes/extensions to media records.

ALTER TABLE media_files 
ADD COLUMN IF NOT EXISTS asset_key VARCHAR(255),
ADD COLUMN IF NOT EXISTS source VARCHAR(50) DEFAULT 'user',
ADD COLUMN IF NOT EXISTS theme_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS extension_slug VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_media_files_asset_key ON media_files (asset_key);
CREATE INDEX IF NOT EXISTS idx_media_files_source ON media_files (source);
CREATE INDEX IF NOT EXISTS idx_media_files_theme_name ON media_files (theme_name);
CREATE INDEX IF NOT EXISTS idx_media_files_extension_slug ON media_files (extension_slug);
