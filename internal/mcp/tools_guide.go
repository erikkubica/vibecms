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
		"philosophy": "A theme must render correctly from a cold boot with nothing but its own files — no manual DB edits, no magic.",
		"structure": map[string]string{
			"theme.json": "Theme manifest (assets, blocks, templates)",
			"layouts/":   "Go html/template files (base.html, etc.)",
			"partials/":  "Reusable template fragments",
			"blocks/":    "Content block templates and block.json definitions",
			"assets/":    "Static files (CSS, JS, images) - referenced via theme-asset:<key>",
			"templates/": "Full page demo templates (JSON)",
			"scripts/":   "Tengo hooks, filters, and seeding (theme.tengo)",
		},
		"rules": []map[string]any{
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
				"description": "The root 'description' in block.json must summarize visual layout and functional purpose.",
			},
			{
				"id":          "1.6",
				"title":       "Mandatory Field Help Text",
				"description": "Every field definition must include a 'help' property with instructions for the CMS editor.",
			},
			{
				"id":          "2.1",
				"title":       "Taxonomies -> term field",
				"description": "Never use text fields for tags/categories. Use 'term' field type bound to a taxonomy.",
			},
			{
				"id":          "2.3",
				"title":       "Links/CTAs -> link field",
				"description": "Never split a button into text/url fields. Use the unified 'link' field type.",
			},
			{
				"id":          "6.1",
				"title":       "Portable References",
				"description": "Always reference core entities (nodes, assets, blocks) by slug, never by numeric ID.",
			},
		},
		"field_types": map[string]string{
			"term":     `{"name": "...", "slug": "..."}`,
			"image":    `{"url": "theme-asset:...", "alt": "..."}`,
			"link":     `{"url": "/...", "text": "...", "target": "_self"}`,
			"node":     `{"id": 123, "slug": "...", "title": "..."}`,
			"repeater": "Array of objects matching sub_fields schema.",
		},
		"mcp_resource": "vibecms://guidelines/themes",
	}
}
