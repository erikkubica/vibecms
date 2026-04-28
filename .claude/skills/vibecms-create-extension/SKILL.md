---
name: vibecms-create-extension
description: |
  Use when building a new VibeCMS extension from scratch — adding a feature
  package that owns its own database tables, HTTP routes, business logic,
  admin UI, or event handlers. Triggers: "add an extension that does X",
  "build a Y manager", "this should be an extension, not core",
  scaffolding under `extensions/<slug>/`, deciding between a gRPC plugin
  and a Tengo-only extension, declaring capabilities, wiring an admin
  micro-frontend into the SPA shell, registering custom field types or
  content blocks owned by an extension, or seeding initial data via SQL
  migrations. Pairs with `vibecms-extension-frontend` for the React side.
---

# Creating a VibeCMS Extension

## When to use this skill

Reach for this skill when **the answer to "where does this feature live?" is "in an extension."**

The HARD RULE from `CLAUDE.md` and `README.md`: *if disabling/removing the extension would leave dead code in core, that code belongs in the extension, not in core.* Image optimization, email templates, sitemaps, forms, search — all extensions. Anything feature-specific.

**Don't** use this skill when:
- You're touching `internal/` (kernel work — different rules apply)
- You only need an event hook + an outbound HTTP call (a Tengo script in a theme can be enough)
- You're styling an existing admin page (use `vibecms-extension-frontend`)

## Source of truth

`extensions/README.md` is the contract. This skill is a fast on-ramp; consult the README for the exhaustive reference. Key sections:

| Topic | Section |
|---|---|
| Mental model + boot sequence | §1 |
| Folder anatomy | §2 |
| `extension.json` schema (every field) | §3 |
| Three authoring surfaces (gRPC / Tengo / Admin UI) | §4 |
| **Capability table** (every CoreAPI guard) | §5 |
| gRPC plugin (`Initialize`, `GetSubscriptions`, `HandleEvent`, `HandleHTTPRequest`, `Shutdown`) | §6 |
| Admin proxy vs public proxy | §7 |
| Events (fire-and-forget vs event-with-result) | §8 |
| CoreAPI reference | §9 |
| SQL migrations (idempotency, JSONB normalization) | §10 |
| Tengo scripts | §11 |
| Admin UI authoring | §12 (use `vibecms-extension-frontend`) |
| Custom field types | §13 |
| Content blocks owned by extensions | §14 |
| Settings schema + slot pattern | §15 |
| `extension-asset:<slug>:<key>` mechanism | §16 |
| Lifecycle (scan → activate → run → deactivate) | §17 |
| **The 12 Mandalorian rules** | §18 |
| Troubleshooting table | §19 |
| **Skeleton (copy-paste)** | §20 |

## Decision tree: which kind of extension?

```
Need owned database tables?            → gRPC plugin
Need persistent in-memory state?       → gRPC plugin
Need long-running goroutines?          → gRPC plugin
Need native Go libraries?              → gRPC plugin
Just react to an event + hit an API?   → Tengo-only (see resend-provider)
Just ship blocks/templates?            → declarative-only (see content-blocks)
```

**Reference implementations to read before building:**

| Extension | Why it's the reference |
|---|---|
| `media-manager` | Owned tables + image processing + asset import handshake |
| `forms` | Event-with-result, public routes, FakeHost testing pattern |
| `resend-provider` | Tengo-only, ~20 lines total |
| `content-blocks` | Declarative blocks/templates, no code |
| `hello-extension` | Bare minimum |

## Bootable skeleton

Drop this in `extensions/my-extension/`, restart, activate in the admin.

### File tree

```
extensions/my-extension/
├── extension.json
├── cmd/plugin/main.go
├── migrations/20260101_init.sql
├── admin-ui/
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── index.tsx
│       ├── index.css
│       └── HelloPage.tsx
└── scripts/extension.tengo
```

### `extension.json` (minimum that boots)

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

For the full manifest schema (settings_schema, public_routes, blocks, field_types, slots, assets, etc.) see `extensions/README.md` §3.

### `cmd/plugin/main.go` (the five required methods)

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

### Build

```bash
cd extensions/my-extension
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/my-extension ./cmd/plugin/
cd admin-ui && npm install && npm run build
docker compose restart app
```

Then activate in the admin extension picker and visit `/admin/ext/my-extension/`.

For the React side (`admin-ui/`), use the **`vibecms-extension-frontend`** skill — Tailwind setup, externalize list, `dist/index.css` requirement, `__VIBECMS_SHARED__` shim, etc.

