---
name: vibecms-create-theme
description: |
  Use when building a new VibeCMS theme from scratch — a self-bootstrapping
  marketing site that drops into `themes/<name>/` and seeds its own pages,
  layouts, blocks, partials, taxonomies, settings, and demo content on
  activation. Triggers: "make a marketing site for X", "build a theme for
  Y", "clone hello-vietnam structure", scaffolding under `themes/`,
  designing `theme.json`, writing block `view.html` + `block.json` pairs,
  authoring `theme.tengo` seed scripts, wiring forms via the forms-extension
  handshake, deciding between a layout / partial / block, debugging
  `theme-asset:<key>` resolution, or seeding pages with `blocks_data`.
---

# Creating a VibeCMS Theme

## When to use this skill

Reach for this skill when **the answer to "where does this UI live?" is "in a theme."** Themes own everything a visitor sees on the public site: page chrome, blocks, partials, demo content, on-brand forms.

**Don't** use this skill when:
- You're building a feature that should work across themes (that's an extension — see `vibecms-create-extension`)
- You need a database table (themes don't own tables; extensions do)
- You're styling the admin (admin is a React SPA — see `vibecms-extension-frontend`)

## Source of truth

`themes/README.md` is the contract. Key sections:

| Topic | Section |
|---|---|
| Mental model + boot sequence | §1 |
| Folder anatomy | §2 |
| `theme.json` schema | §3 |
| **Layout vs partial vs block context** (the #1 gotcha) | §4 |
| Block recipe (`block.json` + `view.html` + scoped CSS/JS) | §5 |
| **Field types reference** (every `type` and its `test_data` shape) | §6 |
| Page templates (`templates/*.json`) | §7 |
| Tengo seeding (`scripts/theme.tengo`) | §8 |
| `theme-asset:<key>` mechanism | §9 |
| **Forms-extension handshake** (`forms:upsert` + `forms:render`) | §10 |
| Site settings convention | §11 |
| Template functions reference | §12 |
| Build & live-reload loop | §13 |
| **The 12 Mandalorian rules** | §14 |
| Troubleshooting table | §15 |
| **Skeleton (copy-paste)** | §16 |
| Reference: every block in `hello-vietnam` | §17 |

The reference theme is `themes/hello-vietnam/` — 25 blocks, six demo pages, all the patterns. When in doubt, open it.

## The #1 gotcha: layout vs partial vs block context

**These three template types see different data.** Conflating them is the most common bug.

| Template | Where | Sees |
|---|---|---|
| **Layout** (`layouts/<slug>.html`) | Top-level page chrome | full `.node`, `.app`, `.user` |
| **Partial** (`partials/<slug>.html`) | Reusable fragment via `{{renderLayoutBlock "site-header"}}` | full `.node`, `.app`, `.user`, plus `.partial` |
| **Block view** (`blocks/<slug>/view.html`) | One content block on a page | **only the block's own field values at root** (`{{.heading}}`, NOT `{{.app.settings.foo}}`) |

Inside a block view, `.app` is **not in scope**. To get site-wide data into a block, declare it as a field. To query nodes from a block, use `{{filter "list_nodes" ...}}`.

## Bootable skeleton

Drop this in `themes/my-theme/`, restart, switch to it in the admin.

### File tree

```
themes/my-theme/
├── theme.json
├── layouts/default.html
├── partials/site-header.html
├── partials/site-footer.html
├── blocks/intro/{block.json, view.html}
├── templates/homepage.json
├── assets/styles/theme.css
├── assets/images/hero.webp        # bring your own
└── scripts/theme.tengo
```

### `theme.json`

```json
{
  "name":        "My Theme",
  "version":     "0.1.0",
  "description": "Minimal scaffold",
  "author":      "You",
  "styles":   [ { "handle": "theme-css", "src": "styles/theme.css", "position": "head" } ],
  "scripts":  [],
  "layouts":  [ { "slug": "default", "name": "Default", "file": "default.html", "is_default": true } ],
  "partials": [
    { "slug": "site-header", "name": "Site Header", "file": "site-header.html" },
    { "slug": "site-footer", "name": "Site Footer", "file": "site-footer.html" }
  ],
  "blocks":    [ { "slug": "intro", "dir": "intro" } ],
  "templates": [ { "slug": "homepage", "file": "homepage.json" } ],
  "assets":    [ { "key": "hero", "src": "images/hero.webp", "alt": "Hero photo" } ]
}
```

### `layouts/default.html`

```html
<!doctype html>
<html lang="{{or .node.language_code "en"}}">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{or .node.title "My Theme"}}</title>
{{- range .app.head_styles -}}<link rel="stylesheet" href="{{.}}">{{- end -}}
{{.app.block_styles}}
</head>
<body>
{{renderLayoutBlock "site-header"}}
<main>{{.node.blocks_html}}</main>
{{renderLayoutBlock "site-footer"}}
{{- range .app.foot_scripts -}}<script src="{{.}}" defer></script>{{- end -}}
{{.app.block_scripts}}
</body>
</html>
```

### `blocks/intro/block.json`

```json
{
  "slug": "intro",
  "name": "Intro",
  "description": "Centered headline + body + optional image.",
  "category": "my-theme",
  "field_schema": [
    { "key": "heading", "label": "Heading", "type": "text",     "help": "The H1." },
    { "key": "body",    "label": "Body",    "type": "textarea", "help": "1–3 sentences." },
    { "key": "image",   "label": "Image",   "type": "image",    "help": "Hero photo." }
  ],
  "test_data": {
    "heading": "Welcome.",
    "body":    "This is your new theme. Edit me in the admin.",
    "image":   { "url": "theme-asset:hero", "alt": "Hero photo" }
  }
}
```

### `blocks/intro/view.html`

```html
{{- $img := "" -}}{{- $alt := "" -}}{{- with .image -}}
  {{- with .url -}}{{- $img = . -}}{{- end -}}
  {{- with .alt -}}{{- $alt = . -}}{{- end -}}
{{- end -}}
<section class="intro">
  {{ with .heading }}<h1>{{ . }}</h1>{{ end }}
  {{ with .body }}<p>{{ . }}</p>{{ end }}
  {{ if $img }}<img src="{{ $img }}" alt="{{ $alt }}">{{ end }}
</section>
```

### `scripts/theme.tengo`

```tengo
log      := import("core/log")
settings := import("core/settings")
menus    := import("core/menus")
nodes    := import("core/nodes")

log.info("My Theme initializing…")

seed_setting := func(key, value) {
    existing := settings.get(key)
    if existing == "" || is_error(existing) {
        settings.set(key, value)
    }
}
seed_setting("site.copyright_year", "2026")

// Seed homepage (existence-checked — idempotent)
res := nodes.query({ node_type: "page", slug: "home", limit: 1 })
home_id := 0
if res.total == 0 {
    home := nodes.create({
        title: "Home", slug: "home", node_type: "page", status: "published",
        blocks_data: [
            { type: "intro", fields: {
                heading: "Welcome.",
                body:    "This is your new theme.",
                image:   { url: "theme-asset:hero", alt: "Hero photo" }
            } }
        ]
    })
    home_id = home.id
} else {
    home_id = res.nodes[0].id
}
if home_id > 0 { settings.set("homepage_node_id", string(home_id)) }

// Primary nav (slug-based — survives renames)
menus.upsert({
    slug: "main-nav",
    name: "Primary Navigation",
    items: [ { label: "Home", page: "home" } ]
})

log.info("My Theme initialization complete")
```

## The 12 Mandalorian rules (summary)

Memorize these. Full text in `themes/README.md` §14.

1. **Every block is a complete content type.** `block.json` declares every field its `view.html` reads. Every field has `test_data`. Every field has `help`.
2. **Every template field is gated by `{{with}}`.** No hardcoded fallback strings.
3. **Every image goes through an `image` field with `theme-asset:<key>`.** Never hardcode `/theme/assets/images/...`.
4. **Every image asset is declared in `theme.json` `assets[]`** with real `alt` text.
5. **Every demo page has a matching `templates/<slug>.json`** so editors can re-create it in one click.
6. **`theme.tengo` is idempotent.** Existence checks for content; upserts for menus/settings.
7. **Cross-extension rendering goes through `event`.** Use `event "forms:render"` for forms; never hand-roll a `<form>`.
8. **Settings are namespaced** — `<prefix>.<dot.path>`.
9. **Menu items use `page: "<slug>"`, not `url: "/<slug>"`.**
10. **`safeHTML` only on fields you trust.**
11. **Block-scoped CSS goes in `blocks/<slug>/style.css`.** Site-wide CSS in `assets/styles/theme.css`.
12. **No dead schema fields.** Schema and template must agree.

## Field types — `test_data` shapes

The most common mismatches. Full table in `themes/README.md` §6.

| `type` | `test_data` shape |
|---|---|
| `text`, `textarea`, `richtext`, `select`, `radio`, `color` | `"…"` (string) |
| `number` | `42` |
| `toggle` / `checkbox` | `true` |
| `link` | `{"text": "…", "url": "…", "target": "_self"}` |
| `image` | `{"url": "theme-asset:<key>", "alt": "…"}` ← **NOT a bare string** |
| `gallery` | array of `{url, alt}` objects |
| `term` | `{"slug": "…", "name": "…"}` |
| `node` | `{"slug": "…", "title": "…"}` (engine resolves `id`) |
| `form_selector` | `"<form-slug>"` |
| `repeater` | `[{...}, {...}]` (sub_fields define inner shape) |

**Trap:** in Tengo schemas the field key is `name`; in `block.json` schemas it's `key`. Different parsers, same field types. See §6 + §8.1.

## Top 5 bugs (and how to dodge them)

| Bug | Cause | Fix |
|---|---|---|
| `theme-asset:<key>` shows up as `#ZgotmplZ` in HTML | Asset ref didn't resolve (typo, key not declared, theme not active) | Confirm `assets[]` entry; check media-manager imported the file. |
| Block edit form shows `[object Object]` for a field | `test_data` shape doesn't match the field type (e.g. `image` as a bare string) | Match shape to the field types table. |
| Block schema changes don't appear after restart | `content_hash` matches DB row | Force-resync: `UPDATE block_types SET content_hash = 'force-' \|\| floor(random()*1e6)::text WHERE source = 'theme';` then restart. |
| Form submissions silently disappear | Theme rendered a raw `<form>` instead of using `event "forms:render"` | Use the forms-extension handshake (§10). |
| Layout reads `.app.settings.foo` but value is empty | Setting not seeded, or `seed_setting` short-circuited because key already had `""` | Use `settings.set` directly to overwrite. |

## Forms-extension handshake (the 3-step pattern)

When the theme wants on-brand forms (themed inputs, consistent layout):

**1.** Drop a layout in `forms/<slug>.html` using **the theme's CSS classes**:
```html
<form>
  <label>{{ .name.label }}</label>
  <input class="input" name="{{ .name.id }}" {{ if .name.required }}required{{ end }}>
  <!-- … -->
</form>
```

**2.** Seed the form via `forms:upsert` in `theme.tengo`:
```tengo
events := import("core/events")
assets := import("core/assets")

contact_layout := assets.read("forms/contact.html")
events.emit("forms:upsert", {
    slug: "contact", name: "Contact", layout: contact_layout,
    fields: [
        { id: "email", type: "email", label: "Email", required: true },
        { id: "message", type: "textarea", label: "Message", required: true }
    ]
})
```

**3.** Render from a block or layout via `event "forms:render"`:
```html
{{ safeHTML (event "forms:render" (dict "form_id" "contact")) }}
```

**Never** hand-roll a `<form>` and JS-intercept submit — submissions get silently dropped.

## Useful queries while developing

```bash
# What blocks did the loader register?
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT slug, source, theme_name FROM block_types ORDER BY slug;"

# What did theme.tengo seed?
docker compose logs app --tail=200 | grep -E 'theme|seed'

# Inspect a seeded page's blocks_data
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT slug, jsonb_array_length(blocks_data) AS n_blocks FROM content_nodes WHERE node_type='page';"

# Force-resync block schemas (when content_hash misses a change)
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "UPDATE block_types SET content_hash = 'force-' || floor(random()*1e6)::text WHERE source='theme';"
docker compose restart app
```

## Re-seeding stale demo pages

`theme.tengo` is idempotent — it does NOT overwrite existing pages. After schema changes, delete the rows manually so the seed re-creates them:

```bash
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB <<'SQL'
DELETE FROM menu_items WHERE node_id IN (
  SELECT id FROM content_nodes WHERE node_type = 'page'
    AND slug IN ('home','about','contact')
);
DELETE FROM content_nodes WHERE node_type = 'page'
  AND slug IN ('home','about','contact');
DELETE FROM menus WHERE slug IN ('main-nav','footer-nav');
SQL
docker compose restart app
```

## Next step after the skeleton boots

1. Add more blocks under `blocks/<slug>/`. Declare them in `theme.json` `blocks[]`.
2. Declare every image in `assets[]`; use `theme-asset:<key>` everywhere.
3. Add per-content-type layouts (e.g. `layouts/trip.html`) when `{{.node.fields}}` access is needed.
4. Register Tengo filters in `scripts/filters/` for shared query patterns.
5. For each demo page, ship a matching `templates/<slug>.json` so editors can re-apply.
6. Use the forms-extension handshake instead of hand-rolled forms.

When in doubt, open `themes/hello-vietnam/`. Every pattern in this skill has a working example there.
