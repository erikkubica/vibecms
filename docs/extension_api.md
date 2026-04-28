# VibeCMS Extension API

VibeCMS uses a "Kernel + Extensions" architecture. The core CMS provides the basic infrastructure (Node CRUD, Auth, Database, Event Bus), while everything else—even built-in features like Media Management or Email Delivery—is implemented as an extension.

This document serves as the complete handoff guide for developers building VibeCMS extensions.

---

## 1. Extension Anatomy

An extension lives in its own directory under `extensions/<slug>/`. It can contain any combination of:
1. **`extension.json`** (Required): The manifest.
2. **gRPC Plugin Binary** (Optional): Compiled Go binary for backend logic.
3. **Tengo Scripts** (Optional): Script files in `scripts/` to hook into events and routes without compiling Go code.
4. **React Micro-Frontend** (Optional): Vite build output for admin UI pages.
5. **SQL Migrations** (Optional): In `migrations/`, executed when the extension activates.

### The Manifest (`extension.json`)

```json
{
  "name": "My Cool Extension",
  "slug": "my-cool-ext",
  "version": "1.0.0",
  "author": "Acme Corp",
  "description": "Adds cool features to VibeCMS",
  "priority": 50,
  "capabilities": ["nodes:read", "data:write", "files:write", "log:write"],
  "plugins": {
    "grpc": {
      "command": "./cmd/my-cool-ext/my-cool-ext"
    }
  },
  "admin": {
    "routes": [
      {
        "path": "/",
        "component": "src/main.ts",
        "label": "Cool Settings",
        "icon": "Zap"
      }
    ]
  }
}
```

- **`priority`**: Order in which extensions load (default: 50).
- **`capabilities`**: Core API permissions the extension requests. Enforced at runtime by the CoreAPI wrapper.
- **`plugins`**: Defines where the gRPC binary is located so the PluginManager can start it.
- **`admin`**: Injects paths into the React Admin SPA. The component is loaded via import maps.

---

## 2. Capability System

To prevent extensions from secretly exfiltrating data or wiping the database, you must declare all required capabilities in `extension.json`. The CoreAPI will intercept and reject any call for which the extension lacks permission.

Common capabilities include:
- `nodes:read`, `nodes:write`
- `settings:read`, `settings:write`
- `events:emit`, `events:subscribe`
- `email:send`
- `menus:read`, `menus:write`
- `media:read`, `media:write`
- `users:read`
- `http:fetch` (Outbound requests)
- `log:write`
- `data:read`, `data:write`, `data:exec` (Access to custom SQL tables)
- `files:write` (Local filesystem storage)

---

## 3. Building a gRPC Plugin (Backend)

For robust backend features (complex logic, custom APIs, third-party library integrations), build a gRPC plugin.

### The Plugin Interface

The CMS core and the extension communicate over gRPC. Your plugin implements the `VibeCMSPlugin` interface and is served using HashiCorp's `go-plugin`.

When your plugin starts, it receives a generic `CoreAPI` client from the host.

### Minimal Go Plugin Example

```go
package main

import (
	"context"
	"github.com/hashicorp/go-plugin"
	shared "vibecms/pkg/plugin"
	pb "vibecms/pkg/plugin/coreapipb"
)

// MyExt implements the VibeCMSPlugin plugin interface
type MyExt struct {
	coreAPI shared.CoreAPI
}

func (p *MyExt) Init(api shared.CoreAPI) error {
	p.coreAPI = api
	// e.g. register custom node types here
	return nil
}

// Proxied from /admin/api/ext/<slug>/*
func (p *MyExt) HandleHTTPRequest(ctx context.Context, req *pb.HTTPRequest) (*pb.HTTPResponse, error) {
	if req.Method == "GET" && req.Path == "/hello" {
		// Example of using CoreAPI to verify capabilities work
		p.coreAPI.Log(ctx, "info", "Hello endpoint hit", nil)
		return &pb.HTTPResponse{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(`{"message": "Hello from plugin!"}`),
		}, nil
	}
	
	return &pb.HTTPResponse{StatusCode: 404}, nil
}

func (p *MyExt) Shutdown() error { return nil }

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"grpc": &shared.VibeCMSGRPCPlugin{
				Impl: &MyExt{},
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
```

