# Squilla

A high-performance, AI-native Go-based CMS with a kernel + extension architecture. The core is a minimal kernel providing content nodes, rendering, auth, and a powerful CoreAPI. All features (media, email, SEO, etc.) are extensions — gRPC plugins with their own data, logic, and admin UI.

## Tech Stack
- **Languages:** Go 1.24+
- **Frameworks:** Fiber (routing, middleware), GORM (PostgreSQL ORM)
- **Database:** PostgreSQL 16+ (leveraging JSONB and GIN indexes)
- **Frontend/Admin:** React + TypeScript SPA shell (Vite, Tailwind CSS, shadcn/ui)
- **Templating:** Go `html/template` (layouts, partials, content blocks)
- **Scripting:** Tengo (embedded sandboxed VM for hooks via `core/*` modules)
- **Plugins:** HashiCorp go-plugin (gRPC, bidirectional via GRPCBroker)
- **Storage:** Local Disk (S3 planned)
- **Security:** Ed25519 (license verification), capability-based extension permissions

## Architecture: Kernel + Extensions

**Core = Linux Kernel.** Provides infrastructure only:
- Content nodes (GORM models, CRUD, rendering)
- Authentication, sessions, RBAC
- CoreAPI (35+ methods across 15 domains)
- Extension system (loader, proxy, migrations, public route proxy)
- Theme engine + public site rendering
- Event bus + filter chain

**HARD RULE: If disabling/removing an extension would leave dead code in core, that code belongs in the extension, not core.** Core must never contain feature-specific logic — only generic infrastructure that multiple extensions could use. For example:
- Image optimization, WebP conversion, cache routing → media extension (NOT core)
- Email template management → email extension (NOT core)
- Public route proxy mechanism → core (generic, any extension can use it)
- Filter/event system → core (generic plumbing)

When building new features, always ask: "Does core need this to function as a skeleton CMS, or is this a feature that an extension provides?" If the latter, it goes in the extension — even if it means more work to wire up via gRPC/events/filters.

**Extensions = Debian Packages.** Own their full stack:
- **gRPC plugin** (Go binary) — business logic, handles HTTP requests via `HandleHTTPRequest`
- **Tengo scripts** — event hooks, HTTP routes, filters
- **React micro-frontend** — isolated Vite build loaded via import maps
- **SQL migrations** — own database tables, run on activation
- **Manifest** — declares capabilities, plugins, admin UI routes/menus

**Admin SPA = Pure Shell.** Just auth, sidebar, dashboard, and extension loader. Every feature page is an extension-owned micro-frontend.

### CoreAPI (`internal/coreapi/`)

Single Go interface providing all CMS capabilities to extensions:
- **Nodes:** CRUD + query
- **Node Types:** register, get, list, update, delete (extensions define custom post types)
- **Settings:** get, set, get-all
- **Events:** emit, subscribe (Subscribe / SubscribeResult / SubscribeErr; PublishRequest for sync request/reply with error propagation)
- **Email:** send (kernel routes via event bus → provider plugin; rule matching, template rendering, recipient resolution, and log retention live in the email-manager extension)
- **Menus:** CRUD
- **Routes:** register, remove (Tengo HTTP endpoints)
- **Filters:** register, apply
- **Media:** upload, get, query, delete (kernel keeps no bytes-on-disk fallback — operations route through whichever extension declares `provides:["media-provider"]`; the bundled media-manager fills the slot, but operators can hot-swap an S3/R2/Cloudinary extension by activating it with a higher priority)
- **Users:** get, query (read-only)
- **HTTP:** outbound fetch
- **Log:** leveled logging with caller prefix
- **Data Store:** DataGet, DataQuery, DataCreate, DataUpdate, DataDelete, DataExec (raw SQL)
- **File Storage:** StoreFile, DeleteFile

Three adapters:
1. **Tengo** (`core/*` modules) — for `.tgo` scripts
2. **gRPC** (SquillaHost service via GRPCBroker) — for compiled plugins
3. **Internal** (direct Go calls) — for core code

### Capability System

Extensions declare required capabilities in `extension.json`:
```json
{ "capabilities": ["nodes:read", "data:write", "files:write", "email:send"] }
```
CoreAPI enforces at every call. Internal callers bypass checks.

### Extension HTTP Proxy

Core proxies `/admin/api/ext/{slug}/*` → plugin's `HandleHTTPRequest` RPC. Plugin receives method, path, headers, body, query/path params, user ID. Returns status, headers, body.

## Folder Structure
- `cmd/squilla/`: Application entry point
- `internal/`: Core kernel:
    - `coreapi/`: CoreAPI interface, implementations, adapters (Tengo, gRPC, capability guard)
    - `cms/`: Content service, plugin manager (with provider-tag lookup for `media-provider`/`email.provider`/etc.), extension loader/proxy/migrations
    - `scripting/`: Tengo VM runtime, script callbacks, handler mounting
    - `models/`: GORM models (content_node, menu, user, role, etc.)
    - `events/`: Event bus (Subscribe / SubscribeResult / SubscribeErr; Publish / PublishSync / PublishCollect / PublishRequest)
    - `auth/`: Session auth, RBAC middleware (password reset gracefully degrades when no email provider is active)
    - `db/`: Core migrations and connection pooling
    - `api/`: Response helpers
