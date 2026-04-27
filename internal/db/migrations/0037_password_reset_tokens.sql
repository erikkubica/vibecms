-- Password reset tokens. One row per active reset request.
-- Tokens are 32 random bytes; only the SHA-256 hash is stored. The
-- raw token is sent to the user's email and verified by re-hashing.
-- Single-use is enforced via used_at — a row is consumed by setting
-- used_at on success, then expired rows are pruned by the retention
-- cron (see auth.SessionService.StartCleanupLoop and similar pattern).
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id          SERIAL PRIMARY KEY,
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(64) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    ip_address  VARCHAR(64),
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_user_id ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_expires ON password_reset_tokens(expires_at) WHERE used_at IS NULL;
