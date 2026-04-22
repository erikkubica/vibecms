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
			"registration": "Always register taxonomies BEFORE node types that use them.",
			"idempotency":  "Use page_missing() or node_query checks to avoid duplicate data on script re-runs.",
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

func onboardingGuide() string {
	return `# VibeCMS Theme Onboarding for AI Agents

Welcome! Your task is to build or modify a VibeCMS theme. To succeed without human intervention, you MUST follow this protocol:

## 1. Discovery Phase
- **Read the Guide**: Call 'read_resource' on 'vibecms://guidelines/themes'. This contains both the Rules and the Canonical Examples.
- **Reference Standards**: Use the 'examples' section in the MCP JSON response as your definitive architectural reference.

## 2. Implementation Protocol
- **Schema First**: Define 'block.json' before 'view.html'. 
- **Test Data**: Every field MUST have on-brand 'test_data'. NO LOREM IPSUM.
- **Editor Experience**: Every field MUST have 'help' text instructions.
- **Portability**: Use 'theme-asset:<key>' for images and slugs for node references.

## 3. Data Seeding (theme.tengo)
- Use 'theme.tengo' to register taxonomies, terms, and nodes. 
- Use 'page_missing(slug)' check to ensure idempotency.

## 4. Verification
- Use the 'core.theme.standards' tool to cross-reference your implementation against current rules.
- Ensure 'layouts/base.html' correctly wraps all pages.

Have fun coding! Everything you need is in the guidelines and examples.`
}
