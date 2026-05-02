# Squilla Security

This document describes the **current** security posture: what's implemented, where, and how a kernel or extension developer is expected to use it. For threat-model history and the per-issue audit trail, see commits prefixed `feat(security):`, `feat(auth):`, `feat(secrets):`, and `feat(themes):`.

---

## 1. Defense in Depth Summary

| Layer | Mechanism | Where |
|---|---|---|
| Boot | Refuse to start with unsafe config | `internal/config/config.go::Validate` |
| Transport | TLS expected at edge (Coolify/proxy); HSTS at app level | Fiber middleware |
| Auth | Bcrypt passwords, hashed sessions, account lockout, rate limit | `internal/auth/` |
| Authorization | Per-handler `CapabilityRequired` + capability guard wrapping `CoreAPI` | `internal/auth/`, `internal/coreapi/capability.go` |
| CSRF | JSON-only mutations on `/admin/api/*` | `internal/auth/csrf.go` |
| XSS | bluemonday UGC sanitization at render | `internal/sanitize/richtext.go` |
| Secrets at rest | AES-256-GCM envelope on flagged settings, theme git tokens | `internal/secrets/` |
| Plugin trust | Signed binaries, gRPC handshake validates signature | `pkg/plugin/`, `internal/cms/plugin_manager.go` |
| Outbound HTTP | Scheme allowlist, internal-host blocklist, redirects bounded | `internal/coreapi/impl_http.go` |
| Theme git | HTTPS-only, scheme allowlist, encrypted tokens, HMAC-validated webhook | `internal/cms/theme_*.go` |
| MCP | Bearer token (hashed), per-token rate limit, scope×class ACL, audit log | `internal/mcp/` |
| Logs | Structured slog with request-id correlation, no secret leakage | `internal/logging/` |

---

## 2. Production Boot Gates

`config.Load` validates the environment before the server starts. In `APP_ENV=production`, the boot fails fast on:

| Problem | Error | Remediation |
|---|---|---|
| Empty `SESSION_SECRET` | `SESSION_SECRET unset in production` | Set a random 32+ byte hex string |
| Empty `SQUILLA_SECRET_KEY` | `SQUILLA_SECRET_KEY unset; secret-bearing settings cannot be encrypted` | Generate via `openssl rand -base64 32` |
| Wrong `SQUILLA_SECRET_KEY` length | `secrets init failed: SQUILLA_SECRET_KEY must be 32 raw bytes (base64-encoded): got N bytes, want 32` | Value must base64-decode to **exactly 32 bytes**. Coolify's `SERVICE_BASE64_<NAME>` produces only 24 bytes — override manually with `openssl rand -base64 32`. |
| Default DB password | `DB_PASSWORD is the project default; refusing to start` | Use Coolify magic vars or a real secret |
| `DB_SSLMODE=disable` on a non-internal hostname | `DB_SSLMODE=disable on a public host` | `require` or `verify-full` |
| Empty `CORS_ORIGINS` | `CORS_ORIGINS unset; admin would be open to any origin` | Set the public admin URL list |
| Empty `MONITOR_BEARER_TOKEN` | `MONITOR_BEARER_TOKEN unset; /api/v1/stats unprotected` | Generate via `openssl rand -hex 32` |

Coolify's `coolify-compose.yml` populates all of these via `SERVICE_*` magic variables on first deploy.

---

## 3. Authentication

### 3.1 Sessions
- 32-byte cryptographic random token, hex-encoded; only the SHA-256 hash stored (`token_hash` column).
- Cookie `squilla_session`: `HttpOnly=true`, `SameSite=Lax`, `Secure` when behind TLS (honors `X-Forwarded-Proto` from the trusted proxy).
- Stored fields: `user_id`, `token_hash`, `ip_address`, `user_agent`, `expires_at`.
- Hourly cleanup loop (`SessionService.CleanExpired`) removes expired rows.
- File: `internal/auth/session_svc.go`.

### 3.2 Login
- Bcrypt verification (`bcrypt.DefaultCost=10`; configurable via `BCRYPT_COST` env).
- Constant-time password compare (bcrypt is constant-time by construction).
- Account lockout after N failed attempts (default 5, exponential backoff): `internal/auth/lockout.go`.
- Per-IP rate limiter (default 5 attempts / 15 min): `internal/auth/rate_limit.go`.
- File: `internal/auth/auth_handler.go`, `page_handler.go`.