- `extensions/`: All feature extensions:
    - `media-manager/`: Media library, optimizer, WebP, owns `media_files` (gRPC plugin + React micro-frontend; declares `provides:["media-provider"]`)
    - `email-manager/`: Owns email templates, rules, logs, layouts AND the dispatcher itself — the plugin subscribes to `*` (all events) and matches admin-defined rules (gRPC plugin + React micro-frontend)
    - `seo-extension/`: Owns `/robots.txt`, SEO defaults, AI-crawler policy (gRPC plugin)
    - `sitemap-generator/`: Yoast-style XML sitemaps (gRPC plugin + Tengo scripts)
    - `smtp-provider/` / `resend-provider/`: Email delivery (declare `provides:["email.provider"]`)
    - `forms/`, `content-blocks/`, `hello-extension/`: Reference implementations
- `themes/`: Theme repository (layouts, partials, blocks, assets, `.tgo` scripts)
- `admin-ui/`: React SPA shell (Vite, Tailwind CSS, shadcn/ui)
    - `public/shims/`: Import map shims for extension micro-frontends
    - `src/lib/extension-loader.ts`: Dynamic extension module loading
- `proto/`: Protocol Buffer definitions (plugin + coreapi)
- `pkg/plugin/`: Shared plugin interface, generated proto code
- `storage/`: Local file storage (media uploads)

## Key Conventions
- **Extensions First:** New features should be extensions, not core code. Built-in extensions are the reference implementation for third-party developers.
- **Node-Based Content:** All pages, posts, and entities are `content_nodes` with `blocks_data` JSONB storage.
- **Admin SPA is a Shell:** Only auth, sidebar, dashboard, and extension loader live in the core SPA. Feature pages are extension micro-frontends.
- **Extension Micro-Frontends:** Isolated Vite builds outputting ES modules. Import shared deps (`react`, `@squilla/ui`, `@squilla/api`, `@squilla/icons`) via import map shims from `window.__SQUILLA_SHARED__`.
- **Tengo Modules:** Use `core/*` namespace (core/nodes, core/settings, core/events, etc.). ScriptCallbacks wire events.on, routes.register, filters.add to the engine.
- **Hard-Fail vs. Soft-Fail:**
    - Database connectivity failures → fatal server halt
    - Missing themes or script errors → log warning, continue
    - Extension plugin crashes → isolated, other extensions unaffected
- **Naming:** `snake_case` for Go files, `.html` for templates, `.tgo` for Tengo scripts. Template variables use `snake_case`.
- **Performance:** Atomic operations for hot-swapped config maps and cache. Sub-50ms TTFB for public pages.
- **Docker:** Multi-stage build: Node (admin SPA + extension UIs) → Go (binary + plugin binaries) → Alpine runtime.
- **Binary uploads via MCP:** MCP tools that take binaries bigger than ~5 MB SHOULD provide an `_init`/`_finalize` pair alongside the inline-base64 form. Init returns a presigned URL + token, the client PUTs the bytes to `/api/uploads/<token>` (token IS the auth), then finalize routes through the same install pipeline as the legacy `body_base64` tool. See `docs/extension_api.md#9-presigned-uploads-large-binaries`.
- **404 from themes:** the kernel does not own the 404 page. The public catch-all looks up a layout with the reserved slug `404` from the active theme. If absent, a minimal hardcoded fallback renders. See `themes/README.md` for the layout contract.
- **Settings are schema-driven and per-language:** `internal/settings/builtin.go` declares the kernel-owned schema (general / SEO / advanced / languages / security groups). Each field has a `Translatable` flag. The store reads/writes per-locale rows with default-language fallback. Extensions register their own settings groups via `Registry.RegisterGroup`.
- **Recovery CLI:** `squilla reset-password <email> <new-password>` is a sub-command of the binary, callable without a running server, for cases where the email-based reset flow is unavailable (no email provider configured, broken admin UI, etc.). Lives in `cmd/squilla/main.go` pre-config CLI dispatch.

## Documentation

- **[Extension API](docs/extension_api.md)**: Comprehensive handoff guide for building extensions (gRPC and Tengo).
- **[Scripting API](docs/scripting_api.md)**: Detailed reference for the embedded Tengo scripting engine.
- **[VDUS](docs/vdus.md)**: Squilla Dynamic UI System — how the SDUI layer works (boot manifest, layout trees, SSE, action handler, component registry).
- **[Architecture](docs/architecture.md)**: Kernel + extension architecture reference.

### Plans

- Active plans: `docs/plans/`
- Archived (pre-VDUS): `docs/superpowers/archive/`
