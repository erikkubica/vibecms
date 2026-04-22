# Forms Extension — Production Readiness Roadmap

> **Goal:** Transform the forms extension from alpha MVP to a production-grade,
> GDPR-compliant form builder with file uploads, anti-spam, conditional logic,
> and polished UX matching the rest of the VibeCMS admin UI.

---

## Phase 1 — Critical Fixes & Data Model

### Field Builder
- [x] Merge `name` + `id` into single `key` field (auto-generated from label)
- [x] Fix field key input unfocusing on every keystroke (use index-based key)
- [x] Collapsible card UX matching block-type-editor pattern
+- [x] **Select/Radio options should support `{label, value}` pairs** — not just flat strings
+  - UI: two-column input (value | label) with add/remove rows
+  - Backward compat: `normalizeOptions()` converts old plain strings automatically
+- [x] **Add missing field types:**
+  - `number` — with min, max, step
+  - `url` — with pattern validation
+  - `file` — with allowed_types, max_size, multiple (see Phase 4)
+  - `hidden` — with default_value
+  - `range` — with min, max, step
+  - `gdpr_consent` — special checkbox with consent text (see Phase 3)
+- [x] **Field validation rules:**
+  - `min_length` / `max_length` (text, textarea, url, email, tel)
+  - `min` / `max` / `step` (number, range)
+  - `default_value` (all types)
+  - `help` (help text shown below field)
+- [ ] **Field width/layout:** `width` property (`full` | `half` | `third`) for grid layouts
+- [ ] **Pattern validation:** `pattern` (regex) for text/email/url fields

### Template System
- [x] Fix `{{.fields.name.label}}` render error — add top-level field access by ID
- [x] Fix LayoutTab default template uppercase keys → lowercase
+- [x] Fix LayoutTab templating guide (lowercase keys, shorthand access)
+- [x] **Remove `fields_by_name` from Go backend** — fields are now keyed by `id` only.
+      Each field also accessible at top level: `{{.email.label}}` shorthand.
+- [ ] **Pre-built layout templates** — dropdown with starter layouts:
+  - "Simple stacked" (current default)
+  - "Two-column grid"
+  - "Card style"
+  - "Compact inline"
+- [ ] **Template variable reference panel** — collapsible sidebar in LayoutTab showing
+      all available variables with copy-on-click badges

+### Database Migrations
+- [x] **Add `status` column to `form_submissions`** — `unread` (default) / `read` / `archived`
+      (set on insert in Go backend; DB migration still pending)
+- [ ] **Add `is_gdpr_consent` column to `form_submissions`** — boolean
+- [ ] **Add `ip_address` column to `form_submissions`** — extracted from metadata for easy filtering
+- [ ] **Add index** on `form_submissions(status)` and `form_submissions(created_at DESC)`
+- [ ] **New migration file** `20260423_submissions_status.sql` — add `status TEXT DEFAULT 'unread'` column

---

+## Phase 2 — Anti-Spam & Security
+
+### Honeypot
+- [x] **Go backend:** auto-inject honeypot field into rendered form HTML
+  - Hidden `website_url` input with `display:none` + `tabindex=-1` + `autocomplete=off`
+  - On submission: if honeypot filled → silently discard (log warning, return 200)
+  - Configurable per form: `settings.honeypot_enabled` (default: true)
+- [x] **SettingsTab:** Honeypot toggle added to Spam Protection section
+
+### Rate Limiting
+- [ ] **Go backend:** track submissions per IP in last N minutes
+  - Store in `form_rate_limits` table or in-memory map with TTL
+  - Configurable: `rate_limit` (max submissions per IP per hour, default: 10)
+  - Return 429 with user-friendly message when exceeded
+- [x] **SettingsTab:** Rate Limit input field added to Spam Protection section
+
+### CAPTCHA
+- [ ] **CAPTCHA provider support** — integrate via CoreAPI events/filters:
+  - Support: reCAPTCHA v3, hCaptcha, Cloudflare Turnstile
+  - Go backend: verify CAPTCHA response on submission via HTTP call to provider API
+  - Frontend: auto-inject CAPTCHA script tag + render in form
+- [x] **SettingsTab:** CAPTCHA provider selector + key fields (conditionally shown)
+
+### CSRF Protection
+- [ ] **Go backend:** generate CSRF token per form render
+  - Store in session or double-submit cookie pattern
+  - Validate on POST submission
+  - Auto-inject `<input type="hidden" name="_csrf" value="...">` into rendered forms
+
+### Input Sanitization
+- [x] **Go backend:** validate submitted values
+  - Required field validation
+  - Email format validation via `net/mail.ParseAddress`
+  - Return per-field validation errors: `{error: "VALIDATION_FAILED", fields: {...}}`
+- [ ] Strip HTML tags from text fields (unless explicitly allowed)
+- [ ] Validate URL format for url-type fields
+- [ ] Enforce max length from field definition
+- [ ] Client-side validation in the vibe-form block (see Phase 7)

