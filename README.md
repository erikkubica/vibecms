# VibeCMS

> High-performance, AI-native CMS built in Go. Kernel + extension architecture. MCP-first.

## What Is This

VibeCMS is a content management system designed around one idea: **an AI should be able to build and manage an entire website without human intervention.** The kernel provides content nodes, rendering, auth, and a powerful CoreAPI. Everything else (media, email, SEO, forms) is an extension — gRPC plugins with their own data, logic, and admin UI.

The key differentiator: every CMS operation is exposed as an MCP tool. An AI agent can create node types, seed content, activate themes, and manage extensions through a structured API — no filesystem access, no shell commands, no HTML scraping.

### Architecture in One Sentence

**Core = Linux kernel** (infrastructure only). **Extensions = Debian packages** (own their full stack). **Admin SPA = browser shell** (just loads extension micro-frontends). **Themes = templates + scripts + assets** (registered on activation, no restart needed).

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.24+ |
| HTTP | Fiber (routing, middleware) |
| ORM | GORM (PostgreSQL 16+) |
| Database | PostgreSQL (JSONB, GIN indexes) |
| Admin UI | React + TypeScript (Vite, Tailwind, shadcn/ui) |
| Templates | Go `html/template` |
| Scripting | Tengo (sandboxed VM, `core/*` modules) |
| Plugins | HashiCorp go-plugin (gRPC, bidirectional) |
| Storage | Local disk (S3 planned) |
| Security | Capability-based permissions, Ed25519 license verification |

## Quick Start

```bash
# Clone and run with Docker
git clone <repo-url> && cd vibecms
docker compose up --build

# App runs at http://localhost:3000
# Admin at http://localhost:3000/admin
# Default login: admin@example.com / changeme
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `vibecms` | Database user |
| `DB_PASSWORD` | `vibecms` | Database password |
| `DB_NAME` | `vibecms` | Database name |
| `THEME_PATH` | `themes/default` | Path to active theme |
| `APP_ENV` | `production` | `development` disables template caching |
| `PORT` | `3000` | HTTP port |
| `DATABASE_URL` | _(unset)_ | Optional: `postgres://user:pass@host:port/db?sslmode=disable`. Overrides individual `DB_*` vars when set. |
| `ADMIN_EMAIL` | `admin@vibecms.local` | Email for the auto-seeded admin user (first boot only). |
| `ADMIN_PASSWORD` | _(unset)_ | If unset, a random password is generated on first boot and printed to the app logs **once**. Set this to skip the random one. |

## Deploy on Coolify

VibeCMS ships a `coolify-compose.yml` for zero-config deployment:

1. In Coolify, create a new **Resource → Public Repository** and point it at this repo.
2. Build pack: **Docker Compose**. Compose file: **`coolify-compose.yml`**.
3. Set the **Domain** for the `app` service (Coolify fills `SERVICE_FQDN_APP` and provisions TLS).
4. Click **Deploy**.

Coolify auto-generates the database credentials, session secret, and monitor token via its `SERVICE_*` magic variables — nothing for you to fill in. The first admin password is generated on first boot and printed to the `app` container logs **once** (search the logs for `first-boot admin credentials`). To pre-set credentials, add `ADMIN_EMAIL` and `ADMIN_PASSWORD` env vars to the `app` service before the first deploy.

The pre-built image is published to `ghcr.io/erikkubica/vibecms:latest` (multi-arch, amd64 + arm64).

## Architecture

### Kernel + Extensions

**HARD RULE: If disabling/removing an extension would leave dead code in core, that code belongs in the extension, not core.**

Core provides:
- Content nodes (CRUD, rendering, layout resolution)
- Authentication, sessions, RBAC
- CoreAPI (35+ methods across 15 domains)
- Extension system (loader, proxy, migrations)
- Theme engine + public site rendering
- Event bus + filter chain
- MCP tool server

Extensions own:
- Go plugin binary (business logic, HTTP handling)
- Tengo scripts (event hooks, filters, routes)
- React micro-frontend (admin UI)
- SQL migrations (own tables)
- Manifest (declares capabilities, routes, menus)

### CoreAPI

Single Go interface providing all CMS capabilities. Three adapters:
1. **Tengo** (`core/*` modules) — for `.tgo` scripts
2. **gRPC** (VibeCMSHost service via GRPCBroker) — for compiled plugins
3. **Internal** (direct Go calls) — for core code

See `docs/extension_api.md` for the full reference.

## MCP Tools (AI Interface)

VibeCMS exposes ~50 MCP tools organized by domain. This is how AI agents interact with the CMS.

### Content Management

| Tool | Description |
|------|-------------|
| `core.node.create` | Create a content node |
| `core.node.update` | Update a node by ID |
| `core.node.get` | Get a node by ID/slug/URL |
| `core.node.list` | List nodes with filters |
| `core.node.delete` | Soft-delete a node |
| `core.nodetype.create` | Register a custom post type |
| `core.nodetype.list` | List all node types |
| `core.render.node_preview` | Preview rendered page HTML |

### Theme & Layout

| Tool | Description |
|------|-------------|
| `core.theme.list` | List installed themes |
| `core.theme.active` | Get active theme |
| `core.theme.activate` | Activate a theme (no restart) |
| `core.theme.standards` | Get theme dev standards |
| `core.layout.list` | List registered layouts |
| `core.block_types.list` | List block types |
| `core.guide` | Meta-tool: decision tree + CMS state snapshot |

### Extensions

| Tool | Description |
|------|-------------|
| `core.extension.list` | List extensions (active/inactive) |
| `core.extension.activate` | Activate extension (no restart) |
| `core.extension.deactivate` | Deactivate extension |

