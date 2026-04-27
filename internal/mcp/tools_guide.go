package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"vibecms/internal/coreapi"
)

// registerGuideTools exposes meta-tools that teach an AI client how to use the
// rest of the MCP surface. The goal is to collapse the cold-start problem: one
// call returns a goal→tool decision tree plus current CMS state, so the model
// does not burn 10 discovery calls before it does useful work.
func (s *Server) registerGuideTools() {
	s.addTool(mcp.NewTool("core.guide",
		mcp.WithDescription("META. Call this first when you're new to VibeCMS or unsure which tool to reach for. Returns a goal→tool decision tree (recipes for common journeys) plus a live snapshot of CMS state (active theme, counts, node types, recent nodes). Replaces ~10 discovery calls. Optional topic narrows the response: 'pages' | 'blocks' | 'themes' | 'taxonomies' | 'media' | 'extensions'."),
		mcp.WithString("topic", mcp.Description("Optional: narrow the response to one domain.")),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return s.buildGuide(ctx, stringArg(args, "topic"))
	})

	s.addTool(mcp.NewTool("core.theme.standards",
		mcp.WithDescription("Returns the official VibeCMS theme development standards. Use this to validate theme structure, block definitions (Rule 1.5), and field schemas (Rule 1.6). Always call this before creating or refactoring theme components."),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return themeStandards(), nil
	})

	s.addTool(mcp.NewTool("core.extension.standards",
		mcp.WithDescription("Returns the official VibeCMS extension development standards (manifest schema, capabilities, gRPC plugin lifecycle, admin-UI micro-frontend rules, list-page primitives, SDUI sidebar wiring, lifecycle events). Always call this before creating or refactoring an extension."),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return extensionStandards(), nil
	})
}

// buildGuide assembles the static decision tree alongside a cheap live snapshot
// of the CMS so the AI client has both "what can I do" and "what exists right
// now" in a single payload.
func (s *Server) buildGuide(ctx context.Context, topic string) (map[string]any, error) {
	out := map[string]any{
		"version":    "1",
		"namespace":  "core.<domain>.<verb>",
		"recipes":    guideRecipes(topic),
		"data_shapes": guideShapes(),
		"conventions": guideConventions(),
	}

	// Live snapshot — best-effort. Any failure becomes a note rather than a
	// hard error so the static parts of the guide are always returned.
	snap := map[string]any{}
	if s.deps.ThemeMgmtSvc != nil {
		if t, err := s.deps.ThemeMgmtSvc.GetActive(); err == nil {
			snap["active_theme"] = t
		}
	}
	if s.deps.CoreAPI != nil {
		if types, err := s.deps.CoreAPI.ListNodeTypes(ctx); err == nil {
			summary := make([]map[string]any, 0, len(types))
			for _, nt := range types {
				summary = append(summary, map[string]any{
					"slug":         nt.Slug,
					"label":        nt.Label,
					"label_plural": nt.LabelPlural,
				})
			}
			snap["node_types"] = summary
		}
		if q, err := s.deps.CoreAPI.QueryNodes(ctx, coreapi.NodeQuery{Limit: 5, OrderBy: "updated_at DESC"}); err == nil {
			snap["recent_nodes"] = q
		}
	}
	out["snapshot"] = snap

	// Tool index — flat list of every registered tool with its one-line
	// description. Useful when the AI wants a quick sanity check against the
	// recipes rather than a full capabilities dump.
	out["tool_index"] = s.toolIndex()

	if topic != "" {
		out["topic"] = topic
	}
	return out, nil
}