---

+## Phase 3 — GDPR & Compliance
+
+### Consent Checkbox
+- [x] **New field type: `gdpr_consent`** — special checkbox in BuilderTab:
+  - Auto-sets required=true, disables the required toggle
+  - Configurable `consent_text` with default value
+  - Shows info banner in add-field form
+- [ ] **Go backend:** render GDPR consent as special checkbox in `renderFormHTML`
+  - Consent text rendered as `<label>` with link to privacy policy
+  - Stored as `is_gdpr_consent: true` in submission metadata
+- [ ] **Client-side:** prevent submission if GDPR checkbox unchecked
+
+### Data Retention
+- [x] **SettingsTab:** retention period selector (forever / 30/60/90/180/365 days)
+- [ ] **Go backend:** `CleanupOldSubmissions` method
+  - Called on a schedule (Tengo timer or plugin init + goroutine ticker)
+  - Delete submissions older than retention period
+  - Log count of deleted submissions
+
+### Privacy Settings
+- [x] **SettingsTab:** Privacy Policy URL input
+- [x] **SettingsTab:** Store IP addresses toggle
+
+### Right to Erasure
+- [ ] **SubmissionsList:** "Delete" button per submission (verify existing)
+- [ ] **Bulk delete:** select multiple → delete all
+- [ ] **Anonymize:** instead of delete, replace all field values with `[REDACTED]`
+- [ ] **Export:** single submission as JSON, all submissions as CSV (see Phase 5)

---

+### Multipart Form Data
+- [x] **Go backend:** `parseSubmissionBody()` handles both JSON and `multipart/form-data`
+  - Detects Content-Type, parses accordingly
+  - File fields skipped (placeholder for future file upload support)
+
+## Phase 4 — File Uploads

### Upload Field Type
- [ ] **BuilderTab:** `file` field type with options:
  - `allowed_types`: comma-separated MIME types / extensions (e.g. `pdf,doc,docx,jpg,png`)
  - `max_size`: in MB (default: 5MB)
  - `multiple`: boolean (allow multiple files)
  - `max_files`: when multiple=true (default: 5)
- [ ] **Frontend rendering:** file input with drag-and-drop zone styling
  - Show allowed types and max size as help text
  - Client-side file type + size validation

### Upload Processing (Go Backend)
- [ ] **Go backend:** handle multipart file uploads in submission handler
  - Parse `multipart/form-data` instead of JSON when files present
  - For each file field:
    1. Validate MIME type against `allowed_types`
    2. Validate file size against `max_size`
    3. Store file via `p.host.StoreFile(ctx, path, data)`
    4. Store the returned URL/path in the submission data
  - Submission data for file fields: `{field_id: {name, url, size, mime_type}}`
- [ ] **Go backend:** handle file deletion when submission is deleted
  - Call `p.host.DeleteFile()` for each file in the submission

### File Attachments in Emails
- [ ] **Notification system:** when a notification has `attach_files: true`
  - Fetch file data from stored paths
  - Attach to email (needs CoreAPI EmailRequest extension or inline base64)
  - **Design question:** Current `EmailRequest` only has `To`, `Subject`, `HTML`.
    - Option A: Extend `EmailRequest` with `Attachments []Attachment`
    - Option B: Use a filter/event that the email-manager picks up
    - Option C: Link to file URL in email body (simpler, no proto changes)
  - **Recommendation:** Start with Option C (link to file URLs in email body),
    implement Option A as a future improvement to the CoreAPI proto

### Submission Detail View
- [ ] **SubmissionsList detail dialog:** render file fields as download links
  - Show file name, size, type
  - Clickable link to file URL
  - Image files: thumbnail preview

---

## Phase 5 — Submission Management

### Submissions List Improvements
- [ ] **Pagination** — currently hardcoded limit 100
  - Server-side pagination with `page` and `per_page` params
  - UI: pagination controls at bottom
- [ ] **Filters:**
  - Date range picker (from / to)
  - Status filter (unread / read / all)
  - Form filter (already exists via `form_id` query param)
- [ ] **Sort:** by date (asc/desc), by form name
- [ ] **Status badges:** unread indicator (blue dot), read, archived
- [ ] **Mark as read:** automatic on "View Details" click
- [ ] **Bulk actions:** select multiple → mark read / delete / export

