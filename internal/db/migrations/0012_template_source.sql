-- Add source and theme_name columns to templates table
ALTER TABLE templates ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'custom';
ALTER TABLE templates ADD COLUMN IF NOT EXISTS theme_name VARCHAR(100);
