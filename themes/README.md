# Squilla Themes — The Builder's Guide

> A theme is a **self-bootstrapping marketing site**: drop a folder under `themes/`, restart the app, and a complete demo site appears — pages, layouts, blocks, taxonomies, custom node types, menus, settings, forms, and seeded content. Nothing in the database, no manual admin clicks. The theme owns its full experience.
>
> This document is the contract between **you** (the theme author) and **Squilla** (the engine). Read it once, ship a production-grade theme. Pair it with [`hello-vietnam/`](./hello-vietnam) — the gold-standard reference.

---

## Table of contents

1. [Mental model in 60 seconds](#1-mental-model-in-60-seconds)
2. [Anatomy of a theme](#2-anatomy-of-a-theme)
3. [`theme.json` — the manifest](#3-themejson--the-manifest)
4. [Layouts vs blocks vs partials](#4-layouts-vs-blocks-vs-partials)
5. [Blocks — the recipe](#5-blocks--the-recipe)
6. [Field types reference](#6-field-types-reference)
7. [Page templates (`templates/*.json`)](#7-page-templates-templatesjson)
8. [Tengo seeding (`scripts/theme.tengo`)](#8-tengo-seeding-scriptsthemetengo)
9. [Assets and `theme-asset:` references](#9-assets-and-theme-asset-references)
10. [Forms wiring (forms extension)](#10-forms-wiring-forms-extension)
11. [Site settings convention](#11-site-settings-convention)
11a. [Theme settings (editor-driven)](#11a-theme-settings-editor-driven)
12. [Template functions reference](#12-template-functions-reference)
13. [Build & live-reload loop](#13-build--live-reload-loop)
14. [The Mandalorian rules](#14-the-mandalorian-rules)
15. [Troubleshooting](#15-troubleshooting)
16. [Skeleton: copy-paste a new theme](#16-skeleton-copy-paste-a-new-theme)
17. [Reference: every block in `hello-vietnam`](#17-reference-every-block-in-hello-vietnam)

---

## 1. Mental model in 60 seconds

```
themes/
└── my-theme/
    ├── theme.json          ◄─ manifest. Declares what to register.
    ├── layouts/*.html      ◄─ page chrome (Go html/template).
    ├── partials/*.html     ◄─ reusable layout fragments (header, footer).
    ├── blocks/<slug>/      ◄─ content blocks. block.json + view.html (+ style.css, script.js).
    ├── templates/*.json    ◄─ pre-built page templates editors can "Load" in the admin.
    ├── assets/             ◄─ images/, styles/, scripts/, fonts/.
    ├── forms/*.html        ◄─ form layouts owned by the theme, registered via Tengo + forms-ext.
    └── scripts/
        ├── theme.tengo     ◄─ entry script. Registers node types, taxonomies, settings, menus, seeds pages.
        └── filters/*.tengo ◄─ template filters callable as `{{ filter "name" arg }}`.
```

**Boot sequence** (what happens when the app starts):

1. **Scan** — `themes/*/theme.json` is parsed. Every theme is upserted into the `themes` table (`is_active=false` for new ones; existing rows have metadata refreshed).
2. **Activate** — for the active theme: layouts, partials, blocks, templates, and asset declarations are upserted into the DB. Each row carries a `content_hash` so unchanged files don't churn writes.
3. **Run scripts** — `scripts/theme.tengo` executes once. It registers node types, taxonomies, filters, event handlers, HTTP routes, settings, menus, and seeds demo content. **Idempotent**: subsequent boots do nothing if data exists.
4. **Sync media** — `theme.activated` fires; the `media-manager` extension imports declared `assets[]` images into the media library so they're picked up by `image` fields and `image_url`/`image_srcset` helpers.

**Hot-swap support**: switching themes at runtime emits `theme.deactivated` (cleans up the previous theme's records) → `theme.activated` (registers the new theme's records). The asset registry uses an atomic pointer swap; CSS/JS for the new theme is served instantly with no restart.

---

## 2. Anatomy of a theme

```
themes/my-theme/
├── theme.json
│
├── layouts/
│   ├── default.html               # is_default: true — page chrome for general nodes
│   ├── trip.html                  # specialised layout (per-content-type chrome)
│   └── legal.html                 # alternative layout (e.g. no scripts injected)
│
├── partials/
│   ├── site-header.html           # rendered with {{renderLayoutBlock "site-header"}}
│   └── site-footer.html
│
├── blocks/
│   ├── my-hero/
│   │   ├── block.json             # slug, name, description, field_schema, test_data
│   │   ├── view.html              # Go template; receives the block's fields at root
│   │   ├── style.css              # OPTIONAL — auto-loaded as scoped <style data-block="my-hero">
│   │   └── script.js              # OPTIONAL — auto-loaded as <script data-block="my-hero">
│   └── my-card/
│       ├── block.json
│       └── view.html
│
├── templates/
│   ├── homepage.json              # editors click "Load template" in the admin to populate a page
│   ├── about.json
│   └── contact.json
│
├── assets/
│   ├── images/
│   │   ├── hero.webp              # registered via assets[] in theme.json
│   │   └── about-portrait.webp
│   ├── styles/
│   │   └── theme.css              # registered via styles[] in theme.json
│   └── scripts/
│       └── theme.js               # registered via scripts[] in theme.json
│
├── forms/                         # Optional — only if the theme uses the forms extension
│   ├── contact.html               # Go template, rendered by forms-ext
│   ├── newsletter.html
│   └── trip-order.html
│
└── scripts/
    ├── theme.tengo                # entry — runs once on activation
    └── filters/
        ├── list_nodes.tengo
        └── distinct_field.tengo
```

**File naming rules**:
- Block, layout, partial, template, asset, form **slugs** are `kebab-case`.
- Tengo files are `snake_case.tengo`.
- HTML templates are `kebab-case.html`.
- Site setting keys are `<theme-prefix>.<dot.path>` (e.g. `hv.whatsapp`, `hv.social.instagram`).

---

## 3. `theme.json` — the manifest

Every theme starts here. Every layout, partial, block, template, asset, style, and script that the theme ships **must** be declared. Anything not declared is invisible to the engine.

```jsonc
{
  "name":        "Hello Vietnam",          // human label; also derived → slug "hello-vietnam"
  "version":     "1.0.0",                  // displayed in admin theme picker
  "description": "Warm marketing site …",  // shown in theme picker
  "author":      "Your Studio",

  // CSS files — served from /theme/assets/<src>. Declare everything that should
  // <link rel="stylesheet"> into the layout's {{ range .app.head_styles }}.
  "styles": [
    { "handle": "theme-css", "src": "styles/theme.css?v=2", "position": "head" }
  ],

  // JS files — appended to the layout's {{ range .app.foot_scripts }} in dependency order.
  "scripts": [
    { "handle": "theme-js",  "src": "scripts/theme.js?v=5", "position": "footer", "defer": true, "deps": [] }
  ],

  // Layouts — Go html/template files in layouts/.
  // is_default flags the layout used when a node has no explicit layout pinned.
  // Add "supports_blocks": false to lock a layout to fixed-content nodes (no block editor).
  "layouts": [
    { "slug": "default", "name": "Default Layout", "file": "default.html", "is_default": true },
    { "slug": "trip",    "name": "Trip Detail",    "file": "trip.html" },
    { "slug": "legal",   "name": "Legal / Doc",    "file": "legal.html" }
  ],

  // Partials — reusable layout fragments. Live in partials/. Rendered with
  // {{renderLayoutBlock "site-header"}} from inside a layout.
  "partials": [
    { "slug": "site-header", "name": "Site Header", "file": "site-header.html" },
    { "slug": "site-footer", "name": "Site Footer", "file": "site-footer.html" }
  ],

  // Blocks — directories under blocks/<dir>/. Each must contain block.json + view.html.
  // dir == slug is conventional; only "slug" is required for runtime identity.
  "blocks": [
    { "slug": "hv-hero",       "dir": "hv-hero" },
    { "slug": "hv-categories", "dir": "hv-categories" }
  ],

  // Page templates — JSON files under templates/. Editors load these via the
  // admin "Load template" button to pre-populate a page with blocks.
  "templates": [
    { "slug": "homepage", "file": "homepage.json" },
    { "slug": "about",    "file": "about.json" }
  ],

  // Media assets — files under assets/. The media-manager extension imports
  // these on theme.activated so they're addressable as theme-asset:<key>.
  "assets": [
    { "key": "hero",   "src": "images/hero.webp",   "alt": "…", "width": 1920, "height": 1080 },
    { "key": "about",  "src": "images/about.webp",  "alt": "…" }
  ],

  // Image sizes — named variants the theme depends on. Carried in the
  // theme.activated payload; media-manager upserts them into its sizes
  // table so /media/cache/<name>/<path> URLs resolve. Sizes already owned
  // by the admin (or another source) are NOT clobbered.
  "image_sizes": [
    { "name": "card-thumb",     "width": 480, "height": 320, "mode": "crop" },
    { "name": "showcase-thumb", "width": 450, "height": 350, "mode": "crop" }
  ]
}
```

**Validation cheatsheet:**

| Field | Required | Notes |
|---|---|---|
| `name` | ✓ | Slug derived as lower-cased, dash-joined version. |
| `version` | ✓ | Semver-ish. Bump on schema changes so editors notice. |
| `styles[].position` | optional | `"head"` or `"footer"`. Defaults to `"footer"` for scripts only. |
| `scripts[].deps` | optional | Other handles that must load first. Topologically sorted. |
| `layouts[].is_default` | optional | Exactly one layout should be default. |
| `layouts[].supports_blocks` | optional | Default `true`. Set `false` for chrome-only layouts. |
| `blocks[].dir` | optional | Defaults to slug. Override only if folder name differs. |
| `assets[].key` | ✓ | Must match `^[a-z0-9_-]+$` — referenced in `theme-asset:<key>`. |

---

## 4. Layouts vs blocks vs partials

These three template types **see different data**. Conflating them is the #1 cause of "why is `.app` empty?" bugs.

| Template | Where it lives | Context (data passed in) | Can call `renderLayoutBlock`? | Can call `event`/`filter`? |
|---|---|---|---|---|
| **Layout** | `layouts/<slug>.html` | full `.node`, `.app`, `.user` | ✓ | ✓ |
| **Partial** | `partials/<slug>.html` | full `.node`, `.app`, `.user`, plus `.partial` | ✓ | ✓ |
| **Block view** | `blocks/<slug>/view.html` | **the block's own field values at root** (e.g. `{{.heading}}`) | ✗ | ✓ |

**This is the key gotcha.** Inside a block view, you cannot write `{{.app.settings.foo}}` — `.app` is not in scope. You can call `{{filter "list_nodes" ...}}` to query nodes, or `{{event "forms:render" ...}}` to delegate to another extension. To pass site-wide data into a block, declare it as a field and seed/template-set it explicitly.

### Layout context detail

```go
.node {
    id, status, title, slug, full_url, language_code, node_type
    fields            // the JSONB "fields_data" column, decoded — current node's fields
    seo               // SEO settings map
    excerpt
    featured_image
    taxonomies        // map[taxonomy_slug][]term_slug
    blocks_html       // pre-rendered HTML of all blocks; render with `{{.node.blocks_html}}`
    translations      // language-switcher data
}

.app {
    head_styles []   // resolved theme + extension CSS URLs
    foot_scripts []  // resolved theme + extension JS URLs
    block_styles     // pre-built <style data-block="…"> tags for blocks on this page
    block_scripts    // pre-built <script data-block="…"> tags
    settings {…}     // map[string]string — every site setting, including custom hv.* keys
    menus {…}        // map[slug]→{ items: [{ title, url, target, children }] }
    languages []     // active languages list
    current_lang {…} // language object for the rendered URL
    theme_url        // "/theme/assets" — prefix for static files
}

.user {
    logged_in, id, email, role, full_name
}
```

**Rule of thumb**: anything cross-cutting (site title, brand colour, contact info, social links, copyright, navigation menus) lives in `.app.settings` or `.app.menus`. Anything page-specific (title, blocks, fields) lives in `.node`.

### Partial context detail

A partial sees everything a layout sees, plus a `.partial` map populated with the partial's own field values (declared in `theme.json` partials[].field_schema, if any). Most partials don't declare fields — they just read from `.app` and the current `.node`.

### Block view context detail

A block view's data is the block's `fields` map. **That's it.** From `hv-popular-trips/view.html`:

```html
{{- $limit := .limit -}}{{- if not $limit -}}{{- $limit = 3 -}}{{- end -}}
{{- $trips := filter "list_nodes" (dict "type" "trip" "limit" $limit "order_by" "created_at asc") -}}
{{- if not $trips -}}{{- else -}}
<section>
  {{ with .heading }}<h2>{{ . }}</h2>{{ end }}
  {{- range $trips -}}
    {{- $fd := .fields_data -}}
    <a href="{{ .full_url }}">{{ .title }} — ${{ $fd.price }}</a>
  {{- end -}}
</section>
{{- end -}}
```

Note the dual usage:
- `.heading`, `.limit` — the **block's own** fields, at root.
- `$trips[i].fields_data` — when iterating nodes returned by `filter "list_nodes"`, the field data lives under `fields_data` (the raw DB column name). This is **different** from `.node.fields` in a layout. Two paths, two key names — by design, but worth knowing.

---

## 5. Blocks — the recipe

A block is a self-contained content type: a JSON schema (what the editor fills in), a Go template (how it renders), and optional scoped CSS/JS.

```
blocks/<slug>/
├── block.json       # schema + preview data
├── view.html        # Go template
├── style.css        # OPTIONAL — auto-injected as <style data-block="<slug>">
└── script.js        # OPTIONAL — auto-injected as <script data-block="<slug>">
```

### 5.1 `block.json` — the schema

```jsonc
{
  "slug":        "hv-hero",                         // unique across the install
  "name":        "Home Hero",                       // shown in the admin block picker
  "description": "Punchy multi-line headline …",    // explains layout + intent (1–2 sentences)
  "category":    "hello-vietnam",                   // groups blocks in the picker
  "icon":        "image",                           // OPTIONAL — Lucide-style icon name; defaults to "square"

  "field_schema": [
    {
      "key":   "heading",        // identifier; use snake_case
      "label": "Heading",        // editor-facing label
      "type":  "text",           // see "Field types reference"
      "help":  "The H1.",        // tooltip for the editor
      "required": false          // OPTIONAL — admin enforces; default false
    },
    {
      "key":   "items",
      "label": "Cards",
      "type":  "repeater",
      "help":  "Up to 4 cards.",
      "sub_fields": [             // ← repeater fields go here. KEY MUST BE "sub_fields", not "fields".
        { "key": "title", "label": "Title", "type": "text" },
        { "key": "color", "label": "Color", "type": "select", "options": ["red", "yellow", "green"] }
      ]
    }
  ],

  "test_data": {
    "heading": "Eat, wander, laugh.",
    "items": [
      { "title": "Foodie",    "color": "red" },
      { "title": "Adventure", "color": "green" }
    ]
  }
}
```

**Schema rules** (the engine and admin both rely on these):

| Rule | Why |
|---|---|
| Every field read in `view.html` is declared in `field_schema` | Otherwise the admin form can't edit it. |
| Every field in `field_schema` has a value in `test_data` | The block picker preview, the default-add behaviour, and the renderer canary all depend on it. |
| `test_data` shape exactly matches the field type | E.g. `image` is `{url, alt}` — not a bare string. |
| Repeaters use `sub_fields`, not `fields` | The loader checks for `sub_fields`. |
| `select` / `radio` / `checkbox` options are flat string arrays | Not `[{value, label}]` objects. |
| `description` describes layout + behaviour | Editors lean on it; AI assistants can read it. |
| `help` is set on every non-obvious field | Especially anything with a constrained shape (`color: red\|yellow\|green`). |

### 5.2 `view.html` — the template

```html
{{- /* Brief comment explaining what this block renders. */ -}}
{{- /* Optional: extract image URLs once at the top with the {{with}} idiom. */ -}}
{{- $img := "" -}}{{- $alt := "" -}}{{- with .photo -}}
  {{- with .url -}}{{- $img = . -}}{{- end -}}
  {{- with .alt -}}{{- $alt = . -}}{{- end -}}
{{- end -}}

<section class="my-block">
  {{ with .heading }}<h2>{{ . }}</h2>{{ end }}     {{- /* gate every field, no fallbacks */ -}}
  {{ with .body }}<p>{{ . }}</p>{{ end }}

  {{ with .items }}
    <div class="grid">
      {{ range . }}
        <div class="card card-{{ .color }}">
          <strong>{{ .title }}</strong>
        </div>
      {{ end }}
    </div>
  {{ end }}

  {{- if $img -}}
    <img src="{{ $img }}" alt="{{ $alt }}">
  {{- end -}}
</section>
```

**Template rules**:

| Rule | Example |
|---|---|
| Gate every field with `{{with}}` — no hardcoded fallback content | `{{with .heading}}<h2>{{.}}</h2>{{end}}` ✓ &nbsp; `<h2>{{or .heading "Welcome"}}</h2>` ✗ |
| Empty fields render as nothing — never as canned strings | An unset CTA shouldn't render a "Click here" placeholder. |
| Default values for **non-content** primitives (limits, fallback colours) are OK | `{{- $limit := .limit -}}{{- if not $limit -}}{{- $limit = 3 -}}{{- end -}}` for a query limit is fine; never for visible copy. |
| Use `safeHTML` only for fields explicitly typed as HTML-bearing (`textarea` with HTML-allowed help, `richtext`) | Otherwise Go's auto-escaping protects you from XSS. |
| Use `image_url`/`image_srcset` for responsive imagery from the media library | `<img src="{{image_url $img "medium"}}" srcset="{{image_srcset $img "small" "medium" "large"}}">` |
| Don't reach into `.app` or `.node` from a block — they're not in scope | Use `filter "list_nodes"` to query, or fields to receive data. |

### 5.3 Scoped CSS / JS

Drop `style.css` or `script.js` next to `block.json` and the loader picks them up automatically:

- `<style data-block="<slug>">…</style>` is injected before the closing `</head>` (only on pages that use the block — there's a per-page used-blocks filter).
- `<script data-block="<slug>">…</script>` is injected before `</body>`.

Use this for **block-scoped** styles and behaviours. Site-wide CSS belongs in `assets/styles/theme.css`; site-wide JS in `assets/scripts/theme.js`.

### 5.4 Cross-block patterns

| Goal | Pattern |
|---|---|
| Render a list of nodes (e.g. testimonials, trips) | `{{ filter "list_nodes" (dict "type" "trip" "limit" 3 "order_by" "created_at asc") }}` |
| Fetch one node by ID | `{{ filter "get_node" (dict "id" 42) }}` |
| Derive distinct-tag pills from seeded nodes | `{{ filter "distinct_field" (dict "type" "trip" "field" "tag") }}` |
| Render a form (delegated to forms extension) | `{{ safeHTML (event "forms:render" (dict "form_id" "contact")) }}` |
| Fire an event with a payload | `{{ event "before_main_content" . }}` (returns HTML; layouts mostly use this) |

The filters above (`list_nodes`, `get_node`, `distinct_field`) are **theme-defined Tengo filters**. See [§8 Tengo seeding](#8-tengo-seeding-scriptsthemetengo) for how to register them.

---

## 6. Field types reference

The single source of truth for what `field_schema[].type` accepts and what `test_data` should look like.

| `type` | Editor UI | Stored as | `test_data` shape |
|---|---|---|---|
| `text` | single-line input | string | `"…"` |
| `textarea` | multi-line input | string | `"…"` |
| `richtext` | WYSIWYG editor | string (HTML) | `"<p>…</p>"` |
| `number` | number input | number | `42` |
| `toggle` / `checkbox` | switch / checkbox | boolean | `true` |
| `select` | dropdown | string (one of `options`) | `"red"` |
| `radio` | radio group | string | `"left"` |
| `color` | color picker | string (hex) | `"#FF0000"` |
| `link` | text + URL + target picker | object | `{"text": "Explore", "url": "/trips", "target": "_self"}` |
| `image` | media picker | object | `{"url": "theme-asset:hero", "alt": "Description"}` |
| `gallery` | multi-image picker | array of image objects | `[{"url": "theme-asset:a", "alt": "…"}, {"url": "theme-asset:b", "alt": "…"}]` |
| `term` | taxonomy term picker | object | `{"slug": "foodie", "name": "Foodie"}` (set `taxonomy: "trip_tag"` and `term_node_type: "trip"` in schema) |
| `node` | node picker | object | `{"id": 42, "slug": "hanoi-trip", "title": "Hanoi Street Food"}` (set `node_types: ["trip"]` or `node_type_filter: "trip"` in schema) |
| `form_selector` | forms-ext dropdown | string (form slug) | `"trip-order"` |
| `repeater` | nested array editor | array of objects | `[{...}, {...}]` (sub_fields define inner shape) |

**Helpful schema flags**:

```jsonc
{ "key": "tag",       "type": "term",     "taxonomy": "trip_tag", "term_node_type": "trip" }
{ "key": "color",     "type": "select",   "options": ["red", "yellow", "green", "ink"] }
{ "key": "trip",      "type": "node",     "node_types": ["trip"] }      // restrict picker
{ "key": "order_form","type": "form_selector" }                          // restrict to forms
{ "key": "items",     "type": "repeater", "sub_fields": [ … ] }
```

**`test_data` quality bar**:

- Use real, on-brand copy — no Lorem Ipsum.
- Use `theme-asset:<key>` for images — not absolute URLs and never `/theme/assets/...`.
- Provide a value for every field, even optional ones.
- For `term` fields: `{"slug": "foodie", "name": "Foodie"}` (slug-only is tolerated when the view only reads `.slug`, but ship both for the admin preview).
- For `node` fields: `{"slug": "hanoi-trip", "title": "Hanoi Street Food"}` (the engine resolves `id` at render time). Don't hardcode IDs.

---

## 7. Page templates (`templates/*.json`)

A template is a pre-built blocks layout an editor can apply to a new page in one click. Register every template in `theme.json` `templates[]`; the file lives at `templates/<slug>.json`.

```jsonc
{
  "name":        "Home",                                 // shown to editors
  "description": "Hero, categories, featured trip, …",   // shown to editors
  "thumbnail":   "",                                     // optional — preview image
  "blocks": [
    {
      "type": "hv-hero",                                 // matches a block.slug
      "fields": {                                        // values populated into the new page
        "heading_underlined": "Vietnam",
        "subheading":         "Small-group adventures …",
        "cta_primary":        { "text": "Find Your Adventure", "url": "/trips", "target": "_self" },
        "hero_image":         { "url": "theme-asset:hero-grandma", "alt": "Vietnamese grandma cooking" }
      }
    },
    {
      "type": "hv-categories",
      "fields": {
        "items": [
          { "title": "Foodie",    "tag": { "slug": "foodie" }, "color": "red" },
          { "title": "Adventure", "tag": { "slug": "adventure" }, "color": "green" }
        ]
      }
    }
  ]
}
```

**Template rules**:

- Every `type` matches a block slug declared in `theme.json` `blocks[]`.
- Every field used in the page must be present in `fields` — no implicit defaults.
- Every image is `theme-asset:<key>` so it survives a media-library re-import.
- Templates are **not** seeds — they don't auto-create pages. They populate the block editor when an editor clicks "Load template" on a new page. Use `theme.tengo` for seeding (see next section).

The on-disk JSON gets transformed by the loader into `template.block_config` rows of shape `[{block_type_slug, default_values}]`. The admin editor uses that to hydrate a fresh page.

---

## 8. Tengo seeding (`scripts/theme.tengo`)

Tengo is a small, Go-embedded scripting language. The theme's entry script runs **once on activation** and again on every restart while the theme is active. It's responsible for everything the manifest can't declare statically: custom node types, taxonomies, settings, menus, demo content, and event/filter/route registrations.

**Mandatory rule**: every block in the script is **idempotent**. Re-running theme.tengo on every boot must do nothing if state already exists. Use existence checks for content; Tengo's `upsert` semantics for menus and settings.

### 8.1 Available modules

```tengo
log         := import("core/log")          // log.info("…"), log.warn("…"), log.error("…"), log.debug("…")
nodes       := import("core/nodes")        // nodes.create / get / query / update / delete
nodetypes   := import("core/nodetypes")    // nodetypes.register
taxonomies  := import("core/taxonomies")   // taxonomies.register
settings    := import("core/settings")     // settings.get / set / all (prefix-filter)
menus       := import("core/menus")        // menus.get / list / upsert
filters     := import("core/filters")      // filters.add(name, script_path[, priority])
events      := import("core/events")       // events.emit / subscribe(action, script_path[, priority])
http        := import("core/http")         // http.get / post / put / patch / delete (route registration)
wellknown   := import("core/wellknown")    // wellknown.register("acme-challenge/*", "handlers/foo")
assets      := import("core/assets")       // assets.read("forms/contact.html")  (sandboxed to theme root)
routing     := import("core/routing")      // routing.is_homepage() etc. (only available in event handlers)
```

### 8.2 Common patterns

#### Register a custom node type with field schema

```tengo
nodetypes.register({
    slug: "trip",
    label: "Trip",
    label_plural: "Trips",
    icon: "map",
    description: "A bookable small-group adventure",
    url_prefixes: { en: "trips" },           // /trips/<slug>; can localise per-language
    field_schema: [
        { name: "tag",        type: "term", taxonomy: "trip_tag", term_node_type: "trip",
          help: "Pick a trip tag — managed in Taxonomies." },
        { name: "duration",   type: "text" },
        { name: "price",      type: "number" },
        { name: "hero_image", type: "image" },
        { name: "stops",      type: "repeater",
          sub_fields: [
            { name: "title", type: "text" },
            { name: "brief", type: "textarea" }
          ]
        },
        { name: "order_form", type: "form_selector",
          help: "Form rendered in the trip booking sidebar." }
    ],
    taxonomies: ["trip_tag"]
})
```

Note: in Tengo schemas the field key is `name`, not `key`. Block schemas in `block.json` use `key` because they go through a separate parser. Two different surfaces, same field types.

#### Register a taxonomy

```tengo
taxonomies.register({
    slug: "trip_tag",
    label: "Trip tag",
    label_plural: "Trip tags",
    description: "Foodie / Adventure / Relaxing",
    hierarchical: false,
    node_types: ["trip"]
})
```

> **Known limitation**: there is no `core/terms` Tengo module yet, so taxonomy *terms* (the actual `Foodie`, `Adventure` rows) must be created via the admin or MCP after first boot. Templates that derive pills from seeded node fields (`distinct_field` filter) bypass this gracefully — see hello-vietnam's `hv-trips-filter` block.

#### Seed site settings (idempotent)

```tengo
seed_setting := func(key, value) {
    existing := settings.get(key)
    if existing == "" || is_error(existing) {
        settings.set(key, value)
    }
}

seed_setting("hv.site_tagline",     "Small-group, locally-guided adventures across Vietnam.")
seed_setting("hv.whatsapp",         "+84 12 3456 7890")
seed_setting("hv.social.instagram", "https://instagram.com/hellovietnam")
```

Convention: namespace under your theme's prefix (`hv.*` for Hello Vietnam). Editors override via the Settings admin without losing your seeds.

#### Seed pages with blocks_data (existence check)

```tengo
page_missing := func(slug) {
    r := nodes.query({ node_type: "page", slug: slug, limit: 1 })
    return r.total == 0
}

if page_missing("home") {
    home := nodes.create({
        title: "Home",
        slug: "home",
        node_type: "page",
        status: "published",
        blocks_data: [
            { type: "hv-hero", fields: {
                heading_underlined: "Vietnam",
                subheading: "Small-group adventures…",
                cta_primary:   { text: "Find Your Adventure", url: "/trips", target: "_self" },
                cta_secondary: { text: "Meet the Crew",       url: "/about", target: "_self" },
                hero_image:    { url: "theme-asset:hero-grandma", alt: "Vietnamese grandma cooking" }
            } },
            { type: "hv-categories", fields: { items: [
                { title: "Foodie",    tag: { slug: "foodie" },    cta: { text: "Explore", url: "/trips?tag=foodie" },    color: "red" },
                { title: "Adventure", tag: { slug: "adventure" }, cta: { text: "Explore", url: "/trips?tag=adventure" }, color: "green" }
            ] } }
        ]
    })
    // Pin the homepage
    settings.set("homepage_node_id", string(home.id))
}
```

For collections (e.g. trips, testimonials, crew members), wrap with a count check:

```tengo
seed := func(node_type, label, items, default_layout) {
    if nodes.query({ node_type: node_type, limit: 1 }).total > 0 { return }
    for item in items {
        item.node_type = node_type
        item.status    = "published"
        if !is_undefined(default_layout) && default_layout != "" && (is_undefined(item.layout) || item.layout == "") {
            item.layout = default_layout
        }
        nodes.create(item)
    }
}

seed("trip", "trips", [
    { title: "Hanoi Street Food", slug: "hanoi-street-food",
      fields_data: { tag: "Foodie", price: 45, hero_image: { url: "theme-asset:trip-hanoi", alt: "…" } } }
], "trip")
```

`fields_data` (snake_case) is the Tengo-side property name for the JSONB `fields_data` DB column. From a block's `view.html`, you read this via `.fields_data` after iterating a `filter "list_nodes"` result.

#### Seed menus (slug-based, robust to renames)

```tengo
menus.upsert({
    slug: "main-nav",
    name: "Primary Navigation",
    items: [
        { label: "Home",    page: "home" },     // resolves slug → NodeID at upsert time
        { label: "Trips",   page: "trips" },
        { label: "About",   page: "about" },
        { label: "Contact", url: "/contact",   target: "_self" }   // explicit URL form
    ]
})
```

Use `page: "<slug>"` over `url: "/<slug>"` whenever possible — if an editor renames the page later, the menu item follows automatically.

#### Register Tengo filters

```tengo
filters.add("list_nodes",     "./filters/list_nodes")
filters.add("distinct_field", "./filters/distinct_field")
filters.add("get_node",       "./filters/get_node")
```

Each `<name>.tengo` file lives under `scripts/filters/`. From any template (block view or layout), call `{{ filter "list_nodes" (dict "type" "trip" "limit" 3) }}`. The script receives `value` (the input map) and sets `response` (whatever the template should receive).

Example filter (`scripts/filters/list_nodes.tengo`):

```tengo
nodes := import("core/nodes")
log   := import("core/log")

input := value
if is_undefined(input) { input = {} }

node_type := "article"
if !is_undefined(input.type) { node_type = input.type }
limit := 6
if !is_undefined(input.limit) { limit = input.limit }
order_by := "published_at desc"
if !is_undefined(input.order_by) { order_by = input.order_by }

res := nodes.query({ node_type: node_type, status: "published", limit: limit, order_by: order_by })
if is_error(res) {
    log.warn("list_nodes filter: nodes.query failed")
    response = []
} else {
    response = res.nodes
}
```

#### Register event handlers, HTTP routes, and `/.well-known/*` paths

```tengo
events.subscribe("node.published", "handlers/on_node_published")
events.subscribe("before_main_content", "hooks/hello_world", 10)  // optional priority

http.get("/search", "api/search")           // GET /api/theme/search
http.get("/nodes/:type", "api/nodes_by_type")

wellknown.register("security.txt", "handlers/security_txt")
```

Each handler script lives at the relative path passed in (e.g. `scripts/handlers/on_node_published.tengo`). HTTP handlers receive a `request` variable (`{ method, path, query, params, headers, body, ip }`) and set `response = { status, body, headers }`.

#### Seed forms (forms extension handshake)

See [§10 Forms wiring](#10-forms-wiring-forms-extension).

#### Read packaged files

`core/assets.read(rel_path)` reads a file relative to the theme root with path-traversal protection. Useful for shipping form layouts as separate HTML files instead of inlining them in the script:

```tengo
contact_layout := assets.read("forms/contact.html")
if is_error(contact_layout) {
    log.warn("contact layout missing: " + string(contact_layout))
    contact_layout = ""
}
```

---

## 9. Assets and `theme-asset:` references

The theme's `assets/` folder contains every static file the theme owns — images, fonts, raw JSON fixtures, anything. Two delivery paths:

### 9.1 Direct static path: `/theme/assets/<src>`

Used for:
- The CSS/JS files you declare in `theme.json` `styles[]` and `scripts[]`.
- The favicon (`<link rel="icon" href="/theme/assets/favicon.svg">`).
- Anything you reference by raw URL from a layout or partial that doesn't pass through field-driven render.

### 9.2 Resolved media library reference: `theme-asset:<key>`

This is the **only correct way** to reference theme images from within `image` fields — in `test_data`, in `templates/*.json`, and in `theme.tengo` `blocks_data`. Every image you reference this way must be declared in `theme.json` `assets[]`:

```jsonc
// theme.json
"assets": [
  { "key": "hero-grandma", "src": "images/hero-grandma.webp", "alt": "Vietnamese grandma cooking" }
]

// block.json — test_data
"hero_image": { "url": "theme-asset:hero-grandma", "alt": "Vietnamese grandma cooking" }

// theme.tengo — blocks_data
hero_image: { url: "theme-asset:hero-grandma", alt: "Vietnamese grandma cooking" }
```

**What the engine does at render time**:

1. The active asset registry resolves `theme-asset:hero-grandma` → `/media/theme/hello-vietnam/hero-grandma.webp` (the URL the media-manager extension assigned when it imported the file on `theme.activated`).
2. Resolution happens in `ResolveThemeAssetRefs` before the data hits `view.html` — your template only sees the resolved URL.
3. Path-traversal in keys is rejected (`^[a-z0-9_-]+$` enforced on `key`).

**Why this matters**: the indirection survives theme switches (the resolver swaps atomically), and lets the media-manager reroute the URL through cache/optimisation pipelines without touching your templates.

> **Anti-pattern**: hardcoding `/theme/assets/images/hero.webp` inside a `view.html` instead of declaring an `image` field. It works… until an editor wants to swap the photo, until you switch themes, until the file moves.

### 9.3 Image transformations

`/media/...` URLs run through the `image_url` and `image_srcset` template helpers:

```html
{{ with .hero_image }}
  {{ with .url }}
    <img
      src="{{ image_url . "medium" }}"
      srcset="{{ image_srcset . "small" "medium" "large" }}"
      sizes="(max-width: 600px) 100vw, 50vw"
      alt="{{ $.hero_image.alt }}">
  {{ end }}
{{ end }}
```

Sizes (`small`, `medium`, `large`, `thumbnail`, etc.) are managed by the media-manager extension and are configurable site-wide.

### 9.4 Theme-defined image sizes

A theme may need a variant the default sizes don't cover (a 450×350 showcase thumbnail, an editorial 4:3 card crop). Declare them in `theme.json` `image_sizes[]` and reference them through `image_url`:

```jsonc
// theme.json
"image_sizes": [
  { "name": "showcase-thumb", "width": 450, "height": 350, "mode": "crop" }
]
```

```html
{{- /* In a block view: assume `screenshot` is an `image` field. */ -}}
{{- $url := "" -}}{{- with .screenshot -}}{{- with .url -}}{{- $url = . -}}{{- end -}}{{- end -}}
{{- if $url -}}
  <img src="{{ image_url $url "showcase-thumb" }}"
       width="450" height="350"
       loading="lazy" decoding="async"
       alt="{{ .screenshot.alt }}">
{{- end -}}
```

**What happens at activation**: the theme.activated event carries `image_sizes[]` alongside `assets[]`. The media-manager extension upserts each entry into its `media_image_sizes` table (keyed by `name`, idempotent). If a row with the same name already exists from a different source (e.g. the admin or a previous theme), the manifest entry **does not** clobber it — admin-curated sizes win.

**Field reference**:

| Field | Required | Meaning |
|---|---|---|
| `name` | ✓ | Slug used in the URL: `/media/cache/<name>/<path>`. Must match `^[a-z0-9_-]+$`. |
| `width` / `height` | ✓ | Target dimensions in pixels. |
| `mode` | optional | `"fit"` (default — letterbox/contain) or `"crop"` (cover-and-trim). |
| `quality` | optional | JPEG/WebP quality 1–100. Omit (or 0) to use the site-wide default. |

**Don't ship a manifest size with a name the admin already tweaked** (`thumbnail`, `medium`, `large` are seeded by media-manager and protected). Use a theme-prefixed name (`<prefix>-thumb`, `<prefix>-card`) when in doubt.

**Fallback when the size isn't registered yet**: `image_url` rewrites `/media/foo.jpg` to `/media/cache/<name>/foo.jpg`. If the cache route 404s (size unknown to media-manager), the image won't render. On first activation the size is registered before assets are imported, so this is only relevant for cold deploys where media-manager hasn't booted yet — not a concern in normal use.

---

## 10. Forms wiring (forms extension)

Forms are owned by the **forms extension**, but a theme that wants on-brand forms (themed inputs, custom labels, consistent layout) registers its own form layouts via Tengo and renders them through a single template event.

### 10.1 Three-step handshake

**Step 1 — Drop a layout in `forms/<slug>.html`** (Go template). The forms extension exposes each form field at root via the field's `id`; rich metadata under `.fields_list` etc.:

```html
{{/* forms/contact.html */}}
<form style="display: grid; gap: 16px;">
  <div class="field">
    <label>{{ .name.label }}</label>
    <input class="input" name="{{ .name.id }}" placeholder="{{ .name.placeholder }}" {{ if .name.required }}required{{ end }} />
  </div>
  <div class="field">
    <label>{{ .email.label }}</label>
    <input class="input" name="{{ .email.id }}" type="email" {{ if .email.required }}required{{ end }} />
  </div>
  <div class="field">
    <label>{{ .subject.label }}</label>
    <select class="select" name="{{ .subject.id }}">
      <option value="">{{ .subject.placeholder }}</option>
      {{ range .subject.options }}<option value="{{ .value }}">{{ .label }}</option>{{ end }}
    </select>
  </div>
  <div class="field">
    <label>{{ .message.label }}</label>
    <textarea class="textarea" name="{{ .message.id }}" {{ if .message.required }}required{{ end }}></textarea>
  </div>
  <button class="btn btn-primary" type="submit">Send message</button>
</form>
```

Use **the theme's CSS classes** here (`.input`, `.btn`, `.field`) so the form looks 1:1 with the rest of the site. The forms-extension wraps this layout with success-state markup, honeypot, and the AJAX submit script.

**Step 2 — Seed the form via `forms:upsert`** in `theme.tengo`:

```tengo
core_events := import("core/events")
core_assets := import("core/assets")
log         := import("core/log")

contact_layout := core_assets.read("forms/contact.html")
if is_error(contact_layout) {
    log.warn("contact layout read failed: " + string(contact_layout))
    contact_layout = ""
}

core_events.emit("forms:upsert", {
    slug: "contact",
    name: "Contact",
    force: true,                 // overwrite even if editor changed labels — REMOVE in user-editable themes
    layout: contact_layout,
    settings: {
        success_message:  "Message sent! We'll reply within a day or two.",
        store_ip:         true,
        honeypot_enabled: true
    },
    fields: [
        { id: "name",    type: "text",     label: "Your name",  placeholder: "What should we call you?", required: true },
        { id: "email",   type: "email",    label: "Email",      placeholder: "you@somewhere.com",         required: true },
        { id: "subject", type: "select",   label: "Subject",    placeholder: "Pick one...",
          options: [
            { label: "General question",    value: "general" },
            { label: "Custom trip inquiry", value: "custom_trip" }
          ]
        },
        { id: "message", type: "textarea", label: "Message",    placeholder: "Tell us…", required: true }
    ]
})
```

`force: true` makes the upsert overwrite an existing form on every boot — useful while developing. **Drop `force: true` for shipping** so editor changes to labels/copy survive.

**Step 3 — Render from a block or layout** with the `forms:render` event:

```html
<div class="vibe-form-wrapper" data-form-slug="contact" id="form-contact">
  {{ safeHTML (event "forms:render" (dict "form_id" "contact")) }}
</div>
<link rel="stylesheet" href="/extensions/forms/blocks/vibe-form/style.css">
<script src="/extensions/forms/blocks/vibe-form/script.js" defer></script>
```

The forms-extension owns submission, validation, success state, and persistence. **The theme never intercepts the submit**. (Adding `data-newsletter-form` and a JS handler that fakes a "Thanks!" message is a real-bug pattern — submissions get silently dropped.)

### 10.2 Hidden fields per render

Pass dynamic hidden fields to the form (e.g. the trip ID a booking is for) via the second arg to `forms:render`:

```html
{{ safeHTML (event "forms:render" (dict
    "form_id" "trip-order"
    "hidden"  (dict
        "trip_slug"  $t.slug
        "trip_title" $t.title
        "trip_price" (or $t.fields.price 0)
    ))) }}
```

These end up as `<input type="hidden">` fields in the rendered form and are saved alongside the submission.

---

## 11. Site settings convention

Every theme exposes a handful of cross-cutting strings (tagline, social links, contact details, copyright). The convention:

1. **Namespace under the theme's prefix** — `<prefix>.<dot.path>` (e.g. `hv.whatsapp`, `hv.social.instagram`, `hv.lead_magnet.title`).
2. **Seed them in `theme.tengo`** with the idempotent `seed_setting` helper so editors can override without losing the value.
3. **Read them from layouts/partials** with `{{ index .app.settings "hv.whatsapp" }}`. (The setting is a string; cast/concatenate as needed.)

Example partial reading several:

```html
{{- $s := .app.settings -}}
{{- $tagline := or (index $s "hv.site_tagline") "" -}}
{{- $ig      := or (index $s "hv.social.instagram") "#" -}}
{{- $year    := or (index $s "hv.copyright_year") "2026" -}}
{{- $owner   := or (index $s "hv.copyright_owner") "Hello Vietnam" -}}
<footer>
  <p>{{ $tagline }}</p>
  <a href="{{ $ig }}">Instagram</a>
  <span>© {{ $year }} {{ $owner }}</span>
</footer>
```

Settings **belong to layouts/partials/Tengo**, not to blocks. A block needing site-wide data should declare it as a field — that way it's discoverable in the editor and can vary per page.

---

## 11a. Theme settings (editor-driven)

The convention in §11 covers settings the **theme** owns and seeds. **Theme settings** is the complement: a structured admin UI that lets editors edit theme-specific values (logo, palette swatches, copyright lines, integration keys, …) without touching `theme.tengo`. Each declared page becomes a sidebar entry in the admin under "Theme Settings"; values are stored in the same `site_settings` table under a per-theme namespace and are exposed to layouts, partials, blocks, and Tengo.

### Declaring pages in `theme.json`

Add a `settings_pages` array to the manifest. Each entry references an external JSON file so the manifest stays compact:

```jsonc
{
  "name":    "My Theme",
  "version": "1.0.0",
  // …
  "settings_pages": [
    { "slug": "header", "name": "Header Settings", "file": "settings/header.json", "icon": "PanelTop" },
    { "slug": "footer", "name": "Footer Settings", "file": "settings/footer.json", "icon": "PanelBottom" }
  ]
}
```

| Field | Required | Notes |
|---|---|---|
| `slug` | ✓ | URL slug + storage namespace (`^[a-z0-9_-]+$`). |
| `name` | ✓ | Sidebar label. Falls back to the per-page file's `name`. |
| `file` | ✓ | Path to the schema JSON file, relative to the theme directory. |
| `icon` | optional | Lucide-style icon name; defaults to a generic settings icon. |

A page whose `file` is missing or fails to parse is **skipped with a log warning** — the theme still activates. This is the same soft-fail policy used elsewhere in the loader.

### Per-page schema file

The referenced JSON file declares the page's fields:

```jsonc
// themes/my-theme/settings/header.json
{
  "name":        "Header",
  "description": "Logo, navigation behaviour, and contact CTA shown in the site header.",
  "fields": [
    { "key": "site_label", "label": "Site label",  "type": "text",
      "default": "My Theme", "help": "Falls back to .app.settings.site_name when empty." },
    { "key": "menu_depth", "label": "Menu depth",  "type": "number", "default": 2 },
    { "key": "sticky",     "label": "Sticky header", "type": "toggle", "default": true },
    { "key": "logo",       "label": "Logo",        "type": "image" },
    { "key": "cta_style",  "label": "CTA style",   "type": "select",
      "options": ["primary", "ghost", "link"], "default": "primary" }
  ]
}
```

Field shapes follow the same field-types reference used by blocks (see [§6](#6-field-types-reference)). Custom field types contributed by extensions work out of the box — the loader stores any unrecognised keys (`options`, `placeholder`, taxonomy hints, etc.) opaquely so the renderer's type registry can interpret them.

### Admin UI behaviour

- A "Theme Settings" sidebar group appears as soon as the active theme declares **one or more** pages. Themes with no `settings_pages` show no extra UI.
- Each page renders the same `CustomFieldInput` controls used by node-edit forms — visual parity with the rest of the admin is automatic.
- Values are **per theme**: switching themes hides the previous theme's pages but **preserves the stored rows**. Reactivating later restores the values.
- Saving a page replaces every field's value with the new typed payload; secret-shaped keys are auto-encrypted by the existing settings layer.

### Reading values from layouts and partials

The render context exposes `.theme_settings.<page>.<field>`:

```html
{{- $logo := .theme_settings.header.logo -}}
{{- with $logo.url -}}
  <img src="{{ image_url . "medium" }}" alt="{{ $logo.alt }}">
{{- end -}}

{{ if .theme_settings.header.sticky }}<body class="has-sticky-header">{{ end }}
```

### Reading values from blocks

Blocks don't see `.theme_settings` (they're sandboxed to their own field values), so two helpers are added to block templates:

```html
{{- $logo := themeSetting "header" "logo" -}}
{{- $hdr  := themeSettingsPage "header" -}}
{{ with $logo.url }}<img src="{{ . }}" alt="{{ $logo.alt }}">{{ end }}
{{ with $hdr.site_label }}<span class="brand">{{ . }}</span>{{ end }}
```

Both helpers return `nil` (or an empty map) when the active theme has no matching page/field — gate with `{{ with }}` as you would for any other field.

### Reading values from Tengo

```tengo
ts := import("core/theme_settings")

logo := ts.get("header", "logo")          // single field, coerced to its declared type
all  := ts.all("header")                  // map of every field on the page
pages := ts.pages()                       // [{slug, name, icon}, ...] for the active theme
slug  := ts.active_theme()                // active theme slug, "" when no theme is active
```

The module requires the `theme_settings:read` capability for non-internal callers (themes ship without explicit capability declarations and bypass the gate; extensions list it in `extension.json`'s `capabilities`). Returned shapes follow the field-type table in §6: text-types are strings, `number` is a float, `toggle` is a bool, structured types (`image`, `link`, `repeater`, …) are maps/arrays.

### Field-type mismatch behaviour

Stored values are **never auto-mutated** in the database, even when the schema changes:

- Schema changes from `text` to `media` after a value `"Hello"` was saved → render layer (template, block helpers, Tengo) returns the field's declared `default` and the admin form surfaces a "previous value" hint above the input.
- Saving the page with a new value replaces the stored payload with the new typed value.

This is a deliberate trade-off: forward-compatible (schema evolution doesn't lose data the editor might still want to recover), reverse-compatible (downgrading the schema doesn't corrupt the form).

### Storage model + lifecycle

Every setting is stored in the existing `site_settings` table under the key:

```
theme:<theme-slug>:<page-slug>:<field-key>
```

Lifecycle:

| Event | Effect |
|---|---|
| **Activate** | Registry populated from `theme.json` + per-page files. Bad pages logged & skipped. |
| **Deactivate** | Registry cleared. Stored rows **preserved** for later reactivation. |
| **Delete** | All rows under the theme's prefix wiped via `cms.DeleteThemeSettings`. |

### Sizing guideline

Keep the total fields under **~50 across all pages**. Long forms hurt UX — split into more pages instead of growing one page. The sidebar groups them naturally; a dozen narrowly-scoped pages reads better than one mega-form.

---

## 12. Template functions reference

Available in **every** theme template (layouts, partials, blocks, page templates, even string templates). Provided by the core renderer.

| Func | Signature | What it does |
|---|---|---|
| `safeHTML` | `safeHTML(any) → template.HTML` | Bypass HTML escaping. Use for HTML-bearing fields and `event` results. |
| `safeURL` | `safeURL(any) → template.URL` | Bypass URL escaping. |
| `dict` | `dict(k1, v1, k2, v2, …) → map` | Build a map literal — used to pass args to `filter`/`event`. |
| `list` | `list(a, b, c, …) → []any` | Build a slice literal. |
| `seq` | `seq(n) → [0..n-1]` | Range over an integer count. |
| `mod`/`add`/`sub` | `(a, b int) → int` | Integer math (Go templates have no infix). |
| `split` | `split(sep, s) → []string` | strings.Split. |
| `lastWord`/`beforeLastWord` | `(s) → string` | Useful for split-headline styling. |
| `json` | `json(any) → string` | Pretty-print as JSON (debugging). |
| `deref` | `deref(*T) → T` | Safe pointer deref. |
| `image_url` | `image_url(/media/... , size) → string` | Cached image URL. |
| `image_srcset` | `image_srcset(/media/... , sizes…) → string` | Build a srcset attribute value. |
| `filter` | `filter(name, value) → any` | Run a registered Tengo filter. |
| `event` | `event(name, ctx, args…) → template.HTML` | Fire an event; collect HTML responses. |
| `renderLayoutBlock` | `renderLayoutBlock(slug) → template.HTML` | **Layout/partial only.** Render a partial by slug. |
| `themeSetting` | `themeSetting(page, key) → any` | **Block templates.** One field from a theme settings page (see [§11a](#11a-theme-settings-editor-driven)). |
| `themeSettingsPage` | `themeSettingsPage(page) → map` | **Block templates.** Every field on a theme settings page. |

**Block templates and partials also receive standard Go html/template builtins**: `if`, `range`, `with`, `define`/`block`/`template`, `printf`, `urlquery`, `html`, `js`, `or`, `and`, `not`, `eq`/`ne`/`lt`/`gt`, `len`, `index`, `slice`, etc. See [Go's html/template docs](https://pkg.go.dev/html/template).

---

## 13. Build & live-reload loop

> **Activation is hot.** Switching themes via the admin or `core.theme.activate` MCP call takes effect immediately — no app/container restart, no plugin bounce. The asset registry swaps atomically and the new theme's CSS/JS is served on the next request.
>
> **Drop-in is hot too.** A startup-time fs watcher observes `themes/` and re-runs the registration scan whenever a new directory or `theme.json` appears, so `docker cp`, volume-mount updates, and git pulls register the theme without a restart. It shows up in `core.theme.list` immediately; activate it to make it serve. Ops scripts can call `core.theme.rescan` for an explicit trigger.
>
> The table below covers a different scenario: **editing files of an already-active theme**. The loader hashes file contents at startup and only re-upserts blocks/layouts/partials when the hash changes — which means in-place edits to layouts/blocks need an app restart for the loader to re-hash. (This is a development concern; in production, themes ship as part of the image.)

Themes are loaded fresh on every app boot. Most edits flow through automatically:

| What you edited | What to do | Why |
|---|---|---|
| `assets/styles/theme.css`, `assets/scripts/theme.js` | Restart app (`docker compose restart app`) | Picked up via `theme.json` declarations on next boot. Add `?v=N` to `src` to bust browser cache. |
| `layouts/*.html`, `partials/*.html` | Restart app | Loader hashes contents; re-upserts when changed. |
| `blocks/<slug>/{block.json, view.html, style.css, script.js}` | Restart app | Same hash mechanism — block_types row is updated on next boot. |
| `templates/*.json` | Restart app | Loader re-imports updated templates. |
| `theme.json` | Restart app | New blocks/layouts/partials/assets get registered. Removed entries become **dead rows** in the DB until you switch to another theme and back (which triggers full deregistration). |
| `scripts/theme.tengo` (and any sub-scripts) | Restart app | Re-runs on activation. |
| Existing seeded pages after schema changes | **Re-seed manually** if the existing rows have stale `blocks_data`. The seed is idempotent and does NOT overwrite existing pages. | See below. |

### Force-resyncing block schemas (if hash detection misses)

```sh
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "UPDATE block_types SET content_hash = 'force-' || floor(random()*1e6)::text WHERE source = 'theme';"
docker compose restart app
```

### Re-seeding stale demo pages

If `theme.tengo`'s `blocks_data` schemas changed but the seed already ran on a prior version, the existing pages keep their old fields. To re-seed:

```sh
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB <<'SQL'
DELETE FROM menu_items WHERE node_id IN (
  SELECT id FROM content_nodes WHERE node_type = 'page'
    AND slug IN ('home','about','trips','contact','gallery','legal')
);
DELETE FROM content_nodes WHERE node_type = 'page'
  AND slug IN ('home','about','trips','contact','gallery','legal');
DELETE FROM menus WHERE slug IN ('main-nav','footer-nav');
SQL
docker compose restart app
```

The seed will recreate the pages and re-attach the menus on next boot.

---

## 14. The Mandalorian rules

Twelve rules, the difference between "it works on my machine" and "ships clean from cold boot":

1. **Every block is a complete content type.** `block.json` declares every field its `view.html` reads. Every field has a value in `test_data`. Every field has a `help` line.
2. **Every template field is gated by `{{with}}`.** No hardcoded fallback strings in `view.html`. An empty field renders nothing.
3. **Every image goes through an `image` field with `theme-asset:<key>` test data.** Never hardcode `/theme/assets/images/...` in a `view.html`.
4. **Every image asset is declared in `theme.json` `assets[]`.** With a real `alt` text. The media-manager imports them on activation.
5. **Every page in the demo site has a matching `templates/<slug>.json`.** Editors get a one-click way to re-create your seeded pages.
6. **`theme.tengo` is idempotent.** Existence checks for content; upserts for menus/settings; `force: true` on form upserts only while developing.
7. **Cross-extension rendering goes through `event`.** Use `event "forms:render"` for forms; never hand-roll a `<form>` and JS-intercept the submit.
8. **Settings are namespaced.** `<prefix>.<dot.path>` so editors can override without collision.
9. **Menu items use `page: "<slug>"`, not `url: "/<slug>"`.** Menus survive page renames.
10. **`safeHTML` only on fields you trust.** Prefer Go's auto-escaping. Mark HTML-bearing fields explicitly in `block.json` `help`.
11. **Block-scoped CSS goes in `blocks/<slug>/style.css`.** Site-wide CSS goes in `assets/styles/theme.css`. Don't dump `<style>` blocks inside `view.html`.
12. **No dead schema fields.** If you remove a field from `view.html`, remove it from `block.json`, the `templates/*.json`, and the `theme.tengo` seed. Schema and template must agree.

**This is the way.**

---

## 15. Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Block doesn't appear in the admin block picker | Slug not in `theme.json` `blocks[]`, OR theme not active | `docker compose logs app | grep -i 'theme loaded'` to confirm registration count. Activate the theme in the admin if needed. |
| Block renders empty on the public page | Required field missing in `blocks_data`, or `{{with}}` gate hides everything | Open the page in the admin block editor; verify every field is populated. Check `view.html` for accidental wrapping `{{with}}` over the whole block. |
| Block edit form shows `[object Object]` for a field | `test_data` shape doesn't match the field type (e.g. `image` as a bare string) | Update `test_data` to match the [field types reference](#6-field-types-reference). |
| Block schema changes don't appear after restart | `content_hash` matches DB row (file changed but loader didn't notice) | Force-resync (see [§13 Build loop](#13-build--live-reload-loop)). |
| Seeded page renders with old block structure | Page exists from a previous theme version; seed is idempotent | Delete the page row + menu refs and restart (see [§13](#13-build--live-reload-loop)). |
| `theme-asset:<key>` shows up as `#ZgotmplZ` in HTML | Asset ref didn't resolve (typo, key not declared, theme not active) | Confirm `assets[]` entry; check media-manager imported the file (`/media/theme/<theme>/<file>` URL exists). |
| Form submissions silently disappear | Form rendered as raw `<form>` instead of via `event "forms:render"` | Replace with the [forms-ext handshake](#10-forms-wiring-forms-extension). |
| `{{ filter "list_nodes" … }}` returns nothing | Filter not registered (theme.tengo missed `filters.add`), OR `node_type` typo | Check boot logs for `filters registered`. Match the slug in `nodetypes.register`. |
| Layout reads `.app.settings.hv.whatsapp` but value is empty | Setting not seeded, OR `seed_setting` short-circuited because key already had `""` | Use `settings.set` directly to overwrite, or delete the row from `settings` table and let seed re-run. |
| Tengo script errors at boot but page still renders | `theme.tengo` errors are logged but non-fatal | Check `docker compose logs app | grep WARN` immediately after restart. |
| `renderLayoutBlock` does nothing in a block view | It's a layout-only function; blocks can't render partials | Call from the layout instead, or duplicate the markup into the block. |
| `{{partial …}}` errors | There's no `partial` template function. Use `{{renderLayoutBlock "<slug>"}}`. | Replace the call. |

---

## 16. Skeleton: copy-paste a new theme

The minimum that boots cleanly. Drop this in `themes/my-theme/`, restart, switch to it in the admin.

```
themes/my-theme/
├── theme.json
├── layouts/default.html
├── partials/site-header.html
├── partials/site-footer.html
├── blocks/intro/block.json
├── blocks/intro/view.html
├── templates/homepage.json
├── assets/styles/theme.css
├── assets/images/hero.webp                  # bring your own image
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

### `partials/site-header.html`

```html
{{- $menu := index .app.menus "main-nav" -}}
<header class="site-header">
  <a class="logo" href="/">My Theme</a>
  <nav>
    {{- if $menu -}}{{- range $menu.items -}}
      <a href="{{or .url "#"}}">{{.title}}</a>
    {{- end -}}{{- end -}}
  </nav>
</header>
```

### `partials/site-footer.html`

```html
{{- $year := or (index .app.settings "site.copyright_year") "2026" -}}
<footer class="site-footer">© {{$year}} My Theme.</footer>
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
    { "key": "body",    "label": "Body",    "type": "textarea", "help": "1–3 sentences of intro copy." },
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

### `templates/homepage.json`

```json
{
  "name": "Home",
  "description": "Single intro block.",
  "blocks": [
    {
      "type": "intro",
      "fields": {
        "heading": "Welcome.",
        "body":    "This is your new theme. Edit me in the admin.",
        "image":   { "url": "theme-asset:hero", "alt": "Hero photo" }
      }
    }
  ]
}
```

### `assets/styles/theme.css`

```css
body { font-family: system-ui, sans-serif; max-width: 720px; margin: 2rem auto; padding: 0 1rem; }
.site-header, .site-footer { padding: 1rem 0; border-bottom: 1px solid #ddd; }
.site-footer { border-top: 1px solid #ddd; border-bottom: none; }
.intro img { max-width: 100%; height: auto; }
```

### `scripts/theme.tengo`

```tengo
log      := import("core/log")
settings := import("core/settings")
menus    := import("core/menus")
nodes    := import("core/nodes")

log.info("My Theme initializing…")

// Site settings
seed_setting := func(key, value) {
    existing := settings.get(key)
    if existing == "" || is_error(existing) {
        settings.set(key, value)
    }
}
seed_setting("site.copyright_year", "2026")

// Seed homepage
home_id := 0
res := nodes.query({ node_type: "page", slug: "home", limit: 1 })
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
if home_id > 0 {
    settings.set("homepage_node_id", string(home_id))
}

// Primary nav
menus.upsert({
    slug: "main-nav",
    name: "Primary Navigation",
    items: [ { label: "Home", page: "home" } ]
})

log.info("My Theme initialization complete")
```

That's a complete, bootable theme. Add layouts, blocks, templates from there.

---

## 17. Reference: every block in `hello-vietnam`

The reference theme ships **25 blocks** across six demo pages. Use them as recipes.

| Slug | Where used | What it teaches |
|---|---|---|
| `hv-hero` | homepage | Multi-line headline split across fields, three rotating image fields, sticker badge, dual CTAs. |
| `hv-categories` | homepage | Repeater of category tiles with `term` field + auto-built `/trips?tag=<slug>` links. |
| `hv-featured` | homepage | Query-driven (filter `list_nodes` for one trip), single-block content. |
| `hv-how-it-works` | homepage | Dark-section repeater of steps; per-step accent colour. |
| `hv-popular-trips` | homepage | Query-driven (3 trips), block-level fields for eyebrow/heading/cta. |
| `hv-wall-of-love` | homepage | Query-driven testimonial grid with rotation modulo for visual variety. |
| `hv-lead-magnet` | homepage | Embedded `forms:render` for the newsletter; image field for the freebie mockup. |
| `hv-about-intro` | about | Two-column intro with image field and caption fallback. |
| `hv-timeline` | about | Vertical alternating timeline with per-entry accent. |
| `hv-crew-grid` | about | Polaroid grid of `crew_member` nodes with rotation pattern. |
| `hv-values` | about | Three-column values cards with icon `select`. |
| `hv-impact` | about | Two-column highlight with image, body, green CTA. |
| `hv-contact-band` | about | Dark-themed full-width CTA strip. |
| `hv-trips-filter` | trips | Search input + dynamic pill chips derived from `distinct_field` filter. |
| `hv-staff-pick` | trips | Optional `node` field with auto-fallback to oldest trip. |
| `hv-trips-grid` | trips | Query-driven 3-up grid with empty-state markup, client-side filter hooks. |
| `hv-trips-map` | trips | Mixed: data-bound region list + decorative pin overlay. |
| `hv-custom-journey` | trips | Soft-yellow CTA banner. |
| `hv-contact-intro` | contact | Centered eyebrow + H1 + body. |
| `hv-contact-form` | contact | Repeater of contact cards + delegated form rendering via `event "forms:render"`. |
| `hv-contact-faq` | contact | Side-heading + accordion repeater. |
| `hv-gallery-intro` | gallery | Centered title + dynamic filter pills bound to the masonry block's categories. |
| `hv-gallery-masonry` | gallery | Repeater of photos with per-tile colour, tall-toggle, category, lightbox JS hook. |
| `hv-gallery-cta` | gallery | Yellow band CTA. |
| `hv-legal-content` | legal | Long-form: heading + summary callout (repeater) + sections (repeater) + concern card. |

For trip-detail pages (`/trips/<slug>`), the **layout** `layouts/trip.html` does the rendering — it has full access to `.node.fields` and `.app.settings`, which a block view doesn't. This is the right pattern when you need the current node's full field data for a custom layout: build a layout, not a block.

---

## Appendix: Useful queries for development

```sh
# What blocks are registered?
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT slug, source, theme_name FROM block_types ORDER BY slug;"

# What did theme.tengo seed?
docker compose logs app --tail=200 | grep -E 'theme|seed'

# Inspect a seeded page's blocks_data
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT slug, jsonb_array_length(blocks_data) AS n_blocks FROM content_nodes WHERE node_type='page';"

# What hv.* settings exist?
docker compose exec -T db psql -U $POSTGRES_USER -d $POSTGRES_DB \
  -c "SELECT key, value FROM settings WHERE key LIKE 'hv.%' ORDER BY key;"
```

---

**Got an addition or correction?** Open a PR. The reference is `hello-vietnam/`, the contract is the engine. When they disagree, the engine wins — file an issue.
