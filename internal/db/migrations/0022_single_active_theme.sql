-- Enforce only one active theme at a time at the database level.
CREATE UNIQUE INDEX IF NOT EXISTS idx_themes_single_active ON themes ((true)) WHERE is_active = true;
