# Forms Extension — Developer Reference

The Forms extension (`extensions/forms/`) is a production-grade form builder that ships as part of VibeCMS. This document covers everything an extension author needs to integrate with it or build on top of it.

---

## Public Submit API

### `POST /forms/submit/{slug}`

Submit data to a published form. Accepts both JSON and multipart/form-data.

This is a **public route** — registered on the public Fiber app via the `public_routes` entry in `extension.json`. It does **not** sit under `/admin/api/ext/forms/...` (that prefix is the auth-protected admin proxy). Any browser can post to it directly.

**JSON body:**

```json
{
  "name": "Alice",
  "email": "alice@example.com",
  "message": "Hello!"
}
```

**Multipart/form-data:** Use standard `multipart/form-data` encoding with file inputs mapped to their field ID.

**Response envelope.** Every response — success or error — uses a single, predictable shape:

```jsonc
// Success
{"success": true, "message": "Submission received"}

// Error
{"error": "<MACHINE_CODE>", "message": "Human-readable text", "fields": { /* optional, validation only */ }}
```

| Status | Body | Meaning |
|---|---|---|
| `200 OK` | `{"success": true, "message": "Submission received"}` | Accepted |
| `422 Unprocessable Entity` | `{"error": "VALIDATION_FAILED", "fields": {"email": "Invalid email format"}}` | Per-field validation errors |
| `422 Unprocessable Entity` | `{"error": "CAPTCHA_FAILED", "message": "CAPTCHA verification failed"}` | CAPTCHA token rejected |
| `429 Too Many Requests` | `{"error": "RATE_LIMITED", "message": "Too many submissions. Try again later."}` | Rate limit exceeded |
| `400 Bad Request` | `{"error": "INVALID_BODY", "message": "..."}` | Malformed request body |
| `404 Not Found` | `{"error": "FORM_NOT_FOUND", "message": "Form not found"}` | Unknown slug |
| `500 Internal Server Error` | `{"error": "STORE_FAILED", "message": "..."}` | Server-side persistence failure |

**Spam protection notes:**

- **Honeypot.** When `honeypot_enabled` is true on the form (default), the rendered form contains a hidden `<input name="website_url">`. A non-empty value in this field silently returns `200 OK` with no submission stored — bots see "success" and move on. Real users never see or fill this field. The field name is intentionally generic (`website_url`) rather than `_hp` so naive bots populate it.
- **CAPTCHA.** When `captcha_provider` is `recaptcha` / `hcaptcha` / `turnstile`, include the provider token in the `_captcha_token` form field. The bundled `vibe-form` block's client script (`extensions/forms/blocks/vibe-form/script.js`) handles token retrieval automatically.

---

## Webhook Payload

When a form has `webhook_url` configured, a POST is fired to that URL after each successful submission:

```json
{
  "event": "form.submitted",
  "form_id": 42,
  "form_slug": "contact-us",
  "submission_id": 1001,
  "submitted_at": "2026-04-25T11:00:00Z",
  "data": {
    "name": "Alice",
    "email": "alice@example.com",
    "message": "Hello!"
  },
  "metadata": {
    "ip": "1.2.3.4",
    "user_agent": "Mozilla/5.0 ..."
  }
}
```

---

## `forms:submitted` Event

After every accepted submission the plugin emits `forms:submitted` on the CMS event bus. Other extensions can subscribe to this event in Tengo or Go.

**Payload keys:**

| Key | Type | Description |
|---|---|---|
| `form_id` | `float64` | Numeric ID of the form |
| `form_slug` | `string` | URL-safe slug |
| `submission_id` | `float64` | Numeric ID of the submission row |
| `data` | `map[string]any` | Submitted field values keyed by field ID |
| `metadata` | `map[string]any` | `ip`, `user_agent`, `referer` (if `store_ip` is enabled) |

**Tengo example:**

```tengo
events := import("core/events")
log    := import("core/log")

events.on("forms:submitted", "handlers/on_form_submit")
```

```tengo
// scripts/handlers/on_form_submit.tengo
log := import("core/log")

log.info("Form submitted", {
    form:       event.payload.form_slug,
    submission: event.payload.submission_id
})
```

**Go plugin example (inside `Init`):**