func guideRecipes(topic string) []map[string]any {
	all := []map[string]any{
		{
			"goal":    "Publish a new page with an image",
			"topic":   "pages",
			"steps":   []string{"core.media.upload", "core.node.create", "core.render.node_preview"},
			"notes":   "featured_image is an object {url, alt, ...}, NEVER a string. blocks_data is an array of {type, fields}.",
		},
		{
			"goal":    "Add a custom content block to a theme",
			"topic":   "blocks",
			"steps":   []string{"core.block_types.list", "core.block_types.create", "core.render.block", "core.node.update"},
			"notes":   "html_template reads .fields and .node. Include realistic test_data so the admin preview works.",
		},
		{
			"goal":    "Switch the site's look",
			"topic":   "themes",
			"steps":   []string{"core.theme.list", "core.theme.activate"},
			"notes":   "Theme activation does NOT require a server restart.",
		},
		{
			"goal":    "Add a custom node type (post type)",
			"topic":   "pages",
			"steps":   []string{"core.nodetype.list", "core.nodetype.create", "core.taxonomy.create", "core.node.create"},
			"notes":   "label_plural is required for admin UI. url_prefixes decides public routing.",
		},
		{
			"goal":    "Diagnose a broken public page",
			"topic":   "themes",
			"steps":   []string{"core.theme.active", "core.layout.list", "core.render.node_preview", "core.block_types.get"},
			"notes":   "render.* never fires events; safe to reproduce issues on live content.",
		},
		{
			"goal":    "Tag content with a new taxonomy",
			"topic":   "taxonomies",
			"steps":   []string{"core.taxonomy.create", "core.term.create", "core.node.update"},
			"notes":   "term field values are objects {slug, name}, not bare strings.",
		},
		{
			"goal":    "Wire up an extension (media manager, email, etc.)",
			"topic":   "extensions",
			"steps":   []string{"core.extension.list", "core.extension.activate"},
			"notes":   "Response carries restart_required; some extensions need a process restart before plugin binaries load.",
		},
		{
			"goal":    "Upload and attach media",
			"topic":   "media",
			"steps":   []string{"core.media.upload", "core.media.get", "core.node.update"},
			"notes":   "Use the returned {id, url, slug} — reference media by slug where possible for theme-portable nodes.",
		},
		{
			"goal":    "Verify theme documentation and schema compliance",
			"topic":   "themes",
			"steps":   []string{"core.theme.standards", "core.block_types.get"},
			"notes":   "Rule 1.5 requires block descriptions; Rule 1.6 requires field help text. Always check standards before finalizing a theme.",
		},
		{
			"goal":  "Seed a navigation menu (theme or AI agent)",
			"topic": "themes",
			"steps": []string{"core.theme.standards", "core.menu.upsert"},
			"notes": "MCP: core.menu.upsert({name, slug, items:[{label, page:'<node-slug>'}]}) — page slugs are resolved to NodeIDs server-side, so renaming the target page doesn't break the menu. Tengo equivalent (theme.tengo): import 'core/menus'; menus.upsert({...}). Both paths share the same UpsertMenu CoreAPI.",
		},
		{
			"goal":  "Build a new extension from scratch",
			"topic": "extensions",
			"steps": []string{"core.extension.standards", "core.extension.list", "core.extension.activate"},
			"notes": "extension.json declares capabilities, plugins, admin_ui, public_routes, settings_schema. admin_ui.menu.section routes the sidebar entry into content/design/development/settings; settings_menu auto-injects into Settings. Per-extension Tailwind build is required for Docker; admin shell's @source over extensions/ only helps in local dev.",
		},
		{
			"goal":  "React to schema changes in an extension",
			"topic": "extensions",
			"steps": []string{"core.extension.standards"},
			"notes": "Subscribe to node_type.{created,updated,deleted} and taxonomy.{created,updated,deleted} (added 2026-04-25). Also extension.activated, theme.activated and their deactivated counterparts.",
		},
	}
	if topic == "" {
		return all
	}
	out := make([]map[string]any, 0, len(all))
	for _, r := range all {
		if r["topic"] == topic {
			out = append(out, r)
		}
	}
	return out
}