### CSV Export
- [ ] **Go backend:** `GET /forms/submissions/export?form_id=X&format=csv`
  - Generate CSV with columns = field IDs + metadata (submitted_at, ip)
  - Stream response or generate file + return URL
- [ ] **UI:** "Export CSV" button (already exists but not implemented)
  - Trigger download via `window.open()` or fetch + blob

### Submission Detail Improvements
- [ ] **Better field rendering:**
  - Match field labels (not just raw keys) by looking up form field definitions
  - Render file fields with download links
  - Render checkbox/boolean as ✓ / ✗
  - Render arrays/lists properly
- [ ] **Metadata section:** show IP, user agent, referer, submission time
- [ ] **Actions:** delete, mark unread, print-friendly view

---

+## Phase 6 — Email Notifications (Full Feature)
+
+### Notification Editor Improvements
+- [x] **Comprehensive template variable help** — collapsible reference panel in NotificationsTab
+  showing all variables with monospace formatting
+- [ ] **HTML email template builder** — visual toggle between plain text and HTML
+  - HTML mode: code editor with live preview
+  - Pre-built email templates: "Simple", "Table", "Card"
+- [x] **Auto-responder (confirmation email to submitter):**
+  - Notification type selector: Admin / Auto-Responder
+  - Auto-responder shows "Recipient Field" dropdown (email-type fields)
+  - Hides manual recipients input when auto-responder
+- [x] **Reply-To from field:** dropdown Select listing all email-type fields
+  - Falls back to text input if no email fields exist
+- [x] **CC/BCC fields** on notifications (collapsible section per notification)
+- [x] **Enable/Disable toggle** — Switch in notification card header
+- [ ] **Conditional notifications:** send to different recipients based on field values
+  - e.g. "If subject = 'Sales' → send to sales@, if 'Support' → support@"
+  - UI: condition builder (field → operator → value → recipients)
+
+### Backend Notification Improvements
+- [x] **GoTemplate-based email rendering** — replaced simple string replacement
+  with Go `html/template` using `.FormName`, `.FormSlug`, `.FormID`, `.SubmittedAt`,
+  `{{range .Data}}`, `{{.Field.email}}` etc.
+- [ ] **Email logging** — store sent notification in `form_notification_logs` table
+  - form_id, submission_id, notification_name, recipients, subject, sent_at, status
+  - Link from submission detail to notification log
+- [ ] **Process notifications asynchronously but with error tracking**
+  - Currently uses `go p.triggerNotifications()` — fire-and-forget
+  - Should track success/failure and surface in UI

---

+## Phase 7 — Form Rendering & Frontend Polish
+
+### Form Block (`vibe-form`)
+- [ ] **Client-side validation** before AJAX submit:
+  - Required fields
+  - Email format
+  - URL format
+  - File type/size (for file fields)
+  - Min/max length
+  - Pattern matching
+  - Show inline error messages per field
+- [ ] **Loading/submitting state** — disable button, show spinner (partially done)
+- [ ] **Success state** — animated success message, optional confetti 😄
+- [ ] **Error state** — per-field errors from server, scroll to first error
+- [ ] **Redirect after submission** — support redirect URL from form settings
+- [ ] **Custom CSS class** — form-level CSS class for theme styling hooks
+
+### Preview Tab
+- [x] **Auto-refresh on field/template changes** — 1.5s debounced auto-refresh
+  - Uses `formRef` for stable callback, `isMountedRef` for cleanup
+  - Shows "Auto-updating…" pulse indicator during debounce
+  - Manual refresh button still available for force-refresh
+- [ ] **Device frame toggle** — desktop / tablet / phone preview widths
+- [ ] **Dark mode preview** — toggle to see form in dark theme

### Builder Tab Polish
- [ ] **Drag-and-drop reordering** — replace up/down buttons with actual drag handles
  - Use HTML5 drag API or a lightweight DnD library
- [ ] **Duplicate field** — button to clone a field
- [ ] **Field preview** — mini inline preview of the field type in collapsed state
- [ ] **Empty state illustration** — better visual for "no fields yet"

---

## Phase 8 — Form Management

### Form List
- [ ] **Submission count badge** per form in list view
- [ ] **Last submission date** column
- [ ] **Quick actions:** duplicate, delete, view submissions
- [ ] **Bulk actions:** select multiple → delete

### Form Duplication
- [ ] **Go backend:** `POST /forms/{id}/duplicate`
  - Clone form with name "Copy of {original name}"
  - Generate new slug: `{original-slug}-copy`
  - Deep copy fields, layout, notifications, settings
