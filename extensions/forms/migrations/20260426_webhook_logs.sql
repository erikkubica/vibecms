-- Up
CREATE TABLE IF NOT EXISTS form_webhook_logs (
    id SERIAL PRIMARY KEY,
    form_id INTEGER NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    submission_id INTEGER REFERENCES form_submissions(id) ON DELETE SET NULL,
    url TEXT NOT NULL,
    request_body TEXT,
    status_code INTEGER NOT NULL DEFAULT 0,
    response_body TEXT,
    error TEXT,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_form_webhook_logs_form_id ON form_webhook_logs(form_id);
CREATE INDEX IF NOT EXISTS idx_form_webhook_logs_created_at ON form_webhook_logs(created_at DESC);

-- Down
-- DROP INDEX IF EXISTS idx_form_webhook_logs_created_at;
-- DROP INDEX IF EXISTS idx_form_webhook_logs_form_id;
-- DROP TABLE IF EXISTS form_webhook_logs;