func guideShapes() map[string]string {
	return map[string]string{
		"image":        `{ "url": "/media/...", "alt": "...", "width": 800, "height": 600 }`,
		"link":         `{ "label": "Read more", "url": "/about", "target": "_self" }`,
		"repeater":     `[ { "<sub_field>": "..." }, ... ]`,
		"term":         `{ "slug": "travel", "name": "Travel", "taxonomy": "tag" }`,
		"blocks_data":  `[ { "type": "<slug>", "fields": { ... } }, ... ]`,
		"select/radio": `"<string value>"  // options in schemas MUST be plain strings, not {label,value} objects`,
	}
}

func guideConventions() []string {
	return []string{
		"All list/query tools accept {limit, offset}; default 25, max 200; responses include {total}.",
		"render.* tools are side-effect-free — no events, no view counts, no writes.",
		"restart_required=true in a response means a process restart is needed before plugin binaries fully load.",
		"Prefer typed tools (core.node.query, core.data.query) over core.data.exec (gated raw SQL).",
		"Reference media/layouts by slug when authoring theme-portable content; IDs rotate when themes are reactivated.",
	}
}

// toolIndex returns a flat index of every currently-registered tool with its
// one-line description, captured at registration time by addTool.
func (s *Server) toolIndex() []toolCatalogEntry {
	return s.toolCatalog
}

