-- Migration 0010: Add language_id to email_templates and users for translatable emails.

-- Add language_id to email_templates (NULL = universal fallback)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'email_templates' AND column_name = 'language_id'
    ) THEN
        ALTER TABLE email_templates ADD COLUMN language_id INT REFERENCES languages(id);
    END IF;
END $$;

-- Drop old unique constraint/index on slug alone, replace with composite
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'email_templates' AND constraint_name = 'email_templates_slug_key'
    ) THEN
        ALTER TABLE email_templates DROP CONSTRAINT email_templates_slug_key;
    END IF;
END $$;
DROP INDEX IF EXISTS idx_email_templates_slug;

-- Unique: (slug, language_id) when language_id is NOT NULL
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_templates_slug_lang
ON email_templates (slug, language_id) WHERE language_id IS NOT NULL;

-- Unique: (slug) when language_id IS NULL (universal)
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_templates_slug_universal
ON email_templates (slug) WHERE language_id IS NULL;

-- Add language_id to users (preferred language for emails, NULL = site default)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'language_id'
    ) THEN
        ALTER TABLE users ADD COLUMN language_id INT REFERENCES languages(id);
    END IF;
END $$;