### AI Workflow Examples

**Create a trip booking site from scratch:**
```
1. core.theme.list → find theme ID
2. core.theme.activate(id) → activates theme, seeds node types + content
3. core.node.list(type="trip") → verify trips were created
4. core.render.node_preview(url="/trips/hanoi-street-food") → check rendering
```

**Add a new content type:**
```
1. core.nodetype.create(slug="recipe", label="Recipe", field_schema=[...])
2. core.node.create(node_type="recipe", title="Pho Bo", fields_data={...})
3. core.render.node_preview(url="/recipes/pho-bo") → verify
```

## Folder Structure

```
cmd/vibecms/           Application entry point
internal/              Core kernel
  coreapi/             CoreAPI interface + adapters (Tengo, gRPC, internal)
  cms/                 Content service, theme loader, extension loader
  scripting/           Tengo VM runtime
  models/              GORM models
  auth/                Session auth, RBAC
  events/              Event bus
  mcp/                 MCP tool server (~50 tools)
extensions/            Feature extensions (see extensions/README.md)
themes/                Theme repository
admin-ui/              React SPA shell
proto/                 Protocol Buffer definitions
storage/               Local file storage
```

## Theme Development

Themes are self-contained packages: layouts, partials, blocks, assets, scripts, and page templates.

### Theme Structure
```
themes/my-theme/
  theme.json           Manifest (layouts, blocks, assets, templates)
  layouts/             Page layouts (default.html, trip.html, etc.)
  partials/            Reusable fragments (site-header.html, site-footer.html)
  blocks/              Content blocks (my-hero/view.html + block.json)
  assets/              Static files (images/, styles/, scripts/)
  scripts/             Tengo scripts (theme.tengo entry point)
  templates/           Pre-populated page JSON files
```

### What Happens on Theme Activation

When `core.theme.activate(id)` is called (or `POST /admin/api/themes/:id/activate`):

1. **Previous theme deregistered** — layouts, blocks, partials orphaned (not deleted)
2. **theme.deactivated event** — extensions (e.g., media-manager) purge old theme assets
3. **New theme registered** — layouts, blocks, partials, templates upserted into DB
4. **theme.tengo executed** — registers node types, taxonomies, seeds content, event handlers, filters
5. **theme.activated event** — extensions import new theme's assets
6. **No server restart required**

### CRITICAL: Data Shape Consistency

The #1 source of theme bugs is mismatch between seed data shape and template access patterns.

**Rule: The seed script (theme.tengo) defines the data contract. Templates must match.**

Example bug: seed stores `tag` as a string, template accesses `tag.name`:
```
// theme.tengo seeds:  tag: "Foodie"
// template uses:      {{ with $fd.tag }}{{ .name }}{{ end }}  ← CRASH
// fix:                {{ with $fd.tag }}{{ . }}{{ end }}       ← correct
```

### Template Context: `.fields` vs `.fields_data`

Different template contexts use different key names for node fields:

| Where you are | Access pattern | Example |
|---------------|---------------|---------|
| Layout template (current node) | `.node.fields` | `{{ .node.fields.color }}` |
| Block template (block fields) | `.fields` or direct | `{{ .heading }}` |
| Tengo filter result (list_nodes) | `.fields_data` | `{{ .fields_data.color }}` |

### See Also
- `docs/theming.md` — Complete theming guide (1400+ lines)
- `docs/scripting_api.md` — Tengo scripting reference
- `extensions/README.md` — Extension development guide

## Extension Development

Extensions are feature packages. Two flavors:
- **gRPC plugin** — Go binary, full CoreAPI access, admin UI, HTTP handling
- **Tengo-only** — Just scripts (event hooks, filters, routes)

See `extensions/README.md` for the complete guide and `docs/extension_api.md` for the API reference.

## Key Conventions

- **Extensions First** — New features go in extensions, not core
- **Node-Based Content** — Everything is a `content_node` with `blocks_data` and `fields_data` JSONB
- **Admin SPA is a Shell** — Only auth, sidebar, dashboard. Feature pages are extension micro-frontends
- **Hard-Fail vs Soft-Fail** — DB down → fatal. Missing theme → log warning, continue. Extension crash → isolated
- **Naming** — `snake_case` Go files, `.html` templates, `.tgo` Tengo scripts
- **Performance** — Sub-50ms TTFB target for public pages

## Documentation

| Doc | Description |
|-----|-------------|
| `CLAUDE.md` | AI coding assistant context (architecture, conventions) |
| `docs/architecture.md` | Canonical architectural reference |
| `docs/extension_api.md` | Building extensions (manifests, gRPC plugins, capabilities) |
| `docs/scripting_api.md` | Tengo `core/*` modules for theme + extension scripts |
| `docs/theming.md` | Complete theming guide (layouts, partials, blocks, assets) |
| `docs/forms.md` | Forms extension public API reference |
| `docs/vdus.md` | Server-Driven UI (admin shell + layout trees + SSE) |
| `docs/core_dev_guide.md` | Kernel development workflows |
| `docs/core_features.md` | Exhaustive feature inventory |
| `docs/database-schema.md` | Database schema reference (27 GORM models, 38 migrations) |
| `docs/security.md` | Security posture and PR-time checklist |
| `extensions/README.md` | Extension development guide (alternative entry to `docs/extension_api.md`) |
| `themes/README.md` | Theme development guide (layouts, blocks, partials, Tengo seeding, forms wiring) |

## License

Proprietary. All rights reserved.