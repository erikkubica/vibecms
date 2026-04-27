# VibeCMS Themes

Themes control the public-facing site appearance. Each theme provides layouts, partials, content blocks, static assets, and optional Tengo scripts.

## Structure

```
themes/
  my-theme/
    theme.json              # Theme manifest (name, version, author, assets, templates)
    layouts/                # Page layouts (Go html/template)
      base.html             # Default layout
      blank.html            # No-chrome layout
    partials/               # Reusable template fragments
      site-header.html
      site-footer.html
    blocks/                 # Content block templates
      hero.html
      text.html
    assets/                 # Static files (CSS, JS, images, fonts)
    templates/              # Theme page templates (JSON)
    scripts/                # Tengo hooks and filters
      theme.tengo           # Main entry point for registration/seeding
      filters/              # Custom Tengo filters
```

## How Themes Work

- **Autoregistration**: At startup the core scans `themes/` (mirroring the extension scan) and upserts every theme it finds — drop a directory in, restart, it's available in the admin theme picker. No DB seeding required.
- **Layouts**: Go `html/template` files that wrap page content. Rendered by the core template engine. The default core layout is now seeded with `source='seed'` (migration `0036` retroactively demotes legacy rows) so an activated theme can install its own `base.html`/`blank.html` without colliding with the bootstrap row.
- **Partials**: Included in layouts via `{{ partial "site-header" . }}`.
- **Blocks**: Each content block type maps to a template file. Rendered in sequence to build the page.
- **Assets**: Served statically at `/theme/assets/*`. Referenced via `theme-asset:<key>` in seeds/templates. The asset resolver uses an atomic pointer that swaps on `theme.activated` — switching themes at runtime serves the new theme's assets instantly with no restart and no stale cache. The resolver also rejects path-traversal attempts (`../`).
- **Scripts**: Tengo scripts that register event hooks, filters, custom routes, menus, and `/.well-known/*` handlers.

## Template Functions

Available in all theme templates:
- `{{ partial "name" . }}` — include a partial
- `{{ filter "name" value }}` — apply a filter
- `{{ image_url .url "thumbnail" }}` — get cached/optimized image URL
- `{{ image_srcset .url "medium" "large" }}` — generate srcset attribute
- Standard Go template functions (`if`, `range`, `with`, etc.)

---

# Theme Development Guide

This guide codifies the rules for building a production-grade VibeCMS theme. The golden rule is **the theme must render correctly from a cold boot with nothing but its own files** — no manual DB edits, no hidden fixups, no magic.

## 1. Blocks

### 1.1 Every block must have complete `test_data`
`test_data` is the preview an editor sees, the default payload when a block is added, and the canary for the renderer.

- **Every field in `field_schema` must have a value in `test_data`**, including optional ones.
- Content must be **on-brand** — real place names, real voice. No Lorem Ipsum.
- Values must match the **exact shape** the renderer expects.

### 1.2 Every field must be declared
No field may be read in `view.html` that is not declared in `block.json`'s `field_schema`.

Checklist:
- `field_schema` lists every field `view.html` reads.
- Every field has the **correct type**.
- `test_data` contains every field with realistic values.
- Repeater sub-structures use the key name `sub_fields` (not `fields`).

### 1.3 No fallback defaults — gate each field, not the block
Templates **must not** carry hardcoded fallback content. An unset field means "don't render that piece of UI" — not "show this canned string instead."

**Wrong:**
```go
<h2>{{ or .heading "Welcome to Vietnam" }}</h2>
```

**Right:**
```go
{{ with .heading }}<h2>{{ . }}</h2>{{ end }}
```

### 1.4 Sync the DB after every `block.json` change
Block schemas and templates are cached in the `block_types` DB row. To force a resync during development:
```sh
docker compose exec -T db sh -c 'psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "UPDATE block_types SET content_hash = '\''force-resync-'\''||floor(random()*1000000)::text WHERE source='\''theme'\'';"'
docker compose restart app
```

