# VibeCMS Architecture

VibeCMS is a Go-based, AI-native CMS built on a **kernel + extension** model. The core is a minimal Go binary that provides infrastructure only (content nodes, rendering, auth, CoreAPI, MCP server, VDUS engine). Every user-visible feature — media, email, sitemaps, forms — ships as an independent extension with its own gRPC plugin, admin micro-frontend, database tables, and SQL migrations.

This document is the canonical architectural reference. Companion docs:

- [`extension_api.md`](./extension_api.md) — building extensions (manifests, gRPC plugins, capabilities)
- [`scripting_api.md`](./scripting_api.md) — Tengo `core/*` modules for theme + extension scripts
- [`theming.md`](./theming.md) — building themes (layouts, partials, blocks, assets)
- [`vdus.md`](./vdus.md) — Server-Driven UI for the admin
- [`forms.md`](./forms.md) — Forms extension reference
- [`core_dev_guide.md`](./core_dev_guide.md) — kernel internals & development workflows
- [`core_features.md`](./core_features.md) — exhaustive feature inventory
- [`database-schema.md`](./database-schema.md) — schema reference

---

## 1. The Hard Rule

> **If disabling/removing an extension would leave dead code in core, that code belongs in the extension, not core.**

Core provides generic plumbing that *any* extension could reuse. Feature-specific logic — image optimization, email templates, sitemap XML, form rendering — lives in extensions. Examples:

| Concern | Where it lives |
|---|---|
| Generic event bus, filter chain, plugin loader | Core |
| Public route proxy (any extension can mount routes) | Core |
| Image WebP conversion, thumbnail cache | `media-manager` extension |
| Email rule engine, template editor | `email-manager` extension |
| Sitemap.xml generator | `sitemap-generator` extension |
| Form builder, submission storage | `forms` extension |

Core code that violates this rule is a bug. The recent kernel refactors (commit `eb0c1eb refactor(core): extract SMTP/Resend providers + retention crons + dead-code purge`) moved the last in-core provider implementations out into `smtp-provider` / `resend-provider` extensions.

---

## 2. Topology

```
┌──────────────────────────────────────────────────────────────────────┐
│                        Browser (React Admin SPA)                     │
│   admin-shell  ←  useBoot()  ←  /admin/api/boot                      │
│        │       ←  useSSE()   ←  /admin/api/events  (SSE)             │
│        │       ←  useLayout() ← /admin/api/layout/:page              │
│        └─ extension micro-frontends loaded via import maps           │
└─────────────────────────┬────────────────────────────────────────────┘
                          │ HTTP / SSE
                          ▼
┌──────────────────────────────────────────────────────────────────────┐
│                       Go Kernel (Fiber, Go 1.24)                     │
│                                                                      │
│  Public catch-all  ──▶  PublicHandler  ──▶  RenderContext            │
│   /<lang>/<slug>          (cache, slug → node lookup)                │
│                                                                      │
│  Auth + RBAC      ──▶  CapabilityRequired middleware                 │
│   /admin/api/*         (JSONOnlyMutations CSRF guard)                │
│                                                                      │
│  MCP server       ──▶  Bearer-token + per-token rate limiter         │
│   /mcp                 + scope×class ACL + audit log                 │
│                                                                      │
│  Extension proxy  ──▶  /admin/api/ext/:slug/*  →  plugin gRPC        │
│  Public ext proxy ──▶  manifest-declared public_routes               │
│  Webhooks         ──▶  /api/v1/theme-deploy  (HMAC-validated)        │
│                                                                      │
│  CoreAPI ⇒ NewCapabilityGuard wraps everything plugins/scripts call  │
│                                                                      │
└──────────────────────────┬───────────────────────────┬───────────────┘
                           │ gRPC (HashiCorp go-plugin) │ Tengo VM
                           ▼                            ▼
┌──────────────────────────────────────────────────────────────────────┐
│  Extensions (gRPC plugins)        │  Themes (Tengo scripts)          │
│  bin/<slug>  one process each     │  themes/<slug>/scripts/*.tengo   │
│  HandleHTTPRequest, HandleEvent   │  events.on, filters.add, http.*  │
│  Subscribe to event bus           │  core/* modules via tengo_adapter│
└──────────────────────────────────────────────────────────────────────┘
                           │
                           ▼
                ┌──────────────────────┐
                │ PostgreSQL 16+ JSONB │
                └──────────────────────┘
```

