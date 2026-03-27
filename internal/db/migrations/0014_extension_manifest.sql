-- Add manifest JSONB column to extensions table
ALTER TABLE extensions ADD COLUMN IF NOT EXISTS manifest JSONB NOT NULL DEFAULT '{}';