### Returning Content to Templates (Sync Event with Reply)

Most events are fire-and-forget — handlers run asynchronously and return values are discarded. For events that need to *return rendered content into a Go template* (e.g. `{{event "forms:render" ...}}` from a layout), the event bus provides a synchronous result-collecting path.

**Plugin side (no extra code needed):** any plugin subscribed via `GetSubscriptions()` automatically receives both fire-and-forget events (`Publish`) and result-collecting events (`PublishCollect`). Set `EventResponse.Handled = true` and put your output in `EventResponse.Result` (`[]byte`). Returning `Handled: false` or an empty `Result` means "I had nothing to contribute"; another plugin can still answer.

**Theme/template side:** call `{{event "<name>" (dict ...) }}` and wrap with `safeHTML` if the plugin returns markup:

```go-template
{{ safeHTML (event "forms:render" (dict
    "form_id" "trip-order"
    "hidden"  (dict "trip_slug" .node.slug "trip_price" 45)
)) }}
```

Whatever map you pass becomes the plugin's payload (JSON-marshalled). Multiple plugins subscribing to the same event have their non-empty `Result` strings concatenated in registration order.

**Reference: forms extension events**

| Event | Direction | Payload | Returns |
|-------|-----------|---------|---------|
| `forms:upsert` | template → plugin | `{slug, name, fields, layout?, settings?, force?}` | n/a (fire-and-forget; idempotent on slug) |
| `forms:render` | template → plugin | `{form_id, hidden?}` | rendered form HTML |
| `forms:submitted` | plugin → world | `{form_id, form_slug, submission_id, data, metadata}` | n/a |

The `forms:upsert` pattern is the canonical way for a theme to ship its own forms — see `themes/hello-vietnam/scripts/theme.tengo` for a working example using `core/assets.read` to pull the layout HTML from a theme file.

---

## 4. Scripting (Tengo)

If you don't want to compile Go code, or just need simple lifecycle hooks, you can use the embedded Tengo scripting engine.

Create `scripts/extension.tengo` in your extension directory.

### Example: Inject SEO Defaults

```tengo
// scripts/extension.tengo
events := import("core/events")
log := import("core/log")

log.info("My scripting extension loaded")

events.on("node.created", "handlers/set_seo_defaults")
```

```tengo
// scripts/handlers/set_seo_defaults.tengo
nodes := import("core/nodes")
log := import("core/log")

// `event` is magically injected by the event dispatcher
node_id := event.payload.node_id

// Fetch full node using the CoreAPI nodes module
node := nodes.get(node_id)
if node != undefined {
    nodes.update(node_id, {
        seo_settings: {
            title: node.title + " | Acme Corp",
            index: "noindex"
        }
    })
    log.info("Set default SEO for new node")
}
```

---

## 5. Core API Reference

The `CoreAPI` interface gives you full control over the CMS. In Go plugins, this is accessed via the `shared.CoreAPI` interface injected during `Init()`. In Tengo, these map directly to the `core/*` imports.

### 5.1 Content Nodes
Manage pages, posts, and any custom models.
- **`GetNode(id uint)`**: Fetch single node.
- **`QueryNodes(query NodeQuery)`**: Filter, search, and paginate nodes.
- **`CreateNode(input NodeInput)`**: Create a node.
- **`UpdateNode(id uint, input NodeInput)`**: Update specific node fields.
- **`DeleteNode(id uint)`**: Delete a node.

### 5.2 Node Types
Register new content schemas (e.g., "Product", "Review").
- **`RegisterNodeType(input NodeTypeInput)`**: Register a new custom content type with a JSON-based field schema and icon.
- **`GetNodeType(slug string)`**
- **`ListNodeTypes()`**
- **`UpdateNodeType(slug string, input NodeTypeInput)`**
- **`DeleteNodeType(slug string)`**