---

## 3. Component Map

### 3.1 Kernel (`internal/`)

| Package | Responsibility |
|---|---|
| `coreapi/` | The single API surface every extension consumes. ~80 methods across 17 domains. Wrapped by `NewCapabilityGuard` for plugin/script callers. Three backends: `coreImpl` (direct), `grpc_server.go` (plugins), `tengo_adapter.go` (scripts). |
| `cms/` | Content services: nodes, node types, taxonomies, terms, layouts, partials, templates, block types, languages, themes, extensions, public site rendering. |
| `auth/` | Sessions, login, registration, password reset, RBAC middleware, account lockout, rate limiting, JSON-only CSRF guard. |
| `events/` | Pub-sub event bus (`Publish`, `PublishSync`, `PublishCollect`, `SubscribeAll`). Supports `Unsubscribe` (added in commit `9f9239c`). |
| `scripting/` | Tengo VM lifecycle: load/unload theme + extension scripts, mount HTTP routes, dispatch events/filters with capability propagation. |
| `mcp/` | MCP server at `/mcp` with bearer-token auth, per-token rate limiter, scope×class ACL, audit log, 18 tool domains. |
| `sdui/` | Server-Driven UI: boot manifest, layout-tree generators per page (~16+), SSE broadcaster bridging the event bus to admin clients. |
| `models/` | 27 GORM models. See `database-schema.md`. |
| `db/` | PostgreSQL connection, embedded migrations (38 files: `0001_initial_schema.sql` … `0037_password_reset_tokens.sql`), idempotent seed-on-empty. |
| `email/` | Dispatcher, rules, templates, layouts, logs. Provider implementations live in `extensions/smtp-provider` and `extensions/resend-provider`. |
| `rbac/` | Role admin handler, per-node-type access checks (`NodeAccess.CanRead/CanWrite`). |
| `secrets/` | AES-256-GCM at-rest encryption for sensitive settings (commit `7e29de1`). Master key from `VIBECMS_SECRET_KEY` env. |
| `sanitize/` | bluemonday-based XSS sanitization for richtext fields, applied at render time (commit `55653e5`). |
| `logging/` | Structured `slog` with request-id correlation (commit `dcde556`). Development = human-readable, production = JSON. |
| `config/` | Env-driven `Config` struct with production safety gates (refuses to boot on default credentials, missing `SESSION_SECRET`, etc.). |
| `rendering/` | `html/template` wrapper with custom funcmap (`safeHTML`, `event`, `filter`, `dict`, `image_url`, `image_srcset`). |
| `api/` | Boot manifest endpoint, `/health`, `/stats`, response envelopes. |

### 3.2 Extensions (`extensions/`)

Eight bundled extensions ship in-tree as the reference implementation:

| Slug | Type | Purpose |
|---|---|---|
| `content-blocks` | Static (no plugin) | 40 prebuilt blocks + 10 page templates |
| `email-manager` | gRPC plugin + admin UI | Email templates, rules, logs, layouts |
| `forms` (v2) | gRPC plugin + admin UI + `vibe-form` block + public route | Form builder with CAPTCHA, conditional logic, webhooks, GDPR |
| `hello-extension` | Static | Demo / contract test |
| `media-manager` | gRPC plugin + admin UI | Media library, image optimization, WebP, thumbnail cache |
| `resend-provider` | gRPC plugin + admin UI slot | Resend API email delivery |
| `sitemap-generator` | gRPC plugin | Yoast-style XML sitemaps, rebuilds on content change |
| `smtp-provider` | gRPC plugin + admin UI slot | SMTP email delivery |

### 3.3 Admin SPA (`admin-ui/`)

A pure shell: auth, sidebar, dashboard, extension loader. **Every feature page is rendered from a Server-Driven UI layout tree** returned by `GET /admin/api/layout/:page`. Complex interactions (rich text editors, drag-and-drop trees, code editors) ship as React components — VDUS controls layout orchestration, not interaction.

