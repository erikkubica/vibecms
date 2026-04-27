# VDUS Hardening Plan

**Date:** 2026-04-25
**Branch:** `vdus`
**Status:** PENDING
**Worktree:** No

## Why

The `vdus` branch delivered the SDUI shell, ~40 pages ported to VDUS, the engine at 2,295 lines, and a SSE broadcaster. Manual smoke-test confirms rendering works. What does not work:

- **Reactivity is partial.** Broadcaster only subscribes to `extension.*` and `node_type.*`. Events for `user.*`, `setting.*`, `menu.*`, `layout.*`, `layout_block.*`, `block_type.*`, `node.*`, `theme.*`, `taxonomies:register` are emitted by the backend but never reach the client. Result: updating your username does not refresh the header until a manual page reload.
- **Invalidation is blunt.** `UI_STALE` invalidates `boot` + `layout`; `NODE_TYPE_CHANGED` invalidates `boot` + `node-types` + `nodes`. Nothing is keyed per-entity, so every event nukes more cache than needed and nothing updates list pages for non-node entities.
- **CONFIRM uses `window.confirm()`.** Native browser dialog looks out of place and blocks the thread.
- **No optimistic UI.** Every `CORE_API` save waits for the roundtrip, and there is no save-success toast in the generic action path. User-visible symptom: "I save and nothing visibly happens."

Fix the reactivity + feedback foundation before porting the remaining pages, so every future page inherits working invalidation and feedback instead of each one re-solving it.

## Outcomes

- Update username → header reflects new name within ~200 ms, no refresh.
- Delete / create / update anything → the list you are looking at updates without refresh.
- Toggle an extension → sidebar nav updates.
- Every `CORE_API` mutation shows a success toast on 2xx and an error toast on failure.
- `CONFIRM` action shows a real shadcn Dialog (destructive variant for deletes).
- Every cache key in admin-ui comes from a single factory — no stringly-typed `['foo', id]` scattered across 30 files.

## Design

### 1. Event taxonomy

One SSE event per backend event, typed. Payload carries `entity` + `id` (and `extra` for special cases).

```json
{ "type": "ENTITY_CHANGED", "entity": "user", "id": 42, "op": "updated" }
{ "type": "ENTITY_CHANGED", "entity": "node", "id": 17, "op": "deleted" }
{ "type": "NAV_STALE" }      // extension toggled, theme activated, node_type changed — sidebar needs rebuild
{ "type": "SETTING_CHANGED", "key": "site.title" }
```

`UI_STALE` stays as a coarse escape hatch for events we do not yet recognize, but we stop relying on it.

### 2. Backend wiring

`internal/sdui/broadcaster.go` subscribes to every published event at startup:

| Backend event | SSE output |
|---|---|
| `extension.activated`, `extension.deactivated`, `theme.activated`, `theme.deactivated` | `NAV_STALE` |
| `node_type.created/updated/deleted`, `taxonomies:register` | `NAV_STALE` + `ENTITY_CHANGED(node_type)` |
| `user.registered/updated/deleted/login` | `ENTITY_CHANGED(user)` |
| `node.created/updated/deleted/published/unpublished` | `ENTITY_CHANGED(node)` |
| `menu.created/updated/deleted` | `ENTITY_CHANGED(menu)` |
| `layout.*`, `layout_block.*`, `block_type.*` | `ENTITY_CHANGED(<entity>)` |
| `setting.updated` | `SETTING_CHANGED` with `key` in payload |

The payload `events.Payload` already carries `id` where relevant; we convert it to `SSEEvent.Data`.

### 3. Query key factory

`admin-ui/src/sdui/query-keys.ts`:

```ts
export const qk = {
  boot: () => ['boot'] as const,
  layout: (page: string, params?: Record<string, string>) => ['layout', page, params ?? {}] as const,
  list: (entity: string, filters?: Record<string, unknown>) => ['list', entity, filters ?? {}] as const,
  entity: (entity: string, id: string | number) => ['entity', entity, String(id)] as const,
  settings: () => ['settings'] as const,
};
```

Replace every hand-rolled `queryKey: [...]` in `admin-ui/src/**` and extension UIs with calls into `qk`.

### 4. SSE → invalidation map

`admin-ui/src/hooks/use-sse.ts`:

```ts
const routes: Record<string, (ev: SSEEvent) => QueryKey[]> = {
  NAV_STALE: () => [qk.boot(), ['layout']],
  SETTING_CHANGED: () => [qk.settings(), qk.boot()],
  ENTITY_CHANGED: (ev) => [
    qk.list(ev.entity),
    qk.entity(ev.entity, ev.id),
    // user + node_type also live in boot manifest:
    ...(ev.entity === 'user' || ev.entity === 'node_type' ? [qk.boot()] : []),
  ],
  UI_STALE: () => [['boot'], ['layout']],
};
```

Prefix-match invalidate (TanStack's default) means `qk.list(entity)` invalidates every filter variant.

### 5. CONFIRM dialog

Add `admin-ui/src/components/confirm-dialog.tsx` using shadcn `AlertDialog`. Expose `confirm({title, message, variant}) → Promise<boolean>` via a provider mounted in `main.tsx`. Rewrite the `CONFIRM` branch in `action-handler.ts` to await it.

### 6. Optimistic updates + toasts

In `action-handler.ts`, for every `CORE_API` action:

- If `method` is a mutation (write/update/delete), wrap in `queryClient.executeMutation`-style flow:
  - `onMutate`: snapshot `qk.list(entity)` + `qk.entity(entity, id)`, apply optimistic patch.
  - `onSuccess`: toast success (use action-provided `successMessage` or default "Saved.").
  - `onError`: rollback snapshot, toast the error body.
- Invalidation after success is handled by SSE — no need to double-invalidate.

## Tasks

1. **Plan + docs cleanup** (this file, archive old plans, reconcile VDUS_HANDOFF + vdus.md + CLAUDE.md).
2. **Event taxonomy**: extend `internal/sdui/types.go` (`SSEEvent` fields), rewrite `broadcaster.go` subscriptions.
3. **Query-key factory**: add `query-keys.ts`, migrate admin-ui call sites, then extensions.
4. **SSE routing**: rewrite `use-sse.ts` event handler.
5. **CONFIRM dialog**: add provider, rewrite action handler branch.
6. **Optimistic + toasts**: extend action handler with snapshot/rollback/toast.
7. **Verification**: playwright-cli E2E of username update + delete + extension toggle.

## Non-goals for this pass

- Porting the remaining legacy (non-VDUS) admin pages.
- Decomposing complex components (menu editor, template editor) further into VDUS.
- Phase 2 data binding components (`DataProvider`, `QueryListener`).
- Phase 3 extension filter chain.

Those come after this branch is merged.

## Done when

- [ ] `docker compose up -d` → login → change own username → header updates within one visible frame after the mutation resolves, no page refresh.
- [ ] Delete a node on the list page → row disappears immediately (optimistic), gets reconciled by SSE.
- [ ] Toggle extension → sidebar adds/removes its menu entry within 500 ms of the toggle call.
- [ ] Every save shows a toast.
- [ ] Every delete goes through the shadcn dialog.
- [ ] `grep -r "window.confirm" admin-ui/src extensions/*/admin-ui/src` → no matches.
- [ ] `grep -rn "queryKey:" admin-ui/src | wc -l` decreases to the call sites that use `qk.*`.
