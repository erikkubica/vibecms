# Squilla Theming Guide

The complete reference for building, customizing, and deploying Squilla themes.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Theme Structure](#2-theme-structure)
3. [theme.json Manifest](#3-themejson-manifest)
4. [Layouts](#4-layouts)
5. [Template Functions](#5-template-functions)
6. [Partials (Layout Blocks)](#6-partials-layout-blocks)
7. [Content Blocks](#7-content-blocks)
8. [Page Templates](#8-page-templates)
9. [Assets](#9-assets)
10. [Scripting Integration](#10-scripting-integration)
11. [Template Data Deep Dive](#11-template-data-deep-dive)
12. [Theme Installation & Deployment](#12-theme-installation--deployment)
13. [Best Practices](#13-best-practices)
14. [Complete Example Theme](#14-complete-example-theme)

---

## 1. Overview

A Squilla **theme** is a self-contained package that controls every aspect of how a site looks and behaves on the public-facing side. Themes are loaded at startup from disk, registered into the database, and rendered at runtime through Go's `html/template` engine.

### Key Concepts

- **Single-binary, single-site model** -- Squilla runs as one Go binary serving one site. The active theme is loaded from a directory on disk (configured via the `THEME_PATH` environment variable).
- **A theme is the sum of its parts** -- layouts, partials (layout blocks), content blocks, page templates, static assets, and optional Tengo scripts.
- **Zero-rebuild architecture** -- Themes are hot-loaded from disk and registered into the database. Changing the active theme does not require recompiling the binary or restarting the server.
- **Database-backed rendering** -- Layouts, partials, and block types are stored in the database after being loaded from theme files. This enables the admin UI to display and override them per-language.
- **Go `html/template` engine** -- All HTML rendering (layouts, partials, content blocks) uses the standard Go template engine with a shared context and custom functions.

---

## 2. Theme Structure

Every theme lives in a single directory. The directory name typically matches the theme slug. Here is the complete directory layout:

```
themes/my-theme/
  theme.json                  # Theme manifest (required)
  layouts/                    # Full-page layout templates (.html)
    default.html              # The default layout (DOCTYPE to </html>)
    blank.html                # A minimal layout without header/footer
  partials/                   # Reusable template fragments (.html)
    site-header.html          # Header partial
    site-footer.html          # Footer partial
    primary-nav.html          # Navigation menu partial
    user-menu.html            # Login/logout user menu
    footer-nav.html           # Footer navigation links
    language-switcher.html    # Multi-language switcher dropdown
  blocks/                     # Content block type definitions
    hero/                     # One directory per block type
      block.json              # Block schema definition (required)
      view.html               # Render template (required)
      style.css               # Scoped CSS (optional)
      script.js               # Scoped JS (optional)
    rich-text/
      block.json
      view.html
    faq/
      block.json
      view.html
      style.css
      script.js               # JS with fallback behavior
  templates/                  # Page templates (pre-configured block sets)
    homepage.json
    about.json
    contact.json
    landing.json
  assets/                     # Static files served at /theme/assets/
    styles/
      theme.css               # Global theme stylesheet
    images/
      logo.svg                # Theme logo and images
    scripts/
      custom.js               # Theme-level JavaScript
  scripts/                    # Tengo scripting (.tengo files)
    theme.tengo               # Entry point (registered at startup)
    hooks/
      hello_world.tengo       # Event handler scripts
    handlers/
      on_node_published.tengo # Lifecycle event handlers
    filters/
      site_title_suffix.tengo # Filter chain scripts
    api/
      search.tengo            # Custom REST API endpoints
      nodes_by_type.tengo
    lib/
      helpers.tengo           # Shared utility scripts
```

The only **required** file is `theme.json`. Everything else is optional -- include only what your theme needs.

---

## 3. theme.json Manifest

The `theme.json` file is the heart of every theme. It declares metadata, assets, layouts, partials, blocks, and templates. Squilla reads this file at theme load time and registers everything into the database.

### Full Schema

```json
{
  "name": "My Theme",
  "version": "1.0.0",
  "description": "A short description of what this theme provides",
  "author": "Your Name or Company",
  "styles": [
    {
      "handle": "theme-css",
      "src": "styles/theme.css",
      "position": "head",
      "defer": false,
      "deps": []
    }
  ],
  "scripts": [
    {
      "handle": "alpine-js",
      "src": "scripts/alpine.min.js",
      "position": "head",
      "defer": true,
      "deps": []
    },
    {
      "handle": "theme-js",
      "src": "scripts/theme.js",
      "position": "footer",
      "defer": true,
      "deps": ["alpine-js"]
    }
  ],
  "layouts": [
    {
      "slug": "default",
      "name": "Default Layout",
      "file": "default.html",
      "is_default": true
    },
    {
      "slug": "blank",
      "name": "Blank Layout",
      "file": "blank.html",
      "is_default": false
    }
  ],
  "partials": [
    {
      "slug": "site-header",
      "name": "Site Header",
      "file": "site-header.html"
    },
    {
      "slug": "site-footer",
      "name": "Site Footer",
      "file": "site-footer.html"
    }
  ],
  "blocks": [
    {
      "slug": "hero",
      "dir": "hero"
    },
    {
      "slug": "rich-text",
      "dir": "rich-text"
    }
  ],
  "templates": [
    {
      "slug": "homepage",
      "file": "homepage.json"
    },
    {
      "slug": "about",
      "file": "about.json"
    }
  ]
}
```

### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Human-readable theme name. Used in admin UI and logs. |
| `version` | string | Yes | Semantic version string (e.g. `"1.0.0"`). |
| `description` | string | No | Brief description of the theme. |
| `author` | string | No | Theme author name or organization. |
| `styles` | array | No | CSS assets to register. See [Assets](#9-assets). |
| `scripts` | array | No | JS assets to register. See [Assets](#9-assets). |
| `layouts` | array | No | Layout definitions. See [Layouts](#4-layouts). |
| `partials` | array | No | Partial (layout block) definitions. See [Partials](#6-partials-layout-blocks). |
| `blocks` | array | No | Content block type definitions. See [Content Blocks](#7-content-blocks). |
| `templates` | array | No | Page template definitions. See [Page Templates](#8-page-templates). |

### Asset Definition Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `handle` | string | -- | Unique identifier for dependency resolution. |
| `src` | string | -- | Path relative to the `assets/` directory. |
| `position` | string | `"footer"` | Where to inject: `"head"` or `"footer"`. Styles always go in head. |
| `defer` | bool | `false` | Whether to add the `defer` attribute to script tags. |
| `deps` | array | `[]` | Handles of assets that must load before this one. |

### Layout Definition Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `slug` | string | -- | Unique identifier used in node assignments and code. |
| `name` | string | -- | Human-readable name shown in admin UI. |
| `file` | string | -- | Filename relative to `layouts/` directory. |
| `is_default` | bool | `false` | If `true`, this layout is used when no specific layout is assigned to a node. Exactly one layout should be marked as default. |

### Partial Definition Fields

| Field | Type | Description |
|-------|------|-------------|
| `slug` | string | Unique identifier used with `renderLayoutBlock`. |
| `name` | string | Human-readable name shown in admin UI. |
| `file` | string | Filename relative to `partials/` directory. |

### Block Definition Fields

| Field | Type | Description |
|-------|------|-------------|
| `slug` | string | Unique identifier for this block type. |
| `dir` | string | Directory name under `blocks/` containing `block.json` and `view.html`. |

### Template Definition Fields

| Field | Type | Description |
|-------|------|-------------|
| `slug` | string | Unique identifier for this page template. |
| `file` | string | Filename relative to `templates/` directory (a JSON file). |

---

## 4. Layouts

Layouts define the **full HTML page structure** -- from `<!DOCTYPE html>` through `</html>`. They are Go `html/template` files that wrap the rendered content blocks for each page.

### How Layout Resolution Works

1. If a content node has a specific `layout_id` assigned, that layout is used.
2. Otherwise, the system looks for a language-specific default layout (matching the node's language).
3. If no language-specific layout exists, the universal default layout (`language_id = NULL`) is used.
4. If no layout is found at all, Squilla falls back to legacy file-based template rendering.

### Template Data Context

Every layout receives a flat map with three top-level namespaces: `.app`, `.node`, and `.user`. All keys use **snake_case** for consistency.

#### `.app` -- Application/Global Data

| Key | Type | Description |
|-----|------|-------------|
| `.app.menus` | `map[string]interface{}` | All resolved menus keyed by slug. Each menu has `id`, `slug`, `name`, `language_id`, and `items`. |
| `.app.settings` | `map[string]string` | All site settings keyed by setting key (e.g. `site_name`, `site_description`). |
| `.app.languages` | `[]map` | All active languages. Each entry has: `code`, `slug`, `name`, `native_name`, `flag`, `is_default`, `is_active`, `hide_prefix`. |
| `.app.current_lang` | `map` | The current page's language with the same fields as above. |
| `.app.head_styles` | `[]string` | Resolved CSS URLs to include in `<head>`. |
| `.app.head_scripts` | `[]string` | Resolved JS URLs to include in `<head>`. |
| `.app.foot_scripts` | `[]string` | Resolved JS URLs to include before `</body>`. |
| `.app.block_styles` | `template.HTML` | Pre-built `<style>` tags for all scoped block CSS. Insert directly in `<head>`. |
| `.app.block_scripts` | `template.HTML` | Pre-built `<script>` tags for all scoped block JS. Insert before `</body>`. |
| `.app.theme_url` | `string` | Base URL for theme static assets: `"/theme/assets"`. |

#### `.node` -- Current Page/Content Node Data

| Key | Type | Description |
|-----|------|-------------|
| `.node.id` | `int` | Database ID of the content node. |
| `.node.status` | `string` | Publication status: `"draft"`, `"published"`, `"archived"`. |
| `.node.title` | `string` | Page title. |
| `.node.slug` | `string` | URL slug (e.g. `"about-us"`). |
| `.node.full_url` | `string` | Complete URL path (e.g. `"/en/about-us"`). |
| `.node.blocks_html` | `template.HTML` | All content blocks pre-rendered as a single HTML string. Insert with `{{.node.blocks_html}}`. |
| `.node.fields` | `map[string]interface{}` | Custom fields from the node's `fields_data` JSONB column. |
| `.node.seo` | `map[string]interface{}` | SEO settings: `meta_title`, `meta_description`, `og_image`, etc. |
| `.node.node_type` | `string` | Content type slug (e.g. `"page"`, `"post"`, `"team-member"`). |
| `.node.language_code` | `string` | ISO language code (e.g. `"en"`, `"de"`, `"fr"`). |
| `.node.translations` | `[]map` | Translation siblings for language switcher. Each entry: `language_code`, `language_name`, `flag`, `title`, `full_url`, `is_current`. `nil` if no translation group exists. |

#### `.user` -- Current Visitor/User Data

| Key | Type | Description |
|-----|------|-------------|
| `.user.logged_in` | `bool` | Whether the visitor has an active session. |
| `.user.id` | `int` | User ID (0 if not logged in). |
| `.user.email` | `string` | User email (empty if not logged in). |
| `.user.role` | `string` | Role slug (e.g. `"admin"`, `"editor"`, `"member"`). |
| `.user.full_name` | `string` | User's display name. |

### Example Layout

```html
<!DOCTYPE html>
<html lang="{{.node.language_code}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{or (index .node.seo "meta_title") .node.title "My Site"}}</title>
    {{- with index .node.seo "meta_description" -}}
    <meta name="description" content="{{.}}">
    {{- end -}}

    {{- range .app.head_styles -}}
    <link rel="stylesheet" href="{{.}}">
    {{- end -}}
    {{.app.block_styles}}
</head>
<body>
    {{renderLayoutBlock "site-header"}}

    <main>
        {{event "before_main_content" .}}
        {{.node.blocks_html}}
        {{event "after_main_content" .}}
    </main>

    {{renderLayoutBlock "site-footer"}}

    {{- range .app.foot_scripts -}}
    <script src="{{.}}" defer></script>
    {{- end -}}
    {{.app.block_scripts}}
</body>
</html>
```

---

## 5. Template Functions

Squilla extends Go's `html/template` with custom functions available in all layouts and partials.

### `renderLayoutBlock`

```
{{renderLayoutBlock "slug"}}
```

Includes a **partial** (layout block) by its slug. The partial is resolved from the database, matching the current language first, then falling back to the universal (language-agnostic) version. The partial receives the same template data context as the parent layout.

Partials can nest other partials via `renderLayoutBlock`. Recursion is guarded to a **maximum depth of 5** to prevent infinite loops.

**Example:**
```html
<header>
    {{renderLayoutBlock "primary-nav"}}
    {{renderLayoutBlock "language-switcher"}}
</header>
```

### `event`

```
{{event "event_name" .}}
{{event "event_name" . arg1 arg2}}
```

Fires a **scripting event** and collects the returned HTML from all registered Tengo handlers. The first argument is the event name, the second is typically the full template context (`.`), and additional arguments are passed to handlers.

If no scripting engine is loaded or no handlers are registered, this returns an empty string.

**Example:**
```html
<main>
    {{event "before_main_content" .}}
    {{.node.blocks_html}}
    {{event "after_main_content" .}}
</main>
```

### `filter`

```
{{filter "filter_name" value}}
```

Runs a value through the **scripting filter chain**. Each registered filter script can transform the value before passing it to the next filter in priority order. If no filters are registered for the given name, the original value is returned unchanged.

**Example:**
```html
<title>{{filter "node.title" .node.title}}</title>
```

### `safeHTML`

```
{{safeHTML value}}
```

Marks a string as trusted HTML, preventing Go's template engine from escaping it. Accepts `string`, `template.HTML`, or any value (converted via `fmt.Sprintf`).

**Example:**
```html
<div class="content">{{safeHTML .node.blocks_html}}</div>
```

> **Note:** `.node.blocks_html` is already of type `template.HTML` and does not need `safeHTML`. Use this function for custom string fields that contain HTML.

### `deref`

```
{{deref pointer_value}}
```

Safely dereferences a Go pointer. Returns the pointed-to value, or a zero value (`""` for `*string`, `0` for `*int`) if the pointer is nil. Useful for optional fields that may be stored as pointers.

**Example:**
```html
{{- $subtitle := deref .node.fields.subtitle -}}
{{if $subtitle}}<p class="subtitle">{{$subtitle}}</p>{{end}}
```

### `json`

```
{{json value}}
```

JSON-encodes any value with indentation. Useful for debugging template data or embedding structured data in `<script>` tags.

**Example:**
```html
<script>
  window.__pageData = {{json .node.fields}};
</script>
```

### Standard Go Template Functions

All built-in Go template functions are also available:

- `{{if .condition}}...{{else}}...{{end}}`
- `{{range .items}}...{{end}}`
- `{{with .value}}...{{end}}`
- `{{index .map "key"}}`
- `{{len .slice}}`
- `{{or .value1 .value2 "default"}}`
- `{{eq .a .b}}`, `{{ne .a .b}}`, `{{lt .a .b}}`, `{{gt .a .b}}`
- `{{and .a .b}}`, `{{not .a}}`
- `{{printf "%s: %d" .name .count}}`

---

## 6. Partials (Layout Blocks)

Partials are reusable template fragments included in layouts via `{{renderLayoutBlock "slug"}}`. In Squilla, they are stored as **layout blocks** in the `layout_blocks` database table.

### How Partials Work

1. Theme partials are declared in `theme.json` under the `"partials"` array.
2. At theme load time, Squilla reads each partial's `.html` file and upserts it into the `layout_blocks` table with `source = "theme"`.
3. When a layout calls `{{renderLayoutBlock "site-header"}}`, the renderer:
   - Looks up the layout block by slug, preferring a language-specific version matching the current node's language.
   - Falls back to the universal version (`language_id = NULL`) if no language-specific version exists.
   - Parses the partial's template code with the same function map and data context as the parent layout.
   - Returns the rendered HTML.

### Recursive Nesting

Partials can include other partials. For example, `site-header` might include `primary-nav`, `language-switcher`, and `user-menu`:

```html
<!-- partials/site-header.html -->
<header>
    <div class="flex items-center justify-between">
        <a href="/">{{.app.settings.site_name}}</a>
        {{renderLayoutBlock "primary-nav"}}
        <div class="flex items-center gap-2">
            {{renderLayoutBlock "language-switcher"}}
            {{renderLayoutBlock "user-menu"}}
        </div>
    </div>
</header>
```

Recursion is limited to a **depth of 5** to prevent infinite loops. If a partial tries to include itself or creates a circular chain, rendering stops and a warning is logged.

### Language Overrides

Partials loaded from a theme are created as **universal** (`language_id = NULL`). Through the admin UI, you can create language-specific overrides of any partial. When rendering, the system automatically picks the correct version for the current language.

### Template Context

Partials receive the **exact same data context** as layouts: `.app`, `.node`, and `.user` with all their sub-fields. There is no separate or reduced context for partials.

---

## 7. Content Blocks

Content blocks are the building blocks of page content. Each block type defines a schema (fields) and a rendering template. Content editors assemble pages by adding, configuring, and ordering blocks in the admin UI.

### Directory Structure

Each block type lives in its own directory under `blocks/`:

```
blocks/hero/
  block.json    # Schema definition (required)
  view.html     # Render template (required)
  style.css     # Scoped CSS (optional)
  script.js     # Scoped JS (optional)
```

### block.json Schema

The `block.json` file defines the block's metadata and field schema:

```json
{
  "slug": "hero",
  "label": "Hero",
  "icon": "image",
  "description": "Full-width hero section with heading, subheading, background image, and CTA button",
  "field_schema": [
    {
      "key": "heading",
      "label": "Heading",
      "type": "text",
      "required": true
    },
    {
      "key": "subheading",
      "label": "Subheading",
      "type": "textarea"
    },
    {
      "key": "background_image",
      "label": "Background Image",
      "type": "image"
    },
    {
      "key": "button_text",
      "label": "Button Text",
      "type": "text"
    },
    {
      "key": "button_url",
      "label": "Button URL",
      "type": "text"
    },
    {
      "key": "alignment",
      "label": "Text Alignment",
      "type": "select",
      "options": ["left", "center", "right"],
      "default": "center"
    }
  ],
  "test_data": {
    "heading": "Build Something Amazing",
    "subheading": "A modern CMS designed for speed and flexibility.",
    "background_image": "https://example.com/hero.jpg",
    "button_text": "Get Started",
    "button_url": "/get-started",
    "alignment": "center"
  }
}
```

#### block.json Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `slug` | string | Yes | Unique identifier for this block type (must match the `slug` in `theme.json`). |
| `label` | string | Yes | Human-readable name shown in admin block picker. Falls back to slug if empty. |
| `icon` | string | No | Icon identifier for admin UI (defaults to `"square"`). |
| `description` | string | No | Brief description shown in block picker tooltip. |
| `field_schema` | array | Yes | Array of field definitions (see below). |
| `test_data` | object | No | Sample data for previewing the block in the admin. |

#### Field Schema Types

| Type | Description |
|------|-------------|
| `text` | Single-line text input. |
| `textarea` | Multi-line plain text. |
| `richtext` | Rich text editor (HTML output). Automatically marked as `template.HTML` during rendering. |
| `image` | Image URL (with media library picker in admin). |
| `select` | Dropdown selection. Requires `options` array. |
| `repeater` | Repeating group of sub-fields. Requires `sub_fields` array. |
| `group` | Nested group of sub-fields. Requires `sub_fields` array. |
| `node_selector` | Reference to another content node. Hydrated at render time with full node data. |

### view.html Templates

The `view.html` template renders a single instance of the block. It receives the block's **field values as the root context** -- each field key is directly accessible as a top-level template variable.

```html
{{- /* view.html for the hero block */ -}}

<section class="vb-hero relative w-full overflow-hidden"
  {{- if .background_image }} style="background-image: url('{{ .background_image }}')"{{ end }}>

  <div class="relative z-10 max-w-7xl mx-auto px-4 py-24">
    <h1 class="text-4xl font-bold text-white">{{ .heading }}</h1>

    {{- if .subheading }}
    <p class="text-xl text-gray-200 mt-4">{{ .subheading }}</p>
    {{- end }}

    {{- if and .button_text .button_url }}
    <a href="{{ .button_url }}" class="vb-hero__cta mt-8 inline-block px-8 py-4 text-lg font-semibold rounded-lg">
      {{ .button_text }}
    </a>
    {{- end }}
  </div>
</section>
```

**Key points:**

- Field values are the root context: use `{{.heading}}`, not `{{.fields.heading}}`.
- `richtext` fields are automatically converted to `template.HTML` so they render unescaped.
- Node selector fields are **hydrated** at render time -- a lightweight `{"id": 5}` reference is replaced with the full node data including all custom fields. You can access `{{.author.full_name}}`, `{{.author.avatar}}`, etc.
- Repeater fields are arrays: iterate with `{{range .features}}...{{end}}`.
- The `safeHTML` function is available in block templates for manually marking strings as safe HTML.

### Scoped CSS (`style.css`)

Each block can include a `style.css` file with scoped styles. These are automatically collected and injected as inline `<style data-block="slug">` tags in the page `<head>` via `{{.app.block_styles}}`.

```css
/* style.css for the hero block */

.vb-hero {
  min-height: 500px;
}

.vb-hero:not([style*="background-image"]) {
  background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
}

.vb-hero__overlay {
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.65), rgba(15, 23, 42, 0.80));
}
```

**Convention:** Prefix all CSS class names with `vb-{block-slug}` to prevent style collisions between blocks.

### Scoped JS (`script.js`)

Each block can include a `script.js` file with scoped JavaScript. These are collected and injected as inline `<script data-block="slug">` tags before `</body>` via `{{.app.block_scripts}}`.

```javascript
/* script.js for the faq block */
(function () {
  // Check if Alpine.js handles this; provide fallback if not
  if (window.Alpine) return;

  document.querySelectorAll('.vb-faq-item').forEach(function (item) {
    var button = item.querySelector('.vb-faq-question');
    var answer = item.querySelector('.vb-faq-answer');
    if (!button || !answer) return;

    answer.style.display = 'none';
    button.addEventListener('click', function () {
      var isOpen = answer.style.display !== 'none';
      answer.style.display = isOpen ? 'none' : 'block';
    });
  });
})();
```

**Best practice:** Wrap block scripts in an IIFE to avoid polluting the global scope.

### Block Rendering Pipeline

When a page is rendered, the block rendering pipeline works as follows:

1. The content node's `blocks_data` JSONB column is parsed into an array of `{type, fields}` objects.
2. For each block, the system looks up the matching `BlockType` by slug.
3. Field values are **hydrated** -- node selector references are resolved to full node data with all custom fields flattened into the map.
4. **Rich text marking** -- fields with `type: "richtext"` in the schema are converted from strings to `template.HTML` to prevent double-escaping. This applies recursively into `group` and `repeater` sub-fields.
5. The `view.html` template is parsed and executed with the field values as the root context.
6. All rendered block HTML strings are joined with newlines into `blocks_html`.
7. The combined `blocks_html` is injected into the layout at `{{.node.blocks_html}}`.

---

## 8. Page Templates

Page templates are **pre-configured sets of blocks** with default values. They provide a starting point when creating new pages -- instead of adding blocks one by one, the editor selects a template and gets a pre-populated page.

### Template File Format

Templates are JSON files in the `templates/` directory:

```json
{
  "name": "Homepage",
  "description": "A full homepage with hero, features, testimonials, and CTA",
  "blocks": [
    {
      "type": "hero",
      "fields": {
        "heading": "Build Something Amazing",
        "subheading": "A modern CMS designed for speed and flexibility.",
        "background_image": "https://example.com/hero.jpg",
        "button_text": "Get Started",
        "button_url": "/get-started",
        "alignment": "center"
      }
    },
    {
      "type": "features-grid",
      "fields": {
        "heading": "Everything You Need",
        "features": [
          { "icon": "lightning", "title": "Fast", "description": "Sub-50ms TTFB." },
          { "icon": "blocks", "title": "Block Editor", "description": "Visual page building." }
        ]
      }
    },
    {
      "type": "cta",
      "fields": {
        "heading": "Ready to Get Started?",
        "button_text": "Sign Up",
        "button_url": "/signup"
      }
    }
  ]
}
```

### Template File Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name in the admin template picker. Falls back to slug if empty. |
| `description` | string | Brief description shown in the template picker. |
| `blocks` | array | Ordered array of block configurations. |
| `blocks[].type` | string | Block type slug (must match a registered block type). |
| `blocks[].fields` | object | Default field values for this block instance. |

### How Templates Are Stored

During theme loading, the `{type, fields}` format is converted to `{block_type_slug, default_values}` and stored in the `templates` database table as JSONB `block_config`. Templates are tagged with `source = "theme"` and the theme name for tracking.

---

## 9. Assets

Theme assets are static files (CSS, JS, images, fonts) served from the theme's `assets/` directory at the URL path `/theme/assets/`.

### Registration in theme.json

Styles and scripts are declared in the `"styles"` and `"scripts"` arrays of `theme.json`. Each asset has a `handle`, `src`, and optional `position`, `defer`, and `deps` fields.

```json
{
  "styles": [
    { "handle": "theme-css", "src": "styles/theme.css", "position": "head" },
    { "handle": "google-fonts", "src": "styles/fonts.css", "position": "head" }
  ],
  "scripts": [
    { "handle": "alpine-js", "src": "scripts/alpine.min.js", "position": "head", "defer": true },
    { "handle": "theme-js", "src": "scripts/theme.js", "position": "footer", "defer": true, "deps": ["alpine-js"] }
  ]
}
```

### URL Resolution

All asset `src` paths are relative to the `assets/` directory. They are resolved to URLs with the `/theme/assets/` prefix:

- `"src": "styles/theme.css"` becomes `/theme/assets/styles/theme.css`
- `"src": "scripts/theme.js"` becomes `/theme/assets/scripts/theme.js`

### Positioning

- **Styles** always go in `<head>`, regardless of the `position` field.
- **Scripts** default to `"footer"` position. Set `"position": "head"` to place them in `<head>`.

### Dependency Resolution

Scripts support dependency ordering via the `deps` array. Squilla uses **topological sorting** (Kahn's algorithm) to ensure dependencies load before their dependents.

```json
{
  "scripts": [
    { "handle": "jquery", "src": "scripts/jquery.min.js", "position": "head" },
    { "handle": "slick", "src": "scripts/slick.min.js", "deps": ["jquery"] },
    { "handle": "theme-js", "src": "scripts/theme.js", "deps": ["jquery", "slick"] }
  ]
}
```

This guarantees load order: `jquery` -> `slick` -> `theme-js`.

Circular dependencies are detected and logged as warnings; the circular script is skipped.

### Rendering in Layouts

Use the asset arrays in your layout to render link/script tags:

```html
<head>
    {{- range .app.head_styles -}}
    <link rel="stylesheet" href="{{.}}">
    {{- end -}}
    {{.app.block_styles}}
</head>
<body>
    <!-- page content -->

    {{- range .app.foot_scripts -}}
    <script src="{{.}}" defer></script>
    {{- end -}}
    {{.app.block_scripts}}
</body>
```

### Direct Asset References

In addition to registered assets, you can reference any file in the `assets/` directory using the `theme_url` helper:

```html
<img src="{{.app.theme_url}}/images/logo.svg" alt="Logo">
```

---

## 10. Scripting Integration

Squilla includes an embedded **Tengo** scripting engine that allows themes to add custom logic without modifying Go code or restarting the server. Scripts run in a sandboxed VM with restricted I/O.

### Entry Point

The file `scripts/theme.tengo` is the scripting entry point. It runs once at startup and registers event handlers, filters, and custom API routes.

```tengo
// scripts/theme.tengo
log := import("core/log")
events := import("core/events")
http := import("core/http")
filters := import("core/filters")

log.info("Theme scripts initializing...")

// Register event handlers
events.on("before_main_content", "hooks/hello_world", 10)
events.on("node.published", "handlers/on_node_published")

// Register custom API endpoints at /api/theme/*
http.get("/search", "api/search")
http.get("/nodes/:type", "api/nodes_by_type")

// Register filter chains
filters.add("node.title", "filters/site_title_suffix", 90)

log.info("Theme scripts loaded!")
```

### Events in Layouts

The `{{event "name" .}}` template function connects layouts to the scripting event system. When called, it:

1. Looks up all Tengo handlers registered for the given event name.
2. Runs them in priority order (lower number = runs first).
3. Collects returned HTML fragments.
4. Returns the concatenated HTML.

### Filters in Templates

The `{{filter "name" value}}` function runs a value through all registered filter scripts for the given name. Filters transform values through a chain -- each filter receives the output of the previous one.

### Further Reading

For the complete Tengo scripting API, including available imports, the request/response model for HTTP handlers, and the event lifecycle, see `docs/scripting_api.md`.

---

## 11. Template Data Deep Dive

This section provides practical examples of working with the template data context.

### Accessing Site Settings

```html
<!-- Site name from settings -->
<h1>{{.app.settings.site_name}}</h1>

<!-- Conditional based on a setting -->
{{if .app.settings.google_analytics_id}}
<script async src="https://www.googletagmanager.com/gtag/js?id={{.app.settings.google_analytics_id}}"></script>
{{end}}
```

### Working with Menus

Menus are keyed by slug in `.app.menus`. Each menu has `items`, and each item can have `children` for dropdown sub-menus:

```html
{{- $menu := index .app.menus "main-nav" -}}
{{- if $menu -}}
<nav>
    {{- range $menu.items -}}
    <div>
        <a href="{{.url}}" class="{{.css_class}}">{{.title}}</a>
        {{- if .children -}}
        <ul>
            {{- range .children -}}
            <li><a href="{{.url}}" target="{{.target}}">{{.title}}</a></li>
            {{- end -}}
        </ul>
        {{- end -}}
    </div>
    {{- end -}}
</nav>
{{- end -}}
```

**Menu item fields:** `id`, `title`, `item_type` (`"url"` or `"node"`), `url` (auto-resolved for node items), `target` (`"_self"` or `"_blank"`), `css_class`, `children`, and optionally `node_id`.

### Language Switcher

Use `.node.translations` to build a language switcher. This array is only populated when the current node belongs to a translation group with published siblings:

```html
{{- $translations := .node.translations -}}
{{- if $translations -}}
<div class="language-switcher">
    {{- range $translations -}}
    <a href="{{.full_url}}"
       class="{{if .is_current}}active{{end}}">
        <span>{{.flag}}</span>
        <span>{{.language_name}}</span>
    </a>
    {{- end -}}
</div>
{{- end -}}
```

**Translation entry fields:** `language_code`, `language_name`, `flag` (emoji), `title`, `full_url`, `is_current` (bool).

### SEO Data

Access SEO settings from `.node.seo`:

```html
<head>
    <title>{{or (index .node.seo "meta_title") .node.title "Default Title"}}</title>

    {{- with index .node.seo "meta_description" -}}
    <meta name="description" content="{{.}}">
    {{- end -}}

    {{- with index .node.seo "og_image" -}}
    <meta property="og:image" content="{{.}}">
    {{- end -}}

    <meta property="og:title" content="{{or (index .node.seo "meta_title") .node.title}}">
    <meta property="og:url" content="{{.node.full_url}}">
    <meta property="og:type" content="{{if eq .node.node_type "post"}}article{{else}}website{{end}}">
</head>
```

### Custom Fields

Access node custom fields via `.node.fields`:

```html
{{- with .node.fields.subtitle -}}
<p class="subtitle">{{.}}</p>
{{- end -}}

{{- with .node.fields.featured_image -}}
<img src="{{.}}" alt="{{$.node.title}}">
{{- end -}}
```

### Conditional User Content

Show different content based on login state:

```html
{{if .user.logged_in}}
    <span>Welcome, {{.user.full_name}}</span>
    <a href="/admin">Dashboard</a>
    <a href="/logout">Logout</a>
{{else}}
    <a href="/login">Login</a>
    <a href="/register">Register</a>
{{end}}

{{- /* Show admin-only link */ -}}
{{if eq .user.role "admin"}}
    <a href="/admin/settings">Site Settings</a>
{{end}}
```

### Current Language Info

```html
<html lang="{{.app.current_lang.code}}">

<!-- Show all available languages -->
{{range .app.languages}}
    <span>{{.flag}} {{.native_name}}</span>
{{end}}
```

---

## 12. Theme Installation & Deployment

### Loading at Startup

Squilla loads the active theme from the directory specified by the `THEME_PATH` environment variable:

```bash
export THEME_PATH=/path/to/themes/my-theme
```

At startup, the `ThemeLoader` reads `theme.json` from this directory and:

1. Registers all styles and scripts into the in-memory `ThemeAssetRegistry`.
2. Upserts layouts into the `layouts` database table.
3. Upserts partials into the `layout_blocks` table.
4. Upserts block types into the `block_types` table (including reading `view.html`, `style.css`, `script.js`).
5. Upserts page templates into the `templates` table.
6. Creates or updates a record in the `themes` table.

### Zip Upload

Install a theme via the admin API by uploading a `.zip` archive:

```
POST /admin/api/themes/upload
Content-Type: multipart/form-data
```

The system:
1. Extracts the ZIP to a temp directory (with zip-slip protection).
2. Locates `theme.json` at root level or one directory deep.
3. Reads the `slug` field from the manifest (required for installation).
4. Copies the theme to `themes/{slug}/`.
5. Creates a database record with `source = "upload"`.

### Git-Based Deployment

Install a theme from a Git repository:

```
POST /admin/api/themes/git
{
  "git_url": "https://github.com/org/my-theme.git",
  "branch": "main",
  "token": "github_pat_xxx"
}
```

The system clones the repository (shallow, single-branch) and installs it the same way as a ZIP upload. Git-sourced themes support:

- **Pull updates** (`POST /admin/api/themes/{id}/pull`) -- runs `git pull` and reloads the theme if active.
- **Webhook integration** -- configure your Git provider to call the pull endpoint on push for automatic deployments.
- **Token authentication** -- HTTPS URLs can include an OAuth2 token for private repositories.

### Theme Activation

Only one theme can be active at a time. Activate a theme via:

```
POST /admin/api/themes/{id}/activate
```

This deactivates all other themes and reloads the target theme (re-registering all layouts, partials, blocks, and assets).

### Theme Deactivation and Deletion

```
POST /admin/api/themes/{id}/deactivate
DELETE /admin/api/themes/{id}
```

Active themes cannot be deleted -- deactivate first. Deletion removes both the database record and the filesystem directory.

---

## 13. Best Practices

### Performance

- **Minimize external requests** -- Bundle CSS and JS into theme assets rather than loading from CDNs in production. The default theme uses CDN links for convenience during development.
- **Use scoped block CSS/JS sparingly** -- These are inlined as `<style>` and `<script>` tags on every page that uses the block. For large styles, consider a shared stylesheet in `assets/`.
- **Leverage template caching** -- In production mode (`isDev = false`), templates are parsed once and cached. Avoid dynamic template generation patterns.

### Template Organization

- **One responsibility per partial** -- Keep partials focused. A `site-header` partial should compose navigation and user menu from smaller partials rather than containing everything inline.
- **Use `renderLayoutBlock` for reusable fragments** -- If the same HTML appears in multiple layouts, extract it into a partial.
- **Prefix block CSS classes** -- Use `vb-{block-slug}` convention (e.g. `.vb-hero`, `.vb-hero__cta`) to avoid style collisions.
- **Wrap block JS in IIFEs** -- Prevent global scope pollution from block scripts.

### Asset Management

- **Declare all assets in theme.json** -- Do not hardcode `<link>` or `<script>` tags in layouts unless they are truly external (CDNs). Registered assets participate in dependency resolution and can be managed programmatically.
- **Use dependency ordering** -- If your theme JS depends on jQuery or Alpine.js, declare the dependency in `deps` rather than relying on script tag order.
- **Place non-critical scripts in footer** -- Use `"position": "footer"` (the default) for scripts that do not affect above-the-fold rendering.

### Scripting

- **Keep `theme.tengo` light** -- Use it only for registration. Put actual handler logic in separate files under `hooks/`, `handlers/`, `filters/`, and `api/`.
- **Use priorities wisely** -- Lower numbers run first. Default is 50. Use 10 for early hooks, 90 for late filters.
- **Scripts are sandboxed** -- They cannot access the filesystem, network, or system resources directly. Use the provided `core/*` imports.

### Themes and Multi-Language

- **Theme partials are universal** -- They are created with `language_id = NULL`. For language-specific overrides, create them through the admin UI.
- **Use `.node.translations` for language switchers** -- Do not hardcode language lists; use the dynamic translation data.
- **Test with multiple languages** -- Ensure layouts handle missing translations gracefully (check for `nil` before iterating).

---

## 14. Complete Example Theme

Here is a minimal but complete theme demonstrating all the major features working together.

### theme.json

```json
{
  "name": "Agency Starter",
  "version": "1.0.0",
  "description": "A clean, minimal theme for agency websites",
  "author": "Squilla",
  "styles": [
    { "handle": "theme-css", "src": "styles/main.css", "position": "head" }
  ],
  "scripts": [
    { "handle": "theme-js", "src": "scripts/main.js", "position": "footer", "defer": true }
  ],
  "layouts": [
    { "slug": "default", "name": "Default", "file": "default.html", "is_default": true },
    { "slug": "landing", "name": "Landing Page", "file": "landing.html", "is_default": false }
  ],
  "partials": [
    { "slug": "header", "name": "Header", "file": "header.html" },
    { "slug": "footer", "name": "Footer", "file": "footer.html" },
    { "slug": "nav", "name": "Navigation", "file": "nav.html" }
  ],
  "blocks": [
    { "slug": "hero", "dir": "hero" },
    { "slug": "rich-text", "dir": "rich-text" }
  ],
  "templates": [
    { "slug": "homepage", "file": "homepage.json" }
  ]
}
```

### layouts/default.html

```html
<!DOCTYPE html>
<html lang="{{.app.current_lang.code}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{or (index .node.seo "meta_title") .node.title}} | {{.app.settings.site_name}}</title>
    {{- with index .node.seo "meta_description" -}}
    <meta name="description" content="{{.}}">
    {{- end -}}
    {{- range .app.head_styles -}}
    <link rel="stylesheet" href="{{.}}">
    {{- end -}}
    {{.app.block_styles}}
</head>
<body>
    {{renderLayoutBlock "header"}}

    <main>
        {{event "before_main_content" .}}
        {{.node.blocks_html}}
        {{event "after_main_content" .}}
    </main>

    {{renderLayoutBlock "footer"}}

    {{- range .app.foot_scripts -}}
    <script src="{{.}}" defer></script>
    {{- end -}}
    {{.app.block_scripts}}
</body>
</html>
```

### partials/header.html

```html
<header class="site-header">
    <div class="container">
        <a href="/" class="logo">
            <img src="{{.app.theme_url}}/images/logo.svg" alt="{{.app.settings.site_name}}">
        </a>
        {{renderLayoutBlock "nav"}}
        {{if .user.logged_in}}
            <a href="/admin">Dashboard</a>
        {{else}}
            <a href="/login">Login</a>
        {{end}}
    </div>
</header>
```

### partials/nav.html

```html
{{- $menu := index .app.menus "main-nav" -}}
{{- if $menu -}}
<nav>
    {{- range $menu.items -}}
    <a href="{{.url}}">{{.title}}</a>
    {{- end -}}
</nav>
{{- end -}}
```

### partials/footer.html

```html
<footer class="site-footer">
    <div class="container">
        <p>&copy; 2026 {{.app.settings.site_name}}. All rights reserved.</p>
    </div>
</footer>
```

### blocks/hero/block.json

```json
{
  "slug": "hero",
  "label": "Hero Banner",
  "icon": "image",
  "description": "Full-width hero with heading and call-to-action",
  "field_schema": [
    { "key": "heading", "label": "Heading", "type": "text", "required": true },
    { "key": "subheading", "label": "Subheading", "type": "textarea" },
    { "key": "button_text", "label": "Button Text", "type": "text" },
    { "key": "button_url", "label": "Button URL", "type": "text" }
  ],
  "test_data": {
    "heading": "Welcome to Our Agency",
    "subheading": "We build amazing digital experiences.",
    "button_text": "Contact Us",
    "button_url": "/contact"
  }
}
```

### blocks/hero/view.html

```html
<section class="vb-hero">
    <h1>{{.heading}}</h1>
    {{- if .subheading -}}
    <p>{{.subheading}}</p>
    {{- end -}}
    {{- if and .button_text .button_url -}}
    <a href="{{.button_url}}" class="vb-hero__cta">{{.button_text}}</a>
    {{- end -}}
</section>
```

### blocks/hero/style.css

```css
.vb-hero {
    padding: 6rem 2rem;
    text-align: center;
    background: linear-gradient(135deg, #1e293b, #0f172a);
    color: white;
}
.vb-hero h1 { font-size: 3rem; margin-bottom: 1rem; }
.vb-hero p { font-size: 1.25rem; opacity: 0.8; margin-bottom: 2rem; }
.vb-hero__cta {
    display: inline-block;
    padding: 0.75rem 2rem;
    background: #4f46e5;
    color: white;
    border-radius: 0.5rem;
    text-decoration: none;
}
.vb-hero__cta:hover { background: #4338ca; }
```

### templates/homepage.json

```json
{
  "name": "Homepage",
  "description": "A simple homepage with hero and content area",
  "blocks": [
    {
      "type": "hero",
      "fields": {
        "heading": "Welcome",
        "subheading": "This is your new website. Start editing to make it yours.",
        "button_text": "Learn More",
        "button_url": "/about"
      }
    },
    {
      "type": "rich-text",
      "fields": {
        "content": "<h2>About Us</h2><p>Tell your visitors about your company and what makes you different.</p>"
      }
    }
  ]
}
```

### scripts/theme.tengo

```tengo
log := import("core/log")
events := import("core/events")

log.info("Agency Starter theme initializing...")

// Add a welcome banner before main content
events.on("before_main_content", "hooks/welcome_banner", 10)

log.info("Agency Starter theme loaded!")
```

---

This completes the Squilla theming reference. For questions about the Tengo scripting API, see `docs/scripting_api.md`. For admin UI customization, see `docs/admin_ui.md`.

---

## Appendix A — Common silent-failure modes

A real-world theme port (`docs/theme-build-notes.md`) catalogued ~40
silent failures. Most are now fail-loud — log warnings or hard rejection
at theme load. This appendix is a quick-scan reference. The MCP tool
`core.guide` also returns a machine-readable `gotchas` array covering
the same content.

### Data-shape asymmetries (the silent data-loss family)

| Where | Required key | Wrong (silent) | Note |
|---|---|---|---|
| `nodes.create({...})` top level | `fields_data:` | `fields:` | Now logs a warning on misuse. |
| Block inside `blocks_data: [{type, ...}]` | `fields:` | `fields_data:` | Now logs a warning on misuse. |
| `block.json` `field_schema` entry | `key:` | `name:` | Theme loader now hard-rejects. |
| `nodetypes.register({field_schema:[...]})` | `name:` | `key:` | Auto-falls back to `key` for compatibility but stay consistent. |
| `block.json` `select`/`radio` options | `["a","b"]` | `[{value,label}]` | Theme loader now hard-rejects (used to crash admin with React #31). |
| `term`-typed schema entry | `term_node_type:` set | omitted | Logs a warning at register time; hydration won't match. |
| Term-typed field value | `{slug, name}` object | bare slug string | Admin can't pre-select bare strings; templates handle both. |
| Real taxonomy on a node | `taxonomies: { tax: [slugs] }` | `fields_data: { tax: [...] }` | The taxonomies tab and `tax_query` only see the `taxonomies` JSONB column. |
| Settings template lookup | `index $s "key.with.dots"` or `mustSetting $s "k"` | `.app.settings.key.with.dots` | Settings keys keep their dots — Go templates can't dot-traverse them. |

### Fail-loud helpers

- **`mustSetting $settings "<key>"`** — errors loudly when a required
  setting is missing or empty. Use for any setting your template can't
  render correctly without.
- **`setting $settings "<key>"`** — graceful: returns "" on miss. Use
  for optional settings only.

### Tengo language gotchas

- `log.error("…")` is a **parse error** — `error` is a reserved selector.
  Use `log.warn(…)`, `log.info(…)`, or the alias `log.err(…)`.
- `is_string`, `is_undefined`, `is_error` are built-ins. Use them on
  optional map keys (no exception thrown for missing key).
- Tengo imports are relative without extension: `import("./setup/foo")`.
  Each module needs `export {…}`.
- A bare top-level `return` inside a filter terminates the script.
  Wrap in `if/else` so `response =` is set first.

### Cache & lifecycle

| Change | What invalidates the cache |
|---|---|
| Edit `view.html` / `block.json` | Re-activate the theme (or wait for `content_hash` resync to detect file changes). |
| Edit `layouts/*.html` / `partials/*.html` | Re-activate. Layouts/partials only re-read at activation. |
| `core.settings.set(...)` | Now publishes `setting.updated` and busts the in-process settings cache. |
| `core.theme.activate` | Busts everything: layouts, partials, blocks, settings. |

### Theme HTTP routes

`routes.register("GET", "/docs", "./routes/docs")` mounts the handler at
**`/api/theme/docs`** — NOT at `/docs`. Themes cannot shadow public node
routes. To redirect a bare path, point a menu link directly at the
destination, or use an extension `public_route` (extensions are not
prefixed).

### Filter usage

`{{ filter "name" }}` (no value arg) throws `"wrong number of args"`.
For input-less filters pass an empty dict:
```html
{{ $things := filter "list_things" (dict) }}
```

Filters defined in `scripts/filters/*.tengo` are auto-loaded as importable
modules but **not** registered as named filter handlers — register
explicitly:
```tengo
filters := import("core/filters")
filters.add("list_docs", "./filters/list_docs")
```

### Dev-mode iteration

Set `SQUILLA_DEV_MODE=true` in dev environments. Seeds receive a
top-level `dev_mode` boolean. Branch on it to overwrite-on-reseed for
fast iteration; production stays idempotent because the env var is unset:

```tengo
res := nodes.query({ node_type: "page", slug: "home", limit: 1 })
if res.total > 0 && dev_mode {
    nodes.delete(res.nodes[0].id)
    res = { total: 0 }
}
if res.total == 0 {
    nodes.create({ ... })
}
```

### Production-readiness checklist

- Run `core.theme.checklist({ slug: "<slug>" })` for automated structural
  checks (theme.json validity, schemas, slug prefixing, Tengo gotchas).
- Walk `docs/theme-checklist.md` for the manual checks (admin UX,
  public render, idempotency).
- Don't claim done until both pass.

## Appendix B — Template function reference (verified whitelist)

| Function | Purpose |
|---|---|
| `safeHTML s` | Bypass HTML escaping. Use only on trusted strings or `event` results. |
| `safeURL s` | Bypass URL escaping. |
| `raw s` | Same as `safeHTML`. |
| `dict k1 v1 k2 v2 ...` | Build a map literal. |
| `list a b c ...` | Build a slice. |
| `seq n` | Range `[0..n-1]`. |
| `mod a b` / `add a b` / `sub a b` | Integer math. |
| `json v` | Pretty-print v as JSON. |
| `lastWord s` / `beforeLastWord s` | String helpers. |
| `split sep s` | Split into a slice. |
| `image_url url size` | Cached/optimized image URL. |
| `image_srcset url size1 size2 ...` | Responsive `srcset`. |
| `filter name value` | Run a registered Tengo filter (2 args required). |
| `event name ctx ...` | Fire an event; collect HTML responses. |
| `deref v` | Dereference `*string`/`*int` → bare value. |
| `renderLayoutBlock slug` | Render a partial (layout/partial scope only). |
| `setting settings key` | Settings lookup with empty-on-miss fallback. |
| `mustSetting settings key` | Settings lookup that errors on miss/empty. |

Notably **absent** from the funcmap (these are Go template built-ins, used
without a leading function name): `eq`, `ne`, `lt`, `gt`, `and`, `or`,
`not`, `index`, `len`, `range`, `with`, `if`, `printf`. Common helpers
that don't exist in Squilla: `trimPrefix`, `hasPrefix`, `default`. If you
reach for one of those, write the logic inline or move it to a Tengo
filter.