## The 12 Mandalorian rules (summary)

Memorize these. Full text in `extensions/README.md` §18.

1. **The manifest is the contract.** Anything not in `extension.json` is invisible to the kernel.
2. **Capabilities are minimal.** Declare only what your code calls.
3. **`{"error": code, "message": text}` is the public error envelope.** Always.
4. **Public routes are at the path you declare** — not under `/api/...`.
5. **Tables are prefixed by domain** (`forms`, `media_files`, never collide with core).
6. **JSONB columns come back as strings** — always normalize before iterating.
7. **Asset references survive theme/extension switches.** Use `extension-asset:<slug>:<key>`.
8. **Per-extension Tailwind builds are mandatory for Docker.** Ship your own `dist/index.css`.
9. **Use the design system primitives** (`ListPageShell`, `ListHeader`, etc.).
10. **Filter/sort/view/pagination state lives in URL params.**
11. **Production code under 300 lines per file** (500 hard limit).
12. **Don't reach for a slot pattern when an event-with-result will do.** Pick the looser coupling.

## Top 5 bugs (and how to dodge them)

| Bug | Cause | Fix |
|---|---|---|
| `403 / capability denied` at runtime | Missing entry in `capabilities[]` | Check `extensions/README.md` §5 capability table; add the missing one. |
| Public route returns `404 Not Found` | Assumed it was under `/api/...` | The path you declare is the path users hit. `/forms/submit/contact`, not `/api/ext/forms/submit/contact`. |
| Admin UI loads JS but no styles | Per-extension Tailwind not configured | Add `@tailwindcss/vite` + `cssFileName: "index"`. See `vibecms-extension-frontend`. |
| `[object Object]` in admin UI | JSONB column not normalized | Add `normalizeJSONBFields(row, "fields", "settings", ...)` to the read path. |
| Plugin starts but doesn't get events | `GetSubscriptions()` returns wrong names, OR you wrote them in the manifest's `plugins[].events` (which is informational only) | Subscriptions come from the RPC, not the manifest. |

## Capability cheat sheet (the 24 valid capabilities)

```
nodes:read       nodes:write       nodes:delete
nodetypes:read   nodetypes:write
settings:read    settings:write
events:emit      events:subscribe
email:send
menus:read       menus:write       menus:delete
routes:register
filters:register filters:apply
media:read       media:write       media:delete
users:read
http:fetch
log:write
data:read        data:write        data:delete
files:write      files:delete
```

Declaring a capability that isn't in this list is silently ignored today; reviewers will flag over-declaration. See §5 of the README for which CoreAPI methods each capability gates.

## Useful queries while developing

```bash
# What did the loader register for my extension?
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT slug, version, is_active FROM extensions WHERE slug='my-extension';"

# Which migrations have applied?
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT filename, applied_at FROM extension_migrations WHERE slug='my-extension';"

# Tail the activation log
docker compose logs app --tail=200 | grep -E '\[extensions\]|\[plugins\]|my-extension'

# Hot-deploy a recompiled binary into the running container
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/my-extension ./cmd/plugin/
docker cp bin/my-extension vibecms-app-1:/app/extensions/my-extension/bin/my-extension
docker compose restart app   # required to bounce the plugin process
```

## Testing

The forms extension's **`FakeHost`** (`extensions/forms/cmd/plugin/fakehost_test.go`) is the canonical pattern: an in-memory `coreapi.CoreAPI` implementation that lets you unit-test handlers without spinning up Postgres.

```go
func TestHandleSubmit(t *testing.T) {
    p := &MyPlugin{host: &FakeHost{...}}
    resp, _ := p.handleSubmit(context.Background(), &pb.PluginHTTPRequest{...})
    // assert on resp
}
```

Don't mock the gRPC layer; mock the `CoreAPI` interface. Tests stay fast, deterministic, and runnable on every save.

## Next step after the skeleton boots

1. Add real handlers in `cmd/plugin/handlers_*.go` (split early — 300-line cap).
2. Declare each new capability you call before adding the call.
3. For each public route you add, declare it in `extension.json` `public_routes[]`.
4. For each event you subscribe to, return it from `GetSubscriptions()`.
5. For each admin UI page, add it to `admin_ui.routes[]` and export it from `admin-ui/src/index.tsx`.
6. Run the verifying queries above after each rebuild.

When in doubt, open `extensions/forms/` or `extensions/media-manager/` — they exercise every surface area.
