---
name: squilla-create-theme
description: |
  Use when building a new Squilla theme from scratch — a self-bootstrapping
  marketing site that drops into `themes/<name>/` and seeds its own pages,
  layouts, blocks, partials, taxonomies, settings, and demo content on
  activation. Triggers: "make a marketing site for X", "build a theme for
  Y", "clone hello-vietnam structure", scaffolding under `themes/`,
  designing `theme.json`, writing block `view.html` + `block.json` pairs,
  authoring `theme.tengo` seed scripts, wiring forms via the forms-extension
  handshake, deciding between a layout / partial / block, debugging
  `theme-asset:<key>` resolution, or seeding pages with `blocks_data`.
---

# Creating a Squilla Theme

## When to use this skill

Reach for this skill when **the answer to "where does this UI live?" is "in a theme."** Themes own everything a visitor sees on the public site: page chrome, blocks, partials, demo content, on-brand forms.

**Don't** use this skill when:
- You're building a feature that should work across themes (that's an extension — see `squilla-create-extension`)
- You need a database table (themes don't own tables; extensions do)
- You're styling the admin (admin is a React SPA — see `squilla-extension-frontend`)

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

## Silent failure modes (read this BEFORE writing any Tengo)

Squilla used to swallow these silently. Most are now fail-loud (warnings in
`docker compose logs app`, or hard rejection at theme load). Read them once
so you don't have to debug them.

### The asymmetry cheatsheet

| Where | Wrapping key | Field-key style | Options style | Term value |
|---|---|---|---|---|
| `nodes.create({...})` (top level) | **`fields_data:`** | n/a (your own keys) | n/a | `{slug, name}` object |
| Inside `blocks_data: [{type, ...}]` | **`fields:`** | n/a | n/a | `{slug, name}` object |
| `block.json` `field_schema` | n/a | **`key:`** | **`["a","b"]`** strings only | `term_node_type` required |
| `nodetypes.register({field_schema:[...]})` | n/a | **`name:`** | strings or `{value,label}` ok | same |

**Mismatching `fields:` vs `fields_data:` was the #1 silent data drop.** The
runtime now logs a warning when it sees the wrong one — watch the app logs
during seed runs.

**Object options on a `select` field in block.json** are now rejected at
theme load with a hard error. They used to crash the admin with React error
#31 and silently fail to register the block.

### Other failure modes that now log warnings

- **`type: "term"` field schema entry without `term_node_type:`** — hydration
  silently returns nothing. The CMS now logs at register time. In block.json,
  the theme loader hard-rejects the block.
- **Settings keys keep their dots** in the DB (`squilla.brand.version`).
  Templates that try `.app.settings.squilla.brand.version` get an empty
  result silently. Use `index $s "key"` or — better — `mustSetting`:

```html
{{- $s := .app.settings -}}
<p>Version: {{ mustSetting $s "squilla.brand.version" }}</p>
```

  `mustSetting` errors loudly when the key is missing/empty. `index` returns
  empty silently. Reach for `mustSetting` whenever the value is
  required for the page to make sense.
- **`{{ filter "name" }}` with no value arg** throws "wrong number of args".
  For input-less filters, pass `(dict)`:
  ```html
  {{ $things := filter "list_things" (dict) }}
  ```
- **Default fallbacks like `{{ or .x "Default" }}` mask data bugs.** Don't
  use them. Let empty fields render empty so the bug is loud.

### Tengo language gotchas

- `error` is a reserved selector. **`log.error("…")` is a parse error.**
  Use `log.warn(…)`, `log.info(…)`, or the alias `log.err(…)`.
- `is_string`, `is_undefined`, `is_error` are Tengo built-ins — use them
  for type checks, including on optional map keys.
- Tengo imports are relative without extension: `import("./setup/foo")`
  resolves `scripts/setup/foo.tengo`. Each module needs `export {…}`.
- A bare top-level `return` in a filter terminates the script before
  setting `response`. Use `if/else` branches and let the script fall
  through.

### Theme HTTP routes mount under a prefix

`routes.register("GET", "/docs", "./routes/docs")` ends up at
**`/api/theme/docs`**, NOT `/docs`. Themes cannot shadow public node
routes. To redirect a public path either:

- Point a menu link at the destination URL directly (skipping the page slug
  resolution), or
- Use an extension `public_route` (extensions are not prefixed).

### Caches: when do file changes show up?

| Change | What to do |
|---|---|
| Edit `view.html` (block) | Re-activate the theme: `core.theme.activate` |
| Edit `block.json` `field_schema` | Re-activate. The `content_hash` gates resync — if your edit didn't change the hash, force it (see "Useful queries"). |
| Edit `layouts/*.html` or `partials/*.html` | Re-activate. Layouts/partials are loaded into the renderer cache at activation. |
| `core.settings.set("homepage_node_id", …)` | Now publishes `setting.updated` and busts the cache. Older builds required a re-activate. |
| `core.theme.deploy({body_base64})` | `theme.json` MUST declare `slug:` (regex `[A-Za-z0-9_-]+`). Local `make theme` works without it because the directory name is the slug; the deploy tool requires the field explicitly. |

### Dev-mode iteration loop

Set `SQUILLA_DEV_MODE=true` in your dev environment. Seeds receive a
top-level `dev_mode` boolean. Use it to branch to overwrite-on-reseed for
fast iteration. Production stays safe (idempotent skip-if-exists) because
the env var is unset.

```tengo
nodes := import("core/nodes")

ensure_or_replace_home := func() {
    res := nodes.query({ node_type: "page", slug: "home", limit: 1 })
    if res.total > 0 && dev_mode {
        nodes.delete(res.nodes[0].id)
        res = { total: 0, nodes: [] }
    }
    if res.total == 0 {
        return nodes.create({ ... }).id
    }
    return res.nodes[0].id
}
```

Without `dev_mode`, the same seed is the safe production-idempotent form.

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

## Term-typed field lifecycle

Term fields (a constrained pick from a taxonomy) cause more confusion than
any other field type. The lifecycle, end to end:

1. **Theme registers the taxonomy with `node_types`:**
   ```tengo
   taxonomies := import("core/taxonomies")
   taxonomies.register({
       slug: "doc_section",
       label: "Doc Section",
       label_plural: "Doc Sections",
       hierarchical: false,
       node_types: ["documentation"]
   })
   ```
   To **attach** an existing taxonomy (e.g. core's `category`) to a custom
   node type, **re-register it** with the desired `node_types: [...]` —
   without re-registering, the post-edit form has no selector.

2. **Theme creates terms with `node_type` set:**
   ```tengo
   terms := import("core/terms")
   terms.create({
       node_type: "documentation",
       taxonomy:  "doc_section",
       slug:      "getting-started",
       name:      "Getting Started"
   })
   ```
   The DB unique key is `(node_type, taxonomy, slug)`. Without
   `node_type`, the term won't hydrate when used in a per-node field.

3. **Field schema declares `term_node_type`:**
   ```json
   { "key": "section", "type": "term",
     "taxonomy": "doc_section",
     "term_node_type": "documentation" }
   ```
   Without `term_node_type`, the loader logs a warning and hydration
   silently won't match.

4. **Store the value as an OBJECT, not a bare slug:**
   ```tengo
   nodes.create({
       node_type: "documentation",
       title: "…", slug: "intro", status: "published",
       fields_data: {
           section: { slug: "getting-started", name: "Getting Started" }
       }
   })
   ```
   The admin's term-field component requires the object form to
   pre-select. The hydrator accepts either, but bare strings break the
   admin edit flow.

5. **Templates handle BOTH shapes** (string slug OR hydrated map):
   ```html
   {{- $sec := .node.fields.section -}}
   {{- $secLabel := "" -}}
   {{- if $sec -}}
     {{- with $sec.name }}{{ $secLabel = . }}{{ end -}}
     {{- if not $secLabel -}}{{- with $sec.slug }}{{ $secLabel = . }}{{ end -}}{{- end -}}
     {{- if not $secLabel -}}{{- $secLabel = $sec -}}{{- end -}}
   {{- end -}}
   ```

### Real taxonomies (admin "Taxonomies" tab + tax_query)

For taxonomies you want to surface in the admin's "Taxonomies" tab and
query via `tax_query` (e.g. `category` on `post`), use the **`taxonomies:`**
key on the node, NOT `fields_data`:

```tengo
nodes.create({
    node_type:  "post",
    title:      "Hello",
    slug:       "hello",
    status:     "published",
    taxonomies: { category: ["engineering"] },     // real taxonomy
    fields_data: { excerpt: "…", read_time: "5 min" } // term-typed schema fields
})
```

`taxonomies:` lands in `content_nodes.taxonomies` (JSONB).
`fields_data:` term-typed entries are constrained pickers — they share
storage with regular per-node fields.

## Filter registration

Tengo files in `scripts/filters/` are auto-loaded as importable modules but
**not** as named filter handlers. Register them explicitly in your setup
script:

```tengo
filters := import("core/filters")
filters.add("list_docs", "./filters/list_docs")
filters.add("doc_neighbors", "./filters/doc_neighbors")
```

Then templates can call them:
```html
{{ $docs := filter "list_docs" (dict "section" "getting-started") }}
```

Remember: `filter "name"` with no value argument throws — pass `(dict)` for
filters that take no input.

## Block slug prefixing

Prefix every theme block slug (e.g. `sq-hero`, `mytheme-cta`,
`hv-popular-trips`). Last-write wins per slug, so collisions with
extension-registered blocks (notably `cb-*` from the content-blocks
extension) or other themes' blocks are silent. Prefixing is hygiene, not
optional.

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

## Production-readiness checklist

Before declaring the theme done, **run the automated checklist tool**:

```
core.theme.checklist({ slug: "<your-theme-slug>" })
```

It walks `themes/<slug>/` on disk and reports `pass | fail` for:

- `theme.json` exists, parses, has `slug`, has a default layout.
- Every `block.json` `field_schema` uses `key:` (not `name:`).
- Every `select`/`radio` field has plain string options.
- Every `term`-typed field has `term_node_type` and `taxonomy`.
- No `log.error(` in seed scripts (it's a Tengo parse error).
- No suspect top-level `fields:` literal in scripts that call
  `nodes.create`/`nodes.update`.
- Block slug prefixing.

Read **`docs/theme-checklist.md`** for the full list — including the
manual checks that only a human or a browser-driving agent can verify
(admin pre-selects the right values, public homepage looks right,
re-activation is idempotent). Run the automated tool first, then walk
the manual list.

Don't claim done until both pass. This is the difference between a
theme that one-shots and one that needs a debugging session.

## Update log

This skill was rewritten after a real-world theme port (`themes/squilla`)
hit ~40 silent failures. The retrospective lives in
`docs/theme-build-notes.md`. Most of those gotchas are now fail-loud:
warnings in the app log on misuse, hard rejection at theme load for
schema-level bugs. If you encounter a new silent failure, add it to
that document and propose a fail-loud fix.
