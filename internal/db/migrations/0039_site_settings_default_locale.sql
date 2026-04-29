-- Drop the "all languages" sentinel from site_settings.
--
-- Previously language_code='' meant "shared across all languages, used as a
-- fallback when no per-locale row exists". We now require every row to carry
-- a real language code; reads fall back to the default-language row instead
-- of a synthetic shared row. This keeps the data model uniform with content
-- nodes (every row has a language) and removes a confusing UX option.
--
-- Backfill: rewrite legacy '' rows to whichever language is currently
-- is_default=true. If a per-default-locale row already exists for the same
-- key, the per-locale row wins (operator explicitly set it); the '' row is
-- discarded. Without dedup the UPDATE collides with the composite PK.
--
-- If there's no default language (fresh install before language seeding),
-- '' rows are left alone — the next language seed can revisit.

DO $$
DECLARE
    default_code TEXT;
BEGIN
    SELECT code INTO default_code FROM languages WHERE is_default = TRUE LIMIT 1;
    IF default_code IS NULL THEN
        RETURN;
    END IF;

    -- Drop '' rows whose key already has a row at the default language —
    -- the explicit per-locale row is the source of truth.
    DELETE FROM site_settings AS s
    WHERE s.language_code = ''
      AND EXISTS (
          SELECT 1 FROM site_settings t
          WHERE t.key = s.key AND t.language_code = default_code
      );

    -- Relabel the remaining '' rows to the default language.
    UPDATE site_settings
    SET language_code = default_code
    WHERE language_code = '';
END $$;