Built with React 19, Vite, TypeScript, Tailwind v4, shadcn/ui. Extension micro-frontends are loaded as ES modules via import-map shims that share `react`, `react-dom`, `react-router-dom`, `sonner`, `@vibecms/ui`, `@vibecms/icons`, `@vibecms/api` from `window.__VIBECMS_SHARED__`.

### 3.4 Themes (`themes/`)

Self-contained packages: `theme.json` manifest + `layouts/`, `partials/`, `blocks/`, `templates/`, `assets/`, `scripts/`. Activated via `core.theme.activate(id)` (no restart). On activation: previous theme deregisters, `theme.deactivated` event fires, new theme upserts layouts/blocks/partials/templates into the DB, `theme.tengo` runs (registering node types, seeding content, wiring event handlers and filters), `theme.activated` event fires.

---

## 4. CoreAPI

The single Go interface every extension and theme talks to. Defined in `internal/coreapi/api.go`.

### 4.1 Domains

```
Nodes              GetNode, QueryNodes, CreateNode, UpdateNode, DeleteNode,
                   ListTaxonomyTerms
Node Types         RegisterNodeType, GetNodeType, ListNodeTypes,
                   UpdateNodeType, DeleteNodeType
Taxonomies         RegisterTaxonomy, GetTaxonomy, ListTaxonomies,
                   UpdateTaxonomy, DeleteTaxonomy
Taxonomy Terms     ListTerms, GetTerm, CreateTerm, UpdateTerm, DeleteTerm
Settings           GetSetting, SetSetting, GetSettings
Events             Emit, Subscribe (returns UnsubscribeFunc)
Email              SendEmail
Menus              GetMenu, GetMenus, CreateMenu, UpdateMenu, UpsertMenu, DeleteMenu
Routes             RegisterRoute, RemoveRoute
Filters            RegisterFilter (returns UnsubscribeFunc), ApplyFilters
Media              UploadMedia, GetMedia, QueryMedia, DeleteMedia
Users (read-only)  GetUser, QueryUsers
HTTP (outbound)    Fetch
Log                Log
Data Store         DataGet, DataQuery, DataCreate, DataUpdate, DataDelete, DataExec
File Storage       StoreFile, DeleteFile
```

### 4.2 Three Adapters

```
                ┌──────────────────────────────────┐
                │   CoreAPI interface (api.go)     │
                └──────────────────────────────────┘
                          ▲          ▲          ▲
                          │          │          │
              ┌───────────┘          │          └────────────┐
              │                      │                       │
   ┌──────────────────┐  ┌────────────────────┐  ┌──────────────────────┐
   │ Tengo adapter    │  │ gRPC server        │  │ Internal direct call │
   │ (theme/ext .tgo) │  │ (compiled plugins) │  │ (kernel code)        │
   │ core/* modules   │  │ VibeCMSHost RPC    │  │ no wrapping          │
   └──────────────────┘  └────────────────────┘  └──────────────────────┘
                          │          │
                          ▼          ▼
              ┌──────────────────────────┐
              │  capabilityGuard wraps   │
              │  every method call       │
              └──────────────────────────┘
                          │
                          ▼
              ┌──────────────────────────┐
              │      coreImpl            │
              │ impl_*.go per domain     │
              └──────────────────────────┘
```

The `capabilityGuard` short-circuits when `caller.Type == "internal"` (the kernel itself), so direct internal calls are unguarded. Plugins receive a per-instance gRPC connection whose CallerInfo is populated from the manifest's declared capabilities and the plugin's owned tables.

### 4.3 Capability Strings

Capabilities are declared in `extension.json` as an array of `<domain>:<verb>` strings. Each CoreAPI method is gated by exactly one capability.

```
nodes:read, nodes:write, nodes:delete
nodetypes:read, nodetypes:write, nodetypes:delete
menus:read, menus:write, menus:delete
media:read, media:write, media:delete
data:read, data:write, data:delete, data:exec
files:read, files:write, files:delete
settings:read, settings:write
events:emit, events:subscribe
filters:register, filters:apply
http:fetch
log:write
email:send
users:read
routes:register, routes:remove
```

---

## 5. MCP Server (AI Interface)

Mounted at `/mcp` with permissive CORS (no cookies — bearer-token auth only). Provides ~50 tools across 18 domains so an AI agent can drive the entire CMS without filesystem access or HTML scraping.

