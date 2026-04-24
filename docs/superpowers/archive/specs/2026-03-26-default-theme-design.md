# Default Theme Design Spec

## Overview

A polished, self-contained default theme for VibeCMS that lives in `themes/default/`. It ships 10 content block types with field schemas and test data, 2 layouts, 5 partials (migrated from seed.go), a CSS custom-properties system for easy rebranding, and static image assets to exercise the full theme pipeline.

## Visual Identity

Neutral canvas — slate grays with a single accent color via CSS custom properties. Uses Inter/system-ui fonts. Tailwind CDN stays for utility classes; `theme.css` loads after for custom properties and overrides. An agency rebrands by changing ~5 CSS variables.

## File Structure

```
themes/default/
├── theme.json
├── assets/
│   ├── images/
│   │   ├── logo.svg              # Placeholder site logo
│   │   └── og-default.png        # Default OG/social image placeholder
│   └── styles/
│       └── theme.css             # CSS custom properties + overrides
├── layouts/
│   ├── default.html              # Full page layout (head to footer)
│   └── blank.html                # Content only, no header/footer
├── partials/
│   ├── site-header.html          # Header with nav + user menu
│   ├── site-footer.html          # Footer with nav + copyright
│   ├── primary-nav.html          # Main nav menu
│   ├── user-menu.html            # Auth state menu (login/logout)
│   └── footer-nav.html           # Footer links
├── blocks/
│   ├── hero/
│   │   ├── block.json
│   │   ├── view.html
│   │   └── style.css
│   ├── rich-text/
│   │   ├── block.json
│   │   └── view.html
│   ├── image-text/
│   │   ├── block.json
│   │   ├── view.html
│   │   └── style.css
│   ├── gallery/
│   │   ├── block.json
│   │   ├── view.html
│   │   └── style.css
│   ├── cta/
│   │   ├── block.json
│   │   ├── view.html
│   │   └── style.css
│   ├── features-grid/
│   │   ├── block.json
│   │   ├── view.html
│   │   └── style.css
│   ├── testimonials/
│   │   ├── block.json
│   │   ├── view.html
│   │   └── style.css
│   ├── faq/
│   │   ├── block.json
│   │   ├── view.html
│   │   ├── style.css
│   │   └── script.js
│   ├── contact-form/
│   │   ├── block.json
│   │   ├── view.html
│   │   └── style.css
│   └── stats/
│       ├── block.json
│       ├── view.html
│       └── style.css
```

## theme.json Manifest

```json
{
  "name": "VibeCMS Default",
  "version": "1.0.0",
  "description": "A polished, neutral starter theme with rebrandable CSS custom properties",
  "author": "VibeCMS",
  "styles": [
    { "handle": "theme-css", "src": "assets/styles/theme.css", "position": "head" }
  ],
  "scripts": [],
  "layouts": [
    { "slug": "default", "name": "Default Layout", "file": "layouts/default.html", "is_default": true },
    { "slug": "blank", "name": "Blank Layout", "file": "layouts/blank.html", "is_default": false }
  ],
  "partials": [
    { "slug": "site-header", "name": "Site Header", "file": "partials/site-header.html" },
    { "slug": "site-footer", "name": "Site Footer", "file": "partials/site-footer.html" },
    { "slug": "primary-nav", "name": "Primary Navigation", "file": "partials/primary-nav.html" },
    { "slug": "user-menu", "name": "User Menu", "file": "partials/user-menu.html" },
    { "slug": "footer-nav", "name": "Footer Navigation", "file": "partials/footer-nav.html" }
  ],
  "blocks": [
    { "slug": "hero", "dir": "blocks/hero" },
    { "slug": "rich-text", "dir": "blocks/rich-text" },
    { "slug": "image-text", "dir": "blocks/image-text" },
    { "slug": "gallery", "dir": "blocks/gallery" },
    { "slug": "cta", "dir": "blocks/cta" },
    { "slug": "features-grid", "dir": "blocks/features-grid" },
    { "slug": "testimonials", "dir": "blocks/testimonials" },
    { "slug": "faq", "dir": "blocks/faq" },
    { "slug": "contact-form", "dir": "blocks/contact-form" },
    { "slug": "stats", "dir": "blocks/stats" }
  ]
}
```

## CSS Custom Properties (`theme.css`)

```css
:root {
  --color-accent: #3b82f6;
  --color-accent-hover: #2563eb;
  --color-accent-text: #ffffff;
  --color-surface: #ffffff;
  --color-surface-alt: #f8fafc;
  --color-border: #e2e8f0;
  --color-text: #1e293b;
  --color-text-muted: #64748b;
  --font-heading: 'Inter', system-ui, sans-serif;
  --font-body: 'Inter', system-ui, sans-serif;
  --radius: 0.5rem;
  --max-width: 1200px;
}
```