### 1.5 Every block must have a human-friendly `description`
The `description` field at the root of `block.json` is used in the CMS block picker and by other developers to understand the block's layout and behavior.

- **Bad**: `"description": "Hello Vietnam — About Intro"`
- **Good**: `"description": "A high-impact header for the About page. Features a large, expressive heading and multi-paragraph mission statement."`

The description should summarize the **visual layout** and **functional purpose** of the block.

### 1.6 Every field must have a `help` property
Provide clear, instructional text for the CMS editor via the `help` property. This appears as a hint in the admin interface.

- **Bad**: `{ "key": "eyebrow", "label": "Eyebrow", "type": "text" }`
- **Good**: `{ "key": "eyebrow", "label": "Eyebrow", "type": "text", "help": "Small label shown above the main heading (e.g. 'Our most-booked')." }`

Guidance should be provided for any field where the intent isn't 100% obvious from the label alone.


## 2. Fields — Picking the Right Type

### 2.1 Taxonomies → `term` field
**Never** use a `text` field for tag/category slots. Use a `term` field bound to a taxonomy.
`test_data` shape: `{"name": "Foodie", "slug": "foodie"}`.

### 2.2 `select` / `radio` / `checkbox`
Options must be a **flat array of strings**, not objects.
```json
{ "key": "color", "type": "select", "options": ["red", "yellow", "green"] }
```

### 2.3 Links / CTAs → `link` field
**Never** split a button into text + url + target. Use the `link` field.
`test_data` shape: `{"url": "/trips", "text": "Explore", "target": "_self"}`.

### 2.4 Images → `image` field (always objects)
`test_data` shape: `{"url": "theme-asset:hero-grandma", "alt": "…"}`.
Never flatten to a string; it breaks the admin image picker.

### 2.4b Video / Audio → `video` / `audio` fields
Similar to images, these should be handled as objects if metadata is required, but often `theme-asset:<key>` strings suffice for simple playback.

### 2.5 Field Type Summary

| Intent                      | Field type | Shape in `test_data`                                  |
|-----------------------------|------------|-------------------------------------------------------|
| Taxonomy term               | `term`     | `{"name": "Foodie", "slug": "foodie"}`                |
| Image                       | `image`    | `{"url": "theme-asset:key", "alt": "…"}`              |
| Gallery of plain images     | `gallery`  | `[{"url": "theme-asset:a", "alt": "…"}, …]`           |
| Gallery with captions       | `repeater` | `sub_fields: [image, text]`                           |
| Button / CTA / any URL      | `link`     | `{"url": "/path", "text": "…", "target": "_self"}`    |
| Reference to content        | `node`     | `{"id": 123, "slug": "…", "title": "…"}`              |
| Short heading               | `text`     | `"…"`                                                 |
| Body / paragraph            | `textarea` | `"…"`                                                 |
| Rich text                   | `richtext` | `"…"`                                                 |
| Boolean                     | `toggle`   | `true` / `false`                                      |

## 3. Templates

The theme **must** ship `templates/*.json` files — one per page it demos.

- **Fully Populated**: Templates must contain real content so the page renders perfectly on "Load template".
- **Register in `theme.json`**:
  ```json
  "templates": [
    { "slug": "homepage", "file": "homepage.json" }
  ]
  ```

## 4. Assets & Registration

### 4.1 All media lives in the theme
Commit images, videos, and fonts under `assets/`. No external CDN dependencies for demo content.

### 4.2 Registration in `theme.json`
Every asset and block must be registered in the manifest:
```json
{
  "blocks": [
    { "slug": "hv-hero", "dir": "hv-hero" }
  ],
  "assets": [
    { "key": "hero", "src": "images/hero.jpg", "alt": "Hero description" }
  ]
}
```
Reference assets as `theme-asset:<key>` in your code. This allows the platform to resolve the correct URL even if the file is moved or served via a custom media handler.

## 5. Taxonomies, Menus & Demo Data (Seeding)

