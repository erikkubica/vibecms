# Forms Extension — Status: Production v2.0

All major roadmap items from the original ROADMAP have shipped. The Forms extension is considered production-ready as of v2.0.0.

For the full developer reference (public API, event payloads, condition engine, email templates), see **[docs/forms.md](../../docs/forms.md)**.

## What shipped in v2.0

- Complete field builder: text, email, tel, number, url, range, select, radio, checkbox, textarea, hidden, file, gdpr\_consent
- Field validation rules: required, min/max length, min/max/step, pattern, help text
- Conditional field show/hide (display\_when) with recursive AND/OR condition groups
- Honeypot spam protection (configurable per form)
- In-memory rate limiting (configurable per form, 429 response on breach)
- CAPTCHA support: Cloudflare Turnstile, hCaptcha, reCAPTCHA v3
- File uploads via CoreAPI StoreFile (multipart/form-data + file size/type validation)
- GDPR consent field type with privacy policy URL
- Data retention: background goroutine prunes submissions older than configured threshold
- Email notifications: admin and auto-responder types, Go template variables, CC/BCC, Reply-To
- Conditional notifications (route\_when condition group)
- Webhooks: POST on submission, custom headers, 5s timeout, per-call log row
- `forms:submitted` event emitted on every accepted submission
- Go template layout engine with four starter layouts (stacked, grid, card, inline)
- Submission management: read/unread/archived status, bulk actions, date range filter, CSV export
- Form duplication and JSON import/export
- 242 unit tests via FakeHost (no live database required)
- React admin UI: FormsList, FormEditor (Builder, Layout, Preview, Notifications, Settings tabs), SubmissionsList, FormFieldSelector

## What remains

- **E2E tests** — `e2e/forms.spec.ts` requires live PostgreSQL + SMTP. Not wired to CI yet.
- **Oversize admin-ui files** — `SubmissionsList.tsx`, `NotificationCard.tsx`, `PreviewTab.tsx`, `FormsList.tsx` each exceed the 300-line soft limit. Deferred to a follow-up split pass.
- **CSRF tokens** — honeypot + rate limiter cover the primary vectors; CSRF is a stretch goal.
- **File email attachments** — notifications link to file URLs today; true attachments need a CoreAPI proto change.