```go
p.host.Subscribe(ctx, "forms:submitted", func(payload map[string]any) {
    formSlug, _ := payload["form_slug"].(string)
    submissionID, _ := payload["submission_id"].(float64)
    p.host.Log(ctx, "info",
        fmt.Sprintf("Form %s got submission %d", formSlug, int(submissionID)),
        nil,
    )
})
```

---

## Condition Engine Reference

The Forms extension uses a recursive condition group evaluator for field-level show/hide rules and conditional notification routing. The same engine is exported as `EvaluateGroup` and `EvaluateField` in `cmd/plugin/conditions.go` for use in tests.

### Group Structure

```json
{
  "all": [
    { "field": "subject", "operator": "equals", "value": "Support" }
  ]
}
```

Use `"all"` for AND logic, `"any"` for OR logic. Groups can be nested.

### Supported Operators

| Operator | Meaning |
|---|---|
| `equals` | String or number equality |
| `not_equals` | String or number inequality |
| `contains` | Case-insensitive substring match |
| `not_contains` | Case-insensitive substring not present |
| `gt` / `gte` / `lt` / `lte` | Numeric comparison |
| `in` | Value appears in list (array `value`) |
| `not_in` | Value does not appear in list |
| `matches` | Value matches regex pattern |
| `is_empty` | Field absent, blank string, `false`, or empty collection |
| `is_not_empty` | Opposite of `is_empty` |

---

## Email Notification Templates

Notification bodies are rendered with Go's `html/template`. Available variables:

| Variable | Type | Description |
|---|---|---|
| `{{.FormName}}` | string | Human-readable form name |
| `{{.FormSlug}}` | string | URL-safe slug |
| `{{.FormID}}` | uint | Numeric form ID |
| `{{.SubmittedAt}}` | time.Time | Submission timestamp |
| `{{range .Data}}{{.Key}} — {{.Value}}{{end}}` | slice | All submitted fields |
| `{{index .Field "email"}}` | string | Direct field access by ID (`.Field`, singular — see code) |

`.Fields` was renamed to `.Field` in v2 to match the Go struct. The shorthand `{{.email}}` is **not** available in notification templates — only in the form layout templates rendered by `forms:render`.

---

## Default Form Layouts (Tailwind dependency)

The four bundled form-style presets (`default`/"simple", `grid`, `card`, `inline`) at `extensions/forms/cmd/plugin/templates/*.html` use Tailwind utility classes (`max-w-2xl`, `bg-indigo-600`, `rounded-lg`, …). They render correctly only when the host page already loads Tailwind.

For themes that don't ship Tailwind, the canonical pattern is to **own the form layout from the theme**:

```tengo
// themes/<theme>/scripts/theme.tengo
events := import("core/events")
assets := import("core/assets")

layout := assets.read("forms/contact.html")  // your theme's CSS classes
if is_error(layout) { layout = "" }

events.emit("forms:upsert", {
  slug:   "contact",
  name:   "Contact",
  layout: layout,            // theme-owned HTML, theme CSS classes
  fields: [ /* … */ ]
})
```

See `themes/README.md` §10 (Forms wiring) for the full theme-owned pattern.

---

## Known Limits & Deferred Items

- **E2E tests** (`e2e/forms.spec.ts`) require a live PostgreSQL + SMTP catcher environment and are not run in CI yet. See `e2e/playwright.config.ts` for setup.
- **Oversize admin-ui files**: `FormEditor.tsx` (~400 LOC), `NotificationCard.tsx` (~370 LOC), `FormsList.tsx` (~325 LOC), and `SubmissionDetail.tsx` (~334 LOC) exceed the 300-line soft limit. Each has a natural split point but was deferred to avoid a risky mid-cycle refactor. Tracked for a follow-up pass.
- **File email attachments**: email notifications link to uploaded files rather than attaching them inline. Full attachment support requires a `CoreAPI` proto extension (`EmailRequest.Attachments`).
- **CSRF tokens**: not yet implemented. The honeypot + rate limiter cover the primary spam vectors in the interim.
- **No default-seeded form**: a fresh install with no theme has zero forms, so the bundled `vibe-form` block's preview shows "Failed to load form" until at least one form is created (or seeded by a theme via `forms:upsert`).
