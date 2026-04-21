-- Migration 006: extension ownership metadata on media_files.
--
-- Extensions declare image assets in extension.json under "assets". On
-- activation the media-manager plugin (subscribed to extension.activated)
-- imports those files tagged source='extension'. On deactivation
-- (extension.deactivated) the rows are deleted.
--
-- Sibling to 005_theme_ownership — same pattern, different owner axis.

ALTER TABLE media_files ADD COLUMN IF NOT EXISTS extension_slug TEXT;

CREATE INDEX IF NOT EXISTS idx_media_files_source_extension
  ON media_files(source, extension_slug) WHERE source = 'extension';

CREATE UNIQUE INDEX IF NOT EXISTS uq_media_files_extension_asset_key
  ON media_files(extension_slug, asset_key)
  WHERE source = 'extension' AND asset_key IS NOT NULL;
