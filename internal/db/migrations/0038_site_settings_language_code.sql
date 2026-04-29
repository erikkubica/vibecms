-- Add language_code to site_settings so values can be per-locale.
-- Empty string ('') means "fallback / applies to all languages" — picked over
-- NULL because primary-key columns can't be NULL in Postgres and treating ''
-- as a sentinel keeps lookups simple.
--
-- Read pattern: try (key, current_locale) first, fall back to (key, '').
-- Write pattern: upsert keyed by (key, language_code).

ALTER TABLE site_settings
    ADD COLUMN IF NOT EXISTS language_code VARCHAR(8) NOT NULL DEFAULT '';

ALTER TABLE site_settings DROP CONSTRAINT IF EXISTS site_settings_pkey;

ALTER TABLE site_settings ADD PRIMARY KEY (key, language_code);