### 3.3 Registration
- Public registration creates `member`-role users (not `editor`) since commit `76f6124`.
- Self-registration is gated by `setting.allow_registration=true` (default false in production seeds).

### 3.4 Password Reset
- Real flow (no longer a stub).
- Raw token generated, sent via email; only SHA-256 hash stored in `password_reset_tokens`.
- Tokens single-use (`used_at` set on consumption to detect replay).
- Hourly cleanup of expired/used tokens.
- File: `internal/auth/password_reset.go`.
- **Graceful degradation:** when no `email.provider` plugin is active, the email-based reset flow is unavailable. The handler short-circuits with a clear error rather than panicking. For recovery, operators can use the `squilla reset-password <email> <new-password>` CLI subcommand which writes directly to the DB without going through the dispatcher (commit `c723983`).

### 3.5 Self-Promotion Block
- `UserHandler.Update` strips `role_id` from payloads when `target_user_id == current_user.id` (commit `76f6124`).
- `RoleHandler.Update` requires `manage_roles` capability and refuses to mutate `is_system=true` rows.

### 3.6 Bearer Token (`/api/v1/stats`)
- Constant-time compare via `crypto/subtle.ConstantTimeCompare`.
- Token from `MONITOR_BEARER_TOKEN` env.

---

## 4. Authorization

### 4.1 Two Layers
1. **Per-handler middleware** at the HTTP edge: `auth.CapabilityRequired("manage_users")`, `auth.AdminRequired()`, `auth.JSONOnlyMutations()`.
2. **CoreAPI capability guard** at the API surface: every method wrapped in `capabilityGuard.<Method>` checks `caller.Capabilities[cap]`. See `internal/coreapi/capability.go`.

The guard is wired in `cmd/squilla/main.go:252`:
```go
guardedAPI := coreapi.NewCapabilityGuard(coreAPI)
```
The unguarded `coreAPI` is used **only** by core kernel code (which sets `caller.Type = "internal"` for fail-open). Plugin and Tengo callers always go through the guard.

### 4.2 Per-Table ACL on Extension Data
- `data:read`, `data:write`, `data:delete`, `data:exec` capabilities are checked against the manifest's `data_owned_tables` array (commit `654dae5`).
- An extension declaring ownership over `forms`, `form_submissions` cannot read or write any other extension's tables.

### 4.3 Per-Node-Type Access
- `roles.capabilities` JSONB stores `nodes.<type>.access` and `nodes.<type>.scope` (`read`/`write`/`delete` × `all`/`own`).
- `default_node_access` covers types not explicitly listed.
- `auth.GetNodeAccess(user, nodeType)` → `NodeAccess.CanRead(node)`, `CanWrite(node)`.
- Node `Search` honors the access filter (commit `9f9239c`).

---

## 5. CSRF Protection

`auth.JSONOnlyMutations()` rejects POST/PUT/PATCH/DELETE on `/admin/api/*` unless `Content-Type` is `application/json`. This is sufficient because:

- Browsers cannot send a cross-origin `application/json` request without an explicit CORS preflight.
- `CORS_ORIGINS` is a strict allowlist (admin endpoints are credentialed).
- The session cookie is `SameSite=Lax`, so navigation-initiated requests carry the cookie but a forged form POST is blocked at the body-parser by content-type.

Token-based CSRF middleware is **not** currently implemented. The threat model deems content-type + same-site + strict CORS sufficient for the JSON admin API.

Public extension routes (`extension.json` `public_routes`) are not behind `JSONOnlyMutations` — extensions handling forms must include their own CAPTCHA and/or honeypot defenses (the `forms` extension includes both by default).

---

## 6. XSS Defense

`internal/sanitize/richtext.go` runs bluemonday's UGC policy on richtext fields **at render time**, with these tweaks:

- Strips: `<iframe>`, `<form>`, `<input>`, `<style>`.
- Allows: `<a>` with `rel`/`target`, `class` on all elements, `<img>` with `loading`/`decoding`.

Render-time (not write-time) sanitization means policy can tighten without rewriting stored data.

The Go `html/template` engine auto-escapes by default; `safeHTML` and `safeURL` template helpers explicitly opt out for cases where the kernel knows the value is safe (rendered block output, asset URLs from internal lookups). Treat any new use of `safeHTML`/`safeURL` as requiring a security review.

---

## 7. Secrets at Rest

