# Themes

Themes control the public-facing site appearance. Each theme provides layouts, partials, content blocks, static assets, and optional Tengo scripts.

## Structure

```
themes/
  my-theme/
    theme.json              # Theme manifest (name, version, author)
    layouts/                # Page layouts (Go html/template)
      base.html             # Default layout
      blank.html            # No-chrome layout
    partials/               # Reusable template fragments
      site-header.html
      site-footer.html
      primary-nav.html
    blocks/                 # Content block templates
      hero.html
      text.html
    assets/                 # Static files (CSS, JS, images, fonts)
      css/style.css
      js/main.js
    scripts/                # Tengo (.tgo) hooks and filters
      script.tgo
```

## How Themes Work

- **Layouts**: Go `html/template` files that wrap page content. Rendered by the core template engine.
- **Partials**: Included in layouts via `{{ partial "site-header" . }}`
- **Blocks**: Each content block type maps to a template file. Rendered in sequence to build the page.
- **Assets**: Served statically at `/theme/assets/*`
- **Scripts**: Tengo scripts that register event hooks, filters, and custom routes. Can register custom image sizes, inject data, etc.

## Template Functions

Available in all theme templates:
- `{{ partial "name" . }}` — include a partial
- `{{ filter "name" value }}` — apply a filter
- `{{ image_url .url "thumbnail" }}` — get cached/optimized image URL
- `{{ image_srcset .url "medium" "large" }}` — generate srcset attribute
- Standard Go template functions (`if`, `range`, `with`, etc.)

## Building Themes

Themes are plain files — no build step required. Just create the directory structure and templates. Static assets (CSS/JS) can be pre-built if using a bundler, but that's up to the theme developer.

## Deployment

Themes are deployed as-is. The Dockerfile copies the entire `themes/` directory. No compilation needed — themes are interpreted at runtime by the Go template engine.