| Domain | File | Sample tools |
|---|---|---|
| Nodes | `tools_nodes.go` | `core.node.create`, `.update`, `.list`, `.publish` |
| Node types | `tools_nodetypes.go` | `core.nodetype.create`, `.list` |
| Taxonomies | `tools_taxonomies.go` | `core.taxonomy.*`, `core.term.*` |
| Menus | `tools_menus.go` | `core.menu.*` |
| Media | `tools_media.go` | `core.media.upload`, `.optimize_image` |
| Data | `tools_data.go` | `core.data.*` (extension-scoped tables; `data.exec` requires `VIBECMS_MCP_ALLOW_RAW_SQL=true`) |
| Files | `tools_files.go` | `core.files.*` (theme/extension editing) |
| HTTP | `tools_http.go` | `core.http.fetch` |
| Field types | `tools_field_types.go` | `core.field_types.list` |
| Settings | `tools_settings.go` | `core.settings.*` |
| Events | `tools_events.go` | `core.events.*` |
| Filters | `tools_filters.go` | `core.filters.*` |
| Users | `tools_users.go` | `core.users.list`, `.get` (read-only) |
| Render | `tools_render.go` | `core.render.block`, `.node_preview`, `.layout` |
| Guide | `tools_guide.go` | `core.guide` (meta-tool: decision tree + state snapshot) |
| System | `tools_system.go` | `core.theme.*`, `core.extension.*` |
| Email | `tools_email.go` | `core.email.send` |

### 5.1 Token Model

| Field | Notes |
|---|---|
| `name` | Human label |
| `token_hash` | SHA-256 of the bearer token; raw token shown once at creation |
| `scope` | `read`, `content`, or `full` (subset matrix below) |
| `class` | Same domain enum used to tag tools at registration |
| `rate_limit` | Default 60 req / 10 s window (per-token, in-memory) |
| `expires_at` | Optional |

| Scope | Allowed classes |
|---|---|
| `read` | read |
| `content` | read, content |
| `full` | read, content, full |

### 5.2 Audit Log

Every tool call writes `(token_id, tool, args_hash, status, error_code, duration_ms)` to `mcp_audit_log`. Daily retention sweep keeps the table bounded (commit `eb0c1eb`).

---

## 6. VDUS (Server-Driven UI)

The admin SPA is a **pure rendering engine**. Page layouts are JSON trees emitted by the Go kernel; the React shell walks the tree, looks up each `type` in a component registry, resolves prop bindings, and dispatches Action Objects on user interactions. See [`vdus.md`](./vdus.md) for full details.

### 6.1 Three Endpoints

| Endpoint | Purpose |
|---|---|
| `GET /admin/api/boot` | Boot manifest: user, capabilities, active extensions, navigation tree, node types. Cached by the React shell, invalidated by SSE. |
| `GET /admin/api/layout/:page` | Layout tree for a given admin page (`dashboard`, `list`, `node-editor`, etc.). |
| `GET /admin/api/events` | Persistent SSE stream broadcasting `ENTITY_CHANGED`, `NAV_STALE`, `SETTING_CHANGED`, `NOTIFY` events. |

### 6.2 Action Objects

User interactions are encoded as typed Action Objects, never raw JavaScript:

```json
{
  "type": "SEQUENCE",
  "steps": [
    { "type": "CONFIRM", "message": "Delete this post?" },
    { "type": "CORE_API", "method": "nodes:delete", "params": { "id": "$params.id" } },
    { "type": "INVALIDATE", "keys": ["nodes"] },
    { "type": "TOAST", "message": "Post deleted" },
    { "type": "NAVIGATE", "to": "/admin/content/posts" }
  ]
}
```

Supported types: `CORE_API`, `NAVIGATE`, `TOAST`, `INVALIDATE`, `CONFIRM`, `SET_STORE`, `SEQUENCE`.

### 6.3 SSE Bridge

The broadcaster (`internal/sdui/broadcaster.go`) subscribes to the event bus and translates entity-shaped actions (`<entity>.<op>`) into typed SSE events. The React `useSSE` hook routes those into TanStack Query `invalidateQueries` calls. Per-client buffer cap = 32 with drop-on-full (commit `9f9239c`).

---

## 7. Extension Lifecycle

