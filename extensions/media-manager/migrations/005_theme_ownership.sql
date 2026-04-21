-- Migration 005: theme ownership metadata on media_files.
--
-- Themes declare image assets in theme.json under "assets". On activation,
-- the media-manager plugin (subscribed to theme.activated) imports those
-- files into media_files tagged source='theme'. On deactivation (plugin
-- subscribed to theme.deactivated) those rows are deleted.
--
-- Core never reads or writes these columns; only the media-manager plugin.

ALTER TABLE media_files ADD COLUMN IF NOT EXISTS source        TEXT NOT NULL DEFAULT 'user';
ALTER TABLE media_files ADD COLUMN IF NOT EXISTS theme_name    TEXT;
ALTER TABLE media_files ADD COLUMN IF NOT EXISTS content_hash  TEXT;
ALTER TABLE media_files ADD COLUMN IF NOT EXISTS asset_key     TEXT;

CREATE INDEX IF NOT EXISTS idx_media_files_source_theme
  ON media_files(source, theme_name) WHERE source = 'theme';

CREATE UNIQUE INDEX IF NOT EXISTS uq_media_files_theme_asset_key
  ON media_files(theme_name, asset_key) WHERE source = 'theme' AND asset_key IS NOT NULL;