func themeStandards() map[string]any {
	return map[string]any{
		"philosophy": "The golden rule: A theme must render correctly from a cold boot with nothing but its own files — no manual DB edits, no magic.",
		"structure": map[string]any{
			"theme.json": "Manifest registering blocks, assets, and page templates.",
			"layouts/":   "Page layouts (base.html, blank.html) using Go html/template.",
			"partials/":  "Reusable fragments included via {{ partial \"name\" . }}.",
			"blocks/":    "Content blocks: each with a template (view.html) and schema (block.json).",
			"assets/":    "Static images, CSS, and JS. Use theme-asset:<key> to reference them.",
			"templates/": "Pre-populated page JSON files for demo content.",
			"scripts/":   "Tengo scripts (theme.tengo) for seeding and custom filters.",
		},
		"template_functions": []string{
			"{{ partial \"name\" . }} - Include a partial",
			"{{ filter \"name\" value }} - Apply a Tengo filter",
			"{{ image_url .url \"size\" }} - Optimized image URL",
			"{{ image_srcset .url \"size1\" \"size2\" }} - Responsive srcset",
		},
		"core_rules": []map[string]any{
			{
				"id":          "1.1",
				"title":       "Complete Test Data",
				"description": "Every field in field_schema must have a value in test_data. Content must be on-brand (no Lorem Ipsum).",
			},
			{
				"id":          "1.2",
				"title":       "Field Declaration",
				"description": "No field may be read in view.html that is not declared in block.json's field_schema.",
			},
			{
				"id":          "1.3",
				"title":       "No Fallback Defaults",
				"description": "Templates must not carry hardcoded fallback strings. Gate UI parts with {{ with .field }} instead of {{ or .field 'default' }}.",
			},
			{
				"id":          "1.5",
				"title":       "Human-friendly Block Descriptions",
				"description": "The root 'description' in block.json must summarize visual layout and functional purpose. Mandatory for CMS discovery.",
			},
			{
				"id":          "1.6",
				"title":       "Mandatory Field Help Text",
				"description": "Every field definition must include a 'help' property with instructions for the CMS editor.",
			},
		},
		"examples": map[string]string{
			"theme.json": `{
  "name": "My Theme",
  "slug": "my-theme",
  "blocks": ["my-hero", "my-features"],
  "assets": {
    "styles": ["assets/css/main.css"]
  }
}`,
			"block.json": `{
  "slug": "my-hero",
  "description": "Hero section with title, image and CTA button.",
  "field_schema": [
    { "name": "title", "type": "text", "help": "Catchy headline" },
    { "name": "cta", "type": "link", "help": "Primary call to action" }
  ],
  "test_data": {
    "title": "Welcome to VibeCMS",
    "cta": { "url": "/", "text": "Get Started" }
  }
}`,
			"view.html": `{{ with .title }}<h1>{{ . }}</h1>{{ end }}
{{ with .cta }}<a href="{{ .url }}">{{ .text }}</a>{{ end }}`,
			"theme.tengo": `// Seed a page
if page_missing("home") {
    create_page("home", "Home", "base.html", [
        { "slug": "my-hero", "data": { "title": "Auto Seeded" } }
    ])
}`,
		},
		"field_types": []map[string]any{
			{"type": "term", "intent": "Taxonomies", "shape": `{"name": "...", "slug": "..."}`},
			{"type": "link", "intent": "CTAs/Buttons", "shape": `{"url": "/...", "text": "...", "target": "_self"}`},
			{"type": "image", "intent": "Media", "shape": `{"url": "theme-asset:...", "alt": "..."}`},
			{"type": "node", "intent": "Content Ref", "shape": `{"id": 123, "slug": "...", "title": "..."}`},
			{"type": "gallery", "intent": "Image List", "shape": `[{"url": "...", "alt": "..."}, ...]`},
			{"type": "repeater", "intent": "Nested Lists", "shape": "Array of objects matching sub_fields schema."},
			{"type": "richtext", "intent": "Prose/HTML", "shape": "\"HTML string\""},
			{"type": "toggle", "intent": "Booleans", "shape": "true | false"},
		},
		"seeding_patterns": map[string]string{
			"registration":      "Always register taxonomies BEFORE node types that use them.",
			"idempotency":       "Use page_missing() or node_query checks to avoid duplicate data on script re-runs.",
			"menus":             "Use core/menus → menus.upsert({slug, name, items:[{label, page:'<slug>'}]}). The page:<slug> form resolves to NodeID at upsert so slug renames don't break menus.",
			"wellknown":         "Use core/wellknown to register /.well-known/* handlers (e.g. apple-app-site-association). Unregistered paths return instant 404 via WellKnownRegistry, mounted before the public catch-all.",
			"assets_module":     "Use core/assets to read files from the calling theme/extension's own root: assets.read('forms/trip-order.html') / assets.exists('data/regions.json'). Returns a string, or an error value (wrap with is_error) if missing or path escapes root. Ideal for shipping form layouts, JSON fixtures, or default content as plain files instead of inlining multi-line strings in theme.tengo. Path is relative to the theme/extension dir; absolute paths and ../ traversal are rejected.",
			"forms_seeding":     "Theme-bundled forms: emit core/events 'forms:upsert' with {slug, name, fields, layout, settings, force?}. Idempotent on slug — without force, an existing same-slug form is left alone (admin edits stick); with force:true, theme overwrites on every reload. For theme-styled forms, ship the layout as themes/<theme>/forms/<slug>.html using forms-ext template syntax ({{.field_id.label}}, {{range .options}}…) and theme CSS classes; load it with core/assets.read and pass as the layout field. Form HTML is server-rendered via {{event \"forms:render\" (dict \"form_id\" \"<slug>\" \"hidden\" (dict ...))}} from a layout/block — hidden injects <input type=hidden> before </form> for per-page context (trip_slug, price). Public submit endpoint is /forms/submit/<slug>; the AJAX runtime is /extensions/<slug>/blocks/vibe-form/script.js.",
			"assets_hot_reload": "Theme assets are served via an atomic-pointer resolver that swaps on theme.activated. Runtime theme switches serve the new theme's assets instantly with no restart.",
			"autoregistration":  "Themes in themes/ are autoregistered at startup (mirroring the extension scan). No DB seeding is required to surface a new theme.",
			"layout_seeding":    "The core default layout is seeded with source='seed' (migration 0036). Themes are free to install their own base.html / blank.html without colliding.",
		},
		"portable_refs": []string{
			"Always reference pages/nodes by slug.",
			"Reference blocks by their registered slug (e.g., hv-hero).",
			"Reference assets via theme-asset:<key> prefix.",
		},
		"verification_checklist": []string{
			"All assets import into the media library.",
			"All seeded nodes render (200 status).",
			"Admin edit forms show all fields (no [object Object]).",
			"Templates render identically to seeded pages.",
		},
		"authoritative_resource": "vibecms://guidelines/themes",
	}
}

