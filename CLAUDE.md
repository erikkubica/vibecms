# VibeCMS

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
- **Events:** emit, subscribe
- **Email:** send (via event bus → provider plugin)
- **Menus:** CRUD
- **Routes:** register, remove (Tengo HTTP endpoints)
- **Filters:** register, apply
- **Media:** upload, get, query, delete
- **Users:** get, query (read-only)
- **HTTP:** outbound fetch
- **Log:** leveled logging with caller prefix
- **Data Store:** DataGet, DataQuery, DataCreate, DataUpdate, DataDelete, DataExec (raw SQL)
- **File Storage:** StoreFile, DeleteFile

Three adapters:
1. **Tengo** (`core/*` modules) — for `.tgo` scripts
2. **gRPC** (VibeCMSHost service via GRPCBroker) — for compiled plugins
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
- `cmd/vibecms/`: Application entry point
- `internal/`: Core kernel:
    - `coreapi/`: CoreAPI interface, implementations, adapters (Tengo, gRPC, capability guard)
    - `cms/`: Content service, plugin manager, extension loader/proxy/migrations
    - `scripting/`: Tengo VM runtime, script callbacks, handler mounting
    - `models/`: GORM models (content_node, menu, user, role, etc.)
    - `email/`: Email dispatcher (core infrastructure, not admin management)
    - `events/`: Event bus (publish/subscribe)
    - `auth/`: Session auth, RBAC middleware
    - `db/`: Core migrations and connection pooling
    - `api/`: Response helpers
- `extensions/`: All feature extensions:
    - `media-manager/`: Media library (gRPC plugin + React micro-frontend)
    - `email-manager/`: Email templates, rules, logs (gRPC plugin + React micro-frontend)
    - `sitemap-generator/`: Yoast-style sitemaps (gRPC plugin + Tengo scripts)
    - `smtp-provider/`: SMTP delivery (gRPC plugin)
    - `resend-provider/`: Resend delivery (Tengo script)
    - `hello-extension/`: Demo extension
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
- **Extension Micro-Frontends:** Isolated Vite builds outputting ES modules. Import shared deps (`react`, `@vibecms/ui`, `@vibecms/api`, `@vibecms/icons`) via import map shims from `window.__VIBECMS_SHARED__`.
- **Tengo Modules:** Use `core/*` namespace (core/nodes, core/settings, core/events, etc.). ScriptCallbacks wire events.on, routes.register, filters.add to the engine.
- **Hard-Fail vs. Soft-Fail:**
    - Database connectivity failures → fatal server halt
    - Missing themes or script errors → log warning, continue
    - Extension plugin crashes → isolated, other extensions unaffected
- **Naming:** `snake_case` for Go files, `.html` for templates, `.tgo` for Tengo scripts. Template variables use `snake_case`.
- **Performance:** Atomic operations for hot-swapped config maps and cache. Sub-50ms TTFB for public pages.
- **Docker:** Multi-stage build: Node (admin SPA + extension UIs) → Go (binary + plugin binaries) → Alpine runtime.

## Documentation

- **[Extension API](docs/extension_api.md)**: Comprehensive handoff guide for building extensions (gRPC and Tengo).
- **[Scripting API](docs/scripting_api.md)**: Detailed reference for the embedded Tengo scripting engine.
- **[VDUS](docs/vdus.md)**: VibeCMS Dynamic UI System — how the SDUI layer works (boot manifest, layout trees, SSE, action handler, component registry).
- **[Architecture](docs/architecture.md)**: Kernel + extension architecture reference.

### Plans

- Active plans: `docs/plans/`
- Archived (pre-VDUS): `docs/superpowers/archive/`
