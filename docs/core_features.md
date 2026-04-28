# VibeCMS Core — Complete Feature List

This document is an exhaustive catalog of what the VibeCMS kernel does today, originally derived from a line-by-line review of `internal/`. Each feature is labeled with its current status:

- ✅ **Working** — implemented and exercised on the public/admin surface.
- 🟡 **Partial** — works but has known gaps or incomplete edge cases.
- 🪦 **Dead code** — implemented but never wired into the running app.
- 📦 **Provided by extension** — kernel offers infrastructure; the actual feature ships in `extensions/*`.

> **Status note (2026-04-28):** Many of the 🔴 findings from the original review have since been resolved. The fixed-as-of commits are tagged inline in the relevant subsections, and §21 has been updated with the current critical-issue list. Companion docs: [`core_dev_guide.md`](./core_dev_guide.md) for how to extend or modify the kernel; [`security.md`](./security.md) for the active security posture.

---

## 1. Architectural foundations

VibeCMS is a **kernel + extensions** CMS. The kernel ships infrastructure only; every user-visible feature is delivered either by a built-in extension (in `extensions/`) or by the active theme.

| Boundary | What lives there |
|---|---|
| **Kernel** (`internal/`, `cmd/vibecms/`, `pkg/`) | Content nodes, auth, sessions, RBAC, CoreAPI, event bus, scripting (Tengo), template renderer, theme loader, extension loader/proxy, MCP server, SDUI engine, well-known registry, plugin contract |
| **Extensions** (`extensions/*/`) | Media uploads/optimization, email providers (SMTP/Resend), SEO/sitemap, forms, anything domain-specific |
| **Themes** (`themes/*/`) | Layouts, partials (layout blocks), templates, assets, optional Tengo scripts |
| **Admin SPA shell** (`admin-ui/`) | Auth UI, dashboard, sidebar; every feature page is an extension micro-frontend loaded via SDUI manifest |

The hard rule (per `CLAUDE.md`): *"If disabling/removing an extension would leave dead code in core, that code belongs in the extension."*

Two current violations of this rule are documented in §6.4 (in-core SMTP/Resend) and §10 (in-core media handler).

---

## 2. Content management

### 2.1 Content nodes (`content_nodes` table)
**Status:** ✅ working

The atomic unit of CMS content. Every page, post, product, etc. is a `ContentNode`. All node fields:
- Identity: `id`, `uuid` (Postgres `gen_random_uuid()`), `slug`, `full_url`
- Hierarchy: `parent_id` (self-FK, recursive)
- Type: `node_type` (FK by slug to `node_types.slug`)
- Status: `draft` / `published` / `archived` (string column, *no DB CHECK*)
- Localization: `language_code`, `language_id`, `translation_group_id` (UUID grouping translation siblings)
- Versioning: `version` (incremented on update; not used for optimistic lock yet)
- Authorship: `author_id` (FK to `users.id`) — **currently never populated on create** (see §3.4 "scope='own' broken")
- Layout binding: `layout_id` (FK), `layout_slug` (resilient text reference auto-populated by `BeforeSave` hook)
- Standard fields: `title`, `slug`, `excerpt`, `featured_image` (JSONB), `taxonomies` (JSONB), `seo_settings` (JSONB)
- Editable data: `blocks_data` (JSONB array), `fields_data` (JSONB), `layout_data` (JSONB)
- Timestamps: `created_at`, `updated_at`, `published_at`, `deleted_at` (soft delete via GORM)

Indexes: GIN on `blocks_data`, partial unique on `full_url WHERE deleted_at IS NULL`, B-tree on `(status, language_code)`.

### 2.2 Node CRUD via admin API
**Status:** 🟡 partial

`/admin/api/nodes` — list, get, create, update, delete + per-node-type access checks via `auth.GetNodeAccess`.

Files: `internal/cms/node_handler.go`, `content_svc.go`.

Known gaps:
- `AuthorID` never set on Create → `scope='own'` access mode is unusable.
- `Status` and `node_type` are not validated in Update → users with write on `page` only can switch a node to `post` and circumvent ACL.
- `Search` endpoint bypasses access filter (List filters; Search does not).
- Update is not wrapped in a transaction — revision creation + main update + URL rebuild can leave partial state on failure.

### 2.3 Public API for nodes
**Status:** ✅ working

`GET /api/v1/nodes?node_type=...` — paginated, only `status='published'`, returns ID + title + slug + full_url + excerpt + fields_data.

Caveat: `fields_data` is exposed wholesale — extensions storing private metadata in fields will leak it.

### 2.4 Node revisions (`content_node_revisions`)
**Status:** ✅ working

Every UpdateNode creates a snapshot of `blocks_data` + `seo_settings` keyed by `created_by` (nullable when MCP/extension/system caller).

Known gap: no retention or pruning — revisions grow unbounded.

### 2.5 Translations
**Status:** ✅ working

Sibling nodes share `translation_group_id` (UUID). Endpoint `POST /admin/api/nodes/:id/translations` creates a translation in a target language, copying blocks + fields, slug-conflict-suffixing if needed (`-2`, `-3`).

### 2.6 Node types (`node_types` table)
**Status:** ✅ working

Custom content types beyond page/post. Fields: `slug`, `label`, `label_plural`, `icon`, `description`, `taxonomies` (JSONB), `field_schema` (JSONB), `url_prefixes` (per-language JSONB), `supports_blocks`.

CRUD via `/admin/api/node-types`.

Used by: public URL routing (custom prefix per language), admin SPA layouts, content render context.

Known gap: `RegisterNodeType` (CoreAPI) normalizes field schema; `UpdateNodeType` does not. Updates can produce inconsistent name/key field mirrors.

### 2.7 Slug auto-generation
**Status:** 🟡 partial

`internal/cms/slug.go::Slugify` is the canonical slugifier. `node_handler.go` has its own ASCII-only `slugify` regex that strips Unicode (e.g. `"Crème Brûlée"` → `"cr-me-br-l-e"`).

Recommended: standardize on `cms/slug.go` and add Unicode transliteration.

### 2.8 Full-URL resolution
**Status:** ✅ working

`buildFullURL(node, db)`:
1. Special case: `slug=="index"` or homepage / homepage-translation → `/<lang>` (or `/`).
2. Otherwise: `/<lang>/<node-type-prefix>/<parent-chain>/<slug>`.
3. Honors `language.hide_prefix` (omits language slug).
4. Honors `node_type.url_prefixes[language_code]` for translated type prefixes.

Known gap: `collectParentSlugs` recursion has no cycle/depth limit — admin error setting `parent_id` to ancestor stack-overflows.

---

## 3. Authentication & authorization

### 3.1 Sessions (`sessions` table)
**Status:** ✅ working

- 32-byte cryptographic random token, hex-encoded, only SHA-256 hash stored.
- UUID primary key.
- Stored: `user_id`, `token_hash`, `ip_address`, `user_agent`, `expires_at`.
- Cookie: `vibecms_session`, `HttpOnly=true`, `SameSite=Lax`, `Secure` when `c.Protocol()=="https"`.

Files: `internal/auth/session_svc.go`.

Known gaps:
- `CleanExpired()` defined but never invoked → expired sessions accumulate forever.
- IP/UA captured but not bound — stolen cookies work from anywhere.
- `Secure` flag ignores `X-Forwarded-Proto` → behind TLS-terminating proxies, cookies sent plaintext.