`internal/secrets/secrets.go` provides AES-256-GCM envelope encryption.

- Master key: `SQUILLA_SECRET_KEY` env — a base64 string that decodes to **exactly 32 raw bytes**. Generate with `openssl rand -base64 32` (produces a 44-character string). Hex-encoded keys, 24-byte Coolify magic vars, and arbitrary-length passphrases are all rejected; the secrets service requires exactly 32 bytes after `base64.StdEncoding.DecodeString`.
- Envelope format: `enc:v1:<base64(nonce || ciphertext || tag)>`.
- Fresh random 12-byte nonce per call.
- Encrypted columns:
  - `site_settings.value` for keys matching the secret heuristic.
  - `themes.git_token`.
  - Reserved for future: `extension settings` flagged as secret in their manifest.

### Secret Heuristic
A `site_settings.key` is treated as secret-bearing if (case-insensitive) it contains any of:

```
_password   _key   _token   _apikey   _api_key   _credentials   _secret
```

Reads via `GET /admin/api/settings` redact secret keys (commit `54f573a`) — they return `"<redacted>"` regardless of stored value, unless the caller has explicit elevated capability.

Dev mode (no `SQUILLA_SECRET_KEY`) passes plaintext through and logs a warning; production refuses to boot.

> **Coolify deployment note:** Coolify's `SERVICE_BASE64_<NAME>` magic variable generates a 32-character base64 string, which decodes to only 24 raw bytes — not enough for an AES-256 key. The `coolify-compose.yml` uses `SERVICE_BASE64_SECRETKEY` as the fallback for `SQUILLA_SECRET_KEY`, but the auto-generated value will be rejected on boot. Always override `SQUILLA_SECRET_KEY` manually in the Coolify Environment Variables tab with the output of `openssl rand -base64 32`. See README §Deploy on Coolify for the full procedure.

---

## 8. Plugin Trust

- HashiCorp `go-plugin` v2 protocol with magic cookie `SQUILLA_PLUGIN=squilla`.
- gRPC-only (no NetRPC).
- **Binaries are signed** (commit `654dae5`). The handshake validates the embedded signature against the kernel's public key before allowing the plugin to register.
- Plugin processes are crash-isolated: a panic inside a plugin never kills the kernel.
- Plugin shutdown has a 30-second timeout (`pluginManager.StopAll()` runs `app.ShutdownWithTimeout(30*time.Second)`).

Each plugin receives a per-instance `SquillaHost` gRPC server backed by the **guarded** `CoreAPI`. The plugin's `CallerInfo` is constructed from its manifest's declared capabilities and `data_owned_tables`.

---

## 9. Outbound HTTP (`http.Fetch`)

`CoreAPI.Fetch` is the only path through which plugin/script code can make outbound HTTP requests. Hardening (commit `2344aa1`):

- **Scheme allowlist**: `http`, `https`. Rejects `file://`, `gopher://`, `dict://`, etc.
- **Internal-host blocklist**: rejects `localhost`, `127.0.0.0/8`, `169.254.0.0/16` (link-local, AWS metadata), `10.0.0.0/8` and `192.168.0.0/16` (RFC1918), `::1`, fc00::/7.
- **Redirect bound**: max 5 hops; each hop re-validated.
- **Body cap**: 10 MB default, configurable per call.
- **Timeout**: 30 s default, configurable per call.

Override the blocklist by setting `SQUILLA_ALLOW_PRIVATE_HTTP=true` (development only).

---

## 10. Theme Git Install

Hardening (commit `f4ac40f`):

- **HTTPS-only**: `git_url` must start with `https://`. SSH and `file://` are rejected.
- **Scheme allowlist**: enforced at clone and pull.
- **Encrypted tokens**: `themes.git_token` stored via the secrets envelope.
- **Token never in argv**: tokens injected via `git -c http.extraheader=Authorization:Bearer ...`, not in the URL (no `ps aux` leakage).
- **Hostile-config defense**: post-clone, `.git/config` is reset to a minimal known-good template before any further git operations run.

### Webhook (`POST /api/v1/theme-deploy`)

- HMAC-validated: GitHub `X-Hub-Signature-256`, GitLab `X-Gitlab-Token`. Constant-time compare.
- The legacy `?secret=` query param fallback was removed.
- Rate-limited per-IP.
- Idempotent: duplicate deliveries with the same commit SHA are no-ops.

---

## 11. CORS

