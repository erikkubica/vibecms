# Squilla Admin Shell

The admin SPA is a **pure shell** — auth, sidebar, dashboard, and an
extension loader. Every feature page is rendered either from a Server-Driven
UI layout tree returned by `GET /admin/api/layout/:page`, or from an
extension micro-frontend loaded via import map. There is no per-feature page
logic in this build.

Stack: React 19, TypeScript, Vite, Tailwind v4, shadcn/ui, TanStack Query.

## Layout

```
src/
  api/                client.ts — typed fetch wrapper, language header injection
  components/
    layout/           admin-layout.tsx — top bar + sidebar + outlet
    ui/               design-system primitives (see below)
  hooks/              use-extensions, use-boot, use-sse
  lib/                extension-loader.ts — dynamic ES module + CSS injection
  pages/              one file per shell-owned page (login, node-editor, …)
  sdui/               admin-shell.tsx, renderer.tsx, table-components.tsx, …
                      — VDUS layout-tree walker + built-in component registry
  main.tsx            registers @squilla/ui exports onto window.__SQUILLA_SHARED__
                      so extension micro-frontends can resolve them at runtime
public/
  shims/              import-map shim files (squilla-ui.js, squilla-icons.js,
                      squilla-api.js) referenced by every extension's
                      vite.config.ts as externals
```

## Design system (v2)

The full primitive set lives in `src/components/ui/` and is re-exported on
`window.__SQUILLA_SHARED__.ui` so extensions get the same look without
re-implementing them. Key shared primitives, all consumable by extensions
via `import { Foo } from "@squilla/ui"`:

| Primitive | Role |
|---|---|
| `Titlebar` | Page header used by every editor (block/layout/menu/node/template/term/taxonomy) — title input + status pill + meta row + slot for actions |
| `PublishActions` | Publish/Save sidebar card used by every editor; uniform spacing, button styles, status feedback |
| `SidebarCard` | Editor-sidebar wrapper with consistent padding/heading/separator |
| `TabsCard` | Inset tab strip + transparent footer + content area used by every settings/editor split layout |
| `ListPage` (`ListPageShell`, `ListHeader`, `ListSearch`, `ListTable`, `ListFooter`, `EmptyState`, `Th`, `Tr`, `Td`, `SortableTh`, `Chip`, `StatusPill`, `TitleCell`, `RowActions`, `LoadingRow`) | Listing pages with URL-synced filters, sort, and ajax pagination. Backed by `?sort=<col>:<dir>&page=<n>` query params and shared with the SDUI list renderer. |
| `LanguageSelect`, `LanguageLabel`, `LanguagePicker` | Global language picker — flags removed in `8a28fa1`; uses native names |
| `MetaRow`, `MetaList` | Two-column key/value lists for "last modified", "created by", etc. |
| `PageHeader` | Listing page header used by every list page (title + actions slot, 1200px width clamp) |
| `SaveBar` | Sticky save bar used at the bottom of editors when changes are pending |

Built on top of shadcn/ui primitives (`Button`, `Card`, `Dialog`, `Tabs`,
`Switch`, `Select`, `Popover`, `Command`, `Table`, `DropdownMenu`,
`Checkbox`, `Separator`, `Input`, `Label`, `Textarea`, `Badge`,
`Sonner` toaster).

Form-schema primitives: `CustomFieldInput`, `FieldSchemaEditor`,
`FieldTypePicker`, `SubFieldsEditor`. Rich editing: `RichTextEditor`,
`CodeEditor`, `CodeViewer`, `CodeWindow`, `BlockPicker`.

## VDUS (Server-Driven UI)

`src/sdui/` walks layout trees emitted by the kernel
(`GET /admin/api/layout/:page`) and renders them through a component
registry (`register-builtins.tsx`). Action Objects produced by user
interactions go through the action handler — no inline `onClick` logic.

The shell subscribes to `/admin/api/events` (SSE) via `use-sse.tsx` and
invalidates TanStack Query keys on `ENTITY_CHANGED` / `NAV_STALE` /
`SETTING_CHANGED` events. Reload-free admin updates.

## Extension loader

`src/lib/extension-loader.ts` dynamically imports an extension's compiled
ES module (the entry path declared in its `extension.json` `admin.routes`).
The loader injects a sibling stylesheet (`<link rel="stylesheet">`) **before**
the admin shell stylesheet — order matters because both contribute to the
same `@layer utilities`, and source order wins on ties.

## Build

```bash
npm install
npm run dev          # Vite dev server
npm run build        # Production build → dist/
```

CSS uses Tailwind v4 (`@import "tailwindcss"` in `src/index.css`) with
v2 design tokens — the v1 indigo/slate palette was removed in commit
`54e2e66`.

## Conventions

- Default page width: **1200px clamp** on every listing and editor for visual
  consistency.
- Date formatting: relative ("3 hours ago") on lists, absolute on detail
  views.
- Toasts: top-center, single visible at a time, dismiss button, 2.2 s default
  (commit `61c406c`).
- Sort/page state on listings is URL-synced (`?sort=`, `?page=`,
  `?per_page=`) so refresh + back/forward Just Work and shareable.
