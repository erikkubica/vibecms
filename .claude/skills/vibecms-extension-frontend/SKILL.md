---
name: vibecms-extension-frontend
description: |
  How to build, style, and debug VibeCMS extension admin-UI micro-frontends.
  Use when: (1) building/editing files under `extensions/*/admin-ui/src/**`,
  (2) Tailwind classes silently have no effect in an extension UI (e.g. `pt-3`,
  `gap-x-6`, `sm:grid-cols-2`, `md:col-span-2` produce no visible style),
  (3) extension UI looks correct in `npm run dev` but broken after Docker rebuild,
  (4) hot-deploying a compiled extension into a running container without
  rebuilding the image, (5) any change to shared `@vibecms/ui` primitives
  (Switch, Select, Tabs, etc.) so changes get picked up CMS-wide,
  (6) shipping an extension that uses common Tailwind utilities like `.fixed`,
  `.absolute`, `.grid` and the admin-ui shell layout suddenly breaks
  (sidebar overlapping main, wrong column counts on themes/extensions pages),
  (7) accessing react-router-dom hooks (useSearchParams, useNavigate) or
  sonner toast from inside an extension, (8) building a list page that should
  match the design language of nodes/forms (tabs with counts, search,
  pagination footer).
author: Claude Code
version: 1.1.0
---

# VibeCMS Extension Frontend Build & Styling

## Architecture (one paragraph)

The admin SPA (`admin-ui/`) is a Vite+React+Tailwind v4 shell that loads
extension micro-frontends (`extensions/<slug>/admin-ui/`) as ES modules at
runtime. Extensions externalize `react`, `react-dom`, `react-router-dom`,
`sonner`, `@vibecms/ui`, `@vibecms/api`, `@vibecms/icons` — these are
provided at runtime via `window.__VIBECMS_SHARED__` (set up in
`admin-ui/src/main.tsx`). Extensions ship their own JS **and CSS** from
`dist/`, served by the Go binary at `/admin/api/extensions/<slug>/assets/*`.

## Critical Rule: Each Extension Owns Its Tailwind Build

**Do NOT rely on `admin-ui/src/index.css`'s `@source "../../extensions/*/admin-ui/src/**/*.{ts,tsx}"`** — it works locally but silently fails in Docker because the `frontend` stage in `Dockerfile` only copies `admin-ui/`, not `extensions/`. Tailwind scans nothing extension-related, classes get dropped, layouts collapse.

**Every extension admin-ui MUST:**

1. Have `tailwindcss` + `@tailwindcss/vite` in `devDependencies`.
2. Add `tailwindcss()` to `vite.config.ts` plugins, set `build.lib.cssFileName: "index"`:
   ```ts
   import tailwindcss from "@tailwindcss/vite";
   export default defineConfig({
     plugins: [react(), tailwindcss()],
     build: {
       lib: { entry: "src/index.tsx", formats: ["es"], fileName: "index", cssFileName: "index" },
       rollupOptions: { external: ["react", "react/jsx-runtime", "react-dom", "react-dom/client", "react-router-dom", "sonner", "@vibecms/ui", "@vibecms/api", "@vibecms/icons"] },
       cssCodeSplit: false,
     },
   });
   ```
3. Have `src/index.css` with **only** scanner directives — admin-ui ships the design tokens, base styles, and `@vibecms/ui` `data-slot` overrides:
   ```css
   @import "tailwindcss";
   @source "./**/*.{ts,tsx}";
   ```
4. Import it from the entry: `import "./index.css";` in `src/index.tsx` (top of file, before component imports).

Build emits `dist/index.css` next to `dist/index.js`. The extension loader (`admin-ui/src/lib/extension-loader.ts`) auto-injects `<link rel="stylesheet" href=".../index.css">` when loading the JS — extensions that ship no CSS get a harmless 404.

## Diagnostic Flow: "Tailwind class has no effect"

When a class like `gap-x-6`, `pt-3`, `md:col-span-2`, or `lg:grid-cols-3` produces no styling:

1. **Locate the file using the class.** If it's under `extensions/*/admin-ui/src/`, the extension owns its CSS — check the extension's `dist/index.css` (NOT admin-ui's).
2. **Check the compiled CSS:**
   ```bash
   ESC=$(echo "$CLASS" | sed 's/:/\\\\:/g')
   grep -c "\.${ESC}{" extensions/<slug>/admin-ui/dist/index.css
   ```
   - `1` → CSS exists, problem is browser cache or stale container — `docker cp` fresh assets (see hot-deploy section).
   - `0` → Tailwind scanner missed the file. Verify (a) extension's `vite.config.ts` includes `tailwindcss()` plugin, (b) `src/index.css` `@source` glob covers the file, (c) `src/index.tsx` imports `./index.css`.
3. **NEVER patch with inline `style={{}}` to "fix" a missing class.** The fix is making Tailwind scan correctly. Inline styles hide the underlying build bug and rot.

## CSS Cascade: Extension Stylesheets vs Admin-UI Shell

**THE bug that breaks every admin page when you add Tailwind to an extension.**

When an extension ships its own Tailwind utilities and is loaded into the running admin app, its stylesheet is appended to `<head>` *after* admin-ui's stylesheet. Tailwind v4 puts both extension and admin-ui utilities in the same `@layer utilities`. Within a merged layer, **source order wins**. So the extension's `.fixed { position: fixed }` (used by drawers/modals) sits later in the cascade than admin-ui's `.lg\:relative { position: relative }` and beats it at the `lg` breakpoint.

Real-world symptom: `<aside class="fixed inset-y-0 left-0 ... lg:relative lg:z-auto">` (admin-ui sidebar) becomes permanently `position: fixed` on desktop too. Main content has no left margin to compensate, sidebar overlaps it, every responsive grid (themes 4-col, extensions 3-col) collapses because admin-ui's `lg:grid-cols-N` lost the cascade fight.

**Fix already wired in `admin-ui/src/lib/extension-loader.ts`:**

```ts
// Insert BEFORE the first <link rel="stylesheet"> so admin-ui stays later in
// source order and wins the cascade.
const firstLink = document.head.querySelector('link[rel="stylesheet"]');
if (firstLink) document.head.insertBefore(link, firstLink);
else document.head.insertBefore(link, document.head.firstChild);
```

**Why this matters for extension developers:** if you touch the extension loader, never `appendChild` the extension stylesheet. If you write a new extension that uses `position: fixed`, `grid-cols-*`, etc. (essentially anything), the loader-prepend behavior is what keeps the shell layout intact.

**What's NOT the fix (we tried):** dropping Tailwind preflight via `@import "tailwindcss/utilities.css"` doesn't help — preflight isn't the conflict. The conflict is utility-class source order within the merged `@layer utilities`.

## Accessing Externalized Libraries Inside Extensions

`react-router-dom`, `sonner`, `@vibecms/ui`, `@vibecms/icons`, `@vibecms/api` are externalized in the extension's vite config. The admin-ui shell exposes them on `window.__VIBECMS_SHARED__`. Pattern:

```tsx
const { useSearchParams, useNavigate } =
  (window as any).__VIBECMS_SHARED__.ReactRouterDOM;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

// @vibecms/ui and @vibecms/icons resolve via direct import (vite externalize).
import { Button } from "@vibecms/ui";
import { Upload } from "@vibecms/icons";  // resolves to lucide-react at runtime
```

`__VIBECMS_SHARED__.ui` exposes admin-ui's design-system components: `ListPageShell`, `ListHeader`, `ListSearch`, `ListFooter`, `EmptyState`, `LoadingRow`, `Chip`, `StatusPill`, `Th`, `Td`, `Tr`, `TitleCell`, `RowActions`, `Checkbox`, etc. Always reach for these before rolling your own — they're what makes pages look consistent.

## List Page Pattern (canonical, mirrors nodes/forms/media-library)

```tsx
const { ListPageShell, ListHeader, ListSearch, ListFooter } =
  (window as any).__VIBECMS_SHARED__.ui;

<ListPageShell>
  <ListHeader
    tabs={[{ value: "all", label: "All", count: 42 }, ...]}
    activeTab={activeTab}
    onTabChange={setActiveTab}
    extra={<NewButton />}
  />
  <div className="flex items-center gap-2 mb-2.5 flex-wrap">
    <ListSearch value={search} onChange={setSearch} placeholder="Search…" />
    {/* view switcher, sort pill, density, etc. */}
  </div>
  <YourGridOrTable />
  <ListFooter
    page={page} totalPages={totalPages} total={meta.total}
    perPage={perPage} onPage={setPage} onPerPage={setPerPage} label="files"
  />
</ListPageShell>
```

Drop the page `<h1>` — the active tab pill is the title. ListHeader's `tabs` prop replaces a separate "filter dropdown" — type/status/category becomes a tab row.

## Tab Counts Without an Aggregate Endpoint

When the backend only returns one filtered list per request and you want counts on every tab, fire one parallel `per_page=1` request per tab and read `meta.total`:

```ts
const fetchTabCounts = useCallback(async () => {
  const results = await Promise.all(
    TABS.map(async (t) => {
      const res = await fetchEntities({
        page: 1, per_page: 1,
        filter: t.value === "all" ? undefined : t.value,
        search: searchDebounce || undefined,
      });
      return [t.value, res.meta.total] as const;
    })
  );
  setTabCounts(Object.fromEntries(results));
}, [searchDebounce]);
```

Run on mount + when search changes + after upload/create/delete (alongside the main fetch via `Promise.all([fetchFiles(), fetchTabCounts()])`). 5 small parallel requests cost ~one round-trip; cheaper than adding a backend aggregate endpoint for v1.

## URL-Synced State (filters, view, pagination)

All admin list pages should make their state shareable/refreshable via URL search params. Pattern using `useSearchParams`:

```ts
const [searchParams, setSearchParams] = useSearchParams();

const page = Math.max(1, Number(searchParams.get("page")) || 1);
const view = searchParams.get("view") === "list" ? "list" : "grid";
const sortBy = searchParams.get("sort") || "date_desc";

const updateParams = useCallback((patch: Record<string, string | number | null>, opts: { replace?: boolean; resetPage?: boolean } = {}) => {
  setSearchParams((prev) => {
    const next = new URLSearchParams(prev);
    for (const [k, v] of Object.entries(patch)) {
      if (v === null || v === "" || v === undefined) next.delete(k);
      else next.set(k, String(v));
    }
    if (opts.resetPage) next.delete("page");
    return next;
  }, { replace: opts.replace });
}, [setSearchParams]);

const setView   = (v: string)  => updateParams({ view: v === "grid" ? null : v });
const setSearch = (s: string)  => updateParams({ q: s || null }, { replace: true, resetPage: true });
const setSort   = (s: string)  => updateParams({ sort: s === "date_desc" ? null : s }, { resetPage: true });
const setPage   = (p: number | ((prev: number) => number)) => {
  const next = typeof p === "function" ? p(page) : p;
  updateParams({ page: next === 1 ? null : next });
};
```

Two conventions worth keeping:
1. **Default values omit the param** (`?view=grid` is implicit — you only ever see `?view=list`). Keeps URLs clean and shareable.
2. **Search input uses `replace: true`** so each keystroke doesn't pollute browser history.
3. **Filter changes call `resetPage: true`** so users don't end up on page 7 of a 2-page filtered result.

## Sortable Column Headers Sync With Sort Dropdown

If the page has both a "Sort by" dropdown (grid view) and clickable column headers (list view), back BOTH with the same `?sort=` URL param. The column header is just a different control onto the same state — cycle direction on click, show arrow indicator on the active column. Keeps one mental model and works across view switches.

## Editing Shared UI Primitives

Files under `admin-ui/src/components/ui/` (Switch, Select, Tabs, Button, etc.) are exposed to extensions via `__VIBECMS_SHARED__.ui`. Editing them changes behavior CMS-wide and across every extension. **Always rebuild admin-ui** after touching these:

```bash
cd admin-ui && npm run build
```

Common edits already made:
- `switch.tsx`: `data-[state=checked]:bg-indigo-600` (was `slate-900` — invisible black on white).
- `select.tsx`: SelectTrigger `w-full` (was `w-fit` — selects didn't fill rows like other inputs).
- `tabs.tsx`: TabsTrigger `cursor-pointer` (clickable controls must show pointer).

## Hot-Deploy Without Rebuilding the Docker Image

Saves the user 30+ seconds per iteration. Use after `npm run build` confirms changes:

```bash
docker cp /path/to/admin-ui/dist/. <container>:/app/admin-ui/dist/
docker cp /path/to/extensions/<slug>/admin-ui/dist/. <container>:/app/extensions/<slug>/admin-ui/dist/
```

The Go binary serves these as static files — **no container restart needed**. User must hard-refresh (Cmd+Shift+R) to bypass cached `index.html`/JS/CSS.

Find the container with `docker ps --format "table {{.Names}}\t{{.Image}}"` (typically `vibecms-app-1`).

## Editor Layout Pattern (canonical, mirrors `admin-ui/src/pages/node-editor.tsx`)

For any edit page (form, node, term, …):

```tsx
<div className="space-y-4">
  <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
    <div className="space-y-4 min-w-0">
      {/* 1. Compact pill: back arrow + title input | / | slug input + Auto/Edit toggle */}
      {/* 2. Tabs (rounded-xl bg-slate-100 p-1, white-on-active) */}
      {/* 3. TabsContent for each tab */}
    </div>
    <div className="space-y-4">
      {/* Sidebar: Publish card (Save / Cancel), Actions card */}
    </div>
  </div>
</div>
```

The pill belongs **inside the left column**, not full-width above the grid. Don't use 2/3+1/3 grids — use `minmax(0,1fr)_320px` so the sidebar is fixed and the main column fluid. `min-w-0` on the main column prevents flex/grid overflow from long content.

## Extension Build Verification Checklist

After any extension UI change:

- [ ] `cd extensions/<slug>/admin-ui && npm run build` completes clean
- [ ] `dist/index.js` AND `dist/index.css` both present
- [ ] `dist/index.css` contains the expected class (grep test above)
- [ ] If admin-ui shared primitives also changed: `cd admin-ui && npm run build`
- [ ] Hot-deploy with `docker cp` (faster than full Docker rebuild)
- [ ] Browser hard-refresh

## References

- `docs/extension_api.md` §6 — Micro-frontend conventions (CSS section)
- `admin-ui/src/lib/extension-loader.ts` — Module + CSS loader
- `admin-ui/src/main.tsx` — `__VIBECMS_SHARED__` shim setup
- `extensions/forms/admin-ui/` — Reference implementation
- `Dockerfile` `frontend` and `ext-frontend` stages — why centralized Tailwind scan fails
