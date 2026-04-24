# Layout, Template & Menu System Design

**Date:** 2026-03-26
**Status:** Approved
**Mental Model:** WordPress + ACF Pro, JSON-first with AI

## Overview

A complete layout management system for VibeCMS enabling full page customization from `<head>` to `</footer>`. Different pages/nodes can have different layouts with a WordPress-like template resolution cascade. Includes reusable layout blocks (partials), a hierarchical menu editor with submenu support, and a comprehensive theme asset pipeline.

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Template resolution | Hybrid: convention cascade + explicit override | WordPress-like defaults with per-node control |
| Layout editing | Code editor (upgradeable to zone markers later) | MVP simplicity, zone convention documented for future UI |
| Menu reference | Direct slug in templates (`{{renderMenu "slug"}}`) | Simpler than location registry |
| Menu nesting | 3 levels max (depth 0-2) | Covers 99% of use cases, simpler UI |
| Data model | Separate tables (layouts, layout_blocks, menus) | Future-proof, each entity evolves independently |
| Template context | Global `.app` namespace + `.node` for current page | No helper functions except `renderLayoutBlock` |
| Theme registration | `theme.json` manifest | Programmatic registration of layouts, partials, blocks, assets |
| Language support | Per-entity language_code with default-language fallback | Same pattern as content_nodes |
| Primary keys | SERIAL (integer) | Consistent with all existing tables (users, content_nodes, block_types, etc.) |
| Template engine | Go `html/template` | Already used by existing rendering pipeline in `template_renderer.go` |

## Data Model

### New Tables

#### `layouts`

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| slug | VARCHAR(255) | NOT NULL |
| name | VARCHAR(255) | NOT NULL |
| description | TEXT | |
| language_code | VARCHAR(10) | NOT NULL, FK → languages |
| template_code | TEXT | NOT NULL |
| source | VARCHAR(20) | DEFAULT 'custom' — 'theme' or 'custom' |
| theme_name | VARCHAR(100) | NULL |
| is_default | BOOLEAN | DEFAULT false |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

- **UNIQUE:** `(slug, language_code)`
- **INDEX:** `(source, theme_name)`
- **PARTIAL UNIQUE INDEX:** `CREATE UNIQUE INDEX layouts_one_default_per_lang ON layouts (language_code) WHERE is_default = true;` — enforces at most one default layout per language

#### `layout_blocks` (partials)

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| slug | VARCHAR(255) | NOT NULL |
| name | VARCHAR(255) | NOT NULL |
| description | TEXT | |
| language_code | VARCHAR(10) | NOT NULL, FK → languages |
| template_code | TEXT | NOT NULL |
| source | VARCHAR(20) | DEFAULT 'custom' — 'theme' or 'custom' |
| theme_name | VARCHAR(100) | NULL |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

- **UNIQUE:** `(slug, language_code)`
- **INDEX:** `(source, theme_name)`

#### `menus`

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| slug | VARCHAR(255) | NOT NULL |
| name | VARCHAR(255) | NOT NULL |
| language_code | VARCHAR(10) | NOT NULL, FK → languages |
| version | INT | DEFAULT 1 — incremented on each save, used for optimistic locking |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

- **UNIQUE:** `(slug, language_code)`

#### `menu_items`

| Column | Type | Constraints |
|--------|------|-------------|
| id | SERIAL | PK |
| menu_id | INT | FK → menus ON DELETE CASCADE |
| parent_id | INT | NULL, self-referencing FK, ON DELETE SET NULL |
| title | VARCHAR(255) | NOT NULL |
| item_type | VARCHAR(20) | NOT NULL — 'node', 'url', 'anchor' |
| node_id | INT | NULL, FK → content_nodes |
| url | VARCHAR(2048) | NULL |
| target | VARCHAR(20) | DEFAULT '_self' |
| css_class | VARCHAR(255) | NULL |
| sort_order | INT | DEFAULT 0 |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

- **INDEX:** `(menu_id, sort_order)`
- **INDEX:** `(menu_id, parent_id)`
- **INDEX:** `(node_id)` — for URL sync when nodes change

Note: `depth` is not stored — it is computed server-side from the `parent_id` tree during bulk save validation. The server enforces max 3 levels (depth 0-2) by walking the parent chain and rejecting saves that exceed it.

### Modified Tables

#### `content_nodes` — add column

| Column | Type | Constraints |
|--------|------|-------------|
| layout_id | INT | NULL, FK → layouts ON DELETE SET NULL |

Explicit layout override. NULL = use cascade resolution.

#### `block_types` — add columns

| Column | Type | Constraints |
|--------|------|-------------|
| theme_name | VARCHAR(100) | NULL |
| view_file | VARCHAR(255) | NULL — theme file path |
| block_css | TEXT | NULL — scoped CSS from style.css |
| block_js | TEXT | NULL — scoped JS from script.js |

Source field updated to support: 'custom' | 'theme'.

## Template Resolution Cascade

When rendering a content node with `language_code=de`, `node_type=post`, `slug=hello-world`:

```
1. node.layout_id is set?           → use that layout directly
2. layout slug="layout-post-hello-world" lang=de?
3. layout slug="layout-post-hello-world" lang=en?  (default fallback)
4. layout slug="layout-post" lang=de?
5. layout slug="layout-post" lang=en?              (default fallback)
6. layout is_default=true lang=de?
7. layout is_default=true lang=en?                 (default fallback)
8. hardcoded minimal HTML                          (last resort)
```

First match wins. Steps 2-7 interleave language-specific → default-language for each cascade level.

## Template Rendering Context

### Global `.app` namespace

```
.app.menus["main-nav"]           → menu tree {items: [{title, url, children: [...]}]}
.app.menus["footer-links"]       → another menu
.app.settings.site_name          → from site_settings table
.app.settings.homepage_id        → etc.
.app.languages                   → all available languages
.app.currentLang                 → current language object
.app.headStyles                  → resolved CSS <link> tags for <head>
.app.headScripts                 → resolved JS <script> tags for <head>
.app.footScripts                 → resolved JS <script> tags for before </body>
.app.blockStyles                 → CSS from blocks used on this page
.app.blockScripts                → JS from blocks used on this page
```

### Current page `.node` namespace

```
.node.title
.node.slug
.node.full_url
.node.blocks_html                → rendered blocks output (HTML string)
.node.fields                     → ACF-style fields_data
.node.seo                        → SEO settings
.node.node_type                  → "page", "post", etc.
.node.parent                     → parent node (if any)
.node.children                   → child nodes (if any)
.node.language_code
```

### Template Functions

Only one custom function:

- `{{renderLayoutBlock "slug"}}` — renders a layout block (partial) with current context. Resolves language automatically (requested lang → default lang fallback). **Recursion guard:** max 5 levels of nested `renderLayoutBlock` calls. If exceeded, renders empty string and logs a warning. This prevents infinite loops from circular partial references (e.g., `header` → `nav` → `header`).

All other data is accessed via `.app` and `.node` — no helper functions.

### Example Layout

```html
<!DOCTYPE html>
<html lang="{{.app.currentLang.Code}}">
<head>
    <meta charset="UTF-8">
    <title>{{.node.seo.title}} | {{.app.settings.site_name}}</title>
    {{range .app.headStyles}}<link rel="stylesheet" href="{{.}}">{{end}}
    {{range .app.headScripts}}<script src="{{.}}"></script>{{end}}
    {{.app.blockStyles}}
</head>
<body>
    {{renderLayoutBlock "header"}}

    <main>
        {{.node.blocks_html}}
    </main>

    {{renderLayoutBlock "footer"}}

    {{range .app.footScripts}}<script src="{{.}}" defer></script>{{end}}
    {{.app.blockScripts}}
</body>
</html>
```

### Example Partial (primary-nav)

```html
<nav class="main-nav">
    {{$menu := index .app.menus "main-nav"}}
    {{if $menu}}
        <ul>
        {{range $menu.Items}}
            <li class="{{.CSSClass}}">
                <a href="{{.URL}}" target="{{.Target}}">{{.Title}}</a>
                {{if .Children}}
                <ul class="submenu">
                    {{range .Children}}
                    <li><a href="{{.URL}}">{{.Title}}</a>
                        {{if .Children}}
                        <ul class="submenu-l2">
                            {{range .Children}}
                            <li><a href="{{.URL}}">{{.Title}}</a></li>
                            {{end}}
                        </ul>
                        {{end}}
                    </li>
                    {{end}}
                </ul>
                {{end}}
            </li>
        {{end}}
        </ul>
    {{end}}
</nav>
```

## Theme Structure

```
themes/{theme-name}/
├── theme.json                    ← manifest: metadata + registrations
├── layouts/
│   ├── default.html              ← full page: <head> → </body>
│   ├── post.html                 ← layout-post
│   └── blank.html                ← minimal, no chrome
├── partials/
│   ├── header.html               ← layout block: site header
│   ├── footer.html               ← layout block: site footer
│   └── primary-nav.html          ← layout block: navigation
├── blocks/
│   ├── hero/
│   │   ├── block.json            ← field schema (ACF-style JSON)
│   │   ├── view.html             ← render template
│   │   ├── preview.html          ← admin preview (optional)
│   │   └── style.css             ← block-scoped CSS (optional)
│   ├── text/
│   │   ├── block.json
│   │   └── view.html
│   └── gallery/
│       ├── block.json
│       ├── view.html
│       ├── style.css
│       └── script.js             ← block-scoped JS (lightbox, etc.)
├── assets/
│   ├── css/
│   │   ├── app.css               ← main theme stylesheet
│   │   └── animations.css        ← optional extras
│   ├── js/
│   │   ├── app.js                ← main theme script
│   │   ├── animations.js         ← GSAP, scroll effects, etc.
│   │   └── vendor/               ← third-party libs bundled
│   ├── images/
│   │   ├── logo.svg
│   │   └── fallback-og.jpg
│   ├── icons/
│   │   └── sprite.svg
│   └── fonts/
│       └── inter-var.woff2
└── scripts/                      ← Tengo hooks (existing)
    └── before_render.tgo
```

### theme.json Manifest

```json
{
  "name": "Flavor",
  "version": "1.0.0",
  "description": "Modern agency theme",
  "author": "VibeCMS",

  "styles": [
    { "handle": "theme-app", "src": "assets/css/app.css" },
    { "handle": "theme-animations", "src": "assets/css/animations.css" }
  ],

  "scripts": [
    { "handle": "theme-app", "src": "assets/js/app.js",
      "position": "footer", "defer": true },
    { "handle": "theme-animations", "src": "assets/js/animations.js",
      "position": "footer", "defer": true,
      "deps": ["theme-app"] }
  ],

  "layouts": [
    { "slug": "default", "name": "Default", "file": "layouts/default.html",
      "is_default": true },
    { "slug": "post", "name": "Blog Post", "file": "layouts/post.html" },
    { "slug": "blank", "name": "Blank Canvas", "file": "layouts/blank.html" }
  ],

  "partials": [
    { "slug": "header", "name": "Site Header", "file": "partials/header.html" },
    { "slug": "footer", "name": "Site Footer", "file": "partials/footer.html" },
    { "slug": "primary-nav", "name": "Primary Nav", "file": "partials/primary-nav.html" }
  ],

  "blocks": [
    { "slug": "hero", "dir": "blocks/hero" },
    { "slug": "text", "dir": "blocks/text" },
    { "slug": "gallery", "dir": "blocks/gallery" }
  ]
}
```

### Theme Source Model

**source: "theme"**
- Registered by theme on startup from `theme.json`
- Template code loaded from theme files
- Admin can view but not edit code (read-only in editor)
- Theme update = code updates automatically
- Can be "detached" → becomes source: "custom"

**source: "custom"**
- Created in admin UI
- Template code stored in DB
- Fully editable in code editor
- Not affected by theme changes
- Can clone from theme layouts

## Asset Pipeline

### Registration (startup)

On boot, VibeCMS reads `theme.json` and:
1. Registers all styles/scripts into an in-memory `ThemeAssetRegistry`
2. Resolves dependency order for scripts
3. Serves theme static files at `/theme/assets/*`
4. Registers layouts/partials into DB (upsert by slug + language for source='theme')
5. Registers block types from `blocks/*/block.json` (upsert by slug for source='theme')

### Rendering (per request)

1. Determine which blocks are used on the current page
2. Collect block-scoped CSS/JS for those blocks only
3. Build `.app.headStyles`, `.app.headScripts`, `.app.footScripts`, `.app.blockStyles`, `.app.blockScripts`
4. Pass to layout template for rendering

### In-Memory Asset Registry

```go
type ThemeAsset struct {
    Handle   string   // "theme-app"
    Src      string   // "assets/css/app.css"
    Type     string   // "css" | "js"
    Position string   // "head" | "footer"
    Defer    bool
    Deps     []string // dependency handles
}
```

No DB table for assets — loaded fresh from `theme.json` on every startup.

### Block-Scoped Asset Delivery

Block-scoped CSS/JS (`block_css`, `block_js` in `block_types`) is delivered **inline** — not as linked files:
- CSS: rendered as `<style data-block="hero">...</style>` tags in `<head>`
- JS: rendered as `<script data-block="gallery">...</script>` tags before `</body>`

This avoids needing a file-serving layer for per-block assets. The `.app.blockStyles` and `.app.blockScripts` template variables contain pre-rendered HTML strings (safe HTML), not URLs.

Theme-level assets (`styles`/`scripts` in `theme.json`) are served as files at `/theme/assets/*` and rendered as `<link>`/`<script>` tags with URLs.

## Layout Resolution Caching

Resolved layouts are cached in-process using an atomic `sync.Map` keyed by `node_type + slug + language_code`. Cache is invalidated:
- On layout create/update/delete via admin API
- On theme reload (startup or hot-reload)

This avoids the 8-step cascade DB lookup on every request, consistent with CLAUDE.md guidance on atomic operations for hot-swapped configuration maps.

Menus and layout blocks are also cached in-process by `slug + language_code`, invalidated on mutation.

## Language Fallback

All language-aware entities (layouts, layout_blocks, menus) use the same fallback:

1. Find entity with `slug=X` and `language_code=requested_lang`
2. If not found, find entity with `slug=X` and `language_code=default_lang` (where `languages.is_default = true`)
3. If not found, entity is missing (log warning, use fallback behavior)

Theme-registered layouts/partials are initially created for the default language only. Admin users can create language-specific versions via the UI.

## Admin UI — API Endpoints

### Layouts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/api/layouts` | List layouts (filter: source, language_code) |
| GET | `/admin/api/layouts/:id` | Get layout with template_code |
| POST | `/admin/api/layouts` | Create custom layout |
| PATCH | `/admin/api/layouts/:id` | Update layout (only source='custom') |
| DELETE | `/admin/api/layouts/:id` | Delete layout (only source='custom') |
| POST | `/admin/api/layouts/:id/detach` | Convert theme→custom (clone) |
| POST | `/admin/api/layouts/:id/preview` | Preview layout with sample data |

### Layout Blocks

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/api/layout-blocks` | List partials (filter: source, language_code) |
| GET | `/admin/api/layout-blocks/:id` | Get partial with template_code |
| POST | `/admin/api/layout-blocks` | Create custom partial |
| PATCH | `/admin/api/layout-blocks/:id` | Update partial |
| DELETE | `/admin/api/layout-blocks/:id` | Delete partial |
| POST | `/admin/api/layout-blocks/:id/detach` | Convert theme→custom |

### Menus

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/api/menus` | List menus (filter: language_code) |
| GET | `/admin/api/menus/:id` | Get menu with nested items tree |
| POST | `/admin/api/menus` | Create menu |
| PATCH | `/admin/api/menus/:id` | Update menu metadata |
| DELETE | `/admin/api/menus/:id` | Delete menu + cascade items |
| PUT | `/admin/api/menus/:id/items` | Replace full item tree (bulk save, requires `version` for optimistic locking — returns 409 on conflict) |
| GET | `/admin/api/menus/:id/items` | Get items as flat list with parent_id |

### Node Layout Assignment

| Method | Endpoint | Description |
|--------|----------|-------------|
| PATCH | `/admin/api/nodes/:id` | Existing endpoint — `layout_id` added to payload |

## Admin UI — React Pages

### Layout Editor
- List view: table of layouts grouped by language, badge for source (theme/custom)
- Edit view: Monaco/CodeMirror code editor, syntax highlighting for Go templates
- Available variables panel (`.app.*`, `.node.*`, `renderLayoutBlock`)
- Live preview button (renders with sample data)
- Language selector (create translation of existing layout)

### Layout Block Editor
- Same UI pattern as Layout Editor
- List + code editor + preview

### Menu Editor
- List view: table of menus by language
- Edit view: WordPress-style drag-and-drop tree editor
  - Left panel: "Add items" — search nodes, add custom URL, add anchor
  - Right panel: nested sortable tree of menu items
  - Each item expandable to edit: title, CSS class, target, item type
  - Drag to reorder and nest (max 3 levels)
  - Bulk save (PUT entire tree)

### Node Editor — Layout Picker
- Add layout dropdown to existing node editor
- Shows "Auto (cascade)" as default option + list of available layouts
- Filtered by node's language_code

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Layout not found in cascade | Use hardcoded minimal HTML, log warning |
| Layout block not found | Render empty string, log warning |
| Menu not found | Return empty items array, log warning |
| Template syntax error in layout | Log error, render error page in dev / fallback in prod |
| Template syntax error in partial | Log error, skip partial, render rest of page |
| Theme directory missing | Log warning, skip theme registration, custom-only mode |
| theme.json parse error | Log error, skip theme registration, custom-only mode (per CLAUDE.md: only DB failures are fatal) |
| Circular partial reference | Recursion guard at 5 levels, render empty string, log warning |

## Migration

Single migration file `0008_layouts_menus.sql`:
1. Create `layouts` table with partial unique index for `is_default`
2. Create `layout_blocks` table
3. Create `menus` table (with `version` column)
4. Create `menu_items` table (with `parent_id ON DELETE SET NULL`, no `depth` column)
5. Add `layout_id INT` to `content_nodes` (FK → layouts ON DELETE SET NULL)
6. Add `theme_name`, `view_file`, `block_css`, `block_js` to `block_types`
7. Seed default layout conditionally: `INSERT INTO layouts (..., language_code) SELECT ..., code FROM languages WHERE is_default = true` — uses subquery to resolve default language dynamically rather than hardcoding 'en'