### 3.2 Login / logout / register / forgot password / reset password
**Status:** mixed (see below)

| Endpoint | Status | Notes |
|---|---|---|
| `POST /auth/login` (JSON API) | ✅ | Verifies bcrypt, creates session, sets cookie. |
| `POST /auth/logout` (JSON API, auth-required) | ✅ | Deletes session row, clears cookie. |
| `GET /me` (JSON API, auth-required) | ✅ | Returns user + capabilities. |
| `POST /auth/login-page` (form) | ✅ | Same as JSON login but redirects with flash cookies. |
| `POST /auth/register` (form) | 🟡 | Creates `member`-role users (commit `76f6124`); gated by `setting.allow_registration` (default false in production seeds). |
| `POST /auth/forgot-password` (form) | ✅ | Real flow. SHA-256 hashed token in `password_reset_tokens`; emailed via configured rule (`76f6124`). |
| `POST /auth/reset-password` (form) | ✅ | Single-use token consumption; replays detected via `used_at` (`76f6124`). |
| `GET /logout` (form) | ✅ | Now POST-only since `76f6124` (CSRF safe). |

Files: `internal/auth/auth_handler.go` (JSON), `page_handler.go` (form-based).

### 3.3 User management
**Status:** ✅ working

`/admin/api/users` (CRUD). Capability `manage_users` required for create/list/delete; users can update their own profile.

Known gap: a user with `manage_users` can promote themselves to admin via `PATCH /users/<their_id> {"role_id": <admin_id>}` — no privilege-level guard.

### 3.4 RBAC: roles, capabilities, per-node-type access
**Status:** ✅ working (with caveats)

Roles (table `roles`):
- `slug`, `name`, `description`, `is_system`, `capabilities` (JSONB).

Built-in roles (seed.go): `admin`, `editor`, `author`, `member`.

`Capabilities` JSONB shape:
```json
{
  "admin_access": true,
  "manage_users": true,
  "manage_roles": true,
  "manage_settings": true,
  "manage_menus": true,
  "manage_layouts": true,
  "manage_email": true,
  "default_node_access": {"access": "write", "scope": "all"},
  "nodes": {
    "post": {"access": "read", "scope": "own"}
  },
  "email_subscriptions": ["user.registered", "node.published"]
}
```

Helpers: `auth.HasCapability`, `auth.RoleRequired`, `auth.CapabilityRequired`, `auth.GetNodeAccess`, `NodeAccess.CanRead/CanWrite/CanAccessNode`.

Known broken pieces:
- `scope='own'` is **unusable** because `AuthorID` is never set on Create.
- Many admin endpoints (settings, taxonomies, terms, menus, languages, layouts, block types, templates, cache) have no `CapabilityRequired` middleware — only `AuthRequired`.
- `IsAdmin` hardcodes slug `"admin"` (`auth/rbac_middleware.go`); renaming the admin role breaks it.

### 3.5 Password hashing
**Status:** 🟡 partial

`bcrypt.DefaultCost=10` everywhere (`page_handler.go:148`, `user_handler.go:154,234`, `seed.go:275`). OWASP recommends 12+ on modern hardware. Should be configurable via `BCRYPT_COST` env.

### 3.6 Bearer-token auth for monitor endpoint
**Status:** 🟡 partial

`api.BearerTokenRequired(token)` middleware — used only by `GET /api/v1/stats`.

Known gap: comparison via plain `!=` (timing oracle). Fix: `crypto/subtle.ConstantTimeCompare`.

---

## 4. Multi-language support

### 4.1 Languages (`languages` table)
**Status:** ✅ working

Fields: `code`, `slug`, `name`, `native_name`, `flag`, `is_default`, `is_active`, `hide_prefix`, `sort_order`.

