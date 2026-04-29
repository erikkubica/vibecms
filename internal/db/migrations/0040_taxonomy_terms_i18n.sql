-- Add per-language storage to taxonomy terms.
--
-- Mirrors the content_nodes pattern: each language version of a term is its
-- own row, linked to siblings via translation_group_id. Slug uniqueness is
-- per (node_type, taxonomy, language_code) so the EN "documentation" and
-- the PT "documentação" can both exist without colliding.

ALTER TABLE taxonomy_terms
    ADD COLUMN IF NOT EXISTS language_code VARCHAR(10) NOT NULL DEFAULT 'en',
    ADD COLUMN IF NOT EXISTS translation_group_id UUID;

-- Backfill: stamp every existing row with the site's default language.
-- Existing terms predate i18n, so they all belong to that language.
DO $$
DECLARE
    default_code TEXT;
BEGIN
    SELECT code INTO default_code FROM languages WHERE is_default = TRUE LIMIT 1;
    IF default_code IS NOT NULL THEN
        UPDATE taxonomy_terms
        SET language_code = default_code
        WHERE language_code = 'en' OR language_code = '';
    END IF;
END $$;

-- Replace the old slug-uniqueness constraint with one that's
-- language-scoped, so translations don't collide on slug.
ALTER TABLE taxonomy_terms
    DROP CONSTRAINT IF EXISTS taxonomy_terms_node_type_taxonomy_slug_key;

ALTER TABLE taxonomy_terms
    ADD CONSTRAINT taxonomy_terms_node_type_taxonomy_slug_lang_key
    UNIQUE (node_type, taxonomy, slug, language_code);

CREATE INDEX IF NOT EXISTS idx_taxonomy_terms_translation_group
    ON taxonomy_terms (translation_group_id)
    WHERE translation_group_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_taxonomy_terms_language
    ON taxonomy_terms (node_type, taxonomy, language_code);