Two policies are mounted in parallel (commit `ace0066`):

| Path | Policy |
|---|---|
| `/mcp` | Permissive: `Access-Control-Allow-Origin: *`, no credentials. Bearer-token auth means cookies are irrelevant. |
| Everything else (admin + public) | Strict: origins must match `CORS_ORIGINS` env (comma-separated), `AllowCredentials=true`, methods/headers allowlisted. |

`SERVICE_URL_APP` from Coolify is normalized to a list at startup (commits `279a2be`, `e84ffff`) — bare hostnames don't crash startup.

---

## 12. MCP Server (`/mcp`)

### Tokens
- Format: `vcms_<32 hex bytes>`.
- Stored as SHA-256 hash; `token_prefix` (first 8 chars) kept for log identification.
- Created via `POST /admin/api/mcp/tokens`; raw value returned **once** in the response.
- Per-token rate limiter: 60 req/10 s default, in-memory (process-local). Backed by `golang.org/x/time/rate`.

### Scope × Class ACL
| Scope | Allowed classes |
|---|---|
| `read` | read |
| `content` | read, content |
| `full` | read, content, full |

`tools_data.go` `core.data.exec` (raw SQL) requires both `full` scope and `SQUILLA_MCP_ALLOW_RAW_SQL=true` env.

### Audit Log
Every tool call writes `(token_id, tool, args_hash, status, error_code, duration_ms)` to `mcp_audit_log`. Raw args are not stored (only SHA-256 hash) so PII does not leak. Daily retention sweep keeps the table bounded.

### Presigned Upload Tokens (`/api/uploads/<token>`)

For binaries above ~5 MB, MCP tools use a three-step presigned-upload flow
(`core.<kind>.upload_init` → `PUT /api/uploads/<token>` →
`core.<kind>.upload_finalize`). The PUT route is mounted as a **raw
`http.Handler` outside Fiber middleware** — there is no session, no bearer
header, and no capability check at the HTTP edge. The token in the URL is
the entire access-control story.

| Property | Value |
|---|---|
| Token entropy | 64 hex chars (32 random bytes via `crypto/rand`) |
| TTL | ~15 minutes (`expires_at`) |
| Single-use | Token row goes from `pending` → `uploaded` → `finalized` atomically |
| Bound to user | `pending_uploads.user_id` is set at `_init`; `_finalize` runs in the issuing user's auth context, so capability checks at finalize time still apply |
| Bound to kind | `media`, `theme`, or `extension` only — finalize for the wrong kind is rejected |
| Size cap | Per-kind, env-tunable (`SQUILLA_MEDIA_MAX_MB`, `SQUILLA_THEME_MAX_MB`, `SQUILLA_EXTENSION_MAX_MB`) |
| Cleanup | Background ticker deletes expired rows + temp files every 5 minutes |
| Streamed | Body written straight to `data/pending/<token>.bin` while computing SHA-256 — no full-buffer-in-memory |
| Error codes | 404 unknown, 410 expired, 409 already-uploaded / in-progress, 413 too large |

**Why it's safe without Fiber auth:** the threat is unauthenticated parties
discovering a valid token. With 256 bits of entropy and a 15-minute window
the search space is far beyond brute-force reach. A successful upload still
needs the issuing user's auth context to call `_finalize`; the bytes on
disk without finalize are inert (the cleanup ticker reaps them).

Tables: `pending_uploads` (migration `0042_pending_uploads.sql`). Code:
`internal/uploads/`.

### Schema-driven settings & Capability propagation through providers

Two architectural changes from commit `7e49268` worth flagging here:

- **Settings registry** (`internal/settings/`): every kernel-owned setting
  is declared in `builtin.go` with an explicit `Translatable` flag and a
  group (general / SEO / advanced / languages / security). The registry
  drives both the admin UI and the per-language storage rules in `store.go`
  — non-translatable fields read/write the default-language row directly,
  translatable fields use the per-locale composite-PK lookup with
  default-language fallback. Extensions register their own groups via
  `Registry.RegisterGroup`, and the same secret-key heuristic from
  `internal/secrets/` applies to extension-owned keys without extra wiring.