CRUD via `/admin/api/languages`. Auto-unsets other defaults when a new default is set (race-prone — see #12 review).

### 4.2 Translation chains
**Status:** ✅ working

Nodes share `translation_group_id`. The render context loads all siblings of a node, used by language-switcher widgets. URL fallback chain (per `language_svc.go::Update`): when `slug` or `hide_prefix` change, **all** content nodes for that language are walked and re-saved with new `full_url`.

Performance caveat: walking N nodes one by one is O(N) writes; not transactional.

### 4.3 URL prefix rules
**Status:** ✅ working

- Default language with `hide_prefix=true` → `/about` (no prefix).
- Otherwise → `/en/about`, `/fr/a-propos`.
- Custom node types add a translated prefix → `/en/blog/my-post`.

---

## 5. Content blocks, layouts, templates

VibeCMS uses a Tailwind/Alpine-based composable block model. Four entity types collaborate to render a page.

### 5.1 Block types (`block_types` table)
**Status:** ✅ working

A reusable component (e.g. "hero", "feature-grid", "login-form"). Fields: `slug`, `label`, `icon`, `description`, `field_schema` (JSONB), `html_template` (Go html/template), `test_data`, `source` (custom/seed/theme/system), `theme_name`, `view_file`, `block_css`, `block_js`, `content_hash`, `cache_output`.

Editor preview endpoint: `POST /admin/api/block-types/preview` renders sample HTML in active theme chrome.

CRUD: `/admin/api/block-types`.

### 5.2 Layout blocks ("partials") (`layout_blocks` table)
**Status:** ✅ working

Reusable HTML chunks composable into layouts via `{{renderLayoutBlock "<slug>"}}`. Fields: `slug`, `name`, `description`, `language_id`, `template_code`, `source`, `theme_name`, `field_schema`, `content_hash`.

Built-in seeded blocks: `primary-nav`, `user-menu`, `site-header`, `footer-nav`, `site-footer`.

### 5.3 Layouts (`layouts` table)
**Status:** ✅ working

Page-level templates (full HTML document). Fields: `slug`, `name`, `description`, `language_id`, `template_code`, `source`, `theme_name`, `is_default`, `supports_blocks`, `content_hash`.

Bound to nodes via `content_nodes.layout_slug` (slug reference survives theme cycles).

Recursion limit: `RenderLayout` enforces max 5 levels of nested `renderLayoutBlock` (`rendering/template_renderer.go:331`).

### 5.4 Templates (`templates` table)
**Status:** ✅ working

Block layout templates — describe which block types are placed in which slots. Fields: `slug`, `label`, `description`, `block_config` (JSONB), `source`, `theme_name`, `content_hash`.

### 5.5 Field types
**Status:** ✅ working

20 built-in field types in `internal/cms/field_types/registry.go`:

| Group | Types |
|---|---|
| Basic | text, textarea, richtext, number, range, email, url, date, color |
| Choice | toggle, select, radio, checkbox |
| Media | image, gallery, file |
| Relational | link, node (selector), term (selector) |
| Layout | group, repeater |

Each has structured `HowTo` guidance for both human authors and AI tools (consumed by the MCP `core.field_types.list` tool and the `/admin/api/field-types` endpoint).

Extensions can contribute additional field types via their manifest's `admin_ui.field_types`.

### 5.6 Render pipeline
**Status:** ✅ working

For a public request:
1. `PublicHandler.PageByFullURL` looks up node by `full_url WHERE status='published' AND deleted_at IS NULL`.
2. Resolves layout via `layout_slug` (or default).
3. Loads blocks from `node.blocks_data`, validates against block-type schemas.
4. Renders each block via cached `RenderParsed` (key = `"block:" + slug + ":" + tmplContent` — full source as key, see §10 known gaps).
5. Resolves `theme-asset:<key>` and `extension-asset:<slug>:<key>` references in field data.
6. Wraps rendered blocks in the layout via `RenderLayout`, providing `renderLayoutBlock` template func.
7. Returns HTML with `Content-Type: text/html`.

Cache invalidation: `PublicHandler.SubscribeAll` clears caches on events prefixed `theme.`, `setting.`, `block_type.`, `language.`, or `layout`.

### 5.7 Block output cache
**Status:** 🟡 partial

Blocks marked `cache_output=true` (migration 0019) skip re-rendering if their `(block_slug, content_hash)` was already rendered.

Known gaps:
- Cache is unbounded — DoS via large templates / many distinct blocks.
- Cache key uses full template source — keys grow with template size.
- Wholesale invalidation on every theme/setting/block change.

### 5.8 Template renderer
**Status:** 🟡 partial

`internal/rendering/template_renderer.go` (382 LOC) — html/template wrapper with:
- Layout + page parsing with named-block composition.
- Three caches (full, layout, block) — only `cache` and `blockCache` actively used.
- Custom FuncMap: `safeHTML`, `safeURL`, `event`, `filter`, `dict`, `seq`, `mod/add/sub`, `image_url`, `image_srcset`, `lastWord`, `beforeLastWord`, `split`, `json`, `list`, `deref`.
- Dev mode bypasses cache.

Known gaps:
- `safeHTML`/`safeURL` are global escape-bypass primitives — should be renamed `unsafeHTML`/`unsafeURL`.
- Concurrent parse+execute race in cache-miss path.
- `image_url` hardcodes media-manager extension's `/media/cache/<size>/<path>` URL scheme — kernel knows about an extension's contract.
- No upper bound on cache size.

---

## 6. Email infrastructure

### 6.1 Email rules (`email_rules` table)
**Status:** ✅ working

A rule maps an event action (e.g. `user.registered`) to a template + recipient strategy.

Fields: `action`, `node_type` (nullable filter), `template_id` (FK), `recipient_type` (`actor`/`node_author`/`fixed`/`role`), `recipient_value`, `enabled`.

CRUD via the email-extension; the kernel's `RuleService` is consumed by the dispatcher.

### 6.2 Email templates (`email_templates` table)
**Status:** ✅ working

Fields: `slug`, `name`, `language_id`, `subject_template`, `body_template`, `test_data`.

Language fallback: language-specific → site default → universal NULL.

Rendered via `html/template` for both body **and** subject (see §6.5 known gap).

### 6.3 Email layouts (`email_layouts` table)
**Status:** ✅ working

A "wrapper" template applied to all emails. Body is injected as `.email_body` (marked `template.HTML`).

Language fallback: language-specific → universal NULL.

### 6.4 Email dispatcher & provider plugins
**Status:** ✅ working

Flow:
1. `eventBus.SubscribeAll(emailDispatcher.HandleEvent)` in `cmd/vibecms/main.go`.
2. On any event, `HandleEvent` finds matching rules (filtered by action + optional node_type).
3. For each rule, resolves recipients with their preferred language.
4. Renders subject + body, optionally wrapped in a layout.
5. Calls `sendFunc(SendRequest)` to dispatch synchronously.
6. `sendFunc` is wired to call the active provider plugin via gRPC `HandleEvent("email.send", payload)`.

The hard-rule violation (in-core SMTP/Resend providers) was resolved in commit `eb0c1eb` — `internal/email/smtp.go`, `resend.go`, `provider.go` were removed; both providers now ship only via `extensions/smtp-provider` and `extensions/resend-provider`. `LogService.Resend` re-routes through the same plugin path.

### 6.5 Email logs (`email_logs` table)
**Status:** ✅ working

Every send (success or failure) is logged with: `rule_id`, `template_slug`, `action`, `recipient_email`, `subject`, `rendered_body`, `status`, `error_message`, `provider`.

Hardening (commit `eb0c1eb`):
- Daily retention sweep prunes the table (configurable via `email_log_retention_days` setting).
- Password-reset tokens are stored in the dedicated `password_reset_tokens` table (single-use, hashed) — they never appear in `email_logs.rendered_body`.
- Subject is rendered with `text/template` (no HTML-escape) and validated for CR/LF before send to prevent MIME header injection.
- `LogService.Resend` re-renders against current rule + template, not the stored body — preventing replay of old tokens.

### 6.6 SMTP/Resend providers (in-extension)
**Status:** 📦 in extensions

Extensions: `extensions/smtp-provider`, `extensions/resend-provider`. Both implement `email.send` event handlers via gRPC plugin.

Settings stored in `site_settings` with `ext.<provider>.<key>` prefix.

### 6.7 STARTTLS handling
**Status:** ✅ working (commit `eb0c1eb`)

The SMTP send path (now in `extensions/smtp-provider`) honors `email_smtp_require_tls` (default `true`). When required, the provider refuses to send if STARTTLS is unavailable, blocking MITM downgrade attacks.

---

## 7. Theme system

### 7.1 Theme model (`themes` table)
**Status:** ✅ working

Fields: `slug`, `name`, `description`, `version`, `author`, `source` (`upload`/`git`/`scan`), `git_url`, `git_branch`, `git_token` (plaintext), `is_active`, `path`, `thumbnail`.

DB-level constraint: only one row may have `is_active=true` (migration 0022, partial unique index).

### 7.2 Theme installation
**Status:** 🟡 partial — security gaps

| Method | Endpoint | Status |
|---|---|---|
| Upload (zip) | `POST /admin/api/themes/upload` | ✅ Path-traversal defended at extract |
| Git clone | `POST /admin/api/themes/git` | ✅ Hardened in commit `f4ac40f` |
| Filesystem scan | Boot via `themeMgmtSvc.ScanAndRegister()` | ✅ |

Git path hardening (commit `f4ac40f`):
- HTTPS-only scheme allowlist; rejects `file://`, SSH, internal hosts.
- Token injected via `git -c http.extraheader=Authorization:Bearer ...` rather than the URL — no `ps aux` leakage.
- `.git/config` is reset to a minimal known-good template post-clone, before any further git operations run.
- `themes.git_token` encrypted at rest via the `internal/secrets/` envelope.

### 7.3 Theme activation
**Status:** ✅ working

`POST /admin/api/themes/:id/activate`:
1. Sets `is_active=true` on this theme, `false` on all others.
2. Loads theme via `themeLoader.LoadTheme(path)` — reads `theme.json`, registers layouts/partials/templates/blocks into DB.
3. Loads optional `scripts/theme.tengo` via `scriptEngine.LoadThemeScripts`.
4. Publishes `theme.activated` event with `path` payload.
5. Asset registry refreshes via atomic.Pointer swap.

### 7.4 Theme git pull / hot reload
**Status:** 🟡 partial

`POST /admin/api/themes/:id/pull` runs `git pull` in the theme directory. Same hostile-config concerns as §7.2.

### 7.5 Theme deploy webhook
**Status:** 🟡 partial — auth gaps

`POST /api/v1/theme-deploy` (public, mounted in `main.go:335`) — for GitHub/GitLab post-receive hooks.

Auth: shared secret in `site_settings.theme_webhook_secret`, accepted via `X-Webhook-Secret` header **or** `?secret=<value>` query param.

Known gaps:
- Plain `!=` comparison (timing oracle).
- Query-param fallback leaks secret to access logs.
- No HMAC validation (GitHub provides `X-Hub-Signature-256`; GitLab provides `X-Gitlab-Token`).
- Plaintext secret at rest (no encryption).
- No rate limit.

### 7.6 Theme assets
**Status:** ✅ working (best-of-class)

`internal/cms/theme_assets.go`: registry indexed by `asset_key` (theme-owned) and `<slug>:<asset_key>` (extension-owned).

Public references:
- In templates: `theme-asset:<key>`, `extension-asset:<slug>:<key>`.
- Resolved to `{url, alt, width, height}` at render time via `resolveAssetRefsIn`.
- Both regexes strict: `^[a-z0-9_-]+$` (no path traversal).

Static path: `/theme/assets/*` served by `cmd/vibecms/theme_assets_resolver.go` — uses `atomic.Pointer[string]` for hot-swap on `theme.activated` events. (This is the kernel's only `atomic` usage.)

### 7.7 Theme browser & file editing
**Status:** ✅ working

`GET /admin/api/themes/:id/files?path=...` — recursive directory browse with content read for text files. 1MB max file size (`file_browser.go:12`). Hidden files skipped in directory listings (but readable if requested directly — gap).

### 7.8 Theme purge
**Status:** ✅ working

`themeLoader.PurgeInactiveThemes()` runs on every boot (`main.go:286`) — destructive, no dry-run. Clean-up of inactive themes' DB entries.

---

## 8. Extension system

### 8.1 Extension model (`extensions` table)
**Status:** ✅ working

Fields: `slug`, `name`, `version`, `description`, `author`, `path`, `is_active`, `priority`, `settings` (JSONB), `manifest` (JSONB), `installed_at`, `updated_at`.

### 8.2 Extension manifest (`extension.json`)
**Status:** ✅ working

Declares:
- Identity: `slug`, `name`, `version`, `description`, `author`, `priority`.
- **Capabilities**: array of strings (e.g. `["nodes:read", "data:write", "files:write", "email:send"]`).
- Plugins: list of binary paths + event subscriptions.
- Public routes: HTTP routes mounted at app root (no auth) and proxied to plugin.
- Admin UI: entry point + components + nav items + field-types contributed.
- Assets: theme-asset-style declarations.

⚠️ **Capabilities are currently decorative** (see §13 Critical findings). The capability guard is not wrapped around the CoreAPI given to plugins — extensions get full access regardless of declared capabilities.

### 8.3 Extension lifecycle
**Status:** ✅ working

| State | Triggered by |
|---|---|
| Installed | Filesystem scan or upload finds `extension.json` → row in `extensions` table. |
| Activated | `POST /admin/api/extensions/:slug/activate` → runs migrations, starts plugin processes, loads scripts, registers blocks/templates, publishes `extension.activated`. |
| Deactivated | `POST /admin/api/extensions/:slug/deactivate` → stops plugins, unloads scripts, publishes `extension.deactivated`. |
| Uninstalled | (not yet implemented as an admin endpoint; lifecycle hooks would handle DB cleanup) |

Hot reload supported (no kernel restart).

### 8.4 Extension migrations (`extension_migrations` table)
**Status:** ✅ working

Each extension can ship `migrations/*.sql` files. On activate, the loader runs unapplied migrations in order, tracked by `(extension_slug, filename)` unique key.

### 8.5 Plugin manager (gRPC plugins)
**Status:** ✅ working

Built on HashiCorp `go-plugin` v2 protocol. For each declared plugin in the manifest:
1. Verify binary exists at `<extPath>/<binary>`.
2. Spawn via `exec.Command(binaryPath)`.
3. Handshake: magic cookie `vibecms`.
4. Bidirectional gRPC: kernel registers `VibeCMSHost` service via `GRPCBroker`; plugin connects back to call CoreAPI methods.
5. Plugin implements: `GetSubscriptions`, `HandleEvent(action, payload bytes)`, `HandleHTTPRequest`, `Shutdown`, `Initialize`.

Files: `internal/cms/plugin_manager.go`, `pkg/plugin/plugin.go`, `proto/plugin/vibecms_plugin.proto`.

Hardening:
- ✅ Plugin binaries signed; gRPC handshake validates the signature against the kernel's public key (commit `654dae5`).
- ✅ Per-table ACL: data:* capability checks against manifest's `data_owned_tables` (commit `654dae5`).
- ✅ `app.ShutdownWithTimeout(30 * time.Second)` bounds plugin shutdown.
- 🟡 Context propagation through plugin gRPC calls is partial — some paths still use `context.Background()`. Tracked for follow-up.
- 🟡 No auto-restart on plugin crash. The kernel logs and continues; admins re-activate from the UI.

### 8.6 Extension HTTP proxy (admin)
**Status:** ✅ working

`/admin/api/ext/:slug/*` — `internal/cms/extension_proxy.go`.

Proxies the request to the plugin's `HandleHTTPRequest` RPC, passing:
- Method, path (relative), body, query params, path params.
- User ID (`uint64`).
- Headers — **with `Cookie` and `Authorization` stripped** ✅.
- Adds `X-User-Email`, `X-User-Name` headers for plugin context.

Note: logs response body preview at INFO level (`extension_proxy.go:108-112`) — leaks plugin response content to logs.

### 8.7 Public extension routes
**Status:** 🟡 partial

`internal/cms/public_proxy.go` — registers the routes declared in each active extension's manifest at the **root** of the app (no auth). Same header strip as admin proxy.

Known gap: extension can declare `public_routes: [{path: "/admin/login"}]` and intercept core paths. No namespace enforcement.

### 8.8 Built-in extensions
The kernel ships with several reference extensions in `extensions/` (activation gated by migrations 0018, 0033):
- `media-manager` — media uploads, image optimization, WebP, thumbnail cache, gallery.
- `email-manager` — email rule + log + template UI.
- `sitemap-generator` — Yoast-style sitemap.
- `smtp-provider` — SMTP delivery.
- `resend-provider` — Resend HTTP API delivery.
- `forms-ext` — form rendering and submission via `{{event "forms:render"}}`.
- `hello-extension` — demo / contract test.

---

## 9. Tengo scripting

### 9.1 Script engine
**Status:** ✅ working

`internal/scripting/engine.go` — embedded Tengo VM. Each script execution:
- `script.SetMaxAllocs(50000)` allocation cap.
- `RunContext(ctx, 10*time.Second)` execution timeout.
- `caller := CallerInfo{Slug, Type:"tengo", Capabilities}` — capabilities propagated for the (currently bypassed) guard.

### 9.2 Theme scripts (`themes/<slug>/scripts/theme.tengo`)
**Status:** ✅ working

Loaded after theme activation. Theme scripts have **`Capabilities: nil`** (relies on the guard for denial — currently broken because the guard isn't wrapped).

### 9.3 Extension scripts (`extensions/<slug>/scripts/extension.tengo`)
**Status:** ✅ working

Loaded on extension activation with the manifest-declared capabilities.

### 9.4 Tengo modules exposed
The Tengo adapter (`internal/coreapi/tengo_adapter.go`, 1572 LOC) registers 13 + Tengo-stdlib modules:
- `core/nodes`, `core/menus`, `core/routes`, `core/filters`, `core/http`, `core/log`, `core/nodetypes`, `core/taxonomies`, `core/helpers`, `core/events`, `core/settings`, `core/wellknown`, `core/assets`, `core/routing` (only when render context present).
- Tengo stdlib: passes through whatever is in `stdlib.BuiltinModules`.

Per-module surface mirrors the corresponding CoreAPI methods.

### 9.5 Script-defined HTTP routes
**Status:** ✅ working

Theme/extension scripts can call `routes.register(method, path, scriptPath)`. Routes mounted by `MountHTTPRoutes`:
- Paths containing `.` (e.g. `/sitemap.xml`) → top-level on `app`.
- Other paths → `/api/theme/<path>` group.

Per-request handler:
- Reads body (JSON-decoded if Content-Type allows).
- Reads query params, path params, headers, IP.
- Runs the handler script with `request` injected.
- Returns the script's `response` map as `{status, headers, body, html, text, content_type}`.

Known gaps:
- No path-namespace enforcement — themes can register `/admin/login` and shadow core auth.
- Scripts compile + read from disk on every request (no bytecode cache).

### 9.6 Script event/filter handlers
**Status:** ✅ working

Scripts call `events.on(action, scriptPath, priority)` and `filters.add(name, scriptPath, priority)` to register handlers.

Resolved in commit `9f9239c`: every event/filter registration now keeps the `UnsubscribeFunc` returned by the event bus / filter chain, and `UnloadThemeScripts` / `UnloadExtensionScripts` invoke them before re-registering. Handler dispatch no longer multiplies on reload.

### 9.7 Script-defined `.well-known` endpoints
**Status:** ✅ working

`wellknown.register(path, scriptPath)` → mounts under `/.well-known/<path>`. Used by built-in extensions for `security.txt`, `acme-challenge/*`, `webfinger`, etc.

---

## 10. Media & file storage

### 10.1 Media files (`media_files` table)
**Status:** ✅ schema fine, handlers broken

Fields: `id`, `filename` (UUID-based stored), `original_name`, `mime_type`, `size`, `path` (relative), `url` (full public URL), `width`, `height` (nullable), `alt`, `slug` (unique, backfilled by migration 0031).

### 10.2 In-core media handler
**Status:** 🪦 dead code (and broken)

`internal/cms/media_handler.go` is **never registered in `main.go`** (verified via grep). 195 LOC of unreachable code.

If it WERE registered, it has further bugs:
- Stores files at `storage/uploads/<base36-timestamp>.<ext>` with URL `/storage/uploads/<file>`.
- `/storage` is **not** a static handler in `main.go` — uploaded files would 404.
- No MIME validation, no extension allowlist, filename collision risk on rapid uploads.

### 10.3 Media via extension
**Status:** 📦 in extension

Real media uploads happen through the `media-manager` extension's HTTP endpoints + image optimization pipeline. The kernel's `MediaService` is constructed but **passed as `nil` to `NewCoreImpl`** (`main.go:164`) — so `CoreAPI.UploadMedia` returns "media service not configured" if any extension calls it.

### 10.4 File storage for extensions
**Status:** ✅ working

CoreAPI methods `StoreFile(path, data) (publicURL string, err)` and `DeleteFile(path)`:
- Path-traversal defense via `filepath.Abs("storage")` + `filepath.Clean` + prefix check.
- Files written under `storage/<path>`, served via `/media/*` (mounted in `main.go:343`).
- File mode `0644`.

Known gaps:
- No size limit (gRPC `data []byte` parameter).
- No quota per extension.

### 10.5 File browser
**Status:** ✅ working

`BrowseFilesInDir(baseDir, requestedPath)` — recursive read with path-traversal defense. Used by theme + extension code editors.

Known gap: hidden files are excluded from listings but readable if requested by name.

---

## 11. Site configuration

### 11.1 Site settings (`site_settings` table)
**Status:** 🟡 partial

Key/value store with fields: `key` (PK), `value`, `is_encrypted`, `updated_at`.

CRUD via `/admin/api/settings`:
- `GET /admin/api/settings?prefix=...` — list all (or by prefix).
- `PUT /admin/api/settings` — bulk upsert from JSON body.

Known gaps:
- **No capability gate** beyond `AuthRequired`. Any authenticated user can change any setting.
- **Plaintext secrets**: `is_encrypted` flag exists but no encryption is implemented.
- **GET response includes secrets** verbatim — SMTP password, Resend API key, OAuth tokens. Should redact.
- No allowlist of writable keys — clients can write arbitrary keys.

### 11.2 Well-known endpoint registry
**Status:** ✅ working

`internal/cms/wellknown.go` — extensions can register handlers for `/.well-known/<path>`. Supports exact and prefix matching (with trailing `*`). Routed before public catch-all so unregistered well-known paths return 404 quickly.

### 11.3 Boot manifest endpoint
**Status:** ✅ working

`GET /admin/api/boot` (admin-required) — returns:
- User info + capabilities.
- Active extensions with entry points + components.
- Navigation tree (sidebar).
- Node types.

Built by `internal/sdui/engine.go::GenerateBootManifest`.

---

## 12. Menus

### 12.1 Menu model (`menus`, `menu_items` tables)
**Status:** ✅ working

Menus: `slug`, `name`, `language_id`, `version` (optimistic lock).

Menu items (tree via `parent_id`): `title`, `item_type` (`custom`/`node`), `node_id`, `url`, `target`, `css_class`, `sort_order`.

Tree depth limit: 3 levels (depth 0/1/2).

### 12.2 Menu CRUD
**Status:** 🟡 partial

`/admin/api/menus`:
- `GET /menus` — list with optional `language_id`.
- `GET /menus/:id` — single with nested items.
- `POST /menus` — create.
- `PATCH /menus/:id` — update metadata.
- `PUT /menus/:id/items` — atomic replace items tree (uses optimistic lock via `version`).
- `DELETE /menus/:id` — cascade-deletes items.

Known gaps:
- No `manage_menus` capability check (only `AuthRequired`).
- No URL scheme validation on menu items — `javascript:`, `data:` are accepted.
- `MenuService.cache` (sync.Map) unbounded.
- `ReplaceItems` deletes all items then re-inserts → ID turnover (any system caching item IDs breaks).

### 12.3 Public menu rendering
**Status:** ✅ working

`render_context.go::LoadMenus` does:
1. `menuSvc.ListWithItems(languageID)` — 2 queries total (menus + items).
2. Collects all `node_id`s referenced by items.
3. Single batch fetch of node `full_url`s.
4. Builds nested map per menu slug for templates.

---

## 13. Event bus & filters

### 13.1 Event bus
**Status:** 🟡 partial

`internal/events/bus.go` (153 LOC). Three publish modes:
- `Publish(action, payload)` — async, fan-out to goroutines.
- `PublishSync(action, payload)` — blocks until all handlers return.
- `PublishCollect(action, payload)` — runs result handlers, returns `[]string` outputs.

Two handler types:
- `Handler func(action, payload)` — fire-and-forget.
- `ResultHandler func(...) string` — used by `{{event "forms:render"}}` template integration.

Plus: `SubscribeAll(handler)` — catches every action.

### 13.2 Standard event taxonomy

Action shape: `<entity>.<op>`. Common publishers:

| Action | Publisher |
|---|---|
| `node.created`, `node.updated`, `node.deleted`, `node.published`, `node.unpublished` | `ContentService` |
| `user.registered`, `user.updated`, `user.deleted`, `user.login` | `auth/page_handler.go`, `user_handler.go` |
| `menu.created`, `menu.updated`, `menu.deleted` | `MenuService` |
| `setting.updated` | `SettingsHandler`, node `SetHomepage` |
| `taxonomy.created`, `taxonomy.updated`, `taxonomy.deleted` | `TaxonomyHandler` |
| `theme.activated`, `theme.deactivated` | `ThemeMgmtService` |
| `extension.activated`, `extension.deactivated` | `ExtensionLoader` (via helpers) |
| `email.send` | `Dispatcher` (consumed by provider plugins) |
| `sitemap.rebuild` | `CacheHandler` (consumed by sitemap-generator extension) |

### 13.3 Status

- ✅ **Unsubscribe shipped** — `Subscribe` returns `UnsubscribeFunc` (commit `9f9239c`). Reload no longer accumulates handlers.
- 🟡 **Unbounded goroutine fan-out** on `Publish` — still one goroutine per subscriber (acceptable for current subscriber counts; bounded worker pool is a planned optimization).
- 🟡 **Payload race** — `Payload` map is shared across handlers; the convention is "treat as read-only," but it's not enforced. Mutating from a handler is a bug.
- 🟡 No timeout in `PublishSync` — hung handler blocks publisher.

### 13.4 Filter chain
**Status:** ✅ working

`coreapi.RegisterFilter(name, priority, handler) → UnsubscribeFunc`. Filters run in priority order via `ApplyFilters(name, value)`.

Pointer-compare bug fixed in commit `9f9239c` — unsubscribe now uses opaque IDs assigned at register time.

---

## 14. Database

### 14.1 PostgreSQL connection
**Status:** ✅ working

GORM v2 driver. Pool config: max idle 10, max open 100, conn max lifetime 1h. Logger level: Warn (hardcoded).

### 14.2 Embedded migrations
**Status:** ✅ working

37 SQL migrations in `internal/db/migrations/`, embedded via `//go:embed`. Each runs in its own transaction with bookkeeping in `schema_migrations(filename PK)`.

Known gaps:
- 0 rollback scripts.
- Filename collision: `0012_extensions.sql` + `0012_template_source.sql` (sorted lexicographically, deterministic).
- No `CREATE INDEX CONCURRENTLY` — large-table migrations block writes.
- No CHECK constraints on enum-shaped columns.

### 14.3 Auto-seed on first boot
**Status:** ✅ working

`SeedIfEmpty` checks if users table is empty and runs the full seed. Seeded:
- 4 system roles (admin/editor/author/member) with capabilities JSON.
- Admin user (env-driven email + random password if not set; password printed once to stdout).
- Sample home content node (`/`).
- 5 layout blocks (primary-nav, user-menu, site-header, footer-nav, site-footer).
- 1 default layout.
- 4 auth block types (login/register/forgot/reset forms).
- 4 auth pages (`/login`, `/register`, `/forgot-password`, `/reset-password`).
- 5 email templates (welcome, user-registered-admin, password-reset, node-published, node-created-admin).
- 4 email rules.
- 2 menus (main-nav, footer-nav) with seed items.
- 2 site settings: `site_name`, `site_url`.

Known gap: re-running `Seed` (vs `SeedIfEmpty`) overwrites admin customizations to roles, email templates, auth block types.

---

## 15. Admin SPA & SDUI (Server-Driven UI)

### 15.1 SDUI engine
**Status:** ✅ working

`internal/sdui/engine.go` (2571 LOC — the largest file in core). Per-page layout factories build a JSON `LayoutNode` tree:
- 16+ page builders: dashboard, content_types, taxonomies, node list, taxonomy terms, templates, layouts, block types, layout blocks, menus, themes, extensions.
- Each layout is composable: typed `LayoutNode { Type, Props, Children, Actions }`.
- Actions are a typed enum: `CORE_API`, `NAVIGATE`, `CONFIRM`, `SEQUENCE`, `SET_KEY`, etc. (`types.go`).

Endpoints:
- `GET /admin/api/boot` → `BootManifest`.
- `GET /admin/api/layout/:page` → `LayoutNode` for a given admin page.

### 15.2 SSE broadcaster
**Status:** ✅ working

`GET /admin/api/events` — Server-Sent Events stream. Subscribes to event bus and forwards admin-relevant events:

| SSE event type | Triggered by |
|---|---|
| `NOTIFY` | `notify`, `user.notification` actions |
| `SETTING_CHANGED` | `setting.updated` |
| `NAV_STALE` | extension/theme activation, `node_type.*`, `taxonomies:register` |
| `ENTITY_CHANGED` | dotted entity actions like `layout_block.updated` |

Known gaps:
- Per-client buffer is 32 — slow client blocks publisher.
- Subscribe leaks per #18 (event bus has no unsubscribe).

### 15.3 Action execution
**Status:** ✅ working (with capability caveat)

Actions emitted by SDUI layouts are dispatched by the React SPA. **Server-side capability check at the receiving endpoint is the real boundary** — the SPA may hide buttons based on capabilities, but never trust client-side hiding.

---

## 16. MCP server (AI integration)

### 16.1 Bearer-token API mounted at `/mcp`
**Status:** ✅ working

`internal/mcp/server.go`. Two transports:
- Fiber middleware path (`auth.AuthRequired` + `authMiddleware`).
- Streamable HTTP path (re-validates bearer per request).

### 16.2 MCP tokens (`mcp_tokens`, `mcp_audit_log` tables)
**Status:** ✅ working

Tokens: `vcms_<32 random bytes hex>`. Stored as SHA-256 hash. Display prefix kept for log identification.

Token CRUD via `/admin/api/mcp/tokens` (admin only).

### 16.3 Scope × class ACL
**Status:** ✅ working

| Scope | Allowed classes |
|---|---|
| `read` | read |
| `content` | read, content |
| `full` | read, content, full |

Tools are tagged with their class on registration:
- **read** — `*.get`, `*.query`, `*.list`, render preview.
- **content** — content mutations (nodes, taxonomies, terms, menus, media, files).
- **full** — settings, users, extensions/themes, `data.exec`, `http.fetch`.

`data.exec` (raw SQL) additionally requires `VIBECMS_MCP_ALLOW_RAW_SQL=true` env, then is gated by the kernel's internal-only check at `impl_datastore.go`.

### 16.4 Per-token rate limiter
**Status:** ✅ working

60 req/min, burst 10. Backed by `golang.org/x/time/rate`. Process-local (not cluster-aware).

### 16.5 Audit log
**Status:** ✅ working

Every tool call writes `(token_id, tool, args_hash, status, error_code, duration_ms)`. Indexed for `(token_id, created_at DESC)` and `(tool, created_at DESC)`. Daily retention sweep (commit `eb0c1eb`) prunes entries beyond the configured `mcp_audit_retention_days` setting.

### 16.6 Tools surface
**Status:** ✅ working — extensive

Tools registered per domain (one `tools_*.go` file per domain):

| Domain | Notes |
|---|---|
| `core.nodes.*` | get, query, create, update, delete |
| `core.node_types.*` | register/get/list/update/delete |
| `core.taxonomies.*` + `core.terms.*` | full CRUD |
| `core.menus.*` | get, list, create, update, replace items, delete |
| `core.media.*` | get, query, upload, delete |
| `core.files.*` | store, delete |
| `core.events.*` | emit, subscribe (subscribe is internal-only via callbacks) |
| `core.filters.*` | apply |
| `core.email.*` | send |
| `core.http.*` | fetch (subject to SSRF defenses) |
| `core.log.*` | write |
| `core.settings.*` | get, set, get_all |
| `core.field_types.list` | enumerates the field_types registry |
| `core.users.*` | get, query (read-only) |
| `core.data.*` | get, query, create, update, delete + exec (gated by env+scope+internal check) |
| `core.render.*` | render block / page preview |
| `core.guide.*` | meta-tool listing all registered tools |
| `core.system.*` | system info |

### 16.7 MCP resources
**Status:** ✅ working

The `internal/mcp/resources.go` exposes browsable resources (theme files, extension files, etc.) via the MCP resource protocol.

---

## 17. Plugin contract (kernel ↔ plugin ABI)

### 17.1 HashiCorp go-plugin handshake
**Status:** ✅ working

- Protocol version: 2.
- Magic cookie: `VIBECMS_PLUGIN=vibecms`.
- gRPC-only (no NetRPC).

### 17.2 Plugin protocol (`pkg/plugin/proto/vibecms_plugin.proto`)
**Status:** ✅ working — minimal surface

The plugin must implement:
- `GetSubscriptions() → SubscriptionList` — events the plugin wants to receive.
- `HandleEvent(action, payload bytes) → EventResponse` — invoked when subscribed event fires.
- `HandleHTTPRequest(req) → resp` — invoked when a kernel route proxies to this plugin.
- `Shutdown() → Empty` — graceful shutdown.
- `Initialize(req)` — receives the broker ID for the kernel's `VibeCMSHost` service.

Known gaps:
- All gRPC calls from kernel → plugin use `context.Background()` (no cancellation propagation).
- `Shutdown` has no timeout — hung plugin blocks `pluginManager.StopAll()`.

### 17.3 Host service (kernel exposes CoreAPI to plugin)
**Status:** ⚠️ working but ungated

The `proto/coreapi/vibecms_coreapi.proto` defines the `VibeCMSHost` service. When a plugin starts, the kernel:
1. Allocates a broker ID.
2. Starts a gRPC host server on that ID, registering `VibeCMSHost` backed by an unguarded `coreImpl`.
3. Tells the plugin the broker ID via `Initialize`.
4. Plugin connects back and can call any CoreAPI method.

**The unguarded `coreImpl` is the kernel's #1 security finding** (see §13 below).

---

## 18. CoreAPI surface (60-method interface)

### 18.1 Interface surface

`internal/coreapi/api.go` defines `type CoreAPI interface` with 60 methods across 17 domains:

| Domain | Methods |
|---|---|
| Nodes | GetNode, QueryNodes, ListTaxonomyTerms, CreateNode, UpdateNode, DeleteNode |
| Taxonomies (defs) | RegisterTaxonomy, GetTaxonomy, ListTaxonomies, UpdateTaxonomy, DeleteTaxonomy |
| Terms | ListTerms, GetTerm, CreateTerm, UpdateTerm, DeleteTerm |
| Settings | GetSetting, SetSetting, GetSettings |
| Events | Emit, Subscribe |
| Email | SendEmail |
| Menus | GetMenu, GetMenus, CreateMenu, UpdateMenu, UpsertMenu, DeleteMenu |
| Routes | RegisterRoute, RemoveRoute |
| Filters | RegisterFilter, ApplyFilters |
| Media | UploadMedia, GetMedia, QueryMedia, DeleteMedia |
| Users | GetUser, QueryUsers (read-only) |
| HTTP | Fetch |
| Log | Log |
| Data store | DataGet, DataQuery, DataCreate, DataUpdate, DataDelete, DataExec |
| Node types | RegisterNodeType, GetNodeType, ListNodeTypes, UpdateNodeType, DeleteNodeType |
| File storage | StoreFile, DeleteFile |

### 18.2 Three backing implementations

1. **`coreImpl`** (`internal/coreapi/impl*.go`) — direct Go calls. Used by core code.
2. **`capabilityGuard`** (`capability.go`) — wraps any `CoreAPI`, checks `caller.Capabilities[required_cap]` before delegating. Internal callers bypass the check.
3. **gRPC host server** (`grpc_server.go`) — exposes whatever `CoreAPI` it's constructed with as a gRPC service.

### 18.3 Capability matrix

| Capability | Methods gated |
|---|---|
| `nodes:read` | GetNode, QueryNodes, ListTaxonomyTerms, ListTerms, GetTerm |
| `nodes:write` | CreateNode, UpdateNode, CreateTerm, UpdateTerm, DeleteTerm |
| `nodes:delete` | DeleteNode |
| `nodetypes:read` | GetNodeType, ListNodeTypes, GetTaxonomy, ListTaxonomies |
| `nodetypes:write` | RegisterNodeType, UpdateNodeType, DeleteNodeType, RegisterTaxonomy, UpdateTaxonomy, DeleteTaxonomy |
| `settings:read` / `settings:write` | GetSetting/GetSettings, SetSetting |
| `events:emit` / `events:subscribe` | Emit, Subscribe |
| `email:send` | SendEmail |
| `menus:read` / `menus:write` / `menus:delete` | Get/Create/Update, Delete |
| `routes:register` | RegisterRoute, RemoveRoute |
| `filters:register` / `filters:apply` | RegisterFilter, ApplyFilters |
| `media:read` / `media:write` / `media:delete` | Get/Query/Upload, Delete |
| `users:read` | GetUser, QueryUsers |
| `http:fetch` | Fetch |
| `log:write` | Log |
| `data:read` / `data:write` / `data:delete` | DataGet/DataQuery, DataCreate/DataUpdate/DataExec, DataDelete |
| `files:write` / `files:delete` | StoreFile, DeleteFile |

### 18.4 Capability bypass for internal callers

`CallerInfo.Type == "internal"` short-circuits all checks (`capability.go:19`). `InternalCaller()` is the default returned by `CallerFromContext` when no caller is set in ctx. This is fail-open by design — internal kernel code (admin handlers, render pipeline) operates without capability gating, since enforcement happens at the HTTP edge via `auth.CapabilityRequired`.

The risk is forgetting to call `WithCaller` from a path that should be gated. Per commit `54f573a`, plugin and Tengo callers are always wrapped via `NewCapabilityGuard` at construction time (`cmd/vibecms/main.go:252`), so the only fail-open paths are kernel-internal where capability gating happens earlier in the request flow.

---

## 19. Health, monitoring, observability

### 19.1 Health endpoint
**Status:** 🟡 partial

`GET /api/v1/health` (public) returns `{"status":"up"}`. **Does not actually check DB** — ping is in `/stats` only.

### 19.2 Stats endpoint
**Status:** 🟡 partial

`GET /api/v1/stats` (bearer-token auth):
- Uptime in seconds.
- `runtime.NumGoroutine()`.
- DB ping result (`connected` / `disconnected` / `error`).
- Storage hardcoded to `"ok"` (no actual probe).
- Total node count and published count.

Known gap: bearer comparison is not constant-time.

### 19.3 Logging
**Status:** ✅ working (commit `dcde556`)

Structured `slog` with request-id correlation. Development format = human-readable text; production = JSON to stdout. Every request gets an `X-Request-Id` (generated if absent) propagated through the request context.

The `coreapi.Log` method (callable by extensions) prefixes with `[ext:<slug>]` and writes through the same slog path with the level (`info`/`warn`/`error`/`debug`).

Sensitive-field redaction is enforced by convention: passwords, session tokens, MCP raw tokens, and secret site-setting values must never be logged. Plugin response bodies are no longer logged at INFO (commit `eb0c1eb`).

---

## 20. Build & deployment

### 20.1 Multi-stage Docker build
**Status:** ✅ working

Stages:
1. Node 20 — admin SPA build (`admin-ui/dist`).
2. Node 20 — extension admin-UI builds (`extensions/*/admin-ui/dist`).
3. Go 1.24 — kernel binary + plugin binaries.
4. Alpine 3.20 runtime — assembles binary, templates, themes, extensions, dist artifacts.

Plugin build is **fail-loud** (Dockerfile:42-45) — kernel rejects images with no plugins.

### 20.2 Docker Compose
**Status:** 🟡 partial

`docker-compose.yml` ships with default credentials (`vibecms_secret`, `admin123`) for dev. Comment notes prod config differs (`coolify-compose.yml`).

### 20.3 Makefile
**Status:** ✅ working

Targets: `build`, `run`, `dev`, `test`, `clean`, `db-up`, `db-down`, `migrate`, `seed`, `lint`, `ui` (hot-copy admin UI into running container), `deploy-local`.

### 20.4 CLI subcommands
**Status:** ✅ working

- `vibecms` — start the server.
- `vibecms migrate` — run migrations and exit.
- `vibecms seed` — run full seed and exit.

---

## 21. Critical findings summary

### Resolved since the original audit

| # | Original finding | Fix commit |
|---|---|---|
| 1 | Capability guard bypassed for extensions/themes/scripts | `54f573a`, `654dae5` |
| 2 | Public registration grants `editor` role with admin access | `76f6124` |
| 3 | Forgot/reset password are stubs | `76f6124` (real flow shipped via `password_reset_tokens` table, migration 0037) |
| 4 | UpdateUser self-promotion + RoleHandler mass-assignment | `76f6124` |
| 5 | `config.Load` no production safety guards | `7e29de1` (refuses to boot on default DB password, empty `SESSION_SECRET`, `DB_SSLMODE=disable` on public hosts, etc.) |
| 7 | Status / NodeType not validated on Update → ACL bypass | `9f9239c` |
| 8 | Search bypasses access filter | `9f9239c` |
| 9 | Filter unsubscribe broken (pointer compare on range copies) | `9f9239c` (returns opaque `UnsubscribeFunc` ID) |
| 10 | Event bus has no Unsubscribe | `9f9239c` (Subscribe returns `UnsubscribeFunc`) |
| 11 | SSE buffer of 32 blocks publisher | `9f9239c` (drop-on-full instead of block) |
| 12 | No CSRF protection on admin API | `76f6124` (`auth.JSONOnlyMutations` middleware) |
| 13 | No rate limit / no account lockout on auth | `76f6124` (`auth/lockout.go`, `auth/rate_limit.go`) |
| 14 | Plugin binaries unsigned | `654dae5` (signed handshake) |
| 15 | Theme git install — token leaks, no scheme allowlist, hostile `.git/config` | `f4ac40f` (HTTPS-only, encrypted tokens, `.git/config` reset, HMAC webhook) |
| 16 | STARTTLS downgrade vulnerable | `eb0c1eb` (`email_smtp_require_tls` setting, default true) |
| 17 | SMTP CRLF header injection via subject | `eb0c1eb` |
| 18 | `EmailLog.RenderedBody` stores reset tokens permanently | `eb0c1eb` (retention crons; reset tokens have separate one-shot table) |
| 19 | `is_encrypted` flag decorative — settings plaintext | `7e29de1` (AES-256-GCM via `internal/secrets/`) |
| 20 | SSRF via `impl_http.Fetch` | `2344aa1` (scheme allowlist, internal-host blocklist, redirect bound) |
| — | SMTP/Resend providers in core (hard-rule violation) | `eb0c1eb` (extracted to `extensions/smtp-provider`, `extensions/resend-provider`) |
| — | Files past 500-LOC hard limit | `e6d8551` (split where structurally sensible) |
| — | Plain `log.Printf` everywhere — no levels, no correlation | `dcde556` (structured `slog` with request-id) |
| — | XSS in richtext fields | `55653e5` (bluemonday at render-time) |
| — | Unbounded growth on `mcp_audit_log`, `email_logs`, revisions | `eb0c1eb` (retention sweeps) |
| — | Plugin response body preview leaked into INFO logs | `eb0c1eb` |

### Open items

These were either out of scope for the recent hardening pass or are tracked in `docs/plans/`:

- 🟡 `AuthorID` not always populated on `CoreAPI.CreateNode` (the kernel side is fixed for admin handler create; verify Tengo `nodes.create` path).
- 🟡 Cache key for parsed templates uses full template source — large keys in process memory. LRU with content-hash key is the planned replacement.
- 🟡 No retention on `content_node_revisions` beyond the daily 50-most-recent-per-node sweep — extremely heavy editing still grows the table.
- 🟡 No bytecode cache for Tengo scripts — recompile on every request.
- 🟡 `app.Shutdown` waits up to 30 s but in-flight SSE streams can hold the close longer than the timeout in some edge cases.
- 🟡 Public extension routes can declare arbitrary paths (`extension.json::public_routes`) and could shadow core paths — namespace enforcement is a planned fix.
- 🟡 No tests on the capability guard's per-method matrix (highest-priority test gap; see `core_dev_guide.md` §8).

---

## 22. Cleanup opportunities

The October 2026 hardening pass already removed:
- `internal/cms/admin_handler.go`, `media_handler.go`, dead `node_handler.go` blocks (~800 LOC).
- `internal/email/smtp.go`, `resend.go`, `provider.go` (moved to `extensions/smtp-provider`, `extensions/resend-provider`).
- 11 files that were past the 500-LOC hard limit (split where structurally sensible).

Still open:
- `template_renderer.go::image_url`/`image_srcset` — assume the media-manager extension's URL scheme; should move there.
- `internal/cms/file_browser.go` hidden-file handling needs the listing exclusion replicated on direct read.
- Some `b, _ := json.Marshal(x)` patterns in `impl_*.go` need explicit error handling (low risk; encoders cannot fail on the input shapes used).

---

## 23. What's NOT in the kernel (intentionally)

Per CLAUDE.md's hard rule, these are extension responsibilities and are not in core:

- Image optimization, WebP, thumbnail caching → `media-manager`.
- Email delivery providers → `smtp-provider`, `resend-provider`.
- Sitemap generation → `sitemap-generator`.
- Form rendering and submission → `forms-ext`.
- SEO meta-tag generation → expected to be in a future SEO extension.
- File-system-based content (page-tree YAML) → not present.
- Block library / page builder editor → admin SPA extension UIs.
- E-commerce, multi-tenancy, federation → not present.

The kernel deliberately stays minimal so each capability can ship as a separately-versioned extension.