Tailwind CDN remains in the layout `<head>`. `theme.css` loads after Tailwind so custom properties and overrides take precedence. Block `style.css` files use these variables for consistency.

## Block Designs

### hero
- **Fields:** heading (text), subheading (text), background_image (image), button_text (text), button_url (text), alignment (select: left/center/right)
- **Template:** Full-width section with overlay gradient, responsive text sizing, CTA button using `var(--color-accent)`
- **Scoped CSS:** Overlay gradient, min-height, responsive breakpoints

### rich-text
- **Fields:** body (richtext)
- **Template:** Prose-styled container with proper typography spacing via Tailwind prose classes

### image-text
- **Fields:** image (image), heading (text), body (richtext), image_position (select: left/right), button_text (text), button_url (text)
- **Template:** Two-column layout that stacks on mobile, image position flips via field value
- **Scoped CSS:** Column layout, responsive stacking

### gallery
- **Fields:** images (repeater: image (image), caption (text))
- **Template:** Responsive CSS grid, 2-3 columns depending on viewport
- **Scoped CSS:** Grid layout, hover effects on images

### cta
- **Fields:** heading (text), body (text), button_text (text), button_url (text), style (select: light/dark/accent)
- **Template:** Centered section, style field switches background/text color scheme
- **Scoped CSS:** Style variants using CSS custom properties

### features-grid
- **Fields:** heading (text), subheading (text), features (repeater: icon (text), title (text), description (text))
- **Template:** 3-column responsive grid with icon/title/description cards
- **Scoped CSS:** Card layout, responsive grid

### testimonials
- **Fields:** heading (text), testimonials (repeater: quote (textarea), author_name (text), author_role (text), avatar (image))
- **Template:** Card-based layout with quote styling, avatar with placeholder fallback
- **Scoped CSS:** Quote styling, card layout

### faq
- **Fields:** heading (text), subheading (text), items (repeater: question (text), answer (richtext))
- **Template:** Accordion using Alpine.js `x-data`/`x-show` for toggle behavior
- **Scoped CSS:** Accordion styling, transition animations
- **Script:** Alpine.js accordion toggle logic

### contact-form
- **Fields:** heading (text), subheading (text), email_label (text), message_label (text), button_text (text)
- **Template:** Visual-only form with name, email, message fields. No submission logic.
- **Scoped CSS:** Form field styling using custom properties

### stats
- **Fields:** heading (text), stats (repeater: number (text), label (text), prefix (text), suffix (text))
- **Template:** Horizontal row of large numbers with labels (e.g. "500+ Clients")
- **Scoped CSS:** Number sizing, responsive layout

All blocks include `test_data` in `block.json` with realistic placeholder content.

## Static Assets & Theme URL

### Assets
- `assets/images/logo.svg` — Placeholder SVG logo
- `assets/images/og-default.png` — Placeholder OG image (generated)

### Theme URL
- New static file route: `GET /theme/assets/*` serves files from the active theme's `assets/` directory
- New template variable: `{{.app.theme_url}}` resolves to `/theme/assets` (or the configured base)
- Usage in templates: `<img src="{{.app.theme_url}}/images/logo.svg" alt="...">`

### Implementation
- Add static route in `main.go` or public handler setup
- `ThemeAssetRegistry` already tracks `themeDir` — use it to resolve the filesystem path
- `RenderContext.BuildAppData()` adds `theme_url` to the `.app` map

## Seed Migration Strategy

### Moves to theme (removed from seed.go)
- Layout blocks: primary-nav, user-menu, site-header, site-footer, footer-nav → become theme **partials**
- Default layout → becomes `layouts/default.html`

### Stays in seed.go
- Homepage content node, menus (main-nav, footer-nav), site settings
- Roles (admin, editor, author, member)
- Auth block types (login-form, register-form, forgot-password-form, reset-password-form)

### Conflict resolution
- Theme loads on startup via `ThemeLoader` (existing flow)
- Seed runs after theme load
- Seed skips layout creation if a record with that slug already exists with `source: "theme"`
- Seed skips layout-block creation if a record with that slug already exists with `source: "theme"`
- This means: theme present → theme wins. Theme missing → seed provides fallback.

## Layouts

### default.html
Full HTML document: `<!DOCTYPE html>` through `</html>`. Includes:
- Tailwind CDN in `<head>`
- `{{.app.head_styles}}` for theme CSS
- `{{.app.head_scripts}}` for theme JS
- Site header via `{{layout_block "site-header"}}` (uses existing layout block rendering pipeline)
- `{{.node.blocks_html}}` for page content
- Site footer via `{{layout_block "site-footer"}}`
- `{{.app.foot_scripts}}` for deferred JS
- `{{.app.block_styles}}` and `{{.app.block_scripts}}` for scoped block assets

### blank.html
Same `<head>` setup but no header/footer. Just `{{.node.blocks_html}}` in a centered container. Useful for landing pages, auth pages, or embeds.
