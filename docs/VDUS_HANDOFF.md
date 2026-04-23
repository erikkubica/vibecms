# VDUS Handoff Document

**Date:** 2025-04-23
**Status:** Phase 1 Complete — Dashboard fully SDUI-driven with admin chrome
**Stack:** Go 1.24+ / Fiber v2 / React 19 / TanStack Query v5 / TypeScript / Tailwind 4

---

## What Is VDUS

VDUS (VibeCMS Dynamic UI System) is a Server-Driven UI architecture. The Go Kernel generates JSON layout trees describing admin pages. The React Shell is a rendering engine that walks those trees and renders registered components. Extensions participate by contributing components and modifying layout trees through filter hooks.

The dashboard at `/admin/dashboard` is fully SDUI-driven: sidebar, header, welcome banner, stat cards, and recent content table are all generated from Go and rendered by the recursive renderer.

---

## Architecture Diagram

```
Browser                                        Go Kernel (Fiber)
───────                                        ─────────────────

  React Shell
  ┌──────────────────┐
  │  useBoot()       │──GET /admin/api/boot──→  SDUI Engine
  │  TanStack Query  │                          ├─ GenerateBootManifest()
  │                  │                          └─ DB queries (user, exts, node types, taxonomies)
  │  useSSE()        │──GET /admin/api/events─→ SSE Broadcaster
  │  EventSource     │←──push UI_STALE────────  └─ wired to EventBus
  │                  │←──push heartbeat────────
  │                  │
  │  useLayout(page) │──GET /admin/api/layout/dashboard──→ SDUI Engine
  │  TanStack Query  │←──JSON layout tree────────────────  └─ GenerateLayout()
  │                  │
  │  RecursiveRenderer                        │
  │  ├─ ComponentRegistry                     │
  │  ├─ ActionHandler                         │
  │  ├─ RemoteComponent (extensions)          │
  │  └─ PageStore                             │
  └──────────────────┘
```

---

## API Endpoints

### `GET /admin/api/boot` (Session Auth Required)

Single source of truth for the admin session. Returns user info, active extensions with their component declarations, full navigation tree (flat list with `is_section` markers), and all registered node types.

```json
{
  "data": {
    "version": "1.0.0",
    "user": { "id": 1, "email": "admin@vibecms.local", "full_name": "Admin", "role": "admin", "capabilities": { "manage_users": true } },
    "extensions": [
      { "slug": "media-manager", "name": "Media Manager", "entry": "admin-ui/dist/index.js", "components": ["MediaLibrary"] }
    ],
    "navigation": [
      { "id": "nav-dashboard", "label": "Dashboard", "icon": "LayoutDashboard", "path": "/admin/dashboard" },
      { "id": "section-content", "label": "Content", "is_section": true },
      { "id": "nav-content-page", "label": "Pages", "icon": "FileText", "path": "/admin/pages" },
      { "id": "nav-content-gallery_photo", "label": "Gallery Photo", "icon": "image", "path": "/admin/content/gallery_photo", "children": [
        { "id": "nav-content-gallery_photo-all", "label": "All Gallery Photo", "icon": "image", "path": "/admin/content/gallery_photo" },
        { "id": "nav-content-gallery_photo-tax-gallery_category", "label": "Gallery Category", "icon": "Tags", "path": "/admin/content/gallery_photo/taxonomies/gallery_category" }
      ]},
      { "id": "section-design", "label": "Design", "is_section": true },
      { "id": "nav-ext-email-manager", "label": "Email", "icon": "Mail", "children": [...] }
    ],
    "node_types": [
      { "slug": "post", "label": "Post", "label_plural": "Posts", "icon": "newspaper", "supports_blocks": true }
    ]
  }
}
```

**Navigation structure rules:**
- `is_section: true` = non-clickable section header (Content, Design, Development, Settings)
- Items with `path` only = flat clickable links
- Items with `children` = expandable dropdown. First child is always "All {Type}" linking to the main listing. Remaining children are taxonomy links.
- Extension items without children get a path pointing to `/admin/ext/{slug}/`
- Node type paths: page→`/admin/pages`, post→`/admin/posts`, custom→`/admin/content/{slug}`
- Labels use `label_plural` with fallback to `label`

### `GET /admin/api/layout/:page` (Session Auth Required)

Returns the SDUI layout tree for a page. Dashboard is uncached (live DB stats). Other pages are cached until state-change events fire.

```
GET /admin/api/layout/dashboard
GET /admin/api/layout/list?nodeType=post
```