// extensionStandards returns the structured rule set every extension developer
// (human or AI) should follow. Mirrors the depth of themeStandards but covers
// the gRPC + admin-UI + manifest surface. Kept in lockstep with
// extensions/README.md.
func extensionStandards() map[string]any {
	return map[string]any{
		"philosophy": "Extensions own their full stack: manifest, gRPC plugin, admin-UI micro-frontend, SQL migrations, blocks, public routes. Core is a kernel — if disabling/removing the extension would leave dead code in core, that code belongs in the extension.",
		"manifest": map[string]any{
			"required": []string{"name", "slug", "version"},
			"optional": []string{"author", "description", "priority", "provides", "capabilities", "plugins", "admin_ui", "settings_schema", "blocks", "templates", "layouts", "partials", "public_routes", "assets"},
			"notes": []string{
				"slug MUST match the directory name (kebab-case).",
				"capabilities are enforced on every CoreAPI call. Declare exactly what's needed.",
				"public_routes are mounted on the public Fiber app without auth — proxied to HandleHTTPRequest with user_id=0.",
			},
		},
		"capabilities": []string{
			"nodes:read", "nodes:write", "nodes:delete",
			"nodetypes:read", "nodetypes:write",
			"settings:read", "settings:write",
			"events:emit", "events:subscribe",
			"email:send",
			"menus:read", "menus:write", "menus:delete",
			"routes:register",
			"filters:register", "filters:apply",
			"media:read", "media:write", "media:delete",
			"users:read",
			"http:fetch",
			"log:write",
			"data:read", "data:write", "data:delete",
			"files:write", "files:delete",
		},
		"admin_ui": map[string]any{
			"entry":         "Path to the built ES module, e.g. admin-ui/dist/index.js. The extension loader auto-injects a sibling <link rel='stylesheet'> for dist/index.css when present — DO NOT declare CSS in the manifest.",
			"menu.section":  "One of 'content' (default), 'design', 'development', 'settings'. Honored by the SDUI sidebar engine — items with no/unknown section land at the top level.",
			"settings_menu": "Auto-spliced into the Settings sidebar group. Extensions that only contribute settings can omit menu entirely.",
			"icons":         "Any valid lucide-react icon name (e.g. 'ImageDown', 'Images'). Resolved dynamically; unknown names fall back to 'Puzzle'.",
			"slots":         "Named UI injection points. e.g. smtp-provider injects into email-manager's 'email-settings' slot.",
			"field_types":   "Custom field types registered for use in node type schemas, with shape supports/component metadata.",
			"css_load_order": "The extension loader prepends the extension <link> BEFORE admin-ui's stylesheet so admin-ui's responsive utilities (e.g. lg:relative on <aside>) win the @layer utilities cascade. Do not change this ordering.",
		},
		"tailwind_rule": "Per-extension Tailwind build is mandatory for Docker images. The admin shell's index.css declares an extra @source over extensions/*/admin-ui/src so simple classes work in local dev, but the Docker frontend stage only copies admin-ui/ — extension classes silently drop unless the extension ships its own dist/index.css.",
		"list_page_primitives": []string{
			"ListPageShell", "ListHeader (with tabs={[{value,label,count}]})",
			"ListSearch", "ListFooter (page, totalPages, perPage, total, onPage, onPerPage)",
			"EmptyState", "LoadingRow", "Chip", "StatusPill", "TitleCell",
			"RowActions", "Th", "Td", "Tr", "Checkbox",
			"AccordionRow", "SectionHeader", "CodeWindow",
		},
		"list_page_conventions": []string{
			"Drop the <h1> page title — the active tab pill IS the title.",
			"Tabs replace separate type/status filter dropdowns (?type=image lives in tabs).",
			"All filter / sort / view / pagination state in URL search params; default values omit the param.",
			"Use replace:true on search keystrokes to avoid history pollution; resetPage:true when changing filters.",
			"Reference impl: extensions/media-manager/admin-ui/src/MediaLibrary.tsx and extensions/forms/admin-ui/src/FormsList.tsx.",
		},
		"shared_window_globals": []string{
			"window.__VIBECMS_SHARED__.ReactRouterDOM { useNavigate, useSearchParams, ... }",
			"window.__VIBECMS_SHARED__.Sonner { toast }",
			"window.__VIBECMS_SHARED__.ui (list-page primitives, AccordionRow, SectionHeader, CodeWindow, ...)",
			"@vibecms/ui, @vibecms/api, @vibecms/icons resolve via vite externalize config — import normally.",
		},
		"plugin_interface": []string{
			"Initialize(hostConn) — get VibeCMSHost client; seed defaults idempotently.",
			"GetSubscriptions() — return events to subscribe to.",
			"HandleEvent(action, payload) — handle a fired event.",
			"HandleHTTPRequest(req) — handle proxied admin (/admin/api/ext/<slug>/*) and public route requests.",
			"Shutdown() — cleanup.",
		},
		"lifecycle_events": map[string]string{
			"extension.activated":   "After migrations, scripts, and plugins. Payload: slug, path, version, assets.",
			"extension.deactivated": "Before cleanup. Payload: slug.",
			"theme.activated":       "Replayed for the current theme when an extension activates at runtime. Payload: name, path, version, assets.",
			"theme.deactivated":     "Payload: name.",
			"node_type.created":     "Custom post type registered (added 2026-04-25). Payload: slug.",
			"node_type.updated":     "Custom post type updated. Payload: slug.",
			"node_type.deleted":     "Custom post type removed. Payload: slug.",
			"taxonomy.created":      "Taxonomy registered (added 2026-04-25). Payload: slug.",
			"taxonomy.updated":      "Taxonomy updated. Payload: slug.",
			"taxonomy.deleted":      "Taxonomy removed. Payload: slug.",
		},
		"sdui_reactivity": []string{
			"Typed SSE events route to specific TanStack query keys via a central qk factory (qk.boot, qk.layout, qk.list, qk.entity, qk.settings).",
			"useAuth subscribes through an sse-bus so sidebar user-info refreshes on user.updated without page reload.",
			"CONFIRM action uses shadcn AlertDialog; CORE_API writes toast success/error with per-action overrides.",
		},
		"existing_extensions": map[string]string{
			"media-manager":     "gRPC + React + Tengo. Reference for list-page primitives, drawer, upload modal, URL state, image optimizer settings.",
			"email-manager":     "gRPC + React. Owns 'email-settings' slot that providers inject into.",
			"sitemap-generator": "gRPC + Tengo. Yoast-style sitemaps; rebuild on node.published / node.deleted.",
			"smtp-provider":     "gRPC. Subscribes to email.send; injects settings into email-manager.",
			"resend-provider":   "Tengo-only. Demonstrates no-binary extension via core/http.",
			"forms":             "gRPC + React + Tengo + content block. vibe-form block, form_selector field type, /forms/submit/* public route.",
			"hello-extension":   "Tengo-only minimal demo. Use as starting template.",
			"content-blocks":    "Pure declarative bundle of 40 blocks + 10 templates. No binary.",
		},
		"verification_checklist": []string{
			"Manifest declares exactly the capabilities used (no over-permissioning).",
			"Plugin binary is CGO_ENABLED=0 + statically linked (Alpine container has no glibc).",
			"Admin-UI ships its own dist/index.css (per-extension Tailwind build).",
			"List pages match the canonical pattern: tabs replace dropdowns, URL state, ListPageShell primitives.",
			"Lifecycle subscriptions clean up on extension.deactivated when needed.",
			"Public routes are listed in public_routes — admin routes are auto-mounted at /admin/api/ext/<slug>/*.",
		},
		"authoritative_resource": "vibecms://guidelines/extensions",
	}
}

