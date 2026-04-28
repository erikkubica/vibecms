# VibeCMS Dynamic UI System (VDUS)

## Overview

VDUS is a **Server-Driven UI** architecture where the Go Kernel controls admin page layouts
by emitting JSON layout trees, and the React Shell renders them via a recursive component
registry. Extensions participate by contributing components and modifying layout trees through
filter hooks — no DOM hacking required.

### Core Principles

1. **Server orchestrates layout.** The Go Kernel decides what appears where. The React Shell
   is a rendering engine, not a page builder.
2. **Complex components stay in React.** WYSIWYG editors, block pickers, drag-and-drop trees —
   these ship as real React components. The layout tree says *where* to put them, not *how*
   they work internally.
3. **Actions, not code.** User interactions are expressed as Action Objects (`CORE_API`,
   `NAVIGATE`, `TOAST`), never raw JavaScript strings.
4. **SSE reactivity.** A Server-Sent Events stream pushes state changes (extension toggled,
   node type created) to the Shell, which invalidates TanStack Query caches instantly.
5. **Component library as contract.** The Shell ships ~30 built-in components. Extensions
   reference them by name. New components are registered from extension ESM bundles.

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Browser (React Shell)                       │
│                                                                      │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────────┐ │
│  │ useBoot()    │  │ useSSE()     │  │ RecursiveRenderer          │ │
│  │ TanStack Qry │  │ EventSource  │  │  ├─ ComponentRegistry     │ │
│  │              │  │              │  │  ├─ ActionHandler          │ │
│  │ /admin/api/  │  │ /admin/api/  │  │  ├─ RemoteComponent       │ │
│  │  boot        │  │  events      │  │  └─ PageStore              │ │
│  └──────┬───────┘  └──────┬───────┘  └────────────┬───────────────┘ │
│         │                 │                        │                  │
└─────────┼─────────────────┼────────────────────────┼──────────────────┘
          │ HTTP            │ SSE                    │ HTTP
          ▼                 ▼                        ▼