Dashboard response (real DB data):
```json
{
  "data": {
    "type": "VerticalStack",
    "props": { "gap": 6, "className": "p-6" },
    "children": [
      { "type": "WelcomeBanner", "props": { "title": "Welcome back, Admin", "subtitle": "...", "actionLabel": "Create New Page", "actionPath": "/admin/pages/new" } },
      { "type": "Grid", "props": { "cols": 4, "gap": 4 }, "children": [
        { "type": "StatCard", "props": { "label": "Total Content", "value": "32", "icon": "FileText", "color": "indigo" } },
        { "type": "StatCard", "props": { "label": "Published", "value": "32", "icon": "Eye", "color": "emerald" } },
        { "type": "StatCard", "props": { "label": "Drafts", "value": "0", "icon": "PenLine", "color": "amber" } },
        { "type": "StatCard", "props": { "label": "Users", "value": "1", "icon": "Users", "color": "violet" } }
      ]},
      { "type": "RecentContentTable", "props": { "items": [
        { "id": 43, "title": "Home", "node_type": "page", "status": "published", "updated_at": "2025-04-22" }
      ]}}
    ]
  }
}
```

**Gap values are Tailwind gap utilities:** `gap: 6` = `gap-6` = 1.5rem (24px). Do NOT use raw pixel values.

### `GET /admin/api/events` (Session Auth Required, SSE)

Server-Sent Events stream. Keeps alive with `: heartbeat` comments every 15 seconds.

Event types pushed to client:
- `CONNECTED` — sent on connection
- `UI_STALE` — extension activated/deactivated → Shell invalidates TanStack `boot` and `layout` queries
- `NODE_TYPE_CHANGED` — node type CRUD → Shell invalidates `boot`, `node-types`, `nodes` queries
- `NOTIFY` — push toast notification to UI

---

## File Inventory

### Go Backend (3 new files + 1 handler)

| File | Lines | Purpose |
|------|-------|---------|
| `internal/sdui/types.go` | 84 | Go structs: `BootManifest`, `BootUser`, `BootExt`, `NavItem` (with `IsSection`), `BootNodeType`, `LayoutNode`, `ActionDef`, `SSEEvent` |
| `internal/sdui/engine.go` | 481 | Layout engine: `GenerateBootManifest(user)`, `GenerateLayout(pageSlug, params, userName)`, `buildNavigation(nodeTypes, taxonomies, exts)`, `dashboardLayout(userName)` (live DB queries), `listLayout(nodeType)`, `defaultLayout(pageSlug)`. Cache invalidated on extension/node-type events. Dashboard skips cache. |
| `internal/sdui/broadcaster.go` | 192 | SSE broadcaster wired to EventBus. Buffered channels per client (capacity 32). Heartbeat ticker at 15s. Cleanup inside `SetBodyStreamWriter` callback (NOT defer at handler level — `SetBodyStreamWriter` is async in fasthttp). |
| `internal/api/boot_handler.go` | 73 | Fiber handlers: `GET /boot`, `GET /layout/:page`. Extracts user from `c.Locals("user")` (avoids import cycle with `auth` package). |

### Modified Go Files

| File | Change |
|------|--------|
| `cmd/vibecms/main.go` | Lines 66-68: Create `sduiEngine` + `sduiBroadcaster`. Line 131: Create `bootHandler`. Lines 222-224: Register routes `bootHandler.RegisterRoutes(adminAPI)` and `adminAPI.Get("/events", sduiBroadcaster.Handler())`. |

### React Frontend (9 new files)

