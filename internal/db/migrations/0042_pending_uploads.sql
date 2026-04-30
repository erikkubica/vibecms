-- pending_uploads tracks the three-step presigned-upload flow used by
-- core.<kind>.upload_init / PUT /api/uploads/<token> / core.<kind>.upload_finalize.
-- The token is the auth for the PUT route, so high entropy + a short TTL +
-- a single-row state machine is the whole security story.

CREATE TABLE IF NOT EXISTS pending_uploads (
    token         VARCHAR(64)  PRIMARY KEY,
    kind          VARCHAR(16)  NOT NULL,
    user_id       BIGINT       NOT NULL,
    filename      TEXT         NOT NULL DEFAULT '',
    mime_type     TEXT         NOT NULL DEFAULT '',
    max_bytes     BIGINT       NOT NULL,
    state         VARCHAR(16)  NOT NULL DEFAULT 'pending',
    size_bytes    BIGINT,
    sha256        CHAR(64),
    temp_path     TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ  NOT NULL,
    finalized_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pending_uploads_state_expires
    ON pending_uploads (state, expires_at);