### 5.3 Data Store (SQL Tables)
Extensions can have their own isolated PostgreSQL tables via migrations. Use these APIs to interact with them without raw SQL strings where possible.
- **`DataGet(table string, id uint)`**
- **`DataQuery(table string, query DataStoreQuery)`**: Returns rows based on a WHERE/ORDER condition.
- **`DataCreate(table string, data map[string]any)`**
- **`DataUpdate(table string, id uint, data map[string]any)`**
- **`DataDelete(table string, id uint)`**
- **`DataExec(sql string, args ...any)`**: Run raw SQL statements.

### 5.4 Media & Files
- **`UploadMedia(req MediaUploadRequest)`**: Send raw file bits to the Media Library.
- **`GetMedia(id uint)`**, **`QueryMedia(...)`**, **`DeleteMedia(id uint)`**
- **`StoreFile(path string, data []byte)`**: Save arbitrary blobs to local disk (generates a URL).
- **`DeleteFile(path string)`**

### 5.5 Events & Filters
- **`Emit(action string, payload map[string]any)`**: Fire a generic CMS event.
- **`Subscribe(action string, handler EventHandler)`**: Listen for other events (Go plugins only. Tengo uses `events.on()`).
- **`RegisterFilter(...) / ApplyFilters(...)`**: Hook into content mutations before they render.

### 5.6 External & Utility
- **`SendEmail(req EmailRequest)`**: Pushes to the core email dispatcher.
- **`Fetch(req FetchRequest)`**: Make outbound HTTP requests safely.
- **`Log(level, message string, fields map[string]any)`**: Logs centrally with your extension slug as the prefix.
- **`GetUser(id uint)` / `QueryUsers(query UserQuery)`**: Read-only user access.
- **`GetSetting(key)` / `SetSetting(key, val)`**: Access global site settings.

---

## 6. Developing the Micro-Frontend

Admin UI components for extensions are pure React SPAs transpiled by Vite as an ES module.
Instead of bundling React, they import it dynamically from the global scope defined by the CMS shell.

1. **Setup Vite**: Use standard React + Vite setup.
2. **Build Settings**: Configure `vite.config.ts` to output standard ES modules without hashing.
3. **Include Shims**: The CMS shell injects dependencies via `window.__VIBECMS_SHARED__`. Your Vite build uses this.
4. **Deploy**: Build your React app into the extension's `/admin-ui/dist` folder. The CMS auto-mounts it when an admin visits `/admin/extensions/<your-slug>`.

### CSS / Tailwind

Extensions ship their own compiled CSS — the admin shell does **not** scan extension sources for Tailwind classes. Each extension owns its build.

1. Add `@tailwindcss/vite` and `tailwindcss` as devDependencies.
2. Add the plugin to `vite.config.ts`:
   ```ts
   import tailwindcss from "@tailwindcss/vite";
   plugins: [react(), tailwindcss()],
   build: { lib: { entry: "src/index.tsx", cssFileName: "index", ... } }
   ```
3. Create `src/index.css`:
   ```css
   @import "tailwindcss";
   @source "./**/*.{ts,tsx}";
   ```
4. Import it once from your entry: `import "./index.css";` in `src/index.tsx`.

The build emits `dist/index.css` next to `dist/index.js`. The extension loader auto-injects a `<link rel="stylesheet">` for the sibling CSS when loading the JS entry, so you only need to declare the JS entry in your manifest. Design tokens, base styles, and `@vibecms/ui` component overrides come from the admin shell's stylesheet — your extension CSS only needs to contain the utility classes it actually uses.

**Cascade ordering note:** the loader inserts the extension `<link>` *before* admin-ui's stylesheet in `<head>`, not after. This is critical: both stylesheets put utilities in the same `@layer utilities`, and within a merged layer source order wins. If the extension stylesheet loaded later, its `.fixed` (used by drawers/modals in many extensions) would beat admin-ui's `.lg:relative` on `<aside class="fixed ... lg:relative">`, the desktop sidebar would stay `position: fixed`, main content would have no left offset, and the shell layout would collapse on every admin page. Don't change this insertion order.

