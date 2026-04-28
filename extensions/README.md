# VibeCMS Extensions — The Builder's Guide

> An extension is a **self-contained feature package**: drop a folder under `extensions/`, restart the app, and the whole thing — admin UI, public routes, database tables, custom field types, content blocks, event hooks — wires itself into the kernel without you editing a single core file. The extension owns its full vertical slice.
>
> This document is the contract between **you** (the extension author) and **VibeCMS** (the kernel). Read it once, ship a production-grade extension. Pair it with `extensions/media-manager/` and `extensions/forms/` — the two reference implementations that exercise every surface area of the platform.

---

## Table of contents

1. [Mental model in 60 seconds](#1-mental-model-in-60-seconds)
2. [Anatomy of an extension](#2-anatomy-of-an-extension)
3. [`extension.json` — the manifest](#3-extensionjson--the-manifest)
4. [The three authoring surfaces](#4-the-three-authoring-surfaces)
5. [Capability table](#5-capability-table)
6. [Building the gRPC plugin (Go)](#6-building-the-grpc-plugin-go)
7. [HTTP routing — admin proxy vs public routes](#7-http-routing--admin-proxy-vs-public-routes)
8. [Events: fire-and-forget vs event-with-result](#8-events-fire-and-forget-vs-event-with-result)
9. [CoreAPI reference](#9-coreapi-reference)
10. [SQL migrations](#10-sql-migrations)
11. [Tengo scripts (`scripts/extension.tengo`)](#11-tengo-scripts-scriptsextensiontengo)
12. [Admin UI — micro-frontend authoring](#12-admin-ui--micro-frontend-authoring)
13. [Custom field types](#13-custom-field-types)
14. [Content blocks owned by extensions](#14-content-blocks-owned-by-extensions)
15. [Settings schema and the `email-settings` slot pattern](#15-settings-schema-and-the-email-settings-slot-pattern)
16. [Asset references and the media-manager handshake](#16-asset-references-and-the-media-manager-handshake)
17. [Lifecycle: scan → activate → run → deactivate](#17-lifecycle-scan--activate--run--deactivate)
18. [The Mandalorian rules](#18-the-mandalorian-rules)
19. [Troubleshooting](#19-troubleshooting)
20. [Skeleton: copy-paste a new extension](#20-skeleton-copy-paste-a-new-extension)
21. [Reference: every shipped extension](#21-reference-every-shipped-extension)
22. [Reference dissection: `media-manager`](#22-reference-dissection-media-manager)
23. [Reference dissection: `forms`](#23-reference-dissection-forms)
24. [Appendix: useful queries for development](#appendix-useful-queries-for-development)

---

## 1. Mental model in 60 seconds

VibeCMS is a **kernel + extensions** system. The kernel only ships:

- Content nodes (pages, posts, custom node types) with JSONB block storage
- Authentication, sessions, RBAC
- Theme engine + public site rendering
- An event bus + filter chain
- A capability-guarded **CoreAPI** (35+ methods)
- The extension loader, gRPC plugin manager, public/admin proxies, and Tengo VM

Everything else is an extension: media management, email delivery, forms, sitemaps, content blocks, third-party integrations.

```
extensions/
└── my-extension/
    ├── extension.json    ◄─ manifest. Declares capabilities, plugins, UI, routes, blocks, settings.
    ├── cmd/plugin/       ◄─ Go binary (gRPC plugin). Owns business logic + HTTP handlers + event handlers.
    ├── bin/              ◄─ compiled binary. Built by Docker (or by you, locally).
    ├── admin-ui/         ◄─ React micro-frontend. Built with Vite into an ES module.
    ├── scripts/          ◄─ Tengo scripts (optional). Runs in the kernel's sandbox VM.
    ├── migrations/       ◄─ SQL migrations. Run automatically on activation.
    ├── blocks/           ◄─ Content blocks (optional). Same shape as theme blocks.
    ├── templates/        ◄─ Page templates (optional).
    ├── layouts/          ◄─ Layouts (optional — shared with themes).
    ├── partials/         ◄─ Partials (optional).
    └── assets/           ◄─ Theme-asset-style media files imported on activation.
```

**Boot sequence:**

1. **Scan** — `extensions/*/extension.json` is parsed at startup. Every extension is upserted into the `extensions` table. Built-in extensions (the seven that ship with VibeCMS) auto-activate; everything else stays inactive until a user enables it in the admin.
2. **Activate** — for every active extension:
   - SQL migrations in `migrations/*.sql` run once each (tracked in `extension_migrations`).
   - The Tengo entry script `scripts/extension.tengo` executes.
   - Each `plugins[].binary` becomes a child process via HashiCorp `go-plugin` and is dispensed as a gRPC client.
   - The plugin's `Initialize(grpcConn)` runs; the plugin can now call back into CoreAPI.
   - The plugin's `GetSubscriptions()` is called and every event name is wired to the bus.
   - Block types, templates, layouts, and partials declared in the manifest are upserted.
   - Public routes from `public_routes[]` are mounted on the public Fiber app — **no auth middleware**.
   - The admin proxy mounts the admin route prefix at `/admin/api/ext/{slug}/*` (auth required).
   - `extension.activated` is emitted with the manifest's `assets[]` payload — `media-manager` listens for this and imports the assets into the media library.
   - `theme.activated` is replayed for the current active theme so extensions activated mid-runtime don't miss the theme event.
3. **Runtime** — the plugin process stays alive. Three communication paths flow through it:
   - **Admin HTTP**: browser → `/admin/api/ext/{slug}/*` → admin proxy (auth) → `HandleHTTPRequest` RPC.
   - **Public HTTP**: browser → `<your declared path>` → public proxy (no auth) → `HandleHTTPRequest` RPC.
   - **Events**: kernel event bus → `HandleEvent(action, payload)` RPC. Templates that call `{{event "name" ...}}` use the same path with a result-collecting variant.
   - The plugin can call back into CoreAPI (Nodes, Settings, Email, Files, Data Store, Log, …) via the bidirectional `VibeCMSHost` gRPC service. Every call passes the capability guard.
4. **Deactivation** — `extension.deactivated` fires (so other extensions can clean up); Tengo scripts unload; the plugin's `Shutdown()` is called and the process exits; block/template/layout/partial rows are removed; public and admin routes go cold (return `503` until reactivation).
5. **Crash isolation** — if your plugin panics, only your extension is affected. The kernel and other extensions keep running. You'll see the crash in `docker compose logs app` with the plugin's slug prefix.

**Hot-swap** of binaries works in development: rebuild your binary and restart the app (or `docker compose restart app`). The migrations system is idempotent, so re-activation is safe.

---

## 2. Anatomy of an extension

The full layout for a feature-complete extension (every directory is optional except `extension.json`):

```
extensions/my-extension/
├── extension.json
│
├── cmd/
│   └── plugin/
│       ├── main.go                  # Plugin entry point. Implements ExtensionPlugin.
│       ├── routes.go                # HandleHTTPRequest dispatch.
│       ├── handlers_*.go            # One file per resource (forms, submissions, …).
│       ├── helpers.go               # jsonResponse / jsonError envelopes.
│       └── templates/               # html/template files embedded with //go:embed.
│
├── bin/
│   └── my-extension                 # Compiled binary. Built by Docker (or `go build`).
│
├── admin-ui/
│   ├── package.json                 # devDeps only — vite, react, tailwind plugin.
│   ├── tsconfig.json
│   ├── vite.config.ts               # Externalizes react/sonner/@vibecms/* shims.
│   ├── src/
│   │   ├── index.tsx                # Entry: import "./index.css"; export every routed component.
│   │   ├── index.css                # @import "tailwindcss"; @source "./**/*.{ts,tsx}"
│   │   ├── MyListPage.tsx
│   │   ├── MyEditor.tsx
│   │   ├── MyFieldInput.tsx         # Component for a custom field type.
│   │   └── lib/                     # Internal helpers split out for clarity.
│   └── dist/                        # Build output: index.js + index.css. Loaded by the shell.
│
├── scripts/
│   └── extension.tengo              # Tengo entry. Optional. Runs once on activation.
│
├── migrations/
│   ├── 20250101_init.sql
│   └── 20250215_add_status.sql
│
├── blocks/                          # Optional content blocks.
│   └── my-block/
│       ├── block.json
│       ├── view.html
│       ├── style.css                # Optional — auto-injected per page.
│       └── script.js                # Optional — auto-injected per page.
│
├── templates/                       # Optional page templates.
│   └── my-template.json
│
├── assets/                          # Optional media. Imported into the library on activation.
│   └── images/
│       └── banner.jpg
│
└── preview.svg                      # Optional 256×256 preview shown in the admin extension picker.
```

Minimal extensions can omit almost everything. A Tengo-only extension (e.g. `resend-provider`) is just:

```
extensions/my-tiny-ext/
├── extension.json
└── scripts/
    └── extension.tengo
```

Naming rules:
- Slugs are `kebab-case` and must match the directory name.
- Go files are `snake_case.go`.
- Tengo files are `snake_case.tengo`.
- Migrations are `<sortable-prefix>_<description>.sql`. Date-based (`20260101_init.sql`) is conventional.

---

## 3. `extension.json` — the manifest

The manifest is the contract. Every binary, route, capability, block, custom field type, and admin UI route the extension wires up **must** be declared here. Anything not declared is invisible to the kernel.

### Full schema

```jsonc
{
  // Identity
  "name":        "Media Manager",          // Human-readable. Shown in the admin extension picker.
  "slug":        "media-manager",          // kebab-case; must match the folder name.
  "version":     "1.0.0",                  // Semver. Bump on schema/migration changes.
  "author":      "VibeCMS",
  "description": "Upload, organize, and manage media files.",

  // Loading
  "priority":    50,                       // Lower = earlier. Default 50.
  "provides":    ["media"],                // Free-form feature tags. Other extensions can check `manifest.Provides`.

  // Capabilities — see §5. Every CoreAPI call is guarded; declare the minimum.
  "capabilities": [
    "data:read", "data:write", "data:delete",
    "files:write", "files:delete",
    "settings:read", "settings:write",
    "events:emit",
    "log:write"
  ],

  // gRPC plugin binaries. One entry per binary you ship.
  "plugins": [
    { "binary": "bin/media-manager", "events": [] }
  ],

  // Public (unauthenticated) routes proxied to the plugin. No auth middleware.
  "public_routes": [
    { "method": "GET",  "path": "/media/cache/*" },
    { "method": "POST", "path": "/forms/submit/*" }
  ],

  // Admin UI manifest. The admin shell loads this micro-frontend.
  "admin_ui": {
    "entry": "admin-ui/dist/index.js",     // Path to the built ES module.

    // Sidebar entry. Section routes the menu into a sidebar group:
    // "content" (default), "design", "development", or "settings".
    // Set to `null` to hide from the sidebar entirely.
    "menu": {
      "label":    "Media",
      "icon":     "Image",                 // Any lucide-react icon name (case-sensitive).
      "section":  "content",
      "position": "3",
      "children": [
        { "label": "Library",         "route": "/admin/ext/media-manager/",          "icon": "Images" },
        { "label": "Image Optimizer", "route": "/admin/ext/media-manager/optimizer", "icon": "ImageDown" }
      ]
    },

    // Routes mounted under /admin/ext/{slug}/<path>. The component is the named export
    // from your built ES module (matches `src/index.tsx`'s `export { MediaLibrary }`).
    "routes": [
      { "path": "/",          "component": "MediaLibrary" },
      { "path": "/optimizer", "component": "ImageOptimizerSettings" }
    ],

    // Pages that should appear in the global Settings sidebar group.
    "settings_menu": [
      { "label": "Image Optimizer", "route": "/admin/ext/media-manager/optimizer", "icon": "ImageDown" }
    ],

    // Components injected into named slots provided by other extensions
    // (e.g. smtp-provider injects into email-manager's "email-settings" slot).
    "slots": {
      "email-settings": { "component": "SmtpSettings", "label": "SMTP" }
    },

    // Custom field types. The named component is rendered by the node editor when
    // a field with this type appears. See §13.
    "field_types": [
      {
        "type":        "media",
        "label":       "Media Selector",
        "description": "Select files from the media library",
        "icon":        "Image",
        "group":       "Media",
        "component":   "MediaFieldInput",
        "supports":    ["image", "gallery", "file"]
      }
    ]
  },

  // Settings schema rendered automatically as a form by the admin shell.
  "settings_schema": {
    "host":     { "type": "string",  "label": "SMTP Host", "required": true },
    "port":     { "type": "number",  "label": "SMTP Port", "default": 587 },
    "password": { "type": "string",  "label": "Password",  "sensitive": true },
    "encryption": {
      "type": "string", "label": "Encryption",
      "enum": ["none", "tls", "starttls"], "default": "tls"
    }
  },

  // Content blocks. Same shape as theme blocks. See §14.
  "blocks": [
    { "slug": "vibe-form", "dir": "vibe-form" }
  ],

  // Page templates. Same shape as theme templates.
  "templates": [
    { "slug": "form-submission-receipt", "file": "receipt.json" }
  ],

  // Layouts and partials (rare for extensions; usually theme territory).
  "layouts":  [],
  "partials": [],

  // Owned media assets. Imported into the media library on `extension.activated`
  // by the media-manager extension. Reference with `extension-asset:<slug>:<key>`.
  "assets": [
    { "key": "demo-banner", "src": "images/demo-banner.jpg", "alt": "Demo banner" }
  ]
}
```

### Validation cheatsheet

| Field | Required | Notes |
|---|---|---|
| `name`, `slug`, `version` | ✓ | `slug` must match the folder name. |
| `capabilities` | ✓ for non-trivial extensions | Every CoreAPI call goes through the capability guard. Declaring fewer than you need = runtime denials. Declaring more = code smell that reviewers will catch. |
| `plugins[].binary` | ✓ if you ship Go code | Path relative to extension root. The Dockerfile builds anything under `cmd/plugin/`. |
| `plugins[].events` | optional | Informational only — actual subscriptions come from the plugin's `GetSubscriptions()` RPC at runtime. |
| `public_routes` | ✓ for any path that should bypass auth | Declare even if you also accept admin requests on the same prefix; the public proxy doesn't read the admin route table. |
| `admin_ui.entry` | ✓ if you ship admin UI | The admin shell auto-injects a `<link rel="stylesheet">` for the sibling `index.css` — do **not** declare CSS in the manifest. |
| `admin_ui.menu.section` | optional | Honored by the SDUI sidebar engine: `"content"` (default), `"design"`, `"development"`, `"settings"`. Anything else lands at top level. |
| `admin_ui.routes[].component` | ✓ | Must match a named export from `admin-ui/src/index.tsx`. |
| `admin_ui.field_types[].component` | ✓ | Same — named export from your entry. The node editor renders this for any field with `type: "<your type>"`. |
| `settings_schema[*].sensitive` | optional | When `true`, the admin form masks the value and the kernel never logs it. Required for API keys / passwords. |
| `assets[].key` | ✓ | Must match `^[a-z0-9_-]+$`. Referenced as `extension-asset:<slug>:<key>` in templates and seed data. |
| `priority` | optional | Lower loads first. Use `< 50` to declare you must boot before standard extensions; `> 50` if you depend on others. |

### Field-by-field behavior

#### `capabilities` (the most important field)

Each entry in this list grants one capability via the `capabilityGuard` in `internal/coreapi/capability.go`. Every CoreAPI method consults this set. If your plugin calls `host.SendEmail(...)` and `email:send` isn't in `capabilities`, the call returns `ErrCapabilityDenied` and **nothing else happens** — no email, no error to the user, just a log line.

The capability table lives in §5 of this doc. Use the smallest set that compiles. If you only read settings, declare `settings:read` — never `settings:write` "just in case".

#### `plugins`

Each entry starts one gRPC plugin process. Most extensions need exactly one. Multiple plugins per extension are legal but rare — every plugin in the array shares the manifest's capabilities, so splitting buys you nothing unless the binaries genuinely have different lifetimes.

```jsonc
"plugins": [
  { "binary": "bin/my-extension", "events": [] }
]
```

The `events` field is **informational** — actual event subscriptions come from the plugin's `GetSubscriptions()` RPC at runtime, not from the manifest. Treat it as a hint to humans reading the manifest.

#### `public_routes`

Public routes register on the **public** Fiber app **without auth middleware**. The exact path you declare is the path users hit:

```jsonc
"public_routes": [
  { "method": "POST", "path": "/forms/submit/*" },
  { "method": "GET",  "path": "/media/cache/*" }
]
```

After this declaration:
- `POST https://yoursite.com/forms/submit/contact` reaches your plugin.
- `GET https://yoursite.com/media/cache/medium/2026/03/photo.jpg` reaches your plugin.
- `user_id` in the request payload is always `0` (no auth on public routes).

**Common mistake:** assuming public routes are mounted under `/api/ext/{slug}/...`. They are not. The admin proxy uses `/admin/api/ext/{slug}/*`; the public proxy uses **whatever path you declare**, verbatim. See §7.

#### `admin_ui`

See §12 for a full walkthrough of building the micro-frontend. The `admin_ui` block tells the kernel:
- Where to find the JS entry (`entry`).
- Which named exports to render at which routes (`routes[]`).
- How to add the extension to the sidebar (`menu`, `settings_menu`).
- Which slots in other extensions to inject into (`slots`).
- Which custom field types to register (`field_types[]`).

#### `blocks`, `templates`, `layouts`, `partials`

Extensions can ship the same kinds of presentation assets as themes:
- **`blocks`** — content blocks the page editor can drop into a node.
- **`templates`** — pre-built block sequences an editor can apply with one click.
- **`layouts`** and **`partials`** — page chrome and reusable fragments. These live in `layouts/<file>` and `partials/<file>` relative to the extension root, and are usually only declared by extensions that ship full sub-sites (rare — usually theme territory).

The schema for each is identical to themes (`themes/README.md` §3). For blocks, see §14 below for extension-specific notes.

#### `assets`

Files that should be imported into the media library when the extension activates:

```jsonc
"assets": [
  { "key": "demo-banner", "src": "images/demo-banner.jpg", "alt": "Hello extension demo banner" }
]
```

The media-manager extension subscribes to `extension.activated` and inserts a row into `media_files` for every entry, tagged `source='extension', extension_slug='<your slug>', asset_key='<key>'`. Reference these from blocks, templates, or seeded content with `extension-asset:<slug>:<key>` (see §16).

When the extension deactivates, media-manager removes those rows and deletes the files. The asset registry guarantees that switching extensions doesn't leave dead media references in your DB.

---

## 4. The three authoring surfaces

VibeCMS extensions can write code at three layers, each with a different runtime model:

| Surface | Language | Where it runs | Best for |
|---|---|---|---|
| **gRPC plugin** | Go | Child process started by the kernel; persistent | Heavy logic, custom HTTP endpoints, database tables, anything stateful |
| **Tengo scripts** | Tengo (Go-like, sandboxed) | Embedded VM in the kernel; **fresh VM per call** | Event hooks, one-off lifecycle reactions, simple route handlers |
| **Admin UI** | React + TypeScript | Browser; loaded by the admin SPA shell as an ES module | Every admin-facing UI surface — pages, modals, custom field types, sidebar slots |

You choose per-feature, not per-extension. `media-manager` and `forms` both use **all three** surfaces:
- gRPC plugin — file storage, validation, image processing, captcha verification.
- Tengo — the entry script (`extension.tengo`) that just announces "extension loaded". The current built-in extensions barely use Tengo; it exists so third-party developers can ship logic without compiling Go.
- Admin UI — full library/editor experiences.

`resend-provider` uses **only Tengo** — no Go binary, no admin UI of its own. It just hooks `email.send` and shells out to the Resend API via `core/http`.

`content-blocks` uses **only declarative manifests** — no plugin, no scripts, no admin UI of its own. Just 40 block definitions and 10 templates.

**Rule of thumb:** start with Tengo if your feature can express itself as event hooks. Reach for a gRPC plugin the moment you need:
- Owned database tables (anything beyond CoreAPI's nodes/settings/data store sugar).
- Synchronous HTTP endpoints with request bodies > a few KB.
- Native Go libraries (image processing, crypto, FS access).
- Long-lived state (rate limiters, in-memory caches, background workers).

---

## 5. Capability table

Every CoreAPI method has a guard. The list below is the source of truth — anything not in this table is **not** a real capability. Declaring a non-existent capability is silently ignored, but a future kernel may start rejecting unknown values (and reviewers will flag it).

| Capability | Allows |
|-----------|--------|
| `nodes:read` | `GetNode`, `QueryNodes`, `ListTaxonomyTerms`, `ListTerms`, `GetTerm` |
| `nodes:write` | `CreateNode`, `UpdateNode`, `CreateTerm`, `UpdateTerm`, `DeleteTerm` |
| `nodes:delete` | `DeleteNode` |
| `nodetypes:read` | `GetNodeType`, `ListNodeTypes`, `GetTaxonomy`, `ListTaxonomies` |
| `nodetypes:write` | `RegisterNodeType`, `UpdateNodeType`, `DeleteNodeType`, `RegisterTaxonomy`, `UpdateTaxonomy`, `DeleteTaxonomy` |
| `settings:read` | `GetSetting`, `GetSettings(prefix)` |
| `settings:write` | `SetSetting` |
| `events:emit` | `Emit(action, payload)` |
| `events:subscribe` | `Subscribe(action, handler)` (runtime only — `GetSubscriptions()` at startup is **not** capability-guarded) |
| `email:send` | `SendEmail` |
| `menus:read` | `GetMenu`, `GetMenus` |
| `menus:write` | `CreateMenu`, `UpdateMenu`, `UpsertMenu` |
| `menus:delete` | `DeleteMenu` |
| `routes:register` | `RegisterRoute`, `RemoveRoute` (runtime route registration via Tengo `core/routes` or gRPC. Public routes from `public_routes[]` do **not** require this — they're declarative.) |
| `filters:register` | `RegisterFilter` |
| `filters:apply` | `ApplyFilters` |
| `media:read` | `GetMedia`, `QueryMedia` |
| `media:write` | `UploadMedia` |
| `media:delete` | `DeleteMedia` |
| `users:read` | `GetUser`, `QueryUsers` |
| `http:fetch` | `Fetch` (outbound HTTP) |
| `log:write` | `Log(level, message, fields)` |
| `data:read` | `DataGet`, `DataQuery` |
| `data:write` | `DataCreate`, `DataUpdate`, `DataExec` |
| `data:delete` | `DataDelete` (falls back to `data:write` for backwards compatibility — declare `data:delete` explicitly) |
| `files:write` | `StoreFile` |
| `files:delete` | `DeleteFile` |

**Internal callers** (core Go code) bypass the guard entirely — capabilities only matter for plugin and Tengo callers. There is no per-extension override for "I'm a trusted extension"; built-in extensions go through the same gates as third-party ones.

**Capability denial** returns `ErrCapabilityDenied` from the CoreAPI call. Your plugin should treat this as a programming error (you forgot to declare the capability) and either log it loudly or panic — silent fallback obscures the bug.

---

## 6. Building the gRPC plugin (Go)

### Build command

```bash
cd extensions/my-extension
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/my-extension ./cmd/plugin/
```

**Why these flags:**
- `CGO_ENABLED=0` — required. The runtime image is Alpine; it has no glibc.
- `GOOS=linux GOARCH=amd64` — required for Docker. Skip if you're running the kernel locally on the same arch.
- The output path **must** match the manifest's `plugins[].binary` exactly.

The Dockerfile auto-builds every binary under `extensions/*/cmd/plugin/` during the multi-stage build. CI fails loudly if any plugin can't compile.

### The plugin interface

`pkg/plugin/interface.go` defines what your binary must implement:

```go
type ExtensionPlugin interface {
    Initialize(hostConn *grpc.ClientConn) error
    GetSubscriptions() ([]*pb.Subscription, error)
    HandleEvent(action string, payload []byte) (*pb.EventResponse, error)
    HandleHTTPRequest(req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error)
    Shutdown() error
}
```

| Method | Called when | What you do |
|---|---|---|
| `Initialize` | Once at startup, after the gRPC connection is established. | Wrap `hostConn` in a `coreapi.GRPCHostClient`. Initialize state (rate limiters, background workers, default settings). |
| `GetSubscriptions` | Once at startup, after `Initialize`. | Return the events you want to receive. Names are arbitrary strings — pick whatever the kernel or other extensions emit. |
| `HandleEvent` | Every time a subscribed event fires. | Switch on `action`, dispatch to a per-event handler. Return `{Handled: true}` for fire-and-forget; return `{Handled: true, Result: bytes}` for events called from templates expecting HTML output. |
| `HandleHTTPRequest` | Every time a proxied HTTP request arrives (admin or public). | Route by `req.Path` and `req.Method`. Return a `PluginHTTPResponse` with status code, headers, body. |
| `Shutdown` | Once before the plugin process is killed. | Close any background goroutines (cancel a context). Flush logs. Don't try to call CoreAPI here — the gRPC connection is already torn down. |

### Minimal `main.go`

This is the shape every plugin starts with. Adapted from `extensions/forms/cmd/plugin/main.go`:

```go
package main

import (
    "context"

    goplugin "github.com/hashicorp/go-plugin"
    "google.golang.org/grpc"

    "vibecms/internal/coreapi"
    vibeplugin "vibecms/pkg/plugin"
    coreapipb "vibecms/pkg/plugin/coreapipb"
    pb "vibecms/pkg/plugin/proto"
)

type MyPlugin struct {
    host           coreapi.CoreAPI
    shutdownCancel context.CancelFunc
}

func (p *MyPlugin) Initialize(hostConn *grpc.ClientConn) error {
    p.host = coreapi.NewGRPCHostClient(coreapipb.NewVibeCMSHostClient(hostConn))
    ctx, cancel := context.WithCancel(context.Background())
    p.shutdownCancel = cancel
    p.startBackgroundWorker(ctx)  // optional
    return nil
}

func (p *MyPlugin) GetSubscriptions() ([]*pb.Subscription, error) {
    return []*pb.Subscription{
        {EventName: "node.published", Priority: 50},
        {EventName: "myext:render",   Priority: 0},  // event-with-result; see §8
    }, nil
}

func (p *MyPlugin) HandleEvent(action string, payload []byte) (*pb.EventResponse, error) {
    ctx := context.Background()
    switch action {
    case "node.published":
        return p.handleNodePublished(ctx, payload)
    case "myext:render":
        return p.handleRender(ctx, payload)
    }
    return &pb.EventResponse{Handled: false}, nil
}

func (p *MyPlugin) HandleHTTPRequest(req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
    return p.routeRequest(context.Background(), req)
}

func (p *MyPlugin) Shutdown() error {
    if p.shutdownCancel != nil {
        p.shutdownCancel()
    }
    return nil
}

func main() {
    goplugin.Serve(&goplugin.ServeConfig{
        HandshakeConfig: vibeplugin.Handshake,
        Plugins: map[string]goplugin.Plugin{
            "extension": &vibeplugin.ExtensionGRPCPlugin{Impl: &MyPlugin{}},
        },
        GRPCServer: goplugin.DefaultGRPCServer,
    })
}
```

The `media-manager` plugin uses a slightly different boot path (`VersionedPlugins` map instead of a flat `Plugins` map) — both work, the versioned form is preferred for new plugins so future protocol changes can coexist.

### The host client

`coreapi.NewGRPCHostClient(coreapipb.NewVibeCMSHostClient(hostConn))` is the magic that gives your plugin access to **everything** the kernel can do — content nodes, file storage, settings, email, events, the whole `CoreAPI` interface — over gRPC. Every method takes a `context.Context` first. Every method is capability-guarded.

The `host` field on your plugin struct is your single source of truth. Pass it everywhere; never make a second one.

### File layout for non-trivial plugins

`forms` is the canonical example. Its `cmd/plugin/` directory is split by concern:

```
cmd/plugin/
├── main.go                      # Plugin lifecycle. Tiny.
├── routes.go                    # routeRequest(): the HTTP dispatch tree.
├── handlers_forms.go            # /forms CRUD.
├── handlers_submissions.go      # /submissions CRUD + bulk + filters.
├── handlers_submit.go           # POST /forms/submit/{slug} (the public route).
├── handlers_export.go           # CSV download.
├── handlers_notifications.go    # Test-email endpoint.
├── handlers_preview.go          # Render-without-saving for the editor.
├── render.go                    # Form HTML rendering + post-processing.
├── notifications.go             # Async notification worker.
├── webhooks.go                  # Webhook fire + log.
├── events.go                    # Emit `forms:submitted` after a successful submission.
├── conditions.go                # The shared condition evaluator (server + client mirror).
├── validation.go                # Field-level submission validation.
├── files.go                     # Upload storage + filename sanitization.
├── captcha.go                   # reCAPTCHA / hCaptcha / Turnstile verification.
├── ratelimit.go                 # LRU per-IP token bucket.
├── retention.go                 # Background goroutine for GDPR retention.
├── forms.go                     # JSONB normalization helpers (DB ↔ Go shape).
├── helpers.go                   # jsonResponse / jsonError envelopes.
├── *_test.go                    # 240+ unit tests with FakeHost — no live DB needed.
├── fakehost_test.go             # Test double for CoreAPI used by every other test.
└── templates/
    ├── default_layout.html      # //go:embed-ed default form HTML.
    ├── grid_layout.html
    ├── card_layout.html
    └── inline_layout.html
```

Match this structure for any plugin that exceeds three or four handlers. **Production code stays under 300 lines per file (500 hard limit)** — split early.

### Unit testing without a database

`forms` ships a `FakeHost` test double in `cmd/plugin/fakehost_test.go` that implements `coreapi.CoreAPI` with in-memory maps. Every handler test injects this fake instead of a real gRPC client:

```go
func TestHandleSubmit(t *testing.T) {
    p := &FormsPlugin{host: &FakeHost{...}}
    resp, _ := p.handleSubmit(context.Background(), "contact", &pb.PluginHTTPRequest{...})
    // assert on resp
}
```

This is the recommended pattern. Don't spin up a real Postgres for unit tests — that's e2e territory.

---

## 7. HTTP routing — admin proxy vs public routes

Two completely separate proxies. Different mounts, different auth, same `HandleHTTPRequest` RPC.

### Admin proxy

**Mount path:** `/admin/api/ext/:slug/*`
**Auth:** kernel session middleware. Anonymous requests → `401`.
**Code:** `internal/cms/extension_proxy.go`

The admin proxy is auto-mounted for every active extension at `/admin/api/ext/{slug}/*`. The wildcard portion is what your plugin sees as `req.Path` (with a leading slash).

```
Browser  →  GET /admin/api/ext/forms/submissions?page=2
Kernel   →  Auth check (session cookie) → ExtensionProxy.handleRequest
Kernel   →  PluginHTTPRequest{Path: "/submissions", Method: "GET", QueryParams: {page: "2"}, UserId: 42}
Plugin   →  HandleHTTPRequest(req) → routeRequest(req) → handleSubmissions(req)
```

The proxy:
- Strips `Cookie` and `Authorization` headers before forwarding (no token leakage to plugins).
- Adds `X-User-Email` and `X-User-Name` headers if a user is logged in.
- Sets `req.UserId` to the logged-in user's ID, or `0` if anonymous.
- Sets `req.PathParams["slug"]` to the extension slug.
- Sets `req.PathParams["path"]` to the wildcard tail (mostly redundant with `req.Path`).

### Public proxy

**Mount path:** whatever you declare, verbatim.
**Auth:** none.
**Code:** `internal/cms/public_proxy.go`

```jsonc
"public_routes": [
  { "method": "POST", "path": "/forms/submit/*" },
  { "method": "GET",  "path": "/media/cache/*" }
]
```

After this manifest, your plugin's `HandleHTTPRequest` receives:
- `POST /forms/submit/contact`
- `GET /media/cache/medium/2026/03/photo.jpg`

`req.UserId` is always `0`. There is no auth, no session, no nothing — your plugin is the entire authentication boundary.

**The single most common bug** is assuming public routes are mounted under `/api/...`. They are not. The path you declare is the path users hit. The forms extension's submit endpoint is `/forms/submit/{slug}`, full stop.

### Routing inside the plugin

A plugin handles both proxies through the **same** `HandleHTTPRequest` RPC. `forms/cmd/plugin/routes.go` is the canonical pattern — strip known prefixes, then dispatch:

```go
func (p *FormsPlugin) routeRequest(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
    path := strings.TrimSuffix(req.GetPath(), "/")
    method := strings.ToUpper(req.GetMethod())

    // Tolerate either proxy: admin sends "/submissions"; public sends
    // "/forms/submit/contact". Strip both possible admin prefixes so paths
    // are uniform from here down.
    path = strings.TrimPrefix(path, "/admin/api/ext/forms")
    path = strings.TrimPrefix(path, "/api/ext/forms")
    path = strings.TrimPrefix(path, "/")
    path = strings.TrimSuffix(path, "/")

    // Public submit route
    if strings.HasPrefix(path, "forms/submit/") && method == "POST" {
        return p.handleSubmit(ctx, strings.TrimPrefix(path, "forms/submit/"), req)
    }

    // Admin routes (relative path from proxy)
    if path == "" && method == "GET"  { return p.handleListForms(ctx, req) }
    if path == "" && method == "POST" { return p.handleCreateForm(ctx, req) }
    // … etc
    return jsonError(404, "NOT_FOUND", "Route not found"), nil
}
```

### Response envelope

The shared `helpers.go` file in the forms plugin defines the canonical envelope. Adopt it:

```go
func jsonResponse(status int, data any) *pb.PluginHTTPResponse {
    body, _ := json.Marshal(data)
    return &pb.PluginHTTPResponse{
        StatusCode: int32(status),
        Headers:    map[string]string{"Content-Type": "application/json"},
        Body:       body,
    }
}

func jsonError(status int, code, message string) *pb.PluginHTTPResponse {
    return jsonResponse(status, map[string]string{
        "error":   code,
        "message": message,
    })
}
```

Every error response your plugin emits — admin or public — should use the shape `{"error": "<MACHINE_CODE>", "message": "<human text>"}`. For validation errors with per-field details, add `"fields": {"<field-id>": "<error>"}`. The bundled `vibe-form` client script and the admin UI both read `data.error` and `data.message` — match the shape.

---

## 8. Events: fire-and-forget vs event-with-result

Most events are fire-and-forget. A handful — the ones called from Go templates with `{{ event "name" ... }}` — collect results and inject them into the rendered HTML. Both modes share the same `HandleEvent` RPC.

### Fire-and-forget

Your plugin processes the event and returns nothing meaningful:

```go
func (p *MyPlugin) handleNodePublished(ctx context.Context, payload []byte) (*pb.EventResponse, error) {
    var data struct {
        NodeID    uint   `json:"node_id"`
        NodeTitle string `json:"node_title"`
        NodeType  string `json:"node_type"`
        Slug      string `json:"slug"`
    }
    json.Unmarshal(payload, &data)
    p.host.Log(ctx, "info", "node published: "+data.NodeTitle, nil)
    return &pb.EventResponse{Handled: true}, nil
}
```

The kernel fires this when a node is published. Your plugin reacts. No one waits for your response.

### Event-with-result (template injection)

Templates can call `{{ event "myext:render" (dict "key" "value") }}` and use whatever your plugin returns. The forms extension uses this for the `forms:render` event:

```go-template
{{ safeHTML (event "forms:render" (dict
  "form_id" "trip-order"
  "hidden"  (dict "trip_slug" .node.slug "trip_price" 45)
)) }}
```

Your plugin returns the rendered HTML in `EventResponse.Result`:

```go
func (p *FormsPlugin) handleRenderEvent(ctx context.Context, payload []byte) (*pb.EventResponse, error) {
    var data struct {
        FormID string         `json:"form_id"`
        Hidden map[string]any `json:"hidden"`
    }
    json.Unmarshal(payload, &data)

    form, err := p.lookupForm(ctx, data.FormID)
    if err != nil {
        return &pb.EventResponse{Handled: false, Error: err.Error()}, nil
    }
    html, err := p.renderFormHTML(form)
    if err != nil {
        return &pb.EventResponse{Handled: false, Error: err.Error()}, nil
    }
    if len(data.Hidden) > 0 {
        html = injectHiddenInputs(html, data.Hidden)
    }
    return &pb.EventResponse{Handled: true, Result: []byte(html)}, nil
}
```

The kernel concatenates `Result` from every plugin that subscribed (in priority order) and injects the combined string into the template. This is how:
- `forms` renders form HTML inside any block or layout.
- `email-manager` could render template previews.
- A search extension could inject a search box into a header partial.

Return `{Handled: false}` to opt out — the next plugin in the priority chain gets a turn. Return `{Handled: true, Result: nil}` to claim the event but contribute nothing.

### Subscribe priority

`GetSubscriptions()` returns each subscription with a `Priority` (lower = earlier):

```go
return []*pb.Subscription{
    {EventName: "forms:render", Priority: 0},   // runs first
    {EventName: "node.published", Priority: 50}, // default
}
```

For event-with-result events, all `Result` byte slices are concatenated in priority order. For fire-and-forget events, priority controls the dispatch order but the kernel doesn't aggregate results.

### Lifecycle events emitted by the kernel

These are the events your plugin can subscribe to. Payloads are JSON-encoded; the table shows the keys after `json.Unmarshal`.

| Event | Fires when | Payload keys |
|---|---|---|
| `node.created` | A node is created | `node_id`, `node_title`, `node_type`, `slug`, `language_code` |
| `node.updated` | A node's metadata or fields change | (same as `node.created`) |
| `node.published` | A node transitions to `status=published` | (same as `node.created`) |
| `node.deleted` | A node is deleted | `node_id`, `node_title`, `node_type` |
| `user.registered` | A new user signs up | `user_id`, `email`, `name` |
| `user.deleted` | A user is removed | `user_id`, `email` |
| `theme.activated` | Any theme activates (replayed for active theme on extension activation) | `name`, `path`, `version`, `assets[]` |
| `theme.deactivated` | A theme deactivates | `name` |
| `extension.activated` | Any extension activates | `slug`, `path`, `version`, `assets[]` |
| `extension.deactivated` | An extension deactivates | `slug` |
| `node_type.created` / `.updated` / `.deleted` | Custom post type registered/changed/removed | `slug` |
| `taxonomy.created` / `.updated` / `.deleted` | Taxonomy registered/changed/removed | `slug` |

**Replay semantics:** when an extension activates **after** a theme is already active, the kernel replays `theme.activated` for it. This guarantees that `media-manager` (which imports theme assets on `theme.activated`) doesn't miss the theme of an already-running site.

### Custom events emitted by extensions

Every extension can emit its own events with `host.Emit(ctx, "my-namespace:event", payload)`. Conventional namespacing is `<extension-slug>:<event>`, e.g. `forms:submitted`, `media:sizes_changed`.

**Forms emits:** `forms:submitted` — see `extensions/forms/cmd/plugin/events.go`. Payload keys: `form_id`, `form_slug`, `submission_id`, `data`, `metadata`.

**Forms also subscribes to:** `forms:upsert` (to let themes seed forms via Tengo), `forms:render` (event-with-result for template injection).

**Media-manager emits:** `media:sizes_changed` (so core can refresh its in-memory size registry).

**Media-manager subscribes to:** `theme.activated`, `theme.deactivated`, `extension.activated`, `extension.deactivated` (asset import/purge).

---

## 9. CoreAPI reference

The `coreapi.CoreAPI` interface is the entire surface area your plugin can access. Every method takes a `context.Context` first; every method is capability-guarded; every method has a corresponding Tengo binding under `core/<module>`.

The Go interface lives in `internal/coreapi/api.go`. Below is the practical groupings; for full signatures see the source.

### Nodes

| Method | Capability | What it does |
|---|---|---|
| `GetNode(ctx, id)` | `nodes:read` | Fetch a single node by ID |
| `QueryNodes(ctx, NodeQuery)` | `nodes:read` | Filter/search/paginate |
| `CreateNode(ctx, NodeInput)` | `nodes:write` | Create |
| `UpdateNode(ctx, id, NodeInput)` | `nodes:write` | Patch |
| `DeleteNode(ctx, id)` | `nodes:delete` | Delete |

`Node` carries `ID`, `Title`, `Slug`, `Status`, `NodeType`, `LanguageCode`, `BlocksData`, `FieldsData`, `SeoSettings`, `PublishedAt`, etc. `NodeQuery` supports `Status`, `NodeType`, `LanguageCode`, `Search`, `Page`, `PerPage`.

### Node Types & Taxonomies

| Method | Capability |
|---|---|
| `RegisterNodeType` / `GetNodeType` / `ListNodeTypes` / `UpdateNodeType` / `DeleteNodeType` | `nodetypes:read` / `nodetypes:write` |
| `RegisterTaxonomy` / `GetTaxonomy` / `ListTaxonomies` / `UpdateTaxonomy` / `DeleteTaxonomy` | `nodetypes:read` / `nodetypes:write` |
| `ListTerms` / `GetTerm` / `CreateTerm` / `UpdateTerm` / `DeleteTerm` | `nodes:read` / `nodes:write` |

### Settings

| Method | Capability |
|---|---|
| `GetSetting(ctx, key)` | `settings:read` |
| `GetSettings(ctx, prefix)` | `settings:read` |
| `SetSetting(ctx, key, value)` | `settings:write` |

Settings are key-value strings. Sensitive keys (passwords, API keys) declared in `settings_schema` with `"sensitive": true` are stored encrypted and never logged.

Convention for namespacing: `<extension-slug>:<dot.path>`, e.g. `media:optimizer:jpeg_quality`, `forms:retention_days`. The `email-` prefix is reserved by the email manager.

### Events

| Method | Capability |
|---|---|
| `Emit(ctx, action, payload)` | `events:emit` |
| `Subscribe(ctx, action, handler)` | `events:subscribe` (runtime only — `GetSubscriptions()` does not require this) |

### Email

| Method | Capability |
|---|---|
| `SendEmail(ctx, EmailRequest)` | `email:send` |

`EmailRequest{To []string, Cc []string, Bcc []string, Subject, HTML, Text, ReplyTo}`. The kernel routes through the email manager → provider extension chain.

### Menus

| Method | Capability |
|---|---|
| `GetMenu` / `GetMenus` | `menus:read` |
| `CreateMenu` / `UpdateMenu` / `UpsertMenu` | `menus:write` |
| `DeleteMenu` | `menus:delete` |

### Routes (runtime registration)

| Method | Capability |
|---|---|
| `RegisterRoute` / `RemoveRoute` | `routes:register` |

Used by Tengo scripts that want to add HTTP endpoints at runtime. `public_routes[]` in the manifest is **declarative** and does **not** flow through this API.

### Filters

| Method | Capability |
|---|---|
| `RegisterFilter(ctx, name, priority, handler)` | `filters:register` |
| `ApplyFilters(ctx, name, value)` | `filters:apply` |

Filters are WordPress-style transformation chains. Use them when you want extensions to be able to mutate values mid-render.

### Media

| Method | Capability |
|---|---|
| `UploadMedia(ctx, MediaUploadRequest)` | `media:write` |
| `GetMedia(ctx, id)` / `QueryMedia(ctx, MediaQuery)` | `media:read` |
| `DeleteMedia(ctx, id)` | `media:delete` |

Media calls go through the media-manager extension's plugin. If `media-manager` is deactivated, these calls error.

### Users

| Method | Capability |
|---|---|
| `GetUser(ctx, id)` / `QueryUsers(ctx, UserQuery)` | `users:read` |

Read-only. There is no `CreateUser` from CoreAPI — user provisioning is core-only.

### HTTP

| Method | Capability |
|---|---|
| `Fetch(ctx, FetchRequest)` | `http:fetch` |

`FetchRequest{Method, URL, Headers, Body, Timeout}`. Returns `FetchResponse{StatusCode, Body, Headers}`. The kernel applies a 30s hard cap; pass `Timeout` (in seconds) for shorter limits.

### Log

| Method | Capability |
|---|---|
| `Log(ctx, level, message, fields)` | `log:write` |

Levels: `"debug"`, `"info"`, `"warn"`, `"error"`. Output is prefixed with the extension slug and routed through the kernel's structured logger.

### Data Store (your own SQL tables)

| Method | Capability |
|---|---|
| `DataGet(ctx, table, id)` | `data:read` |
| `DataQuery(ctx, table, DataStoreQuery)` | `data:read` |
| `DataCreate(ctx, table, map)` | `data:write` |
| `DataUpdate(ctx, table, id, map)` | `data:write` |
| `DataDelete(ctx, table, id)` | `data:delete` |
| `DataExec(ctx, sql, args...)` | `data:write` |

`DataStoreQuery` supports:
- `Where` — `map[string]any` for equality conditions.
- `Raw` — string for `WHERE` SQL with `?` placeholders.
- `Args` — `[]any` for `Raw` placeholders.
- `OrderBy`, `Limit`, `Offset`.

`Where` and `Raw` can be combined — the engine `AND`s them. Stick to `?` placeholders for everything user-supplied; never string-concat.

### File Storage

| Method | Capability |
|---|---|
| `StoreFile(ctx, path, data)` | `files:write` |
| `DeleteFile(ctx, path)` | `files:delete` |

`StoreFile(ctx, "media/2026/03/photo.jpg", bytes)` returns the public URL the file is reachable at (e.g. `/media/2026/03/photo.jpg`). Paths are relative to the storage root (`STORAGE_DIR` env, defaults to `storage/`). Path traversal (`..`) is rejected by the host implementation.

---

## 10. SQL migrations

Each `.sql` file in `migrations/` runs **once**, alphabetically. Tracked in `extension_migrations(slug, filename, applied_at)`.

### Naming

```
migrations/
├── 20260101_init.sql
├── 20260201_add_status.sql
└── 20260301_webhook_logs.sql
```

Use a sortable date prefix. Files are applied in lexicographic order, so `20260101_*` runs before `20260201_*`. Once applied, the row in `extension_migrations` prevents re-running, so don't edit a migration after it's shipped — write a new one.

### Idempotency

Every migration **must** be safe to re-run. Use `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, `ALTER TABLE … ADD COLUMN IF NOT EXISTS`. The tracking table prevents re-application in normal flow, but idempotent SQL keeps you safe during development when you reset the DB.

### Conventions

- **Table prefix.** Prefix tables with the extension's domain — `forms`, `form_submissions`, `form_webhook_logs`, `media_files`, `media_image_sizes`. Avoid colliding with core tables (`content_nodes`, `users`, `settings`, `menus`, etc.).
- **Foreign keys.** Reference other extension tables explicitly: `form_id INTEGER NOT NULL REFERENCES forms(id) ON DELETE CASCADE`. Cascade is usually the right choice — orphan submissions are worse than missing ones.
- **JSONB for flexibility, columns for query.** Store flexible payloads (`fields_data`, `notifications`, `settings`) as JSONB. Promote anything you need to filter/index on (e.g. `status`) to its own column.
- **No down migrations.** The migration runner is forward-only. Comment the inverse SQL inline if you want documentation, but you can't actually roll back through the CMS.

### Example: forms initial migration

```sql
-- extensions/forms/migrations/20260422_init.sql

CREATE TABLE IF NOT EXISTS forms (
    id            SERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    slug          TEXT NOT NULL UNIQUE,
    fields        JSONB NOT NULL DEFAULT '[]',
    layout        TEXT NOT NULL DEFAULT '',
    notifications JSONB NOT NULL DEFAULT '[]',
    settings      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS form_submissions (
    id         SERIAL PRIMARY KEY,
    form_id    INTEGER NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    data       JSONB NOT NULL DEFAULT '{}',
    metadata   JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_forms_slug ON forms(slug);
CREATE INDEX IF NOT EXISTS idx_form_submissions_form_id ON form_submissions(form_id);
CREATE INDEX IF NOT EXISTS idx_form_submissions_created_at ON form_submissions(created_at);
```

### JSONB read shape

GORM returns JSONB columns as **strings** through the data store API, not as parsed objects. Every plugin that owns JSONB columns needs a normalization helper. From `extensions/forms/cmd/plugin/forms.go`:

```go
func normalizeForm(row map[string]any) map[string]any {
    return normalizeJSONBFields(row, "fields", "notifications", "settings")
}

func normalizeJSONBFields(row map[string]any, keys ...string) map[string]any {
    for _, key := range keys {
        s, ok := row[key].(string)
        if !ok || s == "" {
            continue
        }
        var parsed any
        if err := json.Unmarshal([]byte(s), &parsed); err == nil {
            row[key] = parsed
        }
    }
    return row
}
```

Call this before returning rows to the admin UI or before iterating in business logic. **This is the single most common source of "why does my admin UI iterate over a string character-by-character" bugs.**

---

## 11. Tengo scripts (`scripts/extension.tengo`)

Tengo is a small, sandboxed scripting language with Go-like syntax. Extensions can use it to hook into events, register routes, and seed data — without compiling Go code.

### When to use Tengo vs gRPC

| Decision factor | Tengo | gRPC plugin |
|---|---|---|
| Need owned database tables? | No (use core nodes/settings/data via CoreAPI) | Yes |
| Need persistent state? | No (fresh VM per call) | Yes |
| Need long-running goroutines? | No | Yes |
| Need to call native Go libraries? | No | Yes |
| Just react to events with HTTP calls / log output / setting writes? | **Yes** | overkill |
| Need request bodies > 1MB? | No (timeout/allocation limits) | Yes |
| Building a simple email provider? | Yes (`resend-provider` is the reference) | overkill |
| Building a media library? | No | Yes (`media-manager` is the reference) |

**Built-in extensions barely use Tengo today.** `media-manager` and `forms` ship a one-line `extension.tengo` that just logs activation. The pattern exists for third-party developers who want to ship logic without a Go toolchain — and for themes, which use Tengo extensively (see `themes/README.md` §8).

### Entry point

`scripts/extension.tengo` runs once at activation. Its job is to register handlers, filters, routes — not to render anything itself.

```tengo
// extensions/sitemap-generator/scripts/extension.tengo
log    := import("core/log")
events := import("core/events")
routes := import("core/routes")

log.info("Sitemap Generator extension loaded!")

events.on("node.published", "handlers/rebuild_sitemap", 10)
events.on("node.deleted",   "handlers/rebuild_sitemap", 10)

routes.register("GET", "/sitemap.xml",       "handlers/serve_index")
routes.register("GET", "/sitemap-:type.xml", "handlers/serve_type")
```

Handler scripts live under `scripts/handlers/<name>.tengo`. The path passed to `events.on` and `routes.register` is relative to `scripts/` and omits the `.tengo` suffix.

### Available `core/*` modules

Every Tengo binding has a CoreAPI counterpart. Capability checks apply.

| Module | Purpose |
|---|---|
| `core/nodes` | `get`, `query`, `create`, `update`, `delete`, `list_taxonomy_terms` |
| `core/nodetypes` | `register`, `get`, `list`, `update`, `delete` |
| `core/taxonomies` | `register`, `get`, `list`, `update`, `delete` |
| `core/menus` | `get`, `list`, `upsert`, `delete` |
| `core/settings` | `get`, `set`, `all` |
| `core/events` | `on(name, script, priority)`, `emit(name, payload)` |
| `core/routes` | `get(path, script)`, `post`, `put`, `patch`, `delete` |
| `core/filters` | `add(name, script, priority)` |
| `core/http` | `fetch(req)` (outbound) |
| `core/email` | `send`, `trigger` |
| `core/log` | `info`, `warn`, `error`, `debug` |
| `core/helpers` | `slugify`, `truncate`, `excerpt`, `escape_html`, `default`, … |
| `core/assets` | `read(path)`, `exists(path)` (sandboxed to extension root) |
| `core/wellknown` | `register("path", "handler-script")` for `.well-known/*` paths |

Full reference: `docs/scripting_api.md`. Theme authors and extension authors share most modules — the difference is `core/assets` reads relative to whichever directory invoked the script.

### Sandboxing

Every Tengo execution runs in a **fresh VM** with hard limits:

| Limit | Value |
|---|---|
| Max allocations | 50,000 per execution |
| Wall-clock timeout | 10 seconds |
| File access | none |
| Network access | none (use `core/http` for whitelisted outbound) |
| Process access | none |

Scripts that exceed limits are killed and an error is logged. Errors **never crash the kernel** — your extension keeps running, the failed script just gets skipped.

### Real-world Tengo extension: `resend-provider`

A complete email provider in 15 lines of Tengo:

```tengo
// extensions/resend-provider/scripts/extension.tengo
events := import("core/events")
log    := import("core/log")

log.info("Resend provider loaded")

events.on("email.send", "handlers/send_via_resend")
```

```tengo
// extensions/resend-provider/scripts/handlers/send_via_resend.tengo
http     := import("core/http")
settings := import("core/settings")
log      := import("core/log")
json     := import("json")

api_key := settings.get("resend_api_key")
if api_key == "" || is_error(api_key) {
    log.warn("Resend API key not configured")
    return
}

body := json.encode({
    from:    event.payload.from,
    to:      event.payload.to,
    subject: event.payload.subject,
    html:    event.payload.html
})

res := http.fetch({
    method:  "POST",
    url:     "https://api.resend.com/emails",
    headers: { "Authorization": "Bearer " + api_key, "Content-Type": "application/json" },
    body:    body,
    timeout: 10
})

if res.status_code >= 400 {
    log.error("Resend send failed: " + string(res.status_code))
}
```

No Go binary, no admin UI of its own — just `extension.json` + Tengo. The provider injects its API-key form into the `email-settings` slot via the manifest's `slots` map.

---

## 12. Admin UI — micro-frontend authoring

Extension admin UIs are **isolated Vite builds** that output a single ES module. The admin SPA shell loads them on demand via dynamic import; shared dependencies are injected through an import map.

### Build setup

```bash
cd extensions/my-extension/admin-ui
# Create the project (one-time)
npm create vite@latest . -- --template react-ts
npm install
# Add Tailwind v4
npm install -D tailwindcss @tailwindcss/vite
```

### `package.json`

Minimal — only devDeps. React, Sonner, the design system, and the icon set come from the host.

```json
{
  "name": "my-extension-admin-ui",
  "private": true,
  "type": "module",
  "scripts": { "build": "vite build" },
  "devDependencies": {
    "@tailwindcss/vite": "^4.2.4",
    "@types/react": "^19.0.0",
    "@vitejs/plugin-react": "^4.3.0",
    "tailwindcss": "^4.2.4",
    "typescript": "^5.6.0",
    "vite": "^6.0.0"
  }
}
```

### `vite.config.ts`

Externalize **everything** the host provides. The list below is the canonical set — copy it as-is.

```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  define: {
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
  build: {
    outDir: "dist",
    lib: {
      entry: "src/index.tsx",
      formats: ["es"],
      fileName: "index",
      cssFileName: "index",          // emits dist/index.css next to dist/index.js
    },
    rollupOptions: {
      external: [
        "react", "react/jsx-runtime",
        "react-dom", "react-dom/client",
        "react-router-dom",
        "sonner",
        "@vibecms/ui", "@vibecms/api", "@vibecms/icons",
      ],
    },
    cssCodeSplit: false,
  },
});
```

### `src/index.tsx`

Two duties: import the Tailwind CSS exactly once, then re-export every component referenced by the manifest's `routes[]`, `slots[]`, or `field_types[]`.

```tsx
import "./index.css";
export { default as MediaLibrary }            from "./MediaLibrary";
export { default as MediaFieldInput }         from "./MediaFieldInput";
export { default as MediaPickerModal }        from "./MediaPickerModal";
export { default as ImageOptimizerSettings }  from "./ImageOptimizerSettings";
```

### `src/index.css`

```css
@import "tailwindcss";
@source "./**/*.{ts,tsx}";
```

That's the entire file. Design tokens, base styles, and `@vibecms/ui` overrides come from the admin shell's own stylesheet — your extension only needs the utility classes it actually uses.

> **Why per-extension Tailwind builds.** The admin shell's stylesheet declares a fallback `@source "../../extensions/*/admin-ui/src/**"` so simple class usage works during local development. **Inside Docker, only `admin-ui/` is copied during the frontend build stage** — the fallback `@source` finds nothing, so any class you didn't compile per-extension silently disappears. Always ship your own CSS.

### Where shared dependencies come from

The admin shell exposes shared modules via `window.__VIBECMS_SHARED__` and an import map. Your code imports them as if they were normal NPM packages:

```tsx
// These resolve through the import map; no shim access needed.
import { Button, Card, Input } from "@vibecms/ui";
import { Upload, Image, Trash2 } from "@vibecms/icons";
import { toast } from "sonner";

// These don't have a typed package wrapper — pull them from the shim.
const { useSearchParams, useNavigate } =
  (window as unknown as { __VIBECMS_SHARED__: any }).__VIBECMS_SHARED__.ReactRouterDOM;
```

The full list:

| Import | What it is | Notes |
|---|---|---|
| `react`, `react-dom`, `react/jsx-runtime` | React 19 | Pinned by the shell |
| `react-router-dom` | Routing primitives | Access via shim — no typed re-export |
| `sonner` | Toast library | `import { toast } from "sonner"` works |
| `@vibecms/ui` | The design system primitives — see below | Direct import works |
| `@vibecms/icons` | Lucide icon components, lazy-loaded | `import { Upload } from "@vibecms/icons"` |
| `@vibecms/api` | API client helpers (admin auth, fetch wrappers) | Direct import |

**Reference for `media-manager`** (`MediaLibrary.tsx`) — uses both styles:

```tsx
import { Button, Dialog, DialogContent, DialogTitle, /* … */ } from "@vibecms/ui";
import { Upload, Loader2, Image as ImageIcon } from "@vibecms/icons";
import { toast } from "sonner";

const SHARED = (window as any).__VIBECMS_SHARED__;
const { useSearchParams } = SHARED.ReactRouterDOM;
const { ListPageShell, ListHeader, ListSearch, ListFooter } = SHARED.ui;
```

The `__VIBECMS_SHARED__.ui` namespace exposes the same components as `@vibecms/ui` (it's the same module), but accessing it via the shim is convenient when destructuring many primitives at once and saves re-listing them in the externalize array. **Both styles work; pick one and be consistent within a file.**

### The design system primitives

`@vibecms/ui` ships the full shadcn-derived component library plus list-page primitives that the entire CMS is built on. Reach for these before rolling your own — they're what makes pages look like the rest of the admin.

**Layout primitives:**
- `ListPageShell` — outer wrapper (handles padding, max-width).
- `ListHeader` — top tabs + extras. Accepts `tabs={[{value, label, count}]}`, `activeTab`, `onTabChange`, `extra`.
- `ListSearch` — debounced search input.
- `ListFooter` — pagination + per-page picker.
- `ListToolbar` — generic toolbar row.
- `EmptyState`, `LoadingRow` — placeholder views.
- `ListTable`, `Th`, `Tr`, `Td` — table primitives.
- `ListCard` — card-grid item.
- `Chip`, `StatusPill`, `TitleCell`, `RowActions` — common cells.
- `SectionHeader`, `Card`, `CardContent`, `Separator` — page sections.
- `AccordionRow`, `CodeWindow` — disclosure / preview primitives.

**Form primitives:** `Button`, `Input`, `Label`, `Select` (`SelectContent`, `SelectItem`, `SelectTrigger`, `SelectValue`), `Switch`, `Checkbox`, `Textarea`, `Tabs` (`TabsList`, `TabsTrigger`, `TabsContent`), `Dialog` (`DialogContent`, `DialogHeader`, `DialogTitle`, `DialogDescription`, `DialogFooter`), `Popover` (`PopoverContent`, `PopoverTrigger`), `Badge`.

The reference implementation of the list-page pattern is `extensions/media-manager/admin-ui/src/MediaLibrary.tsx` — URL-synced filters, view-mode switcher, sort dropdown + sortable column headers, per-tab counts via parallel fetches.

### List page pattern

```tsx
<ListPageShell>
  <ListHeader
    tabs={[
      { value: "all",   label: "All",    count: totalCount },
      { value: "image", label: "Images", count: imageCount },
      // …
    ]}
    activeTab={activeTab}
    onTabChange={setActiveTab}
    extra={<UploadButton />}
  />

  <div className="flex items-center gap-2 mb-2.5 flex-wrap">
    <ListSearch value={search} onChange={setSearch} placeholder="Search…" />
    {/* view / sort / density controls */}
  </div>

  {/* Grid or table */}

  <ListFooter
    page={page} totalPages={totalPages} total={meta.total}
    perPage={perPage} onPage={setPage} onPerPage={setPerPage}
    label="files"
  />
</ListPageShell>
```

**Conventions:**
- **Drop the `<h1>page-title</h1>`** — the active tab pill *is* the title.
- **Tabs replace separate type/status filter dropdowns.** `?type=image` lives in the tab state.
- **All filter / sort / view / pagination state goes in URL search params.** Default values omit the param (`?view=grid` is implicit). Refresh preserves state. Use `replace: true` for search-input keystrokes so they don't pollute history. Use `resetPage: true` when changing filters so users don't strand on page 7 of a 2-page result.
- **Tab counts:** if no aggregate-counts endpoint exists, fire one parallel `per_page=1` fetch per tab on mount (and after any mutation). Cheap and lets you skip backend work for v1.
- **Sortable column headers + sort dropdown share `?sort=`.** Both controls write to the same URL state — clicking a column header updates the dropdown selection, and vice versa.

### Editor page pattern

For any "edit X" page (form editor, node editor, settings page), match the layout used by `admin-ui/src/pages/node-editor.tsx`:

```tsx
<div className="space-y-4">
  <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
    <div className="space-y-4 min-w-0">
      {/* 1. Compact pill header: ArrowLeft + Title input + / + Slug input */}
      {/* 2. Tabs (rounded-xl bg-slate-100 p-1, white-on-active) */}
      {/* 3. <TabsContent> for each tab */}
    </div>
    <div className="space-y-4">
      {/* Sidebar: Publish card with Save / Cancel, optional Actions card */}
    </div>
  </div>
</div>
```

**Rules:**
- The pill header goes **inside** the left column, not full-width above the grid.
- Use `lg:grid-cols-[minmax(0,1fr)_320px]` (fluid main + fixed sidebar). Don't use 2/3 + 1/3 grids.
- `min-w-0` on the main column or long content overflows the grid.
- Listing pages always show an "All (N)" tab via `ListHeader`'s `tabs` prop — even when there's only one filter — to match the rest of the CMS.

### CSS load order — do not break this

The extension loader **prepends** your `<link rel="stylesheet">` tag before `admin-ui`'s stylesheet in `<head>`. Both sheets put utilities in the same `@layer utilities`, and within a merged layer **source order wins**. If your stylesheet loaded later, your `.fixed` (used by drawers/modals) would beat admin-ui's `.lg:relative` on `<aside class="fixed ... lg:relative">` and the desktop sidebar would stay `position: fixed` on every admin page. The whole shell layout would collapse.

The loader handles this for you. Do **not** modify `admin-ui/src/lib/extension-loader.ts` to use `appendChild` for stylesheets without understanding this trade-off.

### Hot-deploy without rebuilding the Docker image

For tight iteration during development, copy built assets directly into the running container:

```bash
# After npm run build in both admin-ui and your extension
docker cp admin-ui/dist/. vibecms-app-1:/app/admin-ui/dist/
docker cp extensions/<slug>/admin-ui/dist/. vibecms-app-1:/app/extensions/<slug>/admin-ui/dist/
```

The Go binary serves these as static files — no container restart needed. Hard-refresh the browser (Cmd+Shift+R) to bypass cached `index.html`.

### Things that bite

| Symptom | Real cause | Fix |
|---|---|---|
| Tailwind class has no effect after Docker build (e.g. `gap-x-6`, `pt-3`, `md:col-span-2`) | Per-extension Tailwind not configured; the admin shell's fallback `@source` doesn't reach into the extension during the Docker frontend build | Add `@tailwindcss/vite` and `cssFileName: "index"`. Verify `dist/index.css` contains `.gap-x-6{...}`. |
| Layout fine in `npm run dev`, broken after `docker build --no-cache` | Same root cause — extension hasn't shipped its own CSS | See above. |
| Inline `style={{padding: 12}}` "fixes" | Patching around a missing Tailwind class — bug rots in place | Make Tailwind scan correctly. Inline styles only for dynamic CSS-variable interpolation (`var(--accent)`). |
| Switches invisible (white on white) when checked | Old shadcn defaults — `data-[state=checked]:bg-slate-900` looked like "off" | Already fixed CMS-wide in `admin-ui/src/components/ui/switch.tsx` (indigo). If you ever touch that file, rebuild admin-ui too. |
| `<Select>` doesn't fill its row like `<Input>` does | shadcn default `w-fit` on `SelectTrigger` | Already fixed CMS-wide to `w-full`. Don't reintroduce `w-fit`. |
| Tabs / clickable controls without pointer cursor | Missing `cursor-pointer` | Add it. Every clickable thing needs it; this is a recurring review note. |
| "Open public form" links on items that have no public URL | Don't link to a public endpoint unless the extension actually serves a public route | Check `public_routes` in the manifest before linking. |
| Sidebar covers main content on every admin page after activating a new extension | Extension stylesheet was injected *after* admin-ui's, putting `.fixed` later in the merged layer cascade | The loader prepends extension `<link>` tags. Don't change to `appendChild`. |

---

## 13. Custom field types

Extensions register custom field types so node-type schemas (and theme blocks) can include first-class custom inputs. The `media` field type is contributed by `media-manager`; the `form_selector` field type is contributed by `forms`.

### Manifest declaration

```jsonc
"admin_ui": {
  "field_types": [
    {
      "type":        "form_selector",
      "label":       "Form Selector",
      "description": "Select a form from the Forms extension",
      "icon":        "ClipboardList",
      "group":       "Forms",
      "component":   "FormFieldSelector",
      "supports":    []
    }
  ]
}
```

| Field | Purpose |
|---|---|
| `type` | The string used in `field_schema[].type`. Must be unique across all activated extensions. |
| `label`, `description`, `icon`, `group` | Cosmetic — shown in the field type picker when adding a field to a node type. |
| `component` | Named export from your `index.tsx`. The node editor renders this for any field with this `type`. |
| `supports` | Optional array of variants. The `media` field type registers `supports: ["image", "gallery", "file"]` so block schemas can use `type: "image"`, `type: "gallery"`, or `type: "file"` and dispatch to the same component. |

### Component contract

The node editor renders your component with three props:

```tsx
interface FieldComponentProps {
  field: {
    key:           string;     // The field's identifier (snake_case)
    label:         string;     // Editor label
    type:          string;     // Your registered type, or one of `supports[]`
    required?:     boolean;
    placeholder?:  string;
    help_text?:    string;
    multiple?:     boolean;
    allowed_types?: string;
    // …plus any custom keys from the field schema
  };
  value:    unknown;            // Current value; shape is up to you
  onChange: (val: unknown) => void;
}
```

Your component is responsible for:
- Rendering whatever editor UI makes sense.
- Validating internally if you want — but you can't block submit; the parent owns submit.
- Calling `onChange(newValue)` whenever the value changes.

The reference implementation is `extensions/media-manager/admin-ui/src/MediaFieldInput.tsx`. It handles single-image (`type: "image"`), gallery (`type: "gallery"`), and arbitrary-file (`type: "file"`) modes from one component, dispatching on `field.type` and `field.multiple`.

### Theme integration

Once registered, field types become available everywhere a `field_schema[]` is parsed — block schemas, theme node types, extension migrations seeding custom node types, etc.:

```jsonc
// In a block.json
"field_schema": [
  { "key": "hero_image", "type": "image" },
  { "key": "gallery",    "type": "gallery" },
  { "key": "order_form", "type": "form_selector" }
]
```

The block editor sees these fields and renders the registered component. No extra wiring per block.

---

## 14. Content blocks owned by extensions

Extensions can ship blocks the same way themes do — `extension.json` declares `blocks[]`, each block has its own directory under `blocks/`, and the extension loader registers them on activation. The schema is identical to theme blocks (see `themes/README.md` §5 for the full recipe).

### Manifest declaration

```jsonc
"blocks": [
  { "slug": "vibe-form", "dir": "vibe-form" }
]
```

### Block directory

```
blocks/vibe-form/
├── block.json     # Schema + test_data (same shape as theme blocks)
├── view.html      # Go template; receives the block's fields at root
├── style.css      # Optional — auto-injected on pages that use the block
└── script.js      # Optional — auto-injected on pages that use the block
```

### When extensions own blocks

- The block's behavior depends on the extension's runtime. `forms`'s `vibe-form` block uses `{{ event "forms:render" ... }}` to delegate rendering to the gRPC plugin.
- The block reads from a database table the extension owns. A blog-archive block would ship with a blog extension.
- The block needs custom field types the extension provides. `vibe-form` uses `type: "form_selector"`.

### When themes own blocks

- Pure presentation, no extension dependency. Most blocks fall here.
- Theme-specific design with no reuse value across themes.

The `content-blocks` extension is a hybrid — it ships a curated library of 40 framework-agnostic blocks (text, media, CTA, features, pricing, layout) that any theme can pick up. Treat it as a building set rather than a "theme blocks vs extension blocks" example.

### The `vibe-form` block reference

```html
{{/* extensions/forms/blocks/vibe-form/view.html */}}
{{ $slug := .form_slug }}{{ if not $slug }}{{ $slug = .form_id }}{{ end }}
{{ if $slug }}
  <div class="vibe-form-wrapper" data-form-slug="{{ $slug }}" id="form-{{ $slug }}">
    {{ if .heading }}
      <h2 class="form-heading">{{ .heading }}</h2>
    {{ end }}

    {{ $rendered := event "forms:render" (dict "form_id" $slug) }}
    {{ if $rendered }}
      {{ safeHTML $rendered }}
    {{ else }}
      <div class="form-error">Failed to load form.</div>
    {{ end }}
  </div>

  <link rel="stylesheet" href="/extensions/forms/blocks/vibe-form/style.css">
  <script src="/extensions/forms/blocks/vibe-form/script.js" defer></script>
{{ end }}
```

The block:
1. Reads its own field (`.form_slug`) from the block's data.
2. Calls `{{ event "forms:render" ... }}` — the result-collecting event handled by the forms plugin.
3. Wraps the rendered HTML with the block's heading and a sibling `<link>`/`<script>` so the form's CSS and JS load on the public page.

This is the canonical pattern for "block delegates rendering to a plugin." Reuse it.

---

## 15. Settings schema and the `email-settings` slot pattern

### `settings_schema`

Declared in the manifest, rendered automatically as a form by the admin shell:

```jsonc
"settings_schema": {
  "host":     { "type": "string", "label": "SMTP Host", "required": true },
  "port":     { "type": "number", "label": "SMTP Port", "default": 587 },
  "password": { "type": "string", "label": "Password",  "sensitive": true },
  "encryption": {
    "type": "string", "label": "Encryption",
    "enum": ["none", "tls", "starttls"], "default": "tls"
  }
}
```

Supported field types: `"string"`, `"number"`, `"boolean"`. Use `"sensitive": true` for passwords and API keys — the admin form masks the value and the kernel encrypts it at rest.

Settings are stored under the key prefix `<extension-slug>:`, accessed via `host.GetSetting(ctx, "smtp-provider:host")`. Convention is to namespace deeper for subsystems within an extension: `media:optimizer:jpeg_quality`.

### The slot pattern

When an extension wants to surface settings **inside another extension's UI**, it injects a component into a named slot:

```jsonc
// extensions/smtp-provider/extension.json
"admin_ui": {
  "slots": {
    "email-settings": { "component": "SmtpSettings", "label": "SMTP" }
  }
}
```

The email-manager admin UI exposes an `email-settings` slot. When the email-manager UI loads, it iterates every active extension, finds the ones contributing to this slot, and renders their components in a tabbed Card. SMTP, Resend, Postmark, and any future provider plug into the same place without email-manager knowing they exist.

The rendered slot component receives no props — it manages its own state by reading/writing settings via the standard fetch pattern:

```tsx
// extensions/smtp-provider/admin-ui/src/SmtpSettings.tsx
export default function SmtpSettings() {
  const [settings, setSettings] = useState<...>(...);

  useEffect(() => {
    fetch("/admin/api/ext/smtp-provider/settings").then(/* … */);
  }, []);

  // …form rendering, save handler
}
```

To declare a slot **for** other extensions to inject into, render a `<SlotHost name="my-slot" />` in your admin UI. The shell scans every active extension's `slots` map and renders matching components inside.

---

## 16. Asset references and the media-manager handshake

Extensions and themes both declare media assets in their manifests under `assets[]`. On activation, the media-manager extension imports those files into the media library and tags them with their owner.

### Declaring assets

```jsonc
// In extension.json
"assets": [
  { "key": "demo-banner",    "src": "images/demo-banner.jpg",    "alt": "Hello extension demo banner" },
  { "key": "feature-mascot", "src": "images/feature-mascot.png", "alt": "Mascot" }
]
```

`key` must match `^[a-z0-9_-]+$`. `src` is relative to the extension directory.

### How import works

1. Extension activates → kernel emits `extension.activated` with payload `{slug, path, version, assets[]}` (each asset entry has `key`, `src`, `abs_path`, `alt`, `width`, `height`).
2. `media-manager` (subscribed to `extension.activated` at priority 50) reads each `abs_path`, computes a SHA-256 content hash, detects MIME, and either:
   - **Inserts** a new row into `media_files` (`source='extension', extension_slug=<slug>, asset_key=<key>, content_hash=<hash>`).
   - **Updates** the existing row if `content_hash` changed.
   - **Skips** if `content_hash` matches (idempotent re-activation).
3. Asset keys present in the database but missing from the new manifest are **purged** — file deleted, row deleted. (See `extensions/media-manager/cmd/plugin/main.go::handleOwnedAssetsActivated`.)
4. On `extension.deactivated`, every row tagged with this extension's slug is deleted along with its underlying file.

### Referencing imported assets

Use the `extension-asset:<slug>:<key>` URI scheme in templates, seed data, and block test data:

```jsonc
// In a block.json's test_data
"hero_image": { "url": "extension-asset:hello-extension:demo-banner", "alt": "Hello banner" }
```

```tengo
// In an extension's Tengo seed
nodes.create({
    blocks_data: [
        { type: "demo-block", fields: {
            image: { url: "extension-asset:my-ext:hero", alt: "Hero" }
        } }
    ]
})
```

The kernel's render pipeline walks the JSON tree and replaces `extension-asset:<slug>:<key>` strings with the resolved URL (`/media/extension/<slug>/<key>.<ext>`) before the data hits a `view.html`. Templates always see a real URL; the indirection survives extension switches.

**Theme equivalent** is `theme-asset:<key>` — same mechanism, different namespace. Themes don't need a slug prefix because only one theme is active at a time.

### Why use the asset reference scheme

- **Survives the file moving.** Media-manager controls the storage path; if it ever changes the layout, references update automatically.
- **Survives WebP conversion.** The media-manager auto-WebP routes (see §22) live behind the `/media/...` URL space, not the extension's storage path.
- **Survives deactivation/reactivation.** The content hash makes re-activation idempotent.
- **Doesn't survive deleting the extension.** That's the desired behavior — when you uninstall an extension, its demo content goes with it.

### Image transformations

Once a file is in `/media/...`, you can serve resized variants through the public cache route:

```html
<img
  src="{{ image_url $.hero_image.url "medium" }}"
  srcset="{{ image_srcset $.hero_image.url "small" "medium" "large" }}"
  sizes="(max-width: 600px) 100vw, 50vw"
  alt="{{ $.hero_image.alt }}">
```

Sizes (`small`, `medium`, `large`, `thumbnail`) live in the `media_image_sizes` table and are configurable from `/admin/ext/media-manager/optimizer`. The cache route `/media/cache/<size>/<path>` is registered as a public route by `media-manager` and resizes on-demand.

---

## 17. Lifecycle: scan → activate → run → deactivate

The full state machine an extension goes through.

### Scan

On every kernel start, `ExtensionLoader.ScanAndRegister()`:
1. Reads every `extensions/*/extension.json`.
2. Upserts a row in the `extensions` table (`slug`, `name`, `version`, `description`, `author`, `path`, `priority`, `manifest`).
3. New extensions default to `is_active=false` (or `true` for built-ins listed in `builtinActiveExtensions`).
4. Existing rows have their metadata refreshed but **`is_active` is preserved** — manual user choices survive across restarts.

### Activate (one-time per extension lifetime)

`ExtensionLoader.Activate(slug)` flips `is_active=true` and reloads the active set. The `Reload` hook in plugin manager:
1. **Migrations.** Every `migrations/*.sql` file is applied in alphabetical order. Already-applied files (per `extension_migrations`) are skipped.
2. **Tengo scripts.** `scripts/extension.tengo` is loaded and executed once. Every `events.on`, `routes.register`, `filters.add` registers handlers for the lifetime of the active session.
3. **gRPC plugins.** Each binary in `plugins[]` is started via HashiCorp `go-plugin`, dispensed as an `ExtensionPlugin`, initialized with a CoreAPI host connection, and queried for `GetSubscriptions()`.
4. **Block / template / layout / partial registration.** `extension_loader.go::loadExtensionBlocks` reads each `blocks[]`, `templates[]`, `layouts[]`, `partials[]` entry and upserts the corresponding row. `theme_name` column is set to the extension slug; `source` is `"extension"`.
5. **Public route mounting.** Every entry in `public_routes[]` is registered on the public Fiber app.
6. **Admin route mounting.** A wildcard route at `/admin/api/ext/{slug}/*` is mounted (always — the admin proxy is per-extension automatic).
7. **`extension.activated` event** fires with `{slug, path, version, assets}`. Other extensions react.
8. **Theme replay.** `theme.activated` is replayed for the currently active theme so the just-activated extension can see it.

### Run

The plugin process is alive and handling RPCs. The kernel:
- Forwards admin/public HTTP requests to `HandleHTTPRequest`.
- Forwards subscribed events to `HandleEvent`.
- Routes CoreAPI calls back into the kernel through the bidirectional gRPC stream.

### Deactivate

`ExtensionLoader.Deactivate(slug)` flips `is_active=false`:
1. **`extension.deactivated` event** fires with `{slug}`. (Other extensions can clean up — `media-manager` purges asset rows, etc.)
2. **Tengo unload.** All registered event handlers, routes, filters tied to this extension are removed.
3. **Plugin shutdown.** The plugin's `Shutdown()` is called, the gRPC connection closes, and the child process is killed.
4. **Block / template / layout / partial unregister.** Rows with `source='extension' AND theme_name=<slug>` are deleted.
5. **Public routes:** the kernel doesn't currently unmount Fiber routes — they remain registered but the proxy returns `503` because `pluginMgr.GetClient(slug)` returns nil.
6. **Admin routes:** same — `/admin/api/ext/{slug}/*` returns `404 extension not found or not running`.

### Reactivation

Repeating `Activate(slug)` runs through the activate sequence again. **Migrations are idempotent** (tracked), block/template/etc. registrations are idempotent (upserts), Tengo scripts run again, plugins boot fresh.

### Crash recovery

If a plugin process dies unexpectedly:
- HashiCorp go-plugin logs the failure.
- Subsequent admin/public requests return `503`.
- Subsequent events to `HandleEvent` are dropped.
- The kernel does **not** automatically restart the plugin; restarting the app reactivates the extension cleanly.

For long-running production sites, treat plugin crashes the way you'd treat any worker crash — alert, investigate, redeploy.

---

## 18. The Mandalorian rules

Twelve rules. The difference between "it works on my machine" and "it ships clean from a cold boot".

1. **The manifest is the contract.** Every binary, route, capability, block, custom field type, and admin UI route the extension wires up is declared in `extension.json`. If it's not there, the kernel can't see it.
2. **Capabilities are minimal.** Declare only what your code calls. Adding a capability is cheap; explaining `data:write` on a read-only extension is not. Reviewers will catch over-declared capabilities.
3. **`jsonError({"error": code, "message": text})` is the public error envelope.** Every error response — admin or public — uses this shape. The client code (admin UI, vibe-form script, third-party callers) reads `data.error` for the machine code and `data.message` for the human text.
4. **Public routes are mounted at the path you declare.** Not under `/api/...`. Not under `/admin/...`. The path you write in `public_routes[].path` is the path users hit. Test at the real URL, not what you wish it was.
5. **Tables are prefixed by the extension's domain.** `forms`, `form_submissions`, `media_files`, `media_image_sizes`, `form_webhook_logs`. Never collide with core tables.
6. **JSONB columns come back as strings** through the data store. Always normalize before iterating. The forms extension's `normalizeForm` / `normalizeSubmission` is the canonical pattern.
7. **Asset references survive theme/extension switches; hardcoded URLs don't.** Use `extension-asset:<slug>:<key>` and `theme-asset:<key>` everywhere. Never hardcode `/media/extension/...` paths in templates or seed data.
8. **Per-extension Tailwind builds are mandatory for Docker.** The admin shell's fallback `@source` works in dev but disappears in the Docker frontend stage. Every admin UI ships its own `dist/index.css`.
9. **Use the design system primitives.** `ListPageShell`, `ListHeader`, `ListSearch`, `ListFooter`, `EmptyState`, `Chip`, `StatusPill`, `RowActions`, etc. Reach for them before rolling your own. The CMS visual consistency depends on it.
10. **All filter / sort / view / pagination state lives in URL params.** Refresh preserves it. Default values omit the param. Use `replace: true` for keystrokes; use `resetPage: true` when changing filters.
11. **Production code under 300 lines per file (500 hard limit).** Test files are exempt. Split early — `forms/cmd/plugin/` is the model. The single biggest source of regression bugs is "I'll refactor the 1500-line file later".
12. **Don't reach for a slot pattern when an event-with-result will do.** Slots couple admin UIs at build time; events couple at runtime. Email providers inject into the `email-settings` slot because they're configuration UIs. Form rendering uses `event "forms:render"` because the consumer is a Go template, not a React component. Pick the looser coupling.

**This is the way.**

---

## 19. Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Extension doesn't appear in the admin extension list | `extension.json` missing or malformed; or the directory is hidden | `docker compose logs app | grep '\[extensions\] scanned'` to confirm scan count. Validate JSON with `jq < extension.json`. |
| Extension appears but won't activate | Migration error, plugin compile error, or capability denial | Check `docker compose logs app | grep -E '\[extensions\]\|\[plugins\]'` for the activation step. |
| `403` / capability denied at runtime | Missing entry in `capabilities[]` | Check the capability table (§5). Add the missing entry, restart, reactivate. |
| Plugin starts but doesn't receive events | `GetSubscriptions()` returns empty, or event name mismatch | Add a `log.info("subscribed to X")` in your `Initialize`. Verify the emitter uses the same name. |
| Public route returns `503 Service Unavailable` | Plugin process not running | `docker compose logs app | grep '<your-slug>'`. The plugin may have crashed at boot — fix and restart. |
| Public route returns `404 Not Found` | Path declared incorrectly, or the extension is inactive | Verify `public_routes[].path` matches the URL exactly. Check `is_active=true` in the DB. |
| Admin route returns `404 extension not found or not running` | Extension is inactive | Activate via the admin UI, or `UPDATE extensions SET is_active=true WHERE slug='<your>'` and restart. |
| Admin UI loads JS but no styles | Per-extension Tailwind not configured, or `dist/index.css` missing | Add `@tailwindcss/vite` and `cssFileName: "index"` to `vite.config.ts`. Confirm `dist/index.css` exists with utility classes. |
| Tab counts don't update after a mutation | Forgot to re-fire the per-tab count fetches | Move the per-tab count effect's deps to include the mutation source. The reference is `MediaLibrary.tsx`'s `tabCounts` effect. |
| `[object Object]` rendered in admin UI | JSONB column not normalized; admin UI iterates the raw JSON string | Add the column to your `normalizeJSONBFields` call. |
| Form submits silently disappear | Theme rendered a raw `<form>` instead of using `event "forms:render"` | Use the forms-extension handshake (see `themes/README.md` §10). |
| Block renders empty on the public page | Required field missing in `blocks_data`, or `{{with}}` gate hides everything | Open the page in the admin block editor; verify every field is populated. |
| `theme-asset:<key>` shows up as `#ZgotmplZ` in HTML | Asset ref didn't resolve (typo, key not declared) | Confirm `assets[]` entry; check media-manager imported the file. |
| `extension-asset:<slug>:<key>` not resolving | Same — extension may not have been re-activated since the asset was added; or the key has invalid characters | Restart the app to re-trigger asset import. Validate `key` against `^[a-z0-9_-]+$`. |
| Plugin compiled fine but won't load | Mismatched protocol version, or built with `CGO_ENABLED=1` | Rebuild with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`. Check that `pkg/plugin/proto/*.go` is up to date with the kernel. |
| Settings form rejects values | Missing field in `settings_schema`, or `"sensitive": true` mismatch | Settings keys outside the schema are accepted by the API but won't render in the auto-form. |
| Slot component doesn't appear in target extension | Target extension doesn't expose that slot, or your `slots` map uses the wrong key | Search the target extension's admin UI for `<SlotHost name="..."/>` to confirm slot names. |
| `ALREADY_RUNNING` 409 from a long-running operation | A previous bulk job is still in progress | Poll the progress endpoint until `running: false`. Reference: `media-manager`'s `/optimizer/reoptimize-progress`. |
| Tests fail with "form layout is empty" | The plugin's `forms:upsert` handler now requires a layout | Pass `layout` (raw HTML) or `style` (`"simple"`/`"grid"`/`"card"`/`"inline"`) in the upsert payload. |

---

## 20. Skeleton: copy-paste a new extension

The minimum that boots cleanly. Drop this in `extensions/my-extension/`, restart, activate in the admin.

```
extensions/my-extension/
├── extension.json
├── cmd/
│   └── plugin/
│       └── main.go
├── migrations/
│   └── 20260101_init.sql
├── admin-ui/
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── index.tsx
│       ├── index.css
│       └── HelloPage.tsx
└── scripts/
    └── extension.tengo
```

### `extension.json`

```json
{
  "name":         "My Extension",
  "slug":         "my-extension",
  "version":      "0.1.0",
  "author":       "You",
  "description":  "Minimal scaffold.",
  "capabilities": ["data:read", "data:write", "log:write"],
  "plugins":      [{ "binary": "bin/my-extension", "events": [] }],
  "admin_ui": {
    "entry":  "admin-ui/dist/index.js",
    "menu":   { "label": "My Extension", "icon": "Puzzle", "section": "content", "position": "9" },
    "routes": [{ "path": "/", "component": "HelloPage" }]
  }
}
```

### `cmd/plugin/main.go`

```go
package main

import (
    "context"
    "encoding/json"

    goplugin "github.com/hashicorp/go-plugin"
    "google.golang.org/grpc"

    "vibecms/internal/coreapi"
    vibeplugin "vibecms/pkg/plugin"
    coreapipb "vibecms/pkg/plugin/coreapipb"
    pb "vibecms/pkg/plugin/proto"
)

type MyPlugin struct {
    host coreapi.CoreAPI
}

func (p *MyPlugin) Initialize(hostConn *grpc.ClientConn) error {
    p.host = coreapi.NewGRPCHostClient(coreapipb.NewVibeCMSHostClient(hostConn))
    return nil
}

func (p *MyPlugin) GetSubscriptions() ([]*pb.Subscription, error) {
    return nil, nil
}

func (p *MyPlugin) HandleEvent(action string, payload []byte) (*pb.EventResponse, error) {
    return &pb.EventResponse{Handled: false}, nil
}

func (p *MyPlugin) HandleHTTPRequest(req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
    body, _ := json.Marshal(map[string]string{"hello": "world"})
    return &pb.PluginHTTPResponse{
        StatusCode: 200,
        Headers:    map[string]string{"Content-Type": "application/json"},
        Body:       body,
    }, nil
}

func (p *MyPlugin) Shutdown() error { return nil }

func main() {
    goplugin.Serve(&goplugin.ServeConfig{
        HandshakeConfig: vibeplugin.Handshake,
        Plugins: map[string]goplugin.Plugin{
            "extension": &vibeplugin.ExtensionGRPCPlugin{Impl: &MyPlugin{}},
        },
        GRPCServer: goplugin.DefaultGRPCServer,
    })
}
```

### `migrations/20260101_init.sql`

```sql
-- Owned table — namespaced by the extension's domain.
CREATE TABLE IF NOT EXISTS my_extension_items (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_my_extension_items_name ON my_extension_items(name);
```

### `admin-ui/package.json`

```json
{
  "name": "my-extension-admin-ui",
  "private": true,
  "type": "module",
  "scripts": { "build": "vite build" },
  "devDependencies": {
    "@tailwindcss/vite": "^4.2.4",
    "@types/react": "^19.0.0",
    "@vitejs/plugin-react": "^4.3.0",
    "tailwindcss": "^4.2.4",
    "typescript": "^5.6.0",
    "vite": "^6.0.0"
  }
}
```

### `admin-ui/tsconfig.json`

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "outDir": "dist",
    "declaration": false
  },
  "include": ["src"]
}
```

### `admin-ui/vite.config.ts`

```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  define: { "process.env.NODE_ENV": JSON.stringify("production") },
  build: {
    outDir: "dist",
    lib: {
      entry: "src/index.tsx",
      formats: ["es"],
      fileName: "index",
      cssFileName: "index",
    },
    rollupOptions: {
      external: [
        "react", "react/jsx-runtime",
        "react-dom", "react-dom/client",
        "react-router-dom",
        "sonner",
        "@vibecms/ui", "@vibecms/api", "@vibecms/icons",
      ],
    },
    cssCodeSplit: false,
  },
});
```

### `admin-ui/src/index.tsx`

```tsx
import "./index.css";
export { default as HelloPage } from "./HelloPage";
```

### `admin-ui/src/index.css`

```css
@import "tailwindcss";
@source "./**/*.{ts,tsx}";
```

### `admin-ui/src/HelloPage.tsx`

```tsx
import { Card, CardContent, Button } from "@vibecms/ui";

export default function HelloPage() {
  return (
    <Card>
      <CardContent className="p-6 space-y-4">
        <h2 className="text-lg font-semibold">My Extension</h2>
        <p className="text-sm text-slate-600">
          Welcome. This is your blank canvas — start by editing
          <code className="ml-1 rounded bg-slate-100 px-1.5 py-0.5 text-xs">admin-ui/src/HelloPage.tsx</code>.
        </p>
        <Button onClick={() => fetch("/admin/api/ext/my-extension/").then(r => r.json()).then(console.log)}>
          Ping plugin
        </Button>
      </CardContent>
    </Card>
  );
}
```

### `scripts/extension.tengo`

```tengo
log := import("core/log")
log.info("My Extension loaded")
```

### Build it

```bash
cd extensions/my-extension
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/my-extension ./cmd/plugin/
cd admin-ui && npm install && npm run build
```

Then `docker compose restart app`, activate in the admin extension picker, and visit `/admin/ext/my-extension/`. You should see the card.

That's a complete, bootable extension. Add database tables, HTTP routes, event handlers, and admin UI pages from there.

---

## 21. Reference: every shipped extension

The seven built-in extensions cover the full spectrum from "5 lines of Tengo" to "2,400 lines of Go + comprehensive React UI". Use them as recipes.

### `media-manager`
**Type:** gRPC plugin + React micro-frontend + Tengo scripts
**Tables:** `media_files`, `media_image_sizes`
**Public routes:** `GET /media/cache/*`, `GET /media/*` (auto-WebP)
**What it teaches:** owned tables with content hashes, on-the-fly image resize + cache, native-CLI optimization (jpegoptim/optipng/pngquant/imagemagick), background goroutines with progress polling, theme/extension asset import via `extension.activated`/`theme.activated`, custom field type registration (`media`, `image`, `gallery`, `file`).

### `forms`
**Type:** gRPC plugin + React micro-frontend + Tengo scripts + content block + custom field type
**Tables:** `forms`, `form_submissions`, `form_webhook_logs`
**Public routes:** `POST /forms/submit/*`
**What it teaches:** event-with-result rendering (`forms:render`), upsert-from-theme pattern (`forms:upsert`), recursive condition engine (server + client mirror in JS), CAPTCHA integration (reCAPTCHA / hCaptcha / Turnstile), per-IP token-bucket rate limiting, file uploads with multipart parsing, GDPR retention worker, webhook delivery with logging, CSV export, JSON import/export with auto slug suffixing, custom field type registration (`form_selector`).

### `email-manager`
**Type:** gRPC plugin + React micro-frontend
**Tables:** owns email-template / email-rule / email-log tables
**What it teaches:** the slot pattern. Exposes the `email-settings` slot for providers to inject into. Doesn't send emails itself — manages templates and rules, then emits `email.send` events that providers (smtp-provider, resend-provider) handle.

### `sitemap-generator`
**Type:** gRPC plugin + Tengo scripts
**Public routes:** registered via Tengo `routes.register("GET", "/sitemap.xml", "handlers/serve_index")`
**What it teaches:** event-driven sitemap rebuilds (`node.published`/`node.deleted`), runtime route registration via Tengo (vs declarative `public_routes[]`), Yoast-style sitemap organization.

### `smtp-provider`
**Type:** gRPC plugin
**What it teaches:** how to be an email provider — subscribe to `email.send`, render templates, push to an external service. Configuration via `settings_schema`, settings UI injected into `email-settings` slot.

### `resend-provider`
**Type:** Tengo-only (no compiled binary)
**What it teaches:** **you don't need a Go binary**. A complete email provider in 20 lines of Tengo: subscribe to `email.send`, fetch the API key from settings, post via `core/http`. Use this as the template for any "react to event, hit external HTTP API" extension.

### `hello-extension`
**Type:** Tengo-only
**What it teaches:** the bare minimum. A manifest, an `extension.tengo` entry point, and a single event handler. The reference for "I just want to inject some HTML into a hook point".

### `content-blocks`
**Type:** Content blocks + templates only (no plugin, no admin UI, no scripts)
**What it teaches:** **declarative-only** extensions. The manifest lists 40 blocks and 10 page templates; the extension loader registers them from disk. No code runs. This is how you ship presentation-only packages — use it as a template for theme-companion extensions.

---

## 22. Reference dissection: `media-manager`

The most code-dense built-in extension. Worth a deep read if you're building anything that touches files, images, or asset import.

### What it owns

- `media_files` — every file in the library (originals + extension/theme-imported assets).
- `media_image_sizes` — named size presets (`thumbnail`, `medium`, `large`, custom).
- `storage/media/...` — the on-disk file tree.
- `storage/cache/images/<size>/...` — the resized variant cache.
- `storage/cache/images/_webp/...` — the auto-WebP cache for original-size images.

### Public routes

- `GET /media/cache/{size}/{path...}` — serve a resized variant. Generates on demand if not cached.
- `GET /media/{path...}` — serve the original, but auto-convert to WebP if `Accept: image/webp` and the optimizer setting allows.

The `*` wildcard captures the rest of the path. The plugin parses the size name out of the prefix, looks up the size definition from `media_image_sizes`, and either serves the cached file or generates and caches it.

**Thundering herd protection:** for the cache-on-miss path, the plugin uses a `sync.Map` of per-path mutexes to ensure only one goroutine generates a given variant at a time. See `MediaManagerPlugin.getPathMutex`.

### Image processing pipeline

On upload (`POST /admin/api/ext/media-manager/upload`):
1. Parse multipart, enforce 50MB hard cap.
2. Detect MIME from content (`http.DetectContentType`) — never trust the client's Content-Type.
3. Validate against an allowlist (`isAllowedMimeType`).
4. Generate a path-safe filename from the MIME type, **not** the original extension.
5. For images: decode to read original dimensions.
6. **Normalize.** If enabled, downscale via ImageMagick CLI (best quality; falls back to `convert` for older installs). Then re-encode via `jpegoptim` / `optipng` / `pngquant` depending on format and quality target.
7. Save the **original bytes** to `storage/media/originals/<path>` first.
8. Save the **optimized bytes** to `storage/media/<path>` and record the URL.
9. Insert into `media_files` with `is_optimized=true`, `original_size`, `optimization_savings`.

The original-bytes backup is what makes the "Restore Original" feature possible — re-optimization re-reads from the original instead of the (already optimized) current file, so quality doesn't degrade across re-runs.

### Asset import (theme + extension assets)

`HandleEvent` subscribes to four lifecycle events:

```go
func (p *MediaManagerPlugin) GetSubscriptions() ([]*pb.Subscription, error) {
    return []*pb.Subscription{
        {EventName: "theme.activated",      Priority: 50},
        {EventName: "theme.deactivated",    Priority: 50},
        {EventName: "extension.activated",  Priority: 50},
        {EventName: "extension.deactivated", Priority: 50},
    }, nil
}
```

The activation handler (`handleOwnedAssetsActivated`):
1. Queries existing `media_files` rows for this owner.
2. For each declared asset:
   - Reads bytes, computes SHA-256, detects MIME.
   - If the row exists and `content_hash` matches → **skip**.
   - If the row exists and hash differs → **update in place** (overwrite path, URL, dimensions, hash).
   - If new → **create row + store file**.
3. Reconciles: any existing row whose `asset_key` is no longer in the manifest gets **deleted** (file + row).

The deactivation handler simply deletes every row tagged with the owner.

This is the canonical pattern for "extension owns external resources tied to other extensions/themes." Match it for any extension that imports artifacts based on lifecycle events.

### Bulk operations with progress polling

Re-optimize-all and restore-all are async. The plugin starts a goroutine, updates a thread-safe progress struct (`bulkJobProgress` with a mutex), and the admin UI polls a progress endpoint:

```
POST /optimizer/reoptimize-all   →  202 Accepted, {data: {running: true, total: 1234, processed: 0}}
GET  /optimizer/reoptimize-progress  →  200 OK, {data: {running: true, total: 1234, processed: 489, total_saved: 12345678, status: "running"}}
GET  /optimizer/reoptimize-progress  →  200 OK, {data: {running: false, status: "done", ...}}
```

A second `POST /optimizer/reoptimize-all` while one is running returns `409 ALREADY_RUNNING`. Match this pattern for any bulk operation longer than a few seconds.

### Capability declaration

```jsonc
"capabilities": [
  "media:read", "media:write", "media:delete",
  "data:read", "data:write",
  "files:write", "files:delete",
  "settings:read", "settings:write",
  "events:emit",
  "log:write"
]
```

What each enables:
- `media:*` — even though media-manager **is** the media implementation, when it calls itself through CoreAPI for theme/extension imports, the capability guard enforces the same rules.
- `data:*` — the owned tables (`media_files`, `media_image_sizes`).
- `files:*` — the actual disk writes.
- `settings:*` — optimizer settings (`media:optimizer:jpeg_quality`, etc.).
- `events:emit` — emits `media:sizes_changed` when sizes change so core can refresh its registry.
- `log:write` — diagnostics.

No `events:subscribe` — even though it subscribes to four events. That's because plugin subscriptions via `GetSubscriptions()` are wired at startup, **before** the capability guard activates. Runtime `Subscribe` calls would need it; declarative subscriptions don't.

---

## 23. Reference dissection: `forms`

The most feature-dense built-in extension by surface area: 17 `cmd/plugin/*.go` files, 22 `admin-ui/src/*.tsx` files, 240+ unit tests, 3 SQL migrations, 4 default form layouts, a content block, and a custom field type.

### What it owns

- `forms` — the form definitions (`fields` JSONB, `layout` string, `notifications` JSONB, `settings` JSONB).
- `form_submissions` — every submission with `data` and `metadata` JSONB.
- `form_webhook_logs` — webhook delivery history.

### Two events handled, two events emitted

```go
func (p *FormsPlugin) GetSubscriptions() ([]*pb.Subscription, error) {
    return []*pb.Subscription{
        {EventName: "forms:render", Priority: 0},   // event-with-result
        {EventName: "forms:upsert", Priority: 0},   // fire-and-forget; idempotent
    }, nil
}
```

- **`forms:render`** — called from templates with `{{ event "forms:render" (dict "form_id" "contact" "hidden" (dict "trip_slug" "...")) }}`. Returns rendered HTML in `EventResponse.Result`.
- **`forms:upsert`** — used by themes to seed forms via Tengo (`themes/hello-vietnam/scripts/theme.tengo` calls `events.emit("forms:upsert", {...})`). Idempotent — checks `slug` first, no-op if exists unless `force: true`.

The plugin emits:
- **`forms:submitted`** — after every accepted submission. Payload: `form_id`, `form_slug`, `submission_id`, `data`, `metadata`. Other extensions (Tengo or gRPC) can subscribe for analytics, CRM sync, etc.

### The submission pipeline

`POST /forms/submit/{slug}` — `handlers_submit.go::handleSubmit`:

1. **Resolve form by slug.** 404 if missing.
2. **Rate limit** by IP using LRU per-IP token bucket (`ratelimit.go`). 429 on breach.
3. **Parse body** as either JSON or `multipart/form-data`. Multipart yields a list of pending file uploads.
4. **Honeypot check.** If field `website_url` is non-empty, return 200 with no DB write — bots see "success" and move on.
5. **CAPTCHA check** if configured (Turnstile / hCaptcha / reCAPTCHA via `core/http`).
6. **Validate** every field against type + required + length + pattern + per-field display_when conditions (`validation.go`).
7. **Store uploaded files** via `host.StoreFile`, replace the field value with `{name, url, size, mime_type}`.
8. **Build metadata** (user-agent, referer, optionally IP per `store_ip` setting).
9. **Insert** into `form_submissions`.
10. **Trigger notifications** (async — separate goroutine).
11. **Emit `forms:submitted`** event.
12. **Fire webhook** (async).
13. Return `{success: true, message: "Submission received"}`.

### The condition engine

`conditions.go` implements a recursive AND/OR group evaluator for both:
- Field-level **show/hide** (`field.display_when`) — affects visibility *and* validation (hidden fields skip required checks).
- Notification **route_when** — skip notifications whose conditions don't match the submission.

The same evaluator is mirrored in JavaScript (`blocks/vibe-form/script.js`) for client-side preview before submit. **The two implementations must match exactly** — the comments in the JS version explicitly note this.

### Default layouts (Tailwind dependence)

The four bundled layouts (`default_layout.html`, `grid_layout.html`, `card_layout.html`, `inline_layout.html`) embed Tailwind utility classes (`max-w-2xl`, `bg-indigo-600`, `rounded-lg`, …). They render correctly only on pages that already load Tailwind.

For themes that don't use Tailwind, the canonical pattern is **theme-owned layouts** seeded via `forms:upsert`:

```tengo
// themes/<theme>/scripts/theme.tengo
events := import("core/events")
assets := import("core/assets")

contact_layout := assets.read("forms/contact.html")  // your theme's CSS classes
if is_error(contact_layout) { contact_layout = "" }

events.emit("forms:upsert", {
  slug:   "contact",
  name:   "Contact",
  layout: contact_layout,
  fields: [ /* … */ ]
})
```

See `themes/README.md` §10 for the full theme-owned form pattern. The `hello-vietnam` theme is the live reference.

### Spam protection layers

Three layers, all configurable per form:

1. **Honeypot** (`honeypot_enabled` setting, default `true`). Renders a hidden `<input name="website_url">`. Bots that auto-fill all fields populate it; the server returns 200 silently with no DB write.
2. **Rate limit** (`rate_limit` setting, default 10/hour per IP). LRU-backed token bucket sized at 10,000 IPs. 429 on breach.
3. **CAPTCHA** (`captcha_provider` setting, default `none`). Supports `recaptcha`, `hcaptcha`, `turnstile`. Token verified via `host.Fetch` against the provider's `siteverify` endpoint.

CSRF protection is documented in `docs/forms.md`; the honeypot + rate limiter cover the primary vectors today.

### GDPR retention

A background goroutine spawned in `Initialize` runs once at boot and again every hour:

```go
func (p *FormsPlugin) startRetentionWorker(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        defer ticker.Stop()
        p.runRetention(ctx)
        for {
            select {
            case <-ctx.Done(): return
            case <-ticker.C:   p.runRetention(ctx)
            }
        }
    }()
}
```

`runRetention` queries each form's `retention_period` setting (in days), finds submissions older than the cutoff, deletes any uploaded files referenced in `data`, then deletes the row. **Cancellation via the shutdown context** — when the kernel deactivates the extension, `Shutdown()` calls `cancel()`, the for-select loop hits `ctx.Done()`, and the goroutine exits cleanly.

This is the canonical pattern for any background work with graceful shutdown. Don't use raw goroutines without a cancel signal.

### File storage

Submission files go to `forms/submissions/<form_id>/<unix_nano>_<safe_name>` via `host.StoreFile`. Filenames are sanitized (`sanitizeFilename`): path separators replaced with underscore, NUL bytes stripped, length capped at 200.

On submission delete (single or bulk), the plugin walks the `data` map for any `{url: "/forms/submissions/...", ...}` shapes and calls `host.DeleteFile` on each before deleting the submission row. The retention worker does the same.

### Notification templates

Notification subjects and bodies are rendered with Go's `html/template`. Available variables:

| Variable | Type | Description |
|---|---|---|
| `{{.FormName}}` | string | |
| `{{.FormSlug}}` | string | |
| `{{.FormID}}` | uint | |
| `{{.SubmittedAt}}` | string (RFC3339) | |
| `{{range .Data}}{{.Key}} — {{.Value}}{{end}}` | slice of `{Key, Label, Value}` | |
| `{{index .Field "email"}}` | map | Direct field access by ID |

The bundled "Send test email" endpoint (`POST /admin/api/ext/forms/{id}/notifications/{idx}/test`) auto-generates sample data from each field's type and renders the template with that — useful for preview without a real submission.

---

## Appendix: useful queries for development

```sh
# What extensions are registered?
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT slug, version, is_active FROM extensions ORDER BY priority, slug;"

# Which migrations have been applied?
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT slug, filename, applied_at FROM extension_migrations ORDER BY applied_at;"

# What blocks are registered, and from where?
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT slug, source, theme_name FROM block_types ORDER BY source, slug;"

# Force-resync block schemas after a manifest change (rare — usually unnecessary)
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "UPDATE block_types SET content_hash = 'force-' || floor(random()*1e6)::text WHERE source = 'extension';"
docker compose restart app

# Which extensions are listening to which events?
docker compose logs app --tail=200 | grep -E '\[plugins\] started'

# Inspect forms data
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT id, slug, name, jsonb_array_length(fields) AS n_fields FROM forms;"

# Inspect media library
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT source, COUNT(*) FROM media_files GROUP BY source;"

# Hot-deploy admin UI changes without rebuilding the Docker image
cd extensions/<slug>/admin-ui && npm run build
docker cp dist/. vibecms-app-1:/app/extensions/<slug>/admin-ui/dist/

# Hot-deploy plugin binary changes
cd extensions/<slug>
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/<slug> ./cmd/plugin/
docker cp bin/<slug> vibecms-app-1:/app/extensions/<slug>/bin/<slug>
docker compose restart app  # restart needed to bounce the plugin process
```

---

**Got an addition or correction?** Open a PR. The reference is `media-manager/` and `forms/`; the contract is the kernel. When they disagree, the kernel wins — file an issue.