- **Provider tags through the plugin manager**: kernel `core.media.*` and
  `SendEmail` no longer call concrete in-kernel services. Plugins declare a
  `provides` array in `extension.json` (`media-provider`, `email.provider`,
  …); the plugin manager indexes them and routes the call to the highest-
  priority active plugin. Capability checks still happen at the gRPC edge
  inside the plugin (the host calls `coreAPI.UploadMedia` with the
  caller's CallerInfo, the guard in `internal/coreapi/capability.go`
  evaluates against the plugin's manifest capabilities, then the call
  proceeds). Hot-swapping providers works because the routing is dynamic
  and the call surface (`MediaProvider` / `EmailProvider` interfaces in
  `internal/coreapi/`) is fixed.

---

## 13. Logging

Structured slog with request-id correlation (commit `dcde556`):

- Development: human-readable text format.
- Production: JSON to stdout for collector ingestion.
- Every request gets an `X-Request-Id` (generated if absent) propagated through the request context.
- `coreapi.Log` (callable by extensions) prefixes with `[ext:<slug>]` and writes through the same slog path.

### Don't Log
- Plaintext passwords (use bcrypt failures with masked email).
- Session cookie values (only token prefix).
- MCP raw token (only `token_prefix`).
- Secret site setting values.
- Plugin response bodies (commit `eb0c1eb` removed the leaky preview log).

### Required Fields on Errors
- Request-id (auto-injected).
- Error category (`auth`, `database`, `external`, `validation`).
- Caller info (when known).

---

## 14. Threat Model Quick Reference

| Threat | Status |
|---|---|
| **Tengo sandbox escape** | Restricted standard library (`os`, `io`, network modules removed); 50k allocation cap; 10 s timeout; per-execution fresh VM. |
| **Credential stuffing** | Bcrypt + per-IP rate limit + account lockout. |
| **JSONB injection** | GORM parameterized queries; field schemas validated against block-type definition before save. |
| **Stored XSS in blocks** | bluemonday at render-time. |
| **CSRF** | JSON-only mutations + strict CORS + SameSite cookie. |
| **MIME header injection** | Subjects rendered with `text/template` then validated for CR/LF (commit `eb0c1eb`). |
| **MITM STARTTLS downgrade** | `email_smtp_require_tls=true` setting (commit `eb0c1eb`); refuses plaintext if STARTTLS unavailable. |
| **Plugin tampering** | Signed binaries + handshake validation. |
| **SSRF via http.Fetch** | Scheme allowlist + internal-host blocklist. |
| **Theme webhook replay** | HMAC validation + rate limit. |
| **Mass-assignment via JSON parse** | Update handlers strip protected fields (`id`, `created_at`, `is_system`, `role_id` for non-self) before `db.Updates`. |
| **Privilege escalation via PATCH /users/me** | Self-edit cannot change `role_id`. |
| **Stored token replay (password reset)** | Single-use; `used_at` set on consumption. |
| **Search bypassing access filter** | Search applies `NodeAccess` filter (commit `9f9239c`). |
| **Filter handler leak** | `RegisterFilter` returns `UnsubscribeFunc`; opaque ID-based unregister (commit `9f9239c`). |
| **Subscribe handler leak** | `Subscribe` returns `UnsubscribeFunc`; bus supports proper unregister. |
| **SSE blocking on slow client** | Bounded buffer (cap 32) with drop-on-full. |

---

## 15. PR-Time Security Checklist

Before merging any change touching kernel code:

- [ ] Capability gate on every admin endpoint that mutates state.
- [ ] DTO for body parsing — no `c.BodyParser(&model)` direct.
- [ ] Mass-assignment safe: protected fields explicitly stripped.
- [ ] Validation: enum fields whitelisted, required fields non-empty, lengths bounded.
- [ ] Constant-time compare for any secret check.
- [ ] No URL injection: scheme allowlist, no leading-wildcard ILIKE on indexed columns.
- [ ] No CRLF in headers: strip `\r\n` from any user-supplied value before writing to a header.
- [ ] Path-traversal defense on FS reads: `filepath.Clean` + prefix check against absolute base.
- [ ] Context propagation: pass `c.UserContext()` through DB/HTTP/script calls.
- [ ] Error wrapping: `fmt.Errorf("...: %w", err)`.
- [ ] No silent `json.Marshal`: handle the error.
- [ ] No new file > 500 LOC.
- [ ] Tests for the new code path (capability denied, invalid input, success).

---

## 16. Reporting Vulnerabilities

Email `security@squilla.local` (placeholder — set in your fork). Disclose privately first; we aim for 7-day acknowledgement and 30-day fix turnaround for critical issues.