### Shared libraries inside extensions

`react`, `react-dom`, `react-router-dom`, `sonner`, `@vibecms/ui`, `@vibecms/icons`, `@vibecms/api` are externalized in your `vite.config.ts`. The admin shell exposes them on `window.__VIBECMS_SHARED__`:

```tsx
const { useSearchParams } = (window as any).__VIBECMS_SHARED__.ReactRouterDOM;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;
import { Button } from "@vibecms/ui";              // resolves via shim
import { Upload } from "@vibecms/icons";           // → lucide-react
```

`__VIBECMS_SHARED__.ui` exposes the design-system list-page primitives (`ListPageShell`, `ListHeader`, `ListSearch`, `ListFooter`, `EmptyState`, `Chip`, `StatusPill`, `TitleCell`, `RowActions`, `Th`, `Td`, `Tr`, etc.) used by every list page in the CMS. Use them so your extension visually matches nodes/forms/media — see `extensions/media-manager/admin-ui/src/MediaLibrary.tsx` for a full reference (URL-synced filters/view/sort, per-tab counts via parallel `per_page=1` fetches, sortable column headers backed by the same `?sort=` URL param as the dropdown).

---

## 7. Reference Implementation: Forms Extension

The Forms extension (`extensions/forms/`) is the canonical example of a production-grade VibeCMS extension. It exercises every major CoreAPI surface area and is the recommended starting point for developers building complex extensions.

### Capability Coverage

| Capability | What it does | Source file |
|---|---|---|
| `http:fetch` | Verify CAPTCHA tokens via provider HTTP APIs | `cmd/plugin/captcha.go` |
| `http:fetch` | Fire outbound webhooks on submission | `cmd/plugin/webhooks.go` |
| `files:write` / `files:delete` | Store and clean up user-uploaded attachments | `cmd/plugin/files.go` |
| `events:emit` | Broadcast `forms:submitted` for other extensions to consume | `cmd/plugin/events.go` |
| `email:send` | Admin notifications and auto-responder emails via Go templates | `cmd/plugin/notifications.go` |
| `data:read` / `data:write` / `data:delete` | Full CRUD on custom `forms`, `form_submissions`, `webhook_logs` tables | `cmd/plugin/forms.go`, `handlers_submissions.go` |
| `settings:read` | Read global site settings for CAPTCHA keys | `cmd/plugin/captcha.go` |
| `log:write` | Structured logging throughout | all `cmd/plugin/` files |

### Notable Patterns

| Pattern | Source |
|---|---|
| `FakeHost` for unit tests (no live database required) | `cmd/plugin/fakehost_test.go` |
| Background goroutine with graceful shutdown via `context.Context` | `cmd/plugin/retention.go` |
| Conditional logic engine (field-level show/hide and notification routing) | `cmd/plugin/conditions.go` |
| Multipart + JSON submission parsing in a single handler | `cmd/plugin/handlers_submit.go` |
| Go `html/template` rendering for email notifications | `cmd/plugin/notifications.go` |

### Subscribing to `forms:submitted` from Another Extension

**Tengo script:**

```tengo
events := import("core/events")
log    := import("core/log")

events.on("forms:submitted", "handlers/on_form_submit")
```

```tengo
// scripts/handlers/on_form_submit.tengo
log := import("core/log")

// event.payload keys: form_id, form_slug, submission_id, data (map), metadata
log.info("Form submitted", {form: event.payload.form_slug})
```

**Go plugin (in `Init`):**

```go
p.host.Subscribe(ctx, "forms:submitted", func(payload map[string]any) {
    formSlug, _ := payload["form_slug"].(string)
    p.host.Log(ctx, "info", "Form submitted: "+formSlug, nil)
})
```

See `docs/forms.md` for the full public API reference.