```
Filesystem scan
  ↓ extensions/<slug>/extension.json found
  ↓ INSERT INTO extensions (slug, manifest, ...) ON CONFLICT DO NOTHING
  ↓
POST /admin/api/extensions/:slug/activate
  ↓ UPDATE extensions SET is_active = true
  ↓ Run pending SQL migrations (extensions/<slug>/migrations/*.sql)
  ↓ Register block_types from manifest into theme asset registry
  ↓ Spawn plugin binary via HashiCorp go-plugin (gRPC handshake)
  ↓ Allocate broker ID; plugin dials back to call VibeCMSHost
  ↓ Load extension Tengo scripts with manifest-declared capabilities
  ↓ Publish extension.activated event
  ↓ Replay theme.activated for the extension's benefit
```

Deactivation reverses each step. Extensions are crash-isolated: a panic inside a plugin process never takes down the kernel.

---

## 8. Render Pipeline

Public page request to `/<lang>/<slug>`:

```
PublicHandler.PageByFullURL
  → DB lookup: WHERE full_url = ? AND status='published' AND deleted_at IS NULL
  → 404 fallback if not found
  → Resolve layout (layout_slug → layouts row, fallback to default)
  → Build TemplateData{App, Node, User} via RenderContext
      → BuildAppData: settings, languages, current_lang, head/foot scripts, block CSS/JS
      → BuildNodeData: theme-asset refs resolved, blocks rendered (cached by content_hash if cache_output=true)
      → LoadMenus: 2 queries total + batch node URL fetch
  → bluemonday-sanitize richtext fields
  → RenderLayout with renderLayoutBlock template func (max 5 levels recursion)
  → Set Content-Type: text/html, Send
```

Cache invalidation: `PublicHandler.SubscribeAll` clears caches on prefix-matched events (`theme.`, `setting.`, `block_type.`, `language.`, `layout`).

---

## 9. Security Posture

This section gives a topology-level view; see [`security.md`](./security.md) for the threat model and the per-control runbook.

| Control | Where |
|---|---|
| **Capability gate on every CoreAPI call** | `internal/coreapi/capability.go` — wraps `coreImpl` for plugin/script callers; internal callers bypass |
| **Per-table ACL on extension data** | `data:*` capability checked against manifest's `data_owned_tables` |
| **Plugin binary signing** | gRPC handshake validates signed binaries (commit `654dae5`) |
| **AES-256-GCM at rest** | Settings matching secret heuristic (`*_password`, `*_key`, `*_token`, etc.) encrypted by `internal/secrets/` |
| **bluemonday XSS sanitization** | Render-time on richtext fields (`internal/sanitize/`) |
| **JSON-only CSRF guard** | `auth.JSONOnlyMutations` rejects POST/PUT/PATCH/DELETE without `Content-Type: application/json` |
| **Account lockout + rate limit** | `auth/lockout.go`, `auth/rate_limit.go` |
| **Production safety gates** | `config.Validate()` refuses to boot in `APP_ENV=production` with default DB password, empty `SESSION_SECRET`, `DB_SSLMODE=disable` on non-internal networks, etc. |
| **Theme git install hardening** | HMAC-validated webhooks, scheme allowlist, encrypted git tokens (commit `f4ac40f`) |
| **Network/proxy hardening** | SSRF defense on outbound `http.Fetch`, scheme allowlist (commit `2344aa1`) |
| **Strict admin CORS, permissive /mcp** | `cmd/vibecms/main.go` (commit `ace0066`) |

---

## 10. Boot Sequence

`cmd/vibecms/main.go` orchestrates startup. Order matters because extensions must be subscribed to lifecycle events before themes activate.

