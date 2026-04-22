-- Up
ALTER TABLE form_submissions ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'unread';
CREATE INDEX IF NOT EXISTS idx_form_submissions_status ON form_submissions(status);

-- Down
-- DROP INDEX IF EXISTS idx_form_submissions_status;
-- ALTER TABLE form_submissions DROP COLUMN IF EXISTS status;