┌──────────────────────────────────────────────────────────────────────┐
│                         Go Kernel (Fiber)                            │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │ BootHandler   │  │ SSE          │  │ SDUI Engine               │  │
│  │ GET /boot     │  │ Broadcaster  │  │  ├─ GenerateBootManifest │  │
│  │ GET /layout/  │  │              │  │  ├─ GenerateLayout       │  │
│  │   :page       │  │ GET /events  │  │  ├─ Cache (invalidated)  │  │
│  │               │  │              │  │  └─ Filter chain          │  │
│  └──────┬────────┘  └──────┬───────┘  └────────────┬─────────────┘  │
│         │                  │                        │                 │
│         ▼                  ▼                        ▼                 │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │                     Event Bus                                 │    │
│  │  extension.activated · extension.deactivated · node_type.*   │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────────┐     │
│  │ Extension Proxy │  │ Plugin Manager │  │ Tengo Script Engine│     │
│  │ /ext/:slug/*   │  │ (gRPC)         │  │ (filters + hooks)  │     │
│  └────────────────┘  └────────────────┘  └────────────────────┘     │
└──────────────────────────────────────────────────────────────────────┘
```

---

## The Boot Manifest

**`GET /admin/api/boot`** — Single source of truth for the admin session.

Returned once on page load. Contains everything the Shell needs to bootstrap:
the current user, active extensions with their component declarations, the full
navigation tree, and all registered node types.

### Schema

```json
{
  "version": "1.0.0",
  "user": {
    "id": 1,
    "email": "admin@example.com",
    "full_name": "Admin",
    "role": "admin",
    "capabilities": {
      "manage_users": true,
      "manage_settings": true,
      "default_node_access": { "access": "write", "scope": "all" },
      "nodes": {
        "page": { "access": "write", "scope": "all" },
        "post": { "access": "write", "scope": "all" }
      }
    }
  },
  "extensions": [
    {
      "slug": "media-manager",
      "name": "Media Manager",
      "entry": "admin-ui/dist/index.js",
      "components": ["MediaLibrary", "Uploader", "MediaGrid"]
    }
  ],
  "navigation": [
    {
      "id": "nav-dashboard",
      "label": "Dashboard",
      "icon": "LayoutDashboard",
      "path": "/admin/dashboard"
    },
    {
      "id": "nav-content",
      "label": "Content",
      "icon": "Database",
      "section": "content",
      "children": [
        { "id": "nav-content-page", "label": "Pages", "icon": "FileText", "path": "/admin/content/page" },
        { "id": "nav-content-post", "label": "Posts", "icon": "FileText", "path": "/admin/content/post" }
      ]
    }
  ],
  "node_types": [
    {
      "slug": "post",
      "label": "Blog Post",
      "label_plural": "Posts",
      "icon": "FileText",
      "supports_blocks": true
    }
  ]
}
```

### Go Implementation

- **Engine:** `internal/sdui/engine.go` — `GenerateBootManifest(user)`
- **Handler:** `internal/api/boot_handler.go` — `GET /admin/api/boot`
- **React hook:** `admin-ui/src/hooks/use-boot.ts` — `useBoot()` (TanStack Query)

### Reactivity

When extensions are activated/deactivated or node types change, the SSE stream pushes
a `UI_STALE` event. The Shell's `useSSE` hook invalidates the `['boot']` query key,
causing `useBoot()` to refetch automatically.

---

## The Layout Tree

**`GET /admin/api/layout/:page`** — Returns a recursive JSON tree describing the page.

The Shell's `RecursiveRenderer` walks this tree, looks up each `type` in the
Component Registry, resolves prop bindings, and renders the corresponding React
component.

### Node Structure

```typescript
interface LayoutNode {
  type: string;                    // Component name to look up in registry
  props?: Record<string, unknown>; // Resolved props (supports $store.*, $params.*)
  children?: LayoutNode[];         // Nested child nodes
  actions?: Record<string, ActionDef>; // Named action handlers
}
```

### Example: Dashboard Page

```json
{
  "type": "VerticalStack",
  "props": { "gap": 6, "className": "p-6" },
  "children": [
    {
      "type": "AdminHeader",
      "props": { "title": "Dashboard" }
    },
    {
      "type": "DashboardWidgets",
      "props": {}
    }
  ]
}
```

### Example: List Page (Posts)

```json
{
  "type": "VerticalStack",
  "props": { "gap": 4 },
  "children": [
    {
      "type": "ListHeader",
      "props": {
        "title": "Posts",
        "newPath": "/admin/content/post/new"
      }
    },
    {
      "type": "ListToolbar",
      "props": { "searchPlaceholder": "Search posts..." }
    },
    {
      "type": "DataTable",
      "props": {
        "endpoint": "nodes",
        "nodeType": "post",
        "columns": []
      }
    }
  ]
}
```

### Example: Node Editor (SDUI for layout, React for the editor)

```json
{
  "type": "SidebarLayout",
  "props": { "sidebarWidth": 320 },
  "children": [
    {
      "type": "VerticalStack",
      "props": { "gap": 6, "className": "p-6" },
      "children": [
        {
          "type": "AdminHeader",
          "props": { "title": "Edit Post", "back": "/admin/content/posts" }
        },
        {
          "type": "NodeEditor",
          "props": {
            "nodeType": "post",
            "nodeId": "$params.id"
          },
          "actions": {
            "onSave": {
              "type": "SEQUENCE",
              "steps": [
                { "type": "CORE_API", "method": "nodes:update", "params": { "id": "$params.id" } },
                { "type": "TOAST", "message": "Post saved!" }
              ]
            },
            "onDelete": {
              "type": "SEQUENCE",
              "steps": [
                { "type": "CONFIRM", "message": "Delete this post?" },
                { "type": "CORE_API", "method": "nodes:delete", "params": { "id": "$params.id" } },
                { "type": "NAVIGATE", "to": "/admin/content/posts" }
              ]
            }
          }
        }
      ]
    },
    {
      "type": "VerticalStack",
      "props": { "gap": 4, "className": "p-4 bg-slate-50" },
      "children": [
        { "type": "NodeStatusPanel", "props": {} },
        { "type": "NodeLanguagePanel", "props": {} },
        { "type": "NodeFeaturedImage", "props": {} },
        { "type": "NodeTaxonomies", "props": {} },
        { "type": "NodeSEOPanel", "props": {} }
      ]
    }
  ]
}
```

The `NodeEditor`, `NodeStatusPanel`, etc. are Tier 3 CMS components — full React
components with their own complex state. The layout tree controls *where* they appear
and *what actions* fire on significant events, but doesn't dictate their internal UX.

### Prop Binding

Props can reference dynamic values using `$`-prefixed paths:

| Syntax | Resolves To |
|--------|------------|
| `$params.id` | URL parameter `:id` |
| `$store.search` | Page store key `search` |
| `$event.data` | Data from the triggering event |

### Go Implementation

- **Engine:** `internal/sdui/engine.go` — `GenerateLayout(pageSlug, params)`
- **Caching:** Layout trees are cached by page slug. Cache is cleared on any
  `extension.activated`, `extension.deactivated`, `node_type.*` event.
- **Filter chain:** TODO — `admin:layout:render` filter will allow extensions to
  modify layout trees before they're sent to the Shell.

---

## Action Handler

Actions are the declarative way to wire user interactions to side effects. The
Shell's action handler processes them sequentially.

### Supported Action Types

| Type | Purpose | Key Fields |
|------|---------|------------|
| `CORE_API` | Call a backend API endpoint | `method` (e.g. `"nodes:delete"`), `params` |
| `NAVIGATE` | Navigate to a new URL | `to` |
| `TOAST` | Show a notification | `message`, `variant` (`success`/`error`/`warning`) |
| `INVALIDATE` | Invalidate TanStack Query caches | `keys` (array of query keys) |
| `CONFIRM` | Show a confirmation dialog | `message` |
| `SET_STORE` | Update a page store value | `key`, `value` |
| `SEQUENCE` | Execute actions in order, stop on failure | `steps` (array of ActionDefs) |

### CORE_API Method Mapping

The `method` field uses the format `resource:action`:

| Method | HTTP | Endpoint |
|--------|------|----------|
| `nodes:list` | GET | `/nodes` |
| `nodes:get` | GET | `/nodes/{id}` |
| `nodes:create` | POST | `/nodes` |
| `nodes:update` | PATCH | `/nodes/{id}` |
| `nodes:delete` | DELETE | `/nodes/{id}` |

### SEQUENCE Execution

Sequences execute steps in order. If a `CONFIRM` step returns `false`
(user cancels), the sequence stops. If any step throws, the sequence stops
and an error toast is shown.

```json
{
  "type": "SEQUENCE",
  "steps": [
    { "type": "CONFIRM", "message": "Delete this post? This cannot be undone." },
    { "type": "CORE_API", "method": "nodes:delete", "params": { "id": "$params.id" } },
    { "type": "INVALIDATE", "keys": ["nodes"] },
    { "type": "TOAST", "message": "Post deleted" },
    { "type": "NAVIGATE", "to": "/admin/content/posts" }
  ]
}
```

### Implementation

- **React:** `admin-ui/src/sdui/action-handler.ts` — `executeActionDef(action, context)`
- The `setNavigate(fn)` function must be called from within a Router context to
  wire React Router's `navigate` for `NAVIGATE` actions.

---

## SSE Nervous System

**`GET /admin/api/events`** — Persistent Server-Sent Events stream.

The SSE stream is the real-time backbone. The Go Kernel's event bus pushes
relevant state changes to all connected admin clients.

### Event Types

| Event | Triggered By | Shell Reaction |
|-------|-------------|----------------|
| `CONNECTED` | Client connects | Log connection |
| `UI_STALE` | Extension activated/deactivated | Invalidate `boot` + `layout` queries |
| `NODE_TYPE_CHANGED` | Node type CRUD | Invalidate `boot` + `node-types` + `nodes` queries |
| `NOTIFY` | `notify` / `user.notification` events | Show toast notification |

### Go Implementation

- **Broadcaster:** `internal/sdui/broadcaster.go`
- Subscribes to event bus: `extension.activated`, `extension.deactivated`,
  `node_type.created`, `node_type.updated`, `node_type.deleted`
- Also subscribes to ALL events via `SubscribeAll` to catch `notify` events
- Uses buffered channels (capacity 10) per client; drops events on slow clients
- Uses Fiber's `SetBodyStreamWriter` for proper SSE streaming

### React Implementation

- **Hook:** `admin-ui/src/hooks/use-sse.ts` — `useSSE()`
- Creates `EventSource` with `withCredentials: true`
- Auto-reconnects on error with 3-second backoff
- Invalidates TanStack Query caches on state change events

---

## Component Registry

The registry maps type name strings to React components. The `RecursiveRenderer`
looks up components by the `type` field in layout tree nodes.

### Built-in Components (registered on app boot)

#### Layout Primitives (Tier 4)

| Name | Purpose |
|------|---------|
| `VerticalStack` | Flex column with configurable gap |
| `HorizontalStack` | Flex row with alignment |
| `Grid` | CSS grid with responsive columns |
| `SidebarLayout` | Two-panel layout (content + sidebar) |
| `TabLayout` | Tab-based layout with panels |
| `Section` | Wrapper with optional title |
| `ScrollRegion` | Scrollable container with max height |
| `Spacer` | Vertical spacing |
| `Divider` | Horizontal rule |

#### Page Composites (Tier 2)

| Name | Purpose |
|------|---------|
| `AdminHeader` | Page title + optional back button |
| `DashboardWidgets` | Dashboard stat card grid |
| `ListHeader` | Title + count badge + "New" button |
| `ListToolbar` | Search input + filter row |
| `DataTable` | Data-bound table (placeholder for now) |

#### UI Primitives (Tier 1)

| Name | Purpose |
|------|---------|
| `VibeButton` | Button with variant support |
| `TextBlock` | Plain text paragraph |
| `CardWrapper` | Styled card container |
| `StatCard` | Dashboard stat card |

#### Feedback (Tier 2)

| Name | Purpose |
|------|---------|
| `LoadingCard` | Skeleton loading placeholder |
| `EmptyState` | Icon + message + CTA |
| `ErrorCard` | Error message display |

### Registering Components

Built-in components are registered in `admin-ui/src/sdui/register-builtins.tsx`:

```typescript
import { registerComponents } from "./registry";

export function registerBuiltinComponents() {
  registerComponents({
    VerticalStack,
    HorizontalStack,
    Grid,
    // ... etc
  });
}
```

Extensions register components from their ESM bundles:

```typescript
// In extension's index.js
window.__VIBECMS_SHARED__.registerComponent("MediaLibrary", MediaLibrary);
```

Or they can be loaded lazily via `RemoteComponent`:

```json
{
  "type": "RemoteComponent",
  "props": {
    "extension": "media-manager",
    "component": "MediaLibrary",
    "context": { "folder": "posts" }
  }
}
```

---

## Page Store

A lightweight reactive key-value store scoped per page. Used for UI state that
needs to be shared between components on the same page (search terms, filter
values, modal open states).

### Usage in Layout Trees

Components can read from and write to the store via prop bindings:

```json
{
  "type": "SearchInput",
  "props": {
    "value": "$store.search",
    "placeholder": "Search..."
  },
  "actions": {
    "onChange": { "type": "SET_STORE", "key": "search", "value": "$event.value" }
  }
}
```

### Implementation

- **React:** `admin-ui/src/sdui/action-handler.ts` — `getPageStore(pageId)`, `clearPageStore(pageId)`
- Simple `Map<string, Map<string, unknown>>` — one inner map per page
- Store values are resolved in the `RecursiveRenderer` via `resolveProps()`

---

## How Extensions Participate

### Simple CRUD Page (no React code)

An extension that just needs a list/detail page returns layout JSON from
its gRPC plugin or Tengo script:

```json
{
  "type": "VerticalStack",
  "props": { "gap": 4, "className": "p-6" },
  "children": [
    { "type": "ListHeader", "props": { "title": "Form Submissions" } },
    {
      "type": "DataTable",
      "props": {
        "endpoint": "ext/forms/submissions",
        "columns": [
          { "key": "name", "label": "Name" },
          { "key": "email", "label": "Email" },
          { "key": "submitted_at", "label": "Date" }
        ]
      }
    }
  ]
}
```

Zero React code. The Shell renders it using built-in components.

### Complex Page (ships React component)

Extensions that need rich interactions ship an ESM bundle and the layout
tree references it:

```json
{
  "type": "RemoteComponent",
  "props": {
    "extension": "media-manager",
    "component": "MediaLibrary"
  }
}
```

### Modifying Existing Pages

Extensions hook into the `admin:layout:render` filter (TODO) to add, remove,
or modify nodes in existing layout trees. For example, the media-manager
extension could inject a `NodeFeaturedImage` panel into the node editor sidebar:

```go
// In the media-manager's Tengo hook or gRPC plugin:
filter.Apply("admin:layout:render", layoutTree)
// Append to sidebar children:
sidebar.Children = append(sidebar.Children, LayoutNode{
    Type: "RemoteComponent",
    Props: map[string]interface{}{
        "extension": "media-manager",
        "component": "NodeFeaturedImage",
    },
})
```

---

## API Reference

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/admin/api/boot` | Session | Boot manifest (user, extensions, nav, node types) |
| GET | `/admin/api/layout/:page` | Session | Layout tree for a page slug |
| GET | `/admin/api/events` | Session | SSE event stream |

### Query Parameters for Layout

```
GET /admin/api/layout/list?nodeType=post
GET /admin/api/layout/dashboard
GET /admin/api/layout/node-editor?nodeType=post&id=123
```

---

## File Structure

### Go Backend

```
internal/sdui/
├── types.go          # BootManifest, LayoutNode, ActionDef, SSEEvent structs
├── engine.go         # Layout generation, caching, navigation building
└── broadcaster.go    # SSE broadcaster wired to event bus

internal/api/
└── boot_handler.go   # GET /boot, GET /layout/:page handlers
```

### React Frontend

```
admin-ui/src/sdui/
├── types.ts              # TypeScript interfaces for all SDUI types
├── registry.ts           # Component registry (name → React component)
├── renderer.tsx           # RecursiveRenderer + LayoutRenderer
├── action-handler.ts     # Action execution engine + page store
├── query-client.ts       # Shared TanStack Query client
└── register-builtins.tsx # Built-in component registration

admin-ui/src/hooks/
├── use-boot.ts           # useBoot() — TanStack Query for boot manifest
├── use-sse.ts            # useSSE() — SSE connection + query invalidation
└── use-layout.ts         # useLayout(page) — TanStack Query for layout trees

admin-ui/src/components/
└── sdui-page.tsx         # Generic SDUI page component
```

---

## Implementation Status

### Done

- [x] Go SDUI types, layout engine, SSE broadcaster, boot + layout API endpoints.
- [x] TanStack Query integration, component registry, recursive renderer with `RemoteComponent`.
- [x] Action handler: `CORE_API`, `NAVIGATE`, `TOAST`, `INVALIDATE`, `CONFIRM`, `SEQUENCE`, `SET_STORE`.
- [x] SSE hook with auto-reconnect.
- [x] Built-in component library (~30 components: layout primitives, page composites, UI primitives, dashboard widgets, generic list table, extensions grid, themes grid).
- [x] Admin shell with sidebar + top bar driven by `useBoot()`.
- [x] ~40 pages ported to SDUI.
- [x] **Hardening pass complete** (commits `9f9239c`, `e53c2b3`):
  - Typed SSE taxonomy: `ENTITY_CHANGED(entity, id, op)`, `NAV_STALE`, `SETTING_CHANGED(key)`, `NOTIFY`, `UI_STALE`.
  - Broadcaster uses `SubscribeAll` + routes every emitted event (user, setting, menu, layout, layout_block, block_type, node, theme, taxonomy, role).
  - Query-key factory replaces ad-hoc keys.
  - Fine-grained invalidation: each SSE event → specific TanStack Query keys.
  - Bounded SSE per-client buffer (cap 32) with drop-on-full.
  - Sidebar filtered by capability (commit `e53c2b3`).

### Open

- [ ] shadcn `AlertDialog`-based `CONFIRM` action (replace `window.confirm`).
- [ ] Optimistic updates + consistent save toasts in the `CORE_API` action path.
- [ ] E2E coverage via `playwright-cli`.
- [ ] `DataProvider` + `QueryListener` components (Phase 2 data binding).
- [ ] `admin:layout:render` filter chain for extension-side layout injection (Phase 3).
- [ ] Extension layout JSON served from gRPC plugins.
- [ ] Layout tree versioning + server-side validation.
- [ ] Component prop TypeScript generation from Go structs.

---

## Design Decisions

### Why TanStack Query + lightweight store (not Zustand/Redux)?

TanStack Query handles the hard part of data fetching: caching, background
refetching, invalidation, and stale-while-revalidate. The page store only
needs to handle UI state (search terms, filter values, modal state). A simple
`Map` is sufficient for this. Adding Zustand would be over-engineering for
what is essentially a key-value bag.

### Why Actions instead of callbacks?

Sending raw JavaScript in JSON is a security risk and prevents the server from
reasoning about what the UI will do. Action Objects let the Go kernel control
the full lifecycle: "when the user clicks delete, confirm first, then call the
API, then navigate away." The Shell just executes the recipe.

### Why not SDUI for everything?

Complex interactive components (rich text editors, drag-and-drop trees,
code editors) have hundreds of state transitions per second. Encoding these
as JSON round-trips would be catastrophically slow. The right boundary is:
SDUI controls *layout orchestration* (where things go, what data flows between
them, what happens on major events), and React controls *interaction* (how
the user manipulates individual components).

### Why SSE instead of WebSockets?

SSE is simpler, works over HTTP/2, and is inherently unidirectional (server →
client). We don't need bidirectional communication — the client already sends
actions via REST API calls. SSE is also easier to proxy through CDNs and load
balancers.