The theme should be self-bootstrapping. Use `theme.tengo` to register custom types, seed navigation menus, and seed demo content.

### 5.0 Menu Seeding (`core/menus`)

The `core/menus` Tengo module exposes `menus.upsert(...)`. Items support either an absolute `url` or a `page: "<slug>"` form — the latter resolves to a `NodeID` at upsert time so that **renaming the page's slug doesn't break the menu**.

```tengo
menus := import("core/menus")

menus.upsert({
    slug: "main-nav",
    name: "Primary Navigation",
    items: [
        { label: "Home",    page: "home" },
        { label: "Trips",   page: "trips" },
        { label: "About",   page: "about" },
        { label: "Contact", url: "/contact", target: "_self" }
    ]
})
```

`hello-vietnam`'s `site-header.html` and `site-footer.html` render directly from the menu tree; `default`'s `theme.tengo` seeds `main-nav` and `footer-nav` on activation. Always seed via slug-based `page:` references — it's the same portability rule as `theme-asset:<key>`.

### 5.0.1 `/.well-known/*` (`core/wellknown`)

`/.well-known/*` is now handled by a pluggable `WellKnownRegistry` mounted **before** the public catch-all, so unregistered paths return an instant 404 instead of triggering six language-prefixed node lookups. Tengo themes (and extensions) register handlers via the new `core/wellknown` module — useful for `apple-app-site-association`, `assetlinks.json`, ACME challenges seeded by themes, etc.

### 5.1 Registration Pattern
Always register taxonomies before the node types that use them:
```tengo
taxonomies.register({
    slug: "trip_tag",
    node_types: ["trip"]
})

nodetypes.register({
    slug: "trip",
    taxonomies: ["trip_tag"],
    field_schema: [ ... ]
})
```

### 5.2 Seeding Pattern (Existence Check)
To avoid duplicate data on script re-runs (activation/deactivation), always check if data exists first:
```tengo
page_missing := func(slug) {
    r := nodes.query({ node_type: "page", slug: slug, limit: 1 })
    return r.total == 0
}

if page_missing("home") {
    nodes.create({
        title: "Home",
        slug: "home",
        node_type: "page",
        blocks_data: [ ... ]
    })
}
```

## 6. Portable Refs (Slugs)

Always reference core entities by **slug**, never by numeric ID.

- **Internal Links**: Use path-based links like `/trips` or `/about` (based on the slug of the page).
- **Block Types**: Reference blocks by their registered slug (e.g., `hv-hero`).
- **Asset Refs**: Use `theme-asset:<key>`.
- **Cross-Node Refs**: If a block has a `node` field, the `test_data` should provide a shape like `{"slug": "hanoi-food-trip"}`.

## 7. Reference Implementation

The **hello-vietnam** theme serves as the gold standard for VibeCMS theme architecture. Refer to it for:
- Complex `theme.tengo` seeding logic.
- Advanced Tengo filters (e.g., `list_nodes` with custom sorting).
- Repeater and nested field schema examples.

## 8. Verification Checklist

- [ ] Every registered asset imports into the media library.
- [ ] Every seeded node renders correctly (200 status).
- [ ] Every block's admin edit form renders all fields (no `[object Object]`).
- [ ] Every template loads and renders identically to the seeded page.

## 9. The Mandalorian Rule

Every block has proper `test_data`. Every field uses the right type. Every theme ships templates that load fully-populated pages. Every asset lives in the repo and is registered for import.

**This is the way.**

---

## 10. Programmatic Access (MCP)

VibeCMS exposes these standards programmatically via its MCP server. AI agents and IDE tools can fetch the latest rules to validate your work automatically.

- **Tool**: `core.theme.standards` — Returns a structured JSON of all rules and common field conventions.
- **Resource**: `vibecms://guidelines/themes` — The authoritative document for theme development.

When working with an AI assistant, you can ask: *"Fetch the VibeCMS theme standards before we start building this block."*