| File | Lines | Purpose |
|------|-------|---------|
| `admin-ui/src/sdui/types.ts` | 78 | TypeScript interfaces: `BootManifest`, `BootUser`, `BootExtension`, `NavItem` (with `is_section`), `BootNodeType`, `LayoutNode`, `ActionDef`, `SSEEventData` |
| `admin-ui/src/sdui/registry.ts` | 35 | `Map<string, ComponentType<any>>` with `registerComponent`, `getComponent`, `registerComponents`. Uses `any` type so the recursive renderer can pass arbitrary props. |
| `admin-ui/src/sdui/renderer.tsx` | 248 | `RecursiveRenderer` + `LayoutRenderer`. Walks layout tree, resolves `$store.*`/`$params.*` prop bindings, looks up components in registry, handles `RemoteComponent` for extension ESM bundles. Uses `React.createElement()` to bypass strict typing. Falls back to `VibeErrorCard` for unknown types. |
| `admin-ui/src/sdui/action-handler.ts` | 212 | `executeActionDef(action, context)` — processes CORE_API (with method→endpoint→HTTP mapping), NAVIGATE, TOAST, INVALIDATE, CONFIRM, SET_STORE, SEQUENCE (stops on CONFIRM cancel or error). Page store: `getPageStore(pageId)` / `clearPageStore(pageId)`. `setNavigate(fn)` must be called from inside Router context. |
| `admin-ui/src/sdui/query-client.ts` | 11 | Shared TanStack `QueryClient` instance (30s stale time, no refetch on focus, 1 retry). Imported directly by `use-sse.ts` instead of using `useQueryClient()` hook to avoid Docker build type issues. |
| `admin-ui/src/sdui/register-builtins.tsx` | 537 | Registers ~20 built-in components: layout primitives (VerticalStack, HorizontalStack, Grid, SidebarLayout, TabLayout, Section, ScrollRegion, Spacer, Divider), page composites (AdminHeader, DashboardWidgets→delegates to StatCard, ListHeader, ListToolbar, DataTable), UI primitives (VibeButton, TextBlock, CardWrapper, StatCard), feedback (LoadingCard, EmptyState, ErrorCard). |
| `admin-ui/src/sdui/sdui-components.tsx` | 354 | Dashboard-specific components: `WelcomeBanner` (gradient hero), `StatCard` (with colored icon backgrounds, Lucide icon lookup by string name), `RecentContentTable` (clickable rows with status badges), `ActivityFeed`, `QuickActions`. Exports shared `iconMap` for reuse. |
| `admin-ui/src/sdui/admin-shell.tsx` | 517 | Full admin page shell with: dark sidebar (bg-[#0f172a], 256px, collapsible), responsive overlay on mobile, navigation from `useBoot()` (flat list with `is_section` headers), icon mapping (30+ Lucide icons), expandable groups with chevron, active route highlighting, user info + logout in footer, top bar with auto-computed breadcrumbs, SSE via `useSSE()`. |
| `admin-ui/src/hooks/use-boot.ts` | 17 | TanStack Query hook: `useBoot()` → `GET /admin/api/boot`, 60s stale time. |
| `admin-ui/src/hooks/use-layout.ts` | 18 | TanStack Query hook: `useLayout(page)` → `GET /admin/api/layout/:page`. |
| `admin-ui/src/hooks/use-sse.ts` | 76 | SSE connection with auto-reconnect (3s backoff). On `UI_STALE`/`NODE_TYPE_CHANGED` → invalidates TanStack Query caches. Imports `queryClient` directly (not via hook). `useRef` requires initial value in React 19 strict mode. |
| `admin-ui/src/components/sdui-page.tsx` | 65 | Generic SDUI page with boot debug strip + layout renderer. Used by `/admin/sdui/:page` test route. |
| `admin-ui/src/pages/sdui-dashboard.tsx` | 26 | Dashboard page: `SduiAdminShell` wrapping `useLayout("dashboard")` + `LayoutRenderer`. |

### Modified React Files

| File | Change |
|------|--------|
| `admin-ui/src/main.tsx` | Added `QueryClientProvider`, `NavigateBridge` (wires `useNavigate` into action handler), `SduiProviders` (calls `useSSE`), `registerBuiltinComponents()` call at module level. |
| `admin-ui/src/App.tsx` | Added `/admin/dashboard` as top-level protected route → `SduiDashboardPage`. Nested `dashboard` route redirects to top-level. Added `/admin/sdui/:page` test route. |

### New Dependency

| Package | Version | Purpose |
|---------|---------|---------|
| `@tanstack/react-query` | ^5.99.2 | Data fetching, caching, invalidation for boot manifest + layout trees |

---

## How the Dashboard Works End-to-End

1. User navigates to `/admin/dashboard`
2. `SduiDashboardPage` renders inside `ProtectedRoute`
3. `SduiAdminShell` calls `useBoot()` → `GET /admin/api/boot` → TanStack caches the manifest
4. Sidebar renders from `boot.navigation` — section headers, node types, taxonomies, extension items
5. Content area calls `useLayout("dashboard")` → `GET /admin/api/layout/dashboard`
6. Go engine queries DB for stats (total/published/draft nodes, users, recent 5 nodes)
7. Go builds layout tree: WelcomeBanner → Grid of StatCards → RecentContentTable
8. `RecursiveRenderer` walks the tree, looks up each `type` in ComponentRegistry
9. Components render with resolved props (stat values come from Go, rendered by StatCard)
10. SSE stream connects → heartbeat every 15s → cache invalidation on state changes

---

## Component Registry (Built-in Components)

All registered in `register-builtins.tsx` and `sdui-components.tsx`:

### Layout (Tier 4)
`VerticalStack`, `HorizontalStack`, `Grid`, `SidebarLayout`, `TabLayout`, `Section`, `ScrollRegion`, `Spacer`, `Divider`

### Page Composites (Tier 2)
`AdminHeader`, `DashboardWidgets`, `ListHeader`, `ListToolbar`, `DataTable`

### UI Primitives (Tier 1)
`VibeButton`, `TextBlock`, `CardWrapper`, `StatCard`

### Dashboard Specific
`WelcomeBanner`, `RecentContentTable`, `ActivityFeed`, `QuickActions`

### Feedback
`LoadingCard`, `EmptyState`, `ErrorCard`

---

## SSE Broadcaster Implementation Notes

**Critical:** `SetBodyStreamWriter` in fasthttp is asynchronous. The handler function returns immediately after registering the callback. Therefore:

- `defer b.Unsubscribe(ch)` MUST live INSIDE the `SetBodyStreamWriter` callback, not at the handler level
- Heartbeat ticker sends `: heartbeat\n\n` every 15 seconds to keep TCP alive
- Channels are NOT closed on unsubscribe — the writer exits via write error or heartbeat, channel is GC'd
- Buffered channels (capacity 32) — events dropped on slow clients

---

## Action Handler

Maps JSON action objects to JavaScript behavior:

| Type | Behavior | Key Fields |
|------|----------|------------|
| `CORE_API` | Fetch to `/admin/api/{endpoint}` | `method` (e.g. `"nodes:delete"` → `DELETE /nodes/{id}`), `params` |
| `NAVIGATE` | `navigate(to)` via React Router | `to` |
| `TOAST` | `sonner` toast | `message`, `variant` (success/error/warning) |
| `INVALIDATE` | TanStack Query cache invalidation | `keys` (query key array) |
| `CONFIRM` | `window.confirm()` (will be shadcn Dialog later) | `message` |
| `SET_STORE` | Update page store key | `key`, `value` |
| `SEQUENCE` | Execute steps in order, stop on CONFIRM cancel or error | `steps` (array of ActionDefs) |

Prop binding: `$params.id` resolves to URL param, `$store.search` resolves to page store value.

---

## Known Issues / TODO

### Must Fix Next
- [ ] `CONFIRM` action uses `window.confirm()` — replace with shadcn Dialog component
- [ ] `DashboardWidgets` component currently delegates to StatCard — needs to actually use boot data for the old non-SDUI dashboard too
- [ ] `SduiAdminShell` sidebar does not show the old admin's "Visit Site" link correctly (it's there but may need the actual site URL from settings)

### Phase 2: Data Binding
- [ ] `DataProvider` component — fetches from API endpoint, provides data to children via context
- [ ] `QueryListener` component — watches URL params, fetches data, updates store
- [ ] `DataTable` full implementation — columns with sorting, pagination, row actions, empty state
- [ ] Reactive page store with subscribers (currently a plain `Map`)

### Phase 3: Extension Integration
- [ ] `admin:layout:render` filter chain in Go — let extensions modify layout trees
- [ ] Extension layout JSON serving from gRPC plugins
- [ ] `RemoteComponent` entry path resolution from extension manifest (currently hardcoded to `index.js`)
- [ ] Update hello-extension to return layout JSON and demonstrate the full loop

### Phase 4: Page Migration
- [ ] Migrate list pages (posts, pages, custom types) to SDUI
- [ ] Migrate settings page to SDUI
- [ ] Migrate user/role management to SDUI
- [ ] Evaluate node editor migration (most complex — keep as React component)

### Phase 5: Polish
- [ ] Layout tree versioning for smarter cache invalidation
- [ ] Server-side layout tree validation
- [ ] Optimistic updates for CORE_API actions
- [ ] Component prop TypeScript generation from Go structs
- [ ] Remove `SduiPage` debug route (`/admin/sdui/:page`) once all pages are migrated
- [ ] Remove old `DashboardPage` import if no longer used

---

## Testing Commands

```bash
# Build and run
docker compose build app && docker compose up -d app

# Test boot manifest
COOKIE=$(curl -s -c - http://localhost:8099/auth/login -H 'Content-Type: application/json' -d '{"email":"admin@vibecms.local","password":"admin123"}' | grep vibecms_session | awk '{print $NF}')
curl -s -b "vibecms_session=$COOKIE" http://localhost:8099/admin/api/boot | python3 -m json.tool

# Test layout tree
curl -s -b "vibecms_session=$COOKIE" http://localhost:8099/admin/api/layout/dashboard | python3 -m json.tool

# Test SSE (hold for 35 seconds to see heartbeats)
curl -s -N --max-time 35 -b "vibecms_session=$COOKIE" http://localhost:8099/admin/api/events

# Check Go compilation
go build ./...

# Check TypeScript compilation (strict mode with tsc -b)
cd admin-ui && npx tsc -b
```

**Login:** `admin@vibecms.local` / `admin123`
**Dashboard:** http://localhost:8099/admin/dashboard
**SDUI test route:** http://localhost:8099/admin/sdui/dashboard (generic, no sidebar)