# VDUS Handoff

**Last updated:** 2026-04-25
**Status:** Phase 1 shell done. ~40 admin pages ported to VDUS. Hardening pass in progress.
**Branch:** `vdus`
**Stack:** Go 1.24+ / Fiber v2 / React 19 / TanStack Query v5 / TypeScript / Tailwind 4

> For the design rationale, architecture diagrams, API schemas, and component catalog see **`docs/vdus.md`** — this file is the delta between "what the design doc describes" and "what's actually in the repo right now", plus the list of known rough edges.

---

## What works today

- `GET /admin/api/boot` returns user, extensions, nav tree, node types, taxonomies.
- `GET /admin/api/layout/:page` returns layout trees for every ported page (dashboard, list pages for nodes / block-types / layouts / layout-blocks / menus / templates / taxonomies / terms / users / roles / languages / mcp-tokens / site-settings / extensions / themes / extension-files / theme-files).
- `GET /admin/api/events` pushes a persistent SSE stream with heartbeats every 15 s and auto-reconnects client-side.
- Admin shell (`admin-ui/src/sdui/admin-shell.tsx`, 588 lines) renders the sidebar + top bar from `useBoot()`.
- Recursive renderer walks layout trees, resolves `$store.*` / `$params.*` bindings, dispatches actions.
- Extensions load via import maps and can register React components into the shared registry.

## What's half-done (tracked in `docs/plans/2026-04-25-vdus-hardening.md`)

1. **SSE only covers extension + node-type events.** Changes to users, settings, menus, layouts, layout-blocks, block-types, nodes, themes, taxonomies are published by the backend but never bridged to SSE, so the UI does not react to them. The hardening plan adds a typed `ENTITY_CHANGED` / `NAV_STALE` / `SETTING_CHANGED` taxonomy.
2. **Invalidation is blunt.** `UI_STALE` nukes `boot` + `layout`, `NODE_TYPE_CHANGED` nukes `boot` + `node-types` + `nodes`. The plan introduces a query-key factory and a precise event → keys map.
3. **`CONFIRM` action uses `window.confirm()`.** Plan swaps in a shadcn `AlertDialog` provider.
4. **No optimistic updates, inconsistent save toasts.** Plan centralizes mutation flow in the action handler with snapshot/rollback + success/error toasts.

## File inventory (current)

### Go (`internal/sdui/`)

| File | Lines | Purpose |
|---|---|---|
| `types.go` | 84 | `BootManifest`, `BootUser`, `BootExt`, `NavItem`, `BootNodeType`, `LayoutNode`, `ActionDef`, `SSEEvent`. Will grow to add typed SSE payload fields. |
| `engine.go` | 2,295 | Layout engine + all per-page layout generators + cache. Grew significantly from the initial 481-line stub as pages were ported. |
| `broadcaster.go` | 192 | SSE broadcaster. Currently only subscribes to `extension.*` and `node_type.*` — this is the main thing the hardening pass expands. |

`internal/api/boot_handler.go` — Fiber routes: `GET /boot`, `GET /layout/:page`.

### React (`admin-ui/src/sdui/`)

| File | Lines | Purpose |
|---|---|---|
| `types.ts` | 78 | TS mirror of Go SDUI types |
| `registry.ts` | 35 | Component registry Map |
| `renderer.tsx` | 248 | `RecursiveRenderer`, `LayoutRenderer`, `RemoteComponent` |
| `action-handler.ts` | 270 | `CORE_API`, `NAVIGATE`, `TOAST`, `INVALIDATE`, `CONFIRM`, `SET_STORE`, `SEQUENCE`. Target of optimistic-update work. |
| `query-client.ts` | 11 | Shared `QueryClient` |
| `register-builtins.tsx` | 562 | Registers ~30 built-in components |
| `sdui-components.tsx` | 366 | Dashboard composites (WelcomeBanner, StatCard, RecentContentTable, ActivityFeed, QuickActions) |
| `admin-shell.tsx` | 588 | Full admin chrome |
| `generic-list-table.tsx` | 356 | Generic list-page table component used by most list layouts |
| `table-components.tsx` | 810 | Cell renderers + table primitives |
| `list-components.tsx` | 204 | List page primitives |
| `extensions-grid.tsx` | 292 | Extensions page |
| `themes-grid.tsx` | 492 | Themes page |

### React hooks (`admin-ui/src/hooks/`)

- `use-boot.ts` — `useBoot()` → `GET /admin/api/boot`, 60 s stale.
- `use-layout.ts` — `useLayout(page)`.
- `use-sse.ts` — SSE connection + query invalidation. Target of event-routing rewrite.

### SDUI-backed pages (`admin-ui/src/pages/sdui-*.tsx`)

Dashboard, block-types (+editor), content-types, extensions, extension-files, languages, layout-blocks (+editor), layouts (+editor), mcp-tokens, menus (+editor), node editor/list, node-type-editor, role-editor, roles, site-settings, taxonomies (+editor), taxonomy-terms, template-editor, templates, term-editor, themes, theme-files, user-editor, users. (Non-`sdui-*` pages in `admin-ui/src/pages/` still exist as fallbacks during the port.)

## Critical implementation notes

These are non-obvious gotchas that caused real bugs during development — do not regress them:

1. **`defer b.Unsubscribe(ch)` must be INSIDE `SetBodyStreamWriter`'s callback**, not at the handler level. `SetBodyStreamWriter` is asynchronous in fasthttp — the handler function returns immediately, so a handler-level defer unsubscribes before the stream has even started.
2. **Never close SSE channels on unsubscribe.** The writer exits via write error or heartbeat; channel is garbage-collected. Double-close panics fasthttp.
3. **Buffered SSE channels (cap 32).** Events drop on slow clients rather than blocking the publisher.
4. **`use-sse.ts` must `useRef(null)` with an initial value** — React 19 strict mode rejects ref-less calls.
5. **`queryClient` is imported directly from `sdui/query-client.ts` inside hooks, not via `useQueryClient()`** — the hook path broke the Docker TS build.
6. **Dashboard layout is uncached.** Every other layout caches until a relevant event fires. Do not add caching to dashboard without handling DB-stat freshness.
7. **Tailwind `gap:` values are utility numbers (`gap: 6` → `gap-6` → 24 px), not pixels.** Do not pass raw pixels.

## Testing commands

```bash
# Build + run
docker compose up -d

# Login + grab session
COOKIE=$(curl -s -c - http://localhost:8099/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"admin@vibecms.local","password":"admin123"}' \
  | grep vibecms_session | awk '{print $NF}')

# Inspect boot + layout
curl -s -b "vibecms_session=$COOKIE" http://localhost:8099/admin/api/boot | jq .
curl -s -b "vibecms_session=$COOKIE" http://localhost:8099/admin/api/layout/dashboard | jq .

# Watch SSE for 35s (confirms heartbeats + events)
curl -s -N --max-time 35 -b "vibecms_session=$COOKIE" http://localhost:8099/admin/api/events

# Compile checks
go build ./...
cd admin-ui && npx tsc -b
```

**Login:** `admin@vibecms.local` / `admin123`

## Pointer

If you are about to port another page or add a new SDUI component, read `docs/vdus.md` sections "The Boot Manifest", "Layout Trees", "Component Registry", "Actions". Then open `docs/plans/2026-04-25-vdus-hardening.md` so you know what interfaces are about to change (query keys, SSE payload shape, action-handler signature) and do not build on top of soon-to-be-deprecated patterns.
