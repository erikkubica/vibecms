-- Migration 0020: Email Layouts
-- Creates email_layouts table for base HTML wrappers applied to outgoing emails.

CREATE TABLE IF NOT EXISTS email_layouts (
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(150) NOT NULL,
    language_id   INT REFERENCES languages(id),
    body_template TEXT NOT NULL,
    is_default    BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One layout per language
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_layouts_lang
    ON email_layouts (language_id) WHERE language_id IS NOT NULL;

-- At most one universal (NULL language) layout
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_layouts_universal
    ON email_layouts ((true)) WHERE language_id IS NULL;

-- Seed default universal layout
INSERT INTO email_layouts (name, language_id, body_template, is_default)
SELECT 'Default Layout', NULL, '<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=0" />
</head>
<body style="margin:0; padding:0; background-color:#f1f5f9; font-family:-apple-system, BlinkMacSystemFont, ''Segoe UI'', Roboto, ''Helvetica Neue'', Arial, sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f1f5f9; padding:32px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px; width:100%; border-radius:12px; overflow:hidden; box-shadow:0 1px 3px rgba(0,0,0,0.1);">
          <!-- Header -->
          <tr>
            <td style="background-color:#2563eb; padding:24px 32px; text-align:center;">
              <h1 style="margin:0; color:#ffffff; font-size:22px; font-weight:600; letter-spacing:-0.02em;">{{.site.site_name}}</h1>
            </td>
          </tr>
          <!-- Content -->
          <tr>
            <td style="background-color:#ffffff; padding:32px;">
              {{.email_body}}
            </td>
          </tr>
          <!-- Footer -->
          <tr>
            <td style="background-color:#f8fafc; padding:20px 32px; text-align:center; border-top:1px solid #e2e8f0;">
              <p style="margin:0; color:#94a3b8; font-size:13px;">&copy; {{.site.site_name}}</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>', true
WHERE NOT EXISTS (
    SELECT 1 FROM email_layouts WHERE language_id IS NULL
);
