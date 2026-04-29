-- Drop the "all languages" sentinel from site_settings.
--
-- Previously language_code='' meant "shared across all languages, used as a
-- fallback when no per-locale row exists". We now require every row to carry
-- a real language code; reads fall back to the default-language row instead
-- of a synthetic shared row. This keeps the data model uniform with content
-- nodes (every row has a language) and removes a confusing UX option.
--
-- Backfill: rewrite legacy '' rows to whichever language is currently
-- is_default=true. If the user later marks a different language as default,
-- already-stored rows stay where they are; the read-fallback follows the new
-- default automatically.
--
-- If there's no default language (fresh install before language seeding),
-- leave '' rows alone — the column still allows '' so nothing breaks; the
-- next migration / language seed can revisit.

UPDATE site_settings
SET language_code = (SELECT code FROM languages WHERE is_default = TRUE LIMIT 1)
WHERE language_code = ''
  AND EXISTS (SELECT 1 FROM languages WHERE is_default = TRUE);