func onboardingGuide() string {
	return `# VibeCMS Onboarding for AI Agents

Welcome! Your task is to build or modify a VibeCMS theme or extension. To
succeed without human intervention, follow the path that matches your task.

## A. Building a Theme

### 1. Discovery
- **Read the Guide**: 'read_resource' on 'vibecms://guidelines/themes'.
- **Tool**: Call 'core.theme.standards' for the structured ruleset.

### 2. Implementation
- **Schema First**: Define 'block.json' before 'view.html'.
- **Test Data**: Every field MUST have on-brand 'test_data'. NO LOREM IPSUM.
- **Editor Experience**: Every field MUST have 'help' text.
- **Portability**: Use 'theme-asset:<key>' for images, slugs for node refs.

### 3. Seeding ('theme.tengo')
- Register taxonomies BEFORE node types that use them.
- Use 'page_missing(slug)' checks for idempotency.
- Seed navigation via 'core/menus' — 'menus.upsert({slug, name, items:
  [{label, page:"<slug>"}]})'. The 'page:<slug>' form resolves to a NodeID
  so renaming the target page does NOT break the menu.
- Register '/.well-known/*' handlers via 'core/wellknown' if needed.

### 4. Verification
- Cross-reference with 'core.theme.standards'.
- Ensure 'layouts/base.html' wraps all pages.
- Themes are autoregistered from 'themes/' on startup — no DB seeding.

## B. Building an Extension

### 1. Discovery
- **Read the Guide**: 'read_resource' on 'vibecms://guidelines/extensions'.
- **Tool**: Call 'core.extension.standards' for the structured ruleset.

### 2. Manifest ('extension.json')
- Declare exactly the 'capabilities' you use — no over-permissioning.
- For sidebar UI use 'admin_ui.menu.section' ('content' | 'design' |
  'development' | 'settings'). Settings-only extensions can rely on
  'settings_menu' and omit 'menu'.
- Icons: any valid 'lucide-react' name; unknowns fall back to 'Puzzle'.

### 3. Plugin Binary (gRPC)
- 'CGO_ENABLED=0' + static linking — Alpine container has no glibc.
- Implement 'Initialize / GetSubscriptions / HandleEvent /
  HandleHTTPRequest / Shutdown'.
- Seed default settings idempotently in 'Initialize'.

### 4. Admin-UI Micro-frontend
- Per-extension Tailwind build is REQUIRED for Docker. The admin shell's
  '@source' over 'extensions/' is local-dev convenience only.
- Use shared list-page primitives: 'ListPageShell', 'ListHeader' (with
  'tabs={[{value,label,count}]}'), 'ListSearch', 'ListFooter', etc.
- All filter/sort/view/page state in URL search params; default values
  omit the param.
- Reference impl: 'extensions/media-manager/admin-ui/src/MediaLibrary.tsx'.

### 5. Lifecycle
- Subscribe to 'extension.activated', 'theme.activated' (replayed on late
  activation), 'node_type.{created,updated,deleted}',
  'taxonomy.{created,updated,deleted}'.
- Clean up on 'extension.deactivated'.

### 6. Verification
- Cross-reference with 'core.extension.standards'.
- Confirm 'public_routes' lists every public path you serve.
- Activate via 'core.extension.activate'; check the response's
  'restart_required' flag.

## Common
- Always prefer typed tools (e.g. 'core.menu.upsert') over raw SQL.
- Reference media/layouts/pages by slug — IDs rotate on theme reactivation.

Everything you need is in 'vibecms://guidelines/themes' and
'vibecms://guidelines/extensions'.`
}