```
1.  Pre-config CLI (catches subcommands like `vibecms migrate` that exit early)
2.  Load config; validate production safety
3.  Init slog (development: text, production: JSON)
4.  Init secrets service (AES-256-GCM master key from VIBECMS_SECRET_KEY)
5.  Connect DB → run migrations → SeedIfEmpty (first-boot admin user)
6.  Create event bus
7.  Create SDUI engine + broadcaster (broadcaster subscribes to bus)
8.  Build Fiber app (50 MB body limit, request-id middleware, strict admin CORS, permissive /mcp CORS)
9.  Construct services: sessions (hourly cleanup), content, node-types, languages,
    blocks, layouts, templates, menus, themes, email (daily log retention)
10. Build asset registries; load block assets from DB
11. Construct theme loader (don't load yet)
12. Build CoreAPI (coreImpl)
13. Wrap with NewCapabilityGuard for plugin/script callers
14. Construct script engine, wire to theme loader
15. Scan extensions → for each active: run migrations, load scripts
16. Construct render context, public handler
17. Register all admin routes (under /admin/api with AuthRequired)
18. Construct MCP server, mount at /mcp
19. Start gRPC plugin manager → for each active extension: start plugins,
    publish extension.activated
20. Wire email send func to active provider plugin
21. Activate theme (LoadTheme + LoadThemeScripts + PurgeInactiveThemes)
22. Mount static asset routes (/admin/assets, /admin/shims, /theme/assets,
    /extensions/<slug>/blocks)
23. Mount public-extension proxy
24. Mount script HTTP routes + .well-known + public catch-all (LAST)
25. Start Listen in goroutine
26. Wait for SIGINT/SIGTERM → cancel bg context → app.ShutdownWithTimeout(30s)
    → pluginManager.StopAll()
```

---

## 11. Why These Choices

### Why a kernel + extension model?

Single-binary CMSes accumulate dead code. Plugin systems with full kernel access leak privilege. By forcing every feature to be an extension with a declared capability list, the kernel stays small and the privilege boundary stays explicit. Disabling an extension actually disables its code path; capability denial is enforced at the API edge, not at the call site.

### Why MCP-first over REST-first?

The product hypothesis is that an AI agent should be able to build and manage a site end-to-end. REST endpoints designed for a human admin SPA are an awkward target for AI — they assume a session cookie, a logged-in user, and tightly-coupled validation. MCP tools are designed for AI: scoped tokens, structured tool descriptions, idempotent operations, audit logs. The admin SPA is built on top of MCP-equivalent CoreAPI calls; both human and AI go through the same capability surface.

### Why VDUS (server-driven UI) for the admin?

A traditional React admin couples UI state to client-side routing and component code. Adding a new node type or activating an extension required a frontend rebuild. With VDUS, layout trees are emitted by the Go kernel, so the navigation, list pages, editors, and even modal dialogs reconfigure instantly when the underlying data model changes. Complex interactions (rich text editors, drag-and-drop trees) still ship as real React components — VDUS controls *where* they appear and *what actions* fire on save, not how they work internally.

### Why Tengo for theme scripting (not WASM)?

Themes need lightweight hooks: register a node type on activation, modify a page title, inject HTML into a hook point. WASM is overkill — Tengo compiles Go-flavored syntax in the same process, sandboxes it (no `os`, no network, 50 k allocation cap, 10 s timeout), and gives theme authors a familiar API.

### Why gRPC plugins for full-fledged extensions?

Crash isolation. A panic inside a Go plugin's HTTP handler kills the plugin process, not the kernel. HashiCorp's `go-plugin` gives us bidirectional gRPC for free, so plugins call back into `VibeCMSHost` (a guarded `CoreAPI` server) without inventing a custom protocol.

---

## 12. What Lives Where (Cheat Sheet)

| I want to... | Look at |
|---|---|
| Add a CoreAPI method | `internal/coreapi/api.go` + `impl_*.go` + `capability.go` + `grpc_server.go` + `tengo_*.go` (see core_dev_guide §3.1) |
| Add an MCP tool | `internal/mcp/tools_<domain>.go` |
| Add a SDUI page | `internal/sdui/engine.go` (or split file) |
| Add a model + migration | `internal/models/` + `internal/db/migrations/00NN_*.sql` |
| Trace a public request | `PublicHandler.PageByFullURL` → `RenderContext.BuildNodeData` → block render → `RenderLayout` |
| Find the capability for a method | `internal/coreapi/capability.go` |
| See full plugin/CoreAPI proto | `proto/plugin/vibecms_plugin.proto`, `proto/coreapi/vibecms_coreapi.proto` |
| Understand the boot order | `cmd/vibecms/main.go` (top to bottom) |
| Check what's been seeded on first boot | `internal/db/seed.go` |
| See built-in field types | `internal/cms/field_types/registry.go` |
| Inspect SSE event types | `internal/sdui/broadcaster.go` |