- [ ] **UI:** "Duplicate" button in form list and editor

### Form Import/Export
- [ ] **Export:** `GET /forms/{id}/export` → JSON file with complete form definition
- [ ] **Import:** `POST /forms/import` with JSON body → create new form
- [ ] **UI:** import/export buttons in form list

### Webhooks
- [ ] **SettingsTab:** webhook URL field
  - `webhook_url`: URL to POST submission data to
  - `webhook_method`: POST (default)
  - `webhook_headers`: custom headers (JSON)
- [ ] **Go backend:** after successful submission + notifications, fire webhook
  - POST JSON payload with form data + metadata
  - Handle timeout (5s) and log failures
  - Retry logic (optional, stretch goal)

### Form Analytics (Stretch)
- [ ] **Submission count** per form over time
- [ ] **Conversion tracking** — views vs submissions (needs view tracking)
- [ ] **Dashboard widget** — recent submissions across all forms

---

## Phase 9 — Documentation & Developer Experience

### Template Variable Reference
- [ ] **In-app help panel** in LayoutTab with all variables, copy-paste friendly
- [ ] **Interactive examples** — click a variable to insert it at cursor position

### Extension API Docs
- [ ] **Update `docs/extension_api.md`** with forms extension architecture
- [ ] **Document form block** (`vibe-form`) in theme development guide
- [ ] **Document public API** — `POST /api/ext/forms/submit/{slug}` for headless usage

### Tengo Scripts
- [ ] **Pre-submission hook** — Tengo script for custom validation
- [ ] **Post-submission hook** — Tengo script for custom processing
- [ ] **Form rendering filter** — modify form HTML before output

---

## Phase 10 — Testing & Quality

### Automated Tests
- [ ] **Go unit tests** — form CRUD, submission handling, notification processing
- [ ] **Go integration tests** — full submission flow with test database
- [ ] **Frontend component tests** — BuilderTab, PreviewTab field interactions

### Manual QA Checklist
- [ ] Create form with all field types → verify rendering
- [ ] Submit form → verify data stored correctly
- [ ] Submit with required fields empty → verify validation
- [ ] Submit with honeypot filled → verify silent discard
- [ ] Exceed rate limit → verify 429 response
- [ ] File upload: valid file → verify stored
- [ ] File upload: invalid type → verify rejected
- [ ] File upload: oversized → verify rejected
- [ ] Email notification sent with correct template variables
- [ ] GDPR consent: cannot submit without checking
- [ ] CSV export: correct columns and data
- [ ] Form duplication: all fields copied
- [ ] Delete form: submissions cascade deleted
- [ ] Preview: matches actual rendered form

---

## Quick Wins (Do First)

These are small changes with outsized impact:

1. ~~**Select options with label+value** — every real form needs this~~ ✅
2. ~~**Honeypot field** — 5 min implementation, blocks 90% of spam~~ ✅
3. ~~**Submission status (read/unread)** — small migration, big UX win~~ ✅
4. ~~**Auto-refresh preview** — remove manual refresh button, debounce~~ ✅
5. **CSV export** — wire up the existing button
6. ~~**Form slug auto-generation from name** — like block-type-editor does it~~ ✅
7. ~~**Reply-to field dropdown** — select email field instead of typing ID~~ ✅
8. ~~**Remove `fields_by_name` from Go backend** — dead code now~~ ✅
9. ~~**GDPR consent field type** — special checkbox with consent text~~ ✅
10. ~~**Template-based notification rendering** — Go templates in emails~~ ✅
11. ~~**Notification type (admin / auto-responder)** — confirmation emails~~ ✅
12. ~~**CAPTCHA settings UI** — provider selector + keys~~ ✅
13. ~~**Missing field types** — number, url, file, hidden, range~~ ✅
14. ~~**Field validation rules** — min/max length, min/max/step, default_value, help~~ ✅

### Remaining Quick Wins

1. **CSV export** — wire up the "Export CSV" button in SubmissionsList
2. **DB migration for `status` column** — `ALTER TABLE form_submissions ADD COLUMN status TEXT DEFAULT 'unread'`
3. **Rate limiting in Go backend** — in-memory map with TTL, respect `rate_limit` setting
4. **Client-side validation** — required, email, URL, min/max length in vibe-form block
5. **Form duplication** — `POST /forms/{id}/duplicate` endpoint

---

*Last updated: 2025-04-23*
*Status: Phase 1 ✅ complete · Phase 2 (honeypot, validation) · Phase 3 (GDPR, retention UI) · Phase 6 (notifications, auto-responder) · 14/19 quick wins done*