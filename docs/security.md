# VibeCMS Security Hardening Documentation

## About This Document

**Purpose:** Security controls, hardening checklist, and threat model for this specific application. Defines the minimum security bar that must be met before production launch.

**How AI tools should use this:** Apply these controls when generating authentication code, API handlers, database queries, and infrastructure configuration; flag any generated code that violates a control listed here.

**Consistency requirements:** Security boundaries must align with architecture.md; authentication model must match api-spec.md auth requirements; sensitive data fields must match database-schema.md column definitions.

VibeCMS is designed to be a fast and secure foundation for agency-managed websites. Because each site runs as its own independent system, a security issue on one site stays contained. However, since the system handles sensitive content, automated AI integrations, and custom scripting, we must ensure the core engine is resilient against common web attacks and specialized threats like script breakouts or LLM prompt injection. This document outlines how we protect the data, the server, and the agency’s reputation.

---

### Threat Model

| Threat | Affected Component | Likelihood | Impact | Mitigation |
|--------|--------------------|------------|--------|------------|
| **Tengo Sandbox Escape** | Tengo Scripting Engine | Low | High | Use restricted standard library; disable `os` and `io` modules; enforce 10ms execution timeout. |
| **Credential Stuffing** | Admin UI Login | High | High | Argon2id hashing; aggressive rate limiting on `/admin/login`; exponential backoff for failed attempts. |
| **JSONB Injection** | Content Node Service | Medium | Medium | Use GORM's parameterized queries; validate all block data against JSON-schemas before SQL insertion. |
| **AI Prompt Injection** | AIService / SEO Gen | Medium | Low | Never pass raw user input directly to LLM; wrap all inputs in fixed system instruction templates. |
| **Monitoring API Key Leak** | Agency Monitoring API | Low | Medium | Keys stored as SHA-256 hashes; read-only access enforced at middleware; IP-based allowlisting recommended. |
| **Stored XSS in Blocks** | Jet Rendering Engine | Medium | High | Use `bluemonday` to sanitize HTML in `blocks_data` on save; use Jet's automatic HTML escaping. |

### Authentication Controls

Based on `api-spec.md`, VibeCMS uses a dual authentication strategy.

*   **Admin UI (Stateful):**
    *   **Storage:** Secure, HttpOnly, SameSite=Lax cookies.
    *   **Expiry:** 24 hours total lifespan; idle timeout is not implemented in MVP.
    *   **Brute-Force Protection:** Max 5 failed attempts per 15 minutes per IP. Lockdown for 30 minutes following threshold breach.
    *   **Session Fixation:** Regenerate `session_id` upon every successful login.
*   **Monitoring API (Stateless):**
    *   **Mechanism:** Static Bearer Token (`vibe_` prefix).
    *   **Storage:** Tokens are stored in `api_tokens` as SHA-256 hashes.
    *   **Key Rotation:** Managed via CLI command `vibecms keys rotate`. Immediate invalidation of all active monitoring keys.

### Authorization Controls

| Role | Access Level | Enforcement Mechanism | Unauthorized Response |
|------|--------------|-----------------------|-----------------------|
| **Admin** | Full System Access | Role-check Middleware | 403 Forbidden |
| **Editor** | Content/Media CRUD only | Middleware + Field Masking | 403 Forbidden |
| **Agency-Manager** | Read-only Monitoring/Logs | Scoped API Keys / Scoped UI | 403 Forbidden |

*   **Mechanism:** RBAC is enforced via a bitmask check in the Fiber/Echo middleware before reaching the handler. 
*   **Unauthorized Request:** Return `403 Forbidden` with body `{"error": "insufficient_permissions"}`. No data from the database should be leaked in the error response.

### Input Validation Checklist

| Endpoint | Field | Allowed Type | Max Length | Error Case |
|----------|-------|--------------|------------|------------|
| `POST /admin/login` | `email` | String (Email format) | 255 | 401 Unauthorized |
| `POST /admin/login` | `password` | String | 72 | 401 Unauthorized |
| `PATCH /api/nodes/:id` | `slug` | String (a-z, 0-9, -) | 255 | 422 Unprocessable Entity |
| `PATCH /api/nodes/:id` | `blocks_data`| JSON (Array) | 2MB | 400 Bad Request |
| `POST /api/media` | `file` | Multipart (WebP/JPG/PNG/SVG) | 10MB | 413 Payload Too Large |
| `GET /api/stats` | `Authorization`| String (Bearer) | 64 | 401 Unauthorized |

*   **JSONB Validation:** Every block in `blocks_data` must be validated against the JSON-schema defined for its `type` using `xeipuuv/gojsonschema`.
*   **SVG Protection:** All uploaded SVGs must pass through a strict XML whitelist to strip `<script>` and `on*` attributes.

### Rate Limiting

Rate limiting is enforced at the middleware level using an in-memory sliding window.

*   **Public Site Content:** No rate limit by default (handled by reverse proxy).
*   **Admin Login (`/admin/login`):** 5 requests per 15 minutes per IP.
*   **Agency Monitoring API (`/api/v1/monitor/*`):** 30 requests per minute per API Key.
*   **AI Suggestions (`/api/ai/*`):** 10 requests per minute per User (to prevent API cost spikes).
*   **Exceeded Response:** HTTP 429. Headers include `Retry-After`. Body: `{"error": "rate_limit_exceeded"}`.

### Secrets Management

*   **Environment Variables:** Used for `DATABASE_URL`, `RESEND_API_KEY`, and `OPENAI_API_KEY`.
*   **Database Encryption:** SMTP passwords and S3 Secrets stored in `site_settings` are encrypted at rest using AES-256-GCM. The key is derived from the machine ID and an environment-provided salt.
*   **License Key:** `vibe.key` sits on the filesystem. If the file is modified or the Ed25519 signature is invalid, the system enters "Soft-Fail" mode.
*   **Compromise Plan:**
    1.  Rotate `MASTER_MONITORING_KEY` in environment variables.
    2.  Invalidate all entries in the `sessions` table via SQL.
    3.  Rotate S3/Resend API keys and update the CMS via the Admin UI.

### Pre-Launch Security Checklist

1.  [ ] Verify `Argon2id` is used for all new user passwords.
2.  [ ] Confirm `os` and `io` modules are stripped from the Tengo VM environment.
3.  [ ] Test that a `403 Forbidden` is returned when an **Editor** tries to access `/admin/settings`.
4.  [ ] Ensure all `PATCH` requests to `content_nodes` are protected by a CSRF token.
5.  [ ] Run a sample Tengo script that attempts a `math.sin` loop to verify 10ms execution timeout.
6.  [ ] Verify that a `vibe_` monitoring key is stored as a hash and cannot be retrieved via the API.
7.  [ ] Check that files uploaded to S3-compatible storage have "Private" ACLs by default.
8.  [ ] Confirm `bluemonday` sanitizer is triggered on any `RichText` field type within a block.
9.  [ ] Audit `jet` templates to ensure no `{{ .Raw }}` or equivalent unsafe filters are used on user input.
10. [ ] Verify that the binary does not run as `root` in the production Dockerfile/service.