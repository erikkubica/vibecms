package mcp

import (
	"context"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"squilla/internal/coreapi"
)

// registerGuideTools exposes meta-tools that teach an AI client how to use the
// rest of the MCP surface. The goal is to collapse the cold-start problem: one
// call returns a goal→tool decision tree plus current CMS state, so the model
// does not burn 10 discovery calls before it does useful work.
//
// Token budget: the default response is engineered to fit in <8 k tokens so
// 125–250 k context models don't blow the per-tool dump threshold. Drilling
// into a topic returns ~10–15 k tokens of focused content; verbose=true is
// the escape hatch that returns the full ~30 k-token reference dump.
func (s *Server) registerGuideTools() {
	s.addTool(mcp.NewTool("core.guide",
		mcp.WithDescription("META — call FIRST. Returns a token-compact menu by default: available_topics, recipe goals (no notes), gotcha topic-keys (no summaries), tools_by_domain (names only), data_shapes, conventions, snapshot. Use { topic:'<topic>' } to drill into a single domain (full recipes + relevant gotchas + tool descriptions). Use { verbose:true } only when you need the entire reference dump (~30 k tokens). Topics: pages | editing | blocks | themes | taxonomies | media | extensions."),
		mcp.WithString("topic", mcp.Description("Narrow to one domain. See available_topics on the menu response.")),
		mcp.WithBoolean("verbose", mcp.Description("Return the full reference dump (recipes, gotchas, editing_playbook, every tool description). Default false to stay token-light.")),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return s.buildGuide(ctx, stringArg(args, "topic"), boolArg(args, "verbose"))
	})

	s.addTool(mcp.NewTool("core.theme.standards",
		mcp.WithDescription("Squilla theme standards: structure, rules, capabilities, lifecycle. Compact by default; pass { verbose:true } for full template examples (theme.json/block.json/view.html/theme.tengo embedded source)."),
		mcp.WithBoolean("verbose", mcp.Description("Include full template examples and seeding patterns. Default false.")),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return themeStandards(boolArg(args, "verbose")), nil
	})

	s.addTool(mcp.NewTool("core.extension.standards",
		mcp.WithDescription("Squilla extension standards: manifest, capabilities, gRPC lifecycle, admin-UI rules, design-system primitives, lifecycle events. Compact by default; pass { verbose:true } for the full reference."),
		mcp.WithBoolean("verbose", mcp.Description("Include full reference (event_modes detail, sdui_reactivity, hot_deploy recipes, etc.). Default false.")),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return extensionStandards(boolArg(args, "verbose")), nil
	})
}

// buildGuide assembles the response in one of three modes:
//
//   - verbose=true        — full reference dump (recipes, gotchas, playbook,
//                           every tool description). ~30 k tokens.
//   - topic=<name>        — focused: full recipes + relevant gotchas +
//                           editing_playbook (when topic=='editing') + tools
//                           for that topic's domains. ~10–15 k tokens.
//   - default (no args)   — compact menu: topic list, recipe goals only,
//                           gotcha topic keys only, tool names grouped by
//                           domain, snapshot. <8 k tokens.
//
// The compact menu is engineered so a 125–250 k context model can call
// core.guide in the first turn without triggering a per-tool token dump.
func (s *Server) buildGuide(ctx context.Context, topic string, verbose bool) (map[string]any, error) {
	snap := s.guideSnapshot(ctx)

	// Verbose escape hatch — return the full reference (legacy behavior).
	if verbose {
		return map[string]any{
			"version":          "1",
			"mode":             "verbose",
			"namespace":        "core.<domain>.<verb>",
			"recipes":          guideRecipes(""),
			"data_shapes":      guideShapes(),
			"conventions":      guideConventions(),
			"gotchas":          guideGotchas(),
			"available_topics": guideTopics(),
			"editing_playbook": editingPlaybook(),
			"snapshot":         snap,
			"tool_index":       s.toolIndex(),
		}, nil
	}

	// Topic mode — focused content for one domain.
	if topic != "" {
		out := map[string]any{
			"version":          "1",
			"mode":             "topic",
			"topic":            topic,
			"namespace":        "core.<domain>.<verb>",
			"recipes":          guideRecipes(topic),
			"gotchas":          gotchasForTopic(topic),
			"data_shapes":      guideShapes(),
			"conventions":      guideConventions(),
			"available_topics": guideTopics(),
			"snapshot":         snap,
			"tools":            s.toolsForTopic(topic),
			"hint":             "Pass verbose:true for the full reference dump. Call core.guide() with no args for the compact menu.",
		}
		if topic == "editing" {
			out["editing_playbook"] = editingPlaybook()
		}
		return out, nil
	}

	// Default — compact menu. Goal: fit in <8 k tokens so the AI can call
	// this first without burning a turn on a token-dumped response.
	return map[string]any{
		"version":   "1",
		"mode":      "menu",
		"namespace": "core.<domain>.<verb>",
		"available_topics": guideTopics(),
		"recipe_index": guideRecipeIndex(""),
		"gotcha_topics": guideGotchaKeys(),
		"data_shapes":   guideShapes(),
		"conventions":   guideConventions(),
		"snapshot":      snap,
		"tools_by_domain": s.toolsByDomain(),
		"next_step": "Call core.guide({topic:'<topic>'}) for full recipes + gotchas + tool descriptions in that domain. Use verbose:true only for the full dump (~30 k tokens).",
	}, nil
}

// guideSnapshot collects the live CMS state — same in every mode. Best-effort:
// any failure becomes a missing key rather than a hard error so the static
// parts of the guide always come back.
//
// IMPORTANT: every projection here is a SUMMARY shape, not the full record.
// The guide must stay token-cheap regardless of how content-heavy the site
// gets — a single page with 7 blocks of fields_data was enough to push the
// whole response past the per-tool dump threshold (52 KB on a real install
// versus 14 KB on a fresh one). Reach for core.node.get / core.theme.get /
// core.block_types.get when full records are actually needed.
func (s *Server) guideSnapshot(ctx context.Context) map[string]any {
	snap := map[string]any{}
	if s.deps.ThemeMgmtSvc != nil {
		if t, err := s.deps.ThemeMgmtSvc.GetActive(); err == nil && t != nil {
			snap["active_theme"] = map[string]any{
				"id":      t.ID,
				"slug":    t.Slug,
				"name":    t.Name,
				"version": t.Version,
				"author":  t.Author,
			}
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
		if q, err := s.deps.CoreAPI.QueryNodes(ctx, coreapi.NodeQuery{Limit: 5, OrderBy: "updated_at DESC"}); err == nil && q != nil {
			recent := make([]map[string]any, 0, len(q.Nodes))
			for _, n := range q.Nodes {
				if n == nil {
					continue
				}
				recent = append(recent, map[string]any{
					"id":            n.ID,
					"slug":          n.Slug,
					"title":         n.Title,
					"node_type":     n.NodeType,
					"status":        n.Status,
					"language_code": n.LanguageCode,
					"full_url":      n.FullURL,
					"updated_at":    n.UpdatedAt,
				})
			}
			snap["recent_nodes"] = map[string]any{
				"nodes": recent,
				"total": q.Total,
				"hint":  "Summary only. Use core.node.get(id) for full blocks_data / fields_data / seo_settings.",
			}
		}
		if sections, err := s.collectNodeSections(ctx); err == nil && len(sections) > 0 {
			snap["sections_by_node_type"] = sections
		}
	}
	if s.deps.BlockTypeSvc != nil {
		if list, err := s.deps.BlockTypeSvc.ListAll(); err == nil {
			out := make([]map[string]any, 0, len(list))
			for _, bt := range list {
				out = append(out, map[string]any{
					"slug":   bt.Slug,
					"label":  bt.Label,
					"source": bt.Source,
				})
			}
			snap["block_types"] = out
		}
	}
	if s.deps.LayoutSvc != nil {
		if rows, _, err := s.deps.LayoutSvc.List(nil, "", 1, 200); err == nil {
			out := make([]map[string]any, 0, len(rows))
			for _, l := range rows {
				out = append(out, map[string]any{
					"id":     l.ID,
					"slug":   l.Slug,
					"name":   l.Name,
					"source": l.Source,
				})
			}
			snap["layouts"] = out
		}
	}
	return snap
}

// guideRecipeIndex returns just goal+topic+steps from each recipe (no notes).
// Used by the compact menu so the AI can see what's available without paying
// the prose cost. The AI drills in via core.guide({topic:...}) for full notes.
func guideRecipeIndex(topic string) []map[string]any {
	all := guideRecipes(topic)
	out := make([]map[string]any, 0, len(all))
	for _, r := range all {
		out = append(out, map[string]any{
			"goal":  r["goal"],
			"topic": r["topic"],
			"steps": r["steps"],
		})
	}
	return out
}

// guideGotchaKeys returns just the topic key of each gotcha. Used by the
// compact menu — the AI can topic-drill via core.guide({topic:'<topic>'}) to
// get full summaries for that area, or call verbose:true for everything.
func guideGotchaKeys() []string {
	all := guideGotchas()
	out := make([]string, 0, len(all))
	for _, g := range all {
		if k, ok := g["topic"].(string); ok {
			out = append(out, k)
		}
	}
	return out
}

// gotchasForTopic returns gotchas relevant to a topic. Most gotchas are
// cross-cutting — we return all of them in topic mode because their summaries
// are short and the cross-domain ones (settings_per_language, presigned_uploads,
// kernel_extensions_boundary, …) are worth surfacing on every topic call.
// Compact menu mode excludes them entirely; the AI sees only the keys.
func gotchasForTopic(topic string) []map[string]any {
	return guideGotchas()
}

// toolDomain extracts "X" from "core.X.Y" (or "core.X" for two-part names).
// Returns "" for malformed names.
func toolDomain(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) < 2 || parts[0] != "core" {
		return ""
	}
	return parts[1]
}

// toolsByDomain returns {domain: [tool-names...]} — names only, no descriptions.
// This replaces the verbose tool_index in the compact menu. ~95 tool names is
// roughly 1 k tokens versus 5–6 k for the full descriptions.
func (s *Server) toolsByDomain() map[string][]string {
	out := map[string][]string{}
	for _, t := range s.toolCatalog {
		d := toolDomain(t.Name)
		if d == "" {
			continue
		}
		out[d] = append(out[d], t.Name)
	}
	for k := range out {
		sort.Strings(out[k])
	}
	return out
}

// topicToolDomains maps a recipe topic to the tool-name domains relevant to it.
// Used by toolsForTopic to scope the tool listing in topic mode.
var topicToolDomains = map[string][]string{
	"pages":      {"node", "nodetype", "render", "layout", "media", "term", "taxonomy"},
	"editing":    {"node", "nodetype", "render", "layout", "term", "taxonomy", "settings", "media"},
	"blocks":     {"block_types", "render"},
	"themes":     {"theme", "layout", "block_types", "render", "settings", "menu"},
	"taxonomies": {"taxonomy", "term", "nodetype", "node"},
	"media":      {"media", "files"},
	"extensions": {"extension", "settings", "data", "events", "filters"},
}

// toolsForTopic returns full tool descriptions for the domains relevant to
// the topic. The AI gets exactly the verbs it needs without trawling all 95.
func (s *Server) toolsForTopic(topic string) []toolCatalogEntry {
	wanted, ok := topicToolDomains[topic]
	if !ok {
		// Unknown topic — return everything so the AI still gets useful info.
		return s.toolCatalog
	}
	wantedSet := map[string]bool{}
	for _, d := range wanted {
		wantedSet[d] = true
	}
	out := make([]toolCatalogEntry, 0, len(s.toolCatalog))
	for _, t := range s.toolCatalog {
		if wantedSet[toolDomain(t.Name)] {
			out = append(out, t)
		}
	}
	return out
}

func guideRecipes(topic string) []map[string]any {
	all := []map[string]any{
		{
			"goal":  "Publish a new page with an image",
			"topic": "pages",
			"steps": []string{"core.media.upload", "core.node.create", "core.render.node_preview"},
			"notes": "featured_image is an object {url, alt, ...}, NEVER a string. blocks_data is an array of {type, fields}.",
		},
		{
			"goal":  "Add a custom content block to a theme",
			"topic": "blocks",
			"steps": []string{"core.block_types.list", "core.block_types.create", "core.render.block", "core.node.update"},
			"notes": "html_template reads .fields and .node. Include realistic test_data so the admin preview works.",
		},
		{
			"goal":  "Switch the site's look",
			"topic": "themes",
			"steps": []string{"core.theme.list", "core.theme.activate"},
			"notes": "Theme activation does NOT require a server restart.",
		},
		{
			"goal":  "Add a custom node type (post type)",
			"topic": "pages",
			"steps": []string{"core.nodetype.list", "core.nodetype.create", "core.taxonomy.create", "core.node.create"},
			"notes": "label_plural is required for admin UI. url_prefixes decides public routing.",
		},
		{
			"goal":  "Diagnose a broken public page",
			"topic": "themes",
			"steps": []string{"core.theme.active", "core.layout.list", "core.render.node_preview", "core.block_types.get"},
			"notes": "render.* never fires events; safe to reproduce issues on live content.",
		},
		{
			"goal":  "Tag content with a new taxonomy",
			"topic": "taxonomies",
			"steps": []string{"core.taxonomy.create", "core.term.create", "core.node.update"},
			"notes": "Term field values are objects {slug, name}, never bare strings. Terms are per-language (rows carry language_code; defaults to site default). Translate via POST /admin/api/terms/<id>/translations {language_code:'<code>'} — source+clone share translation_group_id UUID. Uniqueness: (node_type, taxonomy, slug, language_code).",
		},
		{
			"goal":  "Wire up an extension (media manager, email, etc.)",
			"topic": "extensions",
			"steps": []string{"core.extension.list", "core.extension.activate"},
			"notes": "Activation is hot: HotActivate spawns plugin subprocess, runs migrations, loads scripts, registers blocks — no restart. Dropping a new extension dir on disk (docker cp / volume / git pull) is auto-picked by the fs watcher; core.extension.rescan is the explicit trigger for CI/ops. restart_required always false (reserved flag).",
		},
		{
			"goal":  "Upload and attach media",
			"topic": "media",
			"steps": []string{"core.media.upload", "core.media.get", "core.node.update"},
			"notes": "Use the returned {id, url, slug} — reference media by slug where possible for theme-portable nodes.",
		},
		{
			"goal":  "Verify theme documentation and schema compliance",
			"topic": "themes",
			"steps": []string{"core.theme.standards", "core.block_types.get"},
			"notes": "Rule 1.5 requires block descriptions; Rule 1.6 requires field help text. Always check standards before finalizing a theme.",
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
			"notes": "Subscribe to node_type.{created,updated,deleted} and taxonomy.{created,updated,deleted}, plus extension.activated, theme.activated and their deactivated counterparts.",
		},
		{
			"goal":  "Wire SEO meta tags into a theme's <head>",
			"topic": "themes",
			"steps": []string{
				"Open every layout in themes/<slug>/layouts/*.html",
				"Add `{{.app.head_meta}}` inside <head>, AFTER any hand-written <title> / <meta name=description> and BEFORE the head_styles loop",
				"Strip any hand-rolled <meta property=\"og:*\"> or <meta name=\"twitter:*\"> tags — the kernel emits them now and yours will duplicate",
				"Verify in /admin/site-settings → SEO that defaults (default OG image, og_site_name, twitter_handle, robots_index) are populated",
				"curl -s http://localhost/<slug> | grep -i 'og:' — confirm tags appear",
			},
			"notes": "head_meta resolution: per-node seo → site-wide defaults → node title/excerpt+featured_image. og:image auto-uses featured_image when seo.og_image is empty. Translations emit hreflang automatically. seo_robots_index='false' → noindex,nofollow in both X-Robots-Tag header and <meta name=robots>. 404s force noindex regardless.",
		},
		{
			"goal":  "Ship a custom 404 page with the theme",
			"topic": "themes",
			"steps": []string{
				"Create themes/<slug>/layouts/404.html — a normal layout that renders {{.node.blocks_html}} (the kernel-supplied 404 content lands there)",
				"Register it in theme.json: `{ \"slug\": \"404\", \"file\": \"404.html\", \"name\": \"Not Found\" }`",
				"Drop `{{.app.head_meta}}` into <head> (forced noindex,nofollow on 404s)",
				"core.theme.activate(<slug>) — picks up the new layout",
				"curl -i http://localhost/this-page-does-not-exist — verify status 404 and themed body",
			},
			"notes": "Slug `404` recognized first; legacy alias `error` also accepted. Neither registered → default layout handles 404s. Synthesized not-found body lives in {{.node.blocks_html}} (same shape as a regular page).",
		},
		{
			"goal":  "Build a theme end-to-end (cold-boot, AI one-shot)",
			"topic": "themes",
			"steps": []string{
				"core.theme.standards",
				"core.nodetype.create (custom node types — register BEFORE taxonomies that reference them)",
				"core.taxonomy.create (with node_types: [...] to attach to a node type)",
				"core.term.create (with node_type matching where the term is used)",
				"core.block_types.create (or via theme files: blocks/<slug>/{block.json,view.html})",
				"core.node.create (pages/posts; use fields_data:{} at top level, blocks_data:[{type, fields:{}}])",
				"core.menu.upsert (items use page:'<slug>' for slug-stable menu)",
				"core.theme.activate",
				"core.theme.standards (run again — verify no warnings)",
				"GET / (curl/playwright — confirm public render returns 200, not empty)",
			},
			"notes": "Asymmetries to internalize: (1) node→fields_data, block→fields. (2) block.json field key=`key`, nodetypes.register=`name`. (3) select options = plain strings, never {value,label}. (4) term fields: schema needs term_node_type; values stored as {slug,name}. (5) Settings keys keep dots — `index $s \"k.v\"` or mustSetting. (6) Theme HTTP routes mount at /api/theme/<path>. (7) `dev_mode` global in seeds (SQUILLA_DEV_MODE=true) → overwrite-on-reseed for fast iteration.",
		},
		{
			"goal":  "Add a settings page to a theme",
			"topic": "themes",
			"steps": []string{
				"1. Edit themes/<slug>/theme.json and add a settings_pages entry:\n     { \"slug\": \"header\", \"name\": \"Header Settings\", \"file\": \"settings/header.json\" }",
				"2. Create themes/<slug>/settings/header.json with { name, fields: [...] }.",
				"3. core.theme.activate(<slug>) — picks up the new schema.",
				"4. In templates: {{ .theme_settings.header.logo }}\n   In blocks:    {{ themeSetting \"header\" \"logo\" }}\n   In Tengo:     ts := import(\"core/theme_settings\"); ts.get(\"header\", \"logo\")",
				"5. Storage: theme:<slug>:header:logo in site_settings (auto-encrypted for\n   secret-shaped keys).",
				"6. Admin UI: a \"Theme Settings\" sidebar section appears with one entry per\n   declared page.",
			},
			"notes": "Field shapes = standard field types (text/number/toggle/image/select/repeater/...). Saved-value type mismatches don't mutate the DB — render falls back to declared default; admin shows 'previous value' hint. Tengo callers need theme_settings:read (themes bypass; extensions list in extension.json). Bad pages soft-fail at activation (logged & skipped). Run core.theme.checklist({slug}) to validate before activating.",
		},
		{
			"goal":  "Deploy a theme that lives outside the primary repo",
			"topic": "themes",
			"steps": []string{"core.theme.standards", "core.theme.deploy", "core.render.node_preview"},
			"notes": "Zip (theme.json at root or one level deep), base64-encode, core.theme.deploy({body_base64, activate:true}). Archive lands in data/themes/<slug>/ (persistent volume) via atomic dir swap. Same-slug overrides bundled copy (data wins). 50 MB cap (use core.theme.deploy_init/_finalize for bigger). Slug ∈ [A-Za-z0-9_-]+. Re-deploy refreshes the row in place.",
		},
		{
			"goal":  "Deploy an extension from a local build (no docker cp, no git push)",
			"topic": "extensions",
			"steps": []string{"core.extension.standards", "core.extension.deploy", "core.extension.get"},
			"notes": "Zip extension dir (extension.json + admin-ui/dist + scripts + bin/<plugin> if any), base64-encode, core.extension.deploy({body_base64, activate:true}). Lands in data/extensions/<slug>/ (persistent). Plugin binaries auto-chmod 0755. Pre-build for host OS/arch — no cross-compile. activate:true runs HotActivate (migrations, plugin spawn, scripts, blocks) without restart. 50 MB cap (use core.extension.deploy_init/_finalize for bigger).",
		},
		{
			"goal":  "Author and publish a documentation page (squilla theme)",
			"topic": "editing",
			"steps": []string{
				"core.nodetype.get(slug='documentation') — confirm the field_schema (order, section{name,slug}, summary)",
				"core.guide(topic='editing') — read editing_playbook.docs_page.fields_data and editing_playbook.docs_blocks for shapes",
				"core.layout.list — confirm 'docs' layout exists",
				"core.node.query({node_type:'documentation', limit:200}) — see existing fields_data.section values to reuse a section, or coin a new {name,slug}",
				"core.node.create({node_type:'documentation', language_code:'en', title, slug, status:'published', layout_slug:'docs', excerpt, seo_settings:{meta_title, meta_description}, fields_data:{order, section:{name,slug}, summary}, blocks_data:[...]})",
				"Inspect the response.warnings[] — fix anything flagged before declaring done",
				"core.render.node_preview(id) — verify rendered HTML; or open response.full_url in a browser",
			},
			"notes": "layout_slug:'docs' is REQUIRED — the documentation node-type does not auto-fill a default layout. Sections are an editorial concept (sidebar grouping); the slug must be kebab-case and stable (used as the URL fragment / sidebar key). Reuse an existing section.slug to keep the sidebar grouping coherent.",
		},
		{
			"goal":  "List the existing 'sections' inside a node-type (docs sidebar groups, etc.)",
			"topic": "editing",
			"steps": []string{
				"core.guide(topic='editing') — read snapshot.sections_by_node_type",
				"core.node.query({node_type:'documentation', limit:200}) — fall back when the snapshot is omitted (cold cache)",
			},
			"notes": "There's no dedicated 'list sections' tool because sections are a soft convention on fields_data.section. The guide snapshot aggregates distinct {slug,name} per node-type for cheap discovery.",
		},
		{
			"goal":  "Pick a layout for a content node (set layout_slug)",
			"topic": "editing",
			"steps": []string{"core.layout.list", "core.node.create / core.node.update with layout_slug:'<slug>'"},
			"notes": "If layout_slug is omitted, the node uses the active theme's default layout. Node-type-level defaults are NOT applied at write time; setting layout_slug explicitly is the only reliable way to opt into a non-default layout (e.g. 'docs').",
		},
		{
			"goal":  "Bulk-patch a column across many nodes (normalization sweep)",
			"topic": "editing",
			"steps": []string{
				"core.node.query({node_type:'documentation', limit:200}) — sanity-check what's missing",
				"core.node.update_many({filter:{node_type:'documentation'}, set:{layout_slug:'docs'}}) — only safe top-level columns are accepted: status, layout_slug, language_code",
				"core.node.query — confirm the sweep landed",
			},
			"notes": "Use update_many ONLY for top-level column normalization. Patches to fields_data / blocks_data / seo_settings still need per-node core.node.update so per-node merge logic runs.",
		},
		{
			"goal":  "Browse and restore a node revision",
			"topic": "editing",
			"steps": []string{
				"core.node.revisions(id) — newest-first list (≤100 entries) of {revision_id, version_number, status, language_code, layout_slug, created_at, created_by}",
				"core.node.get(id) — capture the current state's revision_id you'd want to come back to",
				"core.node.revision_restore({id, revision_id}) — applies the snapshot; returns the restored node",
			},
			"notes": "Restore is itself reversible — it snapshots the pre-restore state as a fresh revision before applying the chosen one. Migration 0041 widened revisions from blocks+SEO only to a full snapshot (title/slug/status/language/layout/excerpt/featured_image/fields_data/taxonomies). Pre-0041 rows lack the new columns and restore through a partial-restore code path.",
		},
		{
			"goal":  "Upload a large media file (>5 MB) without base64 overhead",
			"topic": "media",
			"steps": []string{
				"core.media.upload_init({filename, mime_type}) → {upload_url, upload_token, expires_at, max_bytes}",
				"PUT <upload_url> with the raw bytes — NO Authorization header (the token in the URL is the auth)",
				"core.media.upload_finalize({upload_token, sha256?}) — returns the same shape as core.media.upload",
			},
			"notes": "Token TTL ~15 min, single-use, bound to the issuing user. Bytes stream straight to data/pending/<token>.bin while computing SHA-256. Default cap 50 MB (env: SQUILLA_MEDIA_MAX_MB). Same flow exists for themes (core.theme.deploy_init / .deploy_finalize, default 200 MB) and extensions (core.extension.deploy_init / .deploy_finalize, default 200 MB).",
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
		"image":            `{ "url": "/media/...", "alt": "...", "width": 800, "height": 600 }`,
		"link":             `{ "label": "Read more", "url": "/about", "target": "_self" }`,
		"repeater":         `[ { "<sub_field>": "..." }, ... ]`,
		"term":             `{ "slug": "travel", "name": "Travel", "taxonomy": "tag" }  // STORE AS OBJECT, not bare slug — admin's term-field component requires {slug,name} to pre-select. Templates handle both shapes for safety.`,
		"blocks_data":      `[ { "type": "<slug>", "fields": { ... } }, ... ]  // INSIDE a block: use "fields" (not "fields_data")`,
		"node_fields_data": `{ "<field_name>": <value>, ... }  // TOP-LEVEL on a node: use "fields_data" (not "fields"). Asymmetric on purpose; misnaming silently drops data.`,
		"node_taxonomies":  `{ "category": ["engineering"], "tag": ["go", "cms"] }  // For real taxonomies (admin "Taxonomies" tab, tax_query). Term-typed fields go in fields_data instead.`,
		"select/radio":     `"<string value>"  // options in schemas MUST be plain strings, not {label,value} objects (admin crashes with React #31)`,
		"menu_item_in":     `{ "label": "Home", "page": "home" }  // input uses label: + page:'<slug>'`,
		"menu_item_out":    `{ "title": "Home", "url": "/", ... }  // !! Templates iterate menu items and read .title and .url, NOT .label`,
		"settings_lookup":  `{{ index $s "squilla.brand.version" }}  // settings keys keep their dots; use the index fn or mustSetting helper`,
	}
}

// guideGotchas returns machine-readable asymmetries and silent-failure modes.
// Surfaced on every core.guide call so AI agents internalize them up front
// rather than rediscovering each one through trial and error.
func guideGotchas() []map[string]any {
	return []map[string]any{
		{
			"topic":   "fields_vs_fields_data",
			"summary": "Top-level node uses `fields_data:` — blocks inside blocks_data use `fields:`. Mismatching either silently drops the data; nodes.create/update now warn at runtime.",
		},
		{
			"topic":   "block_select_options",
			"summary": "block.json `field_schema` `select`/`radio` options must be plain strings (`[\"a\",\"b\"]`), never `[{value,label}]`. Object options are rejected at theme load and would otherwise crash the admin with React #31.",
		},
		{
			"topic":   "term_field_shape",
			"summary": "Term-typed fields: schema needs `term_node_type`; values must be stored as `{slug, name}` objects (not bare slug strings) so admin pre-selects. Hydration accepts strings on render only.",
		},
		{
			"topic":   "schema_key_vs_name",
			"summary": "block.json field_schema uses `key:` (and `key:` for sub_fields). Tengo nodetypes.register / taxonomies.register use `name:`. Don't mix — empty admin inputs are the symptom.",
		},
		{
			"topic":   "settings_dot_keys",
			"summary": "Settings keys keep their dots in the DB (`squilla.brand.version`). Go templates can't dot-traverse — use `{{ index $s \"key\" }}` or `{{ mustSetting $s \"key\" }}` (loud error on miss).",
		},
		{
			"topic":   "menu_label_vs_title",
			"summary": "menus.upsert input uses `label:` + `page:'<slug>'`. The Tengo→template bridge currently emits `label` to templates; check the active code path before assuming `.title`. Use slug-based `page:` for rename-stable menus.",
		},
		{
			"topic":   "theme_routes_prefix",
			"summary": "routes.register(\"GET\", \"/docs\", …) mounts at `/api/theme/docs`, NOT `/docs`. Themes cannot shadow public node routes; for redirects either point the menu link directly at the destination or use an extension public_route.",
		},
		{
			"topic":   "homepage_and_settings_cache",
			"summary": "site-settings (incl. homepage_node_id) are cached in-process; `core.settings.set` now publishes setting.updated which invalidates the cache. Older builds required theme.activate to bust the cache.",
		},
		{
			"topic":   "log_error_unreachable",
			"summary": "`error` is a reserved Tengo selector — `log.error(\"…\")` is a parse error. Use `log.warn(…)`, `log.info(…)`, or the alias `log.err(…)`. Direct CoreAPI callers (gRPC/Go) still use `error`.",
		},
		{
			"topic":   "theme_layout_cache",
			"summary": "Layouts and partials are loaded into the renderer cache at theme activation. Editing files on disk requires re-activating the theme (or in dev: `make theme` which restarts the container).",
		},
		{
			"topic":   "seed_idempotency",
			"summary": "Production seeds must be idempotent (existence-checked nodes/terms, upsert menus). In dev (SQUILLA_DEV_MODE=true) seeds receive a top-level `dev_mode` boolean and may branch to delete-then-create so AI iteration loops can see template + data changes immediately.",
		},
		{
			"topic":   "no_default_fallbacks",
			"summary": "Don't write `{{ or .x \"Default copy\" }}` in templates — defaults silently mask missing data and turn schema/seed bugs into mystery renders. Strip fallbacks; let empty fields render empty so the bug is loud.",
		},
		{
			"topic":   "block_slug_collisions",
			"summary": "Prefix theme block slugs (e.g. `sq-hero`, `mytheme-cta`) so they don't collide with extension blocks (`cb-*` from content-blocks) or other themes. Last-write wins per slug — collisions are silent.",
		},
		{
			"topic":   "filter_args_required",
			"summary": "`{{ filter \"name\" }}` (no value arg) errors with \"wrong number of args\". For filters that take no input, pass `(dict)` as the value argument.",
		},
		{
			"topic":   "settings_per_language",
			"summary": "site_settings rows have a language_code. Reads scope to caller's locale (X-Admin-Language admin / request locale public) with default-language fallback. Legacy `''` shared sentinel removed (migration 0040 backfilled). Theme settings same model; no `translatable` flag — every field is implicitly per-locale.",
		},
		{
			"topic":   "terms_per_language",
			"summary": "taxonomy_terms are per-language. Uniqueness: (node_type, taxonomy, slug, language_code). Translate via POST /admin/api/terms/<id>/translations {language_code:'<code>'} — source gets fresh translation_group_id UUID, clone joins same group. Route /terms/:id/translations registered before /terms/:nodeType/:taxonomy (Fiber matches by registration order).",
		},
		{
			"topic":   "themes_persistent_data_dir",
			"summary": "Two parallel dirs: image-bundled (themes/, extensions/, read-only) + operator-installed (data/themes/, data/extensions/, persistent volume). Both scanned at boot; data/ wins on slug collision. All writes (theme.deploy, extension.deploy, git install, zip) target data/. Delete refuses rmdir under bundled root. docker-compose: ./data:/app/data; coolify: squilla-data named volume. Missing volume → theme.deploy gets wiped on container restart.",
		},
		{
			"topic":   "theme_activate_pre_flight",
			"summary": "core.theme.activate stat()s theme.json before deregistering the previous theme. Missing manifest → returns error and the previous theme stays intact, preventing 'reactivate to fix it' from wiping all blocks/layouts/templates with nothing to replace them.",
		},
		{
			"topic":   "lost_admin_password",
			"summary": "Recover via CLI: `docker exec -it <app> ./squilla reset-password <email> <new-password>`. Hashes via auth.HashPassword, writes to users.password_hash. Idempotent. Works without SMTP. ADMIN_PASSWORD env only seeds the first-boot user — does NOT reset an existing user.",
		},
		{
			"topic":   "git_install_https_only",
			"summary": "core.theme.git_install (and InstallFromGit) only accepts https:// URLs. SSH-style git@github.com:owner/repo.git is rejected upfront with a clear message — the kernel SSH key would be overprivileged for cloning arbitrary themes. For private repos use a https URL + a personal access token in the token field.",
		},
		{
			"topic":   "kernel_extensions_boundary",
			"summary": "Commit 7e49268 moved features out of core: email dispatcher → extensions/email-manager; /robots.txt + OG/Twitter head_meta → extensions/seo-extension; media_files + optimisation pipeline → extensions/media-manager. CoreAPI surface (SendEmail, UploadMedia, ...) unchanged — calls route to the active plugin that declares the matching `provides` tag (`email.provider`, `media-provider`). 'no provider for X' = activate the right extension. Hot-swap S3/R2/Cloudinary by activating a higher-priority plugin with provides:['media-provider'].",
		},
		{
			"topic":   "presigned_uploads",
			"summary": "Binaries >5 MB: core.<kind>.upload_init → PUT raw bytes to /api/uploads/<token> (no Authorization header — token IS the auth) → core.<kind>.upload_finalize. <kind> ∈ {media, theme, extension}. Tokens 64 chars, single-use, ~15 min TTL, user+kind bound. Caps via SQUILLA_{MEDIA,THEME,EXTENSION}_MAX_MB (50/200/200). Legacy base64 tools still work for tiny payloads.",
		},
		{
			"topic":   "node_revisions_full_snapshot",
			"summary": "Migration 0041 widened content_node_revisions from blocks+SEO only to full snapshots (title, slug, status, language_code, layout_slug, excerpt, featured_image, fields_snapshot, taxonomies_snapshot, version_number). core.node.revision_restore is itself reversible (snapshots pre-restore state). Pre-0041 rows go through a partial-restore path. Daily sweep keeps 50 newest per node.",
		},
		{
			"topic":   "settings_registry_translatable",
			"summary": "Kernel settings live in internal/settings/builtin.go grouped: general/seo/advanced/languages/security. Each field has a Translatable flag — non-translatable reads default-language row directly; translatable uses per-locale composite PK (key, language_code) with default-language fallback. Extensions add groups via Registry.RegisterGroup. core.settings.* hides this; raw site_settings rows show the composite PK.",
		},
	}
}

func guideConventions() []string {
	return []string{
		"All list/query tools accept {limit, offset}; default 25, max 200; responses include {total}.",
		"render.* tools are side-effect-free — no events, no view counts, no writes.",
		"restart_required is currently always false — theme and extension activate/deactivate are hot. Activation spawns/kills only the plugin subprocess, not the app. Brand-new directories dropped into themes/ or extensions/ on disk are picked up automatically by the fs watcher (no restart, no manual rescan); core.theme.rescan / core.extension.rescan exist as explicit triggers for CI/ops.",
		"Prefer typed tools (core.node.query, core.data.query) over core.data.exec (gated raw SQL).",
		"Reference media/layouts by slug when authoring theme-portable content; IDs rotate when themes are reactivated.",
		"For binaries above ~5 MB use the presigned upload pair (core.<kind>.upload_init + .upload_finalize, kind ∈ {media, theme, extension}). The PUT route at /api/uploads/<token> is unauthenticated — the 64-char token IS the auth, single-use, ~15 min TTL.",
		"Bulk-patch nodes via core.node.update_many ONLY for safe top-level columns (status, layout_slug, language_code). For fields_data / blocks_data / seo_settings use per-node core.node.update so per-node merge logic runs.",
		"Revisions are full snapshots since migration 0041 — core.node.revision_restore recreates every editable field, and the pre-restore state is captured as a fresh revision so restore is itself reversible.",
		"Kernel/extensions boundary (commit 7e49268): email dispatcher lives in email-manager, /robots.txt + head_meta in seo-extension, media_files in media-manager. Calls to core.email.send / core.media.* fail with 'no provider' if the corresponding `provides` claimant is inactive — activate the relevant extension or ship a replacement.",
	}
}

// toolIndex returns a flat index of every currently-registered tool with its
// one-line description, captured at registration time by addTool.
func (s *Server) toolIndex() []toolCatalogEntry {
	return s.toolCatalog
}

// guideTopics enumerates every topic referenced by the static recipes plus
// the synthetic 'editing' topic. Returned on every core.guide call so
// clients can discover narrowing options without trial and error.
func guideTopics() []string {
	seen := map[string]bool{}
	for _, r := range guideRecipes("") {
		if t, ok := r["topic"].(string); ok && t != "" {
			seen[t] = true
		}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	// Stable order — easier diff and predictable AI prompts.
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// themeStandards returns the theme rulebook. Compact mode (verbose=false) drops
// the embedded template-source examples and the long seeding_patterns prose —
// keeps philosophy, structure, rules, capabilities, lifecycle, field types, and
// reserved layout slugs. Verbose returns everything.
func themeStandards(verbose bool) map[string]any {
	out := themeStandardsCore()
	if verbose {
		out["examples"] = themeStandardsExamples()
		out["seeding_patterns"] = themeStandardsSeedingPatterns()
		return out
	}
	out["examples_pointer"] = "Pass verbose:true for embedded source of theme.json / block.json / view.html / theme.tengo. The same files live in themes/squilla/ as a working reference."
	out["seeding_pointer"] = "Pass verbose:true for the seeding_patterns map (registration order, idempotency, menus, well-known, assets, forms, hot-reload, autoregistration, layout seeding, terms module, schema field keys)."
	return out
}

func themeStandardsCore() map[string]any {
	return map[string]any{
		"philosophy": "A theme is a self-bootstrapping marketing site: drop a folder under themes/, restart the app, and a complete demo site appears — pages, layouts, blocks, taxonomies, settings, menus, forms, and seeded content. The theme must render correctly from a cold boot with nothing but its own files — no manual DB edits, no magic.",
		"structure": map[string]any{
			"theme.json": "Manifest registering layouts, partials, blocks, page templates, assets, styles, and scripts.",
			"layouts/":   "Page layouts (default.html, trip.html, etc.) using Go html/template. Sees full .node, .app, .user.",
			"partials/":  "Reusable fragments included from a layout via {{ renderLayoutBlock \"slug\" }}. Sees full .node, .app, .user, plus .partial.",
			"blocks/":    "Content blocks: each with a template (view.html), schema (block.json), and optional scoped style.css / script.js. Sees ONLY the block's own field values at root — .app and .node are NOT in scope.",
			"assets/":    "Static images, CSS, JS, fonts. Reference via theme-asset:<key> for declared media; /theme/assets/<src> for declared CSS/JS.",
			"templates/": "Page templates: pre-built block sequences editors can apply with one click. Not seeds — use theme.tengo for seeding.",
			"scripts/":   "scripts/theme.tengo (entry — runs once on activation) plus scripts/filters/<name>.tengo for template-callable filters.",
			"forms/":     "Optional. Theme-owned form layouts as Go templates, registered via the forms-extension handshake (events.emit \"forms:upsert\").",
			"settings/":  "Optional. Per-page JSON schemas referenced from theme.json's settings_pages[]. Each file declares { name, description?, fields: [...] }; field shapes follow the standard field-types reference. Stored under theme:<slug>:<page>:<field> in site_settings (per-theme namespace). Surfaced in admin under a 'Theme Settings' sidebar group; readable from layouts via .theme_settings.<page>.<field>, from blocks via themeSetting/themeSettingsPage helpers, from Tengo via core/theme_settings (capability theme_settings:read).",
		},
		"template_functions": []string{
			"{{ renderLayoutBlock \"slug\" }} — render a partial. LAYOUT/PARTIAL ONLY. There is NO `partial` template function.",
			"{{ filter \"name\" value }} — run a registered Tengo filter (list_nodes, get_node, distinct_field, ...).",
			"{{ event \"name\" ctx }} — fire an event; collect HTML responses (used for forms:render, etc.). Returns template.HTML.",
			"{{ safeHTML s }} — bypass HTML escaping. Use only on fields you trust or on event results.",
			"{{ image_url .url \"size\" }} — cached, optimized image URL (sizes from media-manager).",
			"{{ image_srcset .url \"size1\" \"size2\" ... }} — responsive srcset attribute value.",
			"{{ dict k1 v1 k2 v2 ... }} — build a map literal (used to pass args to filter/event).",
			"{{ list a b c ... }} — build a slice literal.",
			"{{ seq n }} — range over [0..n-1].",
			"{{ add a b }} / {{ sub a b }} / {{ mod a b }} — integer math.",
			"{{ split sep s }} / {{ lastWord s }} / {{ beforeLastWord s }} — string helpers.",
		},
		"context_scoping": map[string]string{
			"layout":  "Sees full .node (id, title, slug, fields, blocks_html, taxonomies, ...), .app (head_styles, foot_scripts, head_meta, block_styles, block_scripts, settings, menus, current_lang, theme_url, ...), .user (logged_in, role, ...).",
			"partial": "Same as layout, plus .partial map populated from any partial-level field_schema.",
			"block":   "Sees ONLY the block's own field values at root, e.g. {{ .heading }}, {{ .items }}. Cannot reach .app or .node. To pull node data into a block, use {{ filter \"list_nodes\" ... }} or {{ filter \"get_node\" ... }}.",
		},
		"head_checklist": map[string]any{
			"intent":  "Every layout that wraps a public page must drop these into <head> so SEO chrome and per-block assets render correctly.",
			"snippet": "<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <title>{{ if and .node.seo .node.seo.meta_title }}{{ .node.seo.meta_title }}{{ else }}{{ .node.title }}{{ end }}</title>\n  {{.app.head_meta}}\n  {{- range .app.head_styles -}}<link rel=\"stylesheet\" href=\"{{.}}\">{{- end -}}\n  {{.app.block_styles}}\n</head>",
			"head_meta": "Pre-built canonical/og:*/twitter:*/robots/hreflang block. Per-node SEO wins; site-wide defaults from Site Settings → SEO fill the gaps; node title/excerpt and featured_image are the final fallback. og:image auto-uses the node's featured_image when seo.og_image is empty. Translations emit hreflang alternates automatically. DO NOT hand-roll og:* / twitter:* tags in your layout — they'll duplicate what the kernel already emits.",
			"head_styles":   "Theme + extension <link rel=stylesheet> URLs. Always render with the {{ range }} loop above.",
			"block_styles":  "Per-block scoped CSS for blocks present on this page (built from blocks/<slug>/style.css).",
		},
		"reserved_layout_slugs": map[string]string{
			"404": "Theme-supplied Page Not Found layout. Register `{ \"slug\": \"404\", \"file\": \"404.html\" }` in theme.json to opt in. The synthesized 404 content lands in `{{.node.blocks_html}}` so a typical 404 layout is your default chrome with blocks_html inside it. head_meta still flows through (with noindex,nofollow forced for 404s). When neither `404` nor the legacy alias `error` is registered, the default layout handles missing pages.",
			"error": "Legacy alias for 404. Recognized for backward compatibility (e.g. squilla theme's existing layouts/error.html); new themes should use `404`.",
		},
		"core_rules": []map[string]any{
			{"id": "1", "title": "Every block is a complete content type.", "description": "block.json declares every field its view.html reads. Every field has a value in test_data. Every field has a help line."},
			{"id": "2", "title": "Every template field is gated by {{with}}.", "description": "No hardcoded fallback strings in view.html. An empty field renders nothing — never canned copy like 'Click here' or 'Welcome'."},
			{"id": "3", "title": "Every image goes through an image field with theme-asset:<key>.", "description": "Never hardcode /theme/assets/images/... in a view.html. Asset references survive theme switches and media-manager URL rewrites; hardcoded paths don't."},
			{"id": "4", "title": "Every image asset is declared in theme.json assets[] with real alt text.", "description": "The media-manager extension imports declared assets on theme.activated. Undeclared images can't be resolved by theme-asset:<key>."},
			{"id": "5", "title": "Every demo page has a matching templates/<slug>.json.", "description": "Editors get a one-click way to re-create your seeded pages."},
			{"id": "6", "title": "theme.tengo is idempotent.", "description": "Existence checks for content (nodes.query → if total == 0 then create); upserts for menus/settings; force:true on forms:upsert only while developing."},
			{"id": "7", "title": "Cross-extension rendering goes through event.", "description": "Use event \"forms:render\" for forms; never hand-roll a <form> and JS-intercept submit (submissions get silently dropped)."},
			{"id": "8", "title": "Settings are namespaced.", "description": "Use <theme-prefix>.<dot.path> (e.g. hv.whatsapp, hv.social.instagram) so editor overrides don't collide with another theme's keys."},
			{"id": "9", "title": "Menu items use page:\"<slug>\", not url:\"/<slug>\".", "description": "menus.upsert resolves slug → NodeID at upsert time, so renames don't break menus."},
			{"id": "10", "title": "safeHTML only on fields you trust.", "description": "Prefer Go's auto-escaping. Mark HTML-bearing fields explicitly in block.json's help. safeHTML on user-controlled strings is XSS."},
			{"id": "11", "title": "Block-scoped CSS goes in blocks/<slug>/style.css.", "description": "Site-wide CSS goes in assets/styles/theme.css. Don't dump <style> blocks inside view.html."},
			{"id": "12", "title": "No dead schema fields.", "description": "If you remove a field from view.html, remove it from block.json, the templates/*.json, and the theme.tengo seed. Schema and template must agree."},
			{"id": "13", "title": "Drop {{.app.head_meta}} into every layout's <head>.", "description": "The kernel composes canonical/og:*/twitter:*/robots/hreflang once per render. Layouts must surface it — without {{.app.head_meta}}, social previews and search engines see only your hand-written <title>/<meta name=description>. Do NOT hand-roll og:* tags; they will duplicate the kernel's output."},
			{"id": "14", "title": "Reserve layout slug \"404\" for missing pages.", "description": "Register `{ \"slug\": \"404\", \"file\": \"404.html\" }` (or legacy alias \"error\") in theme.json. Public renderer auto-uses it for 404 responses. Synthesized 404 content lands in {{.node.blocks_html}}. Falls back to the default layout when absent."},
		},
		"field_types": []map[string]any{
			{"type": "text", "intent": "Single-line input", "shape": `"..."`},
			{"type": "textarea", "intent": "Multi-line input", "shape": `"..."`},
			{"type": "richtext", "intent": "WYSIWYG / HTML", "shape": `"<p>...</p>"`},
			{"type": "number", "intent": "Numeric", "shape": `42`},
			{"type": "toggle", "intent": "Boolean (switch)", "shape": `true | false`},
			{"type": "checkbox", "intent": "Boolean (checkbox)", "shape": `true | false`},
			{"type": "select", "intent": "Dropdown — flat string options", "shape": `"red"  // schema: "options": ["red","yellow","green"]`},
			{"type": "radio", "intent": "Radio group", "shape": `"left"`},
			{"type": "color", "intent": "Color picker", "shape": `"#FF0000"`},
			{"type": "link", "intent": "CTAs/Buttons", "shape": `{"text": "Explore", "url": "/trips", "target": "_self"}`},
			{"type": "image", "intent": "Single image (media-picker)", "shape": `{"url": "theme-asset:<key>", "alt": "..."}`},
			{"type": "gallery", "intent": "Multi-image picker", "shape": `[{"url": "theme-asset:<key>", "alt": "..."}, ...]`},
			{"type": "term", "intent": "Taxonomy term picker (set taxonomy + term_node_type in schema)", "shape": `{"slug": "foodie", "name": "Foodie"}`},
			{"type": "node", "intent": "Node picker (set node_types in schema to restrict)", "shape": `{"slug": "hanoi-trip", "title": "Hanoi Street Food"}  // engine resolves id at render time`},
			{"type": "form_selector", "intent": "Pick a form from forms-extension", "shape": `"<form-slug>"`},
			{"type": "repeater", "intent": "Nested array of sub_fields", "shape": `[{...}, {...}]  // schema: "sub_fields": [...]`},
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
		"authoritative_resource": "squilla://guidelines/themes",
	}
}

// themeStandardsExamples returns the embedded source examples for theme files.
// Heavy (~2 k tokens) — only included when verbose=true.
func themeStandardsExamples() map[string]string {
	return map[string]string{
		"theme.json": `{
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
}`,
		"block.json": `{
  "slug": "intro",
  "name": "Intro",
  "description": "Centered headline + body + optional image.",
  "category": "my-theme",
  "field_schema": [
    { "key": "heading", "label": "Heading", "type": "text",     "help": "The H1." },
    { "key": "body",    "label": "Body",    "type": "textarea", "help": "1-3 sentences of intro copy." },
    { "key": "image",   "label": "Image",   "type": "image",    "help": "Hero photo." }
  ],
  "test_data": {
    "heading": "Welcome.",
    "body":    "This is your new theme.",
    "image":   { "url": "theme-asset:hero", "alt": "Hero photo" }
  }
}`,
		"view.html": `{{- $img := "" -}}{{- $alt := "" -}}{{- with .image -}}
  {{- with .url -}}{{- $img = . -}}{{- end -}}
  {{- with .alt -}}{{- $alt = . -}}{{- end -}}
{{- end -}}
<section class="intro">
  {{ with .heading }}<h1>{{ . }}</h1>{{ end }}
  {{ with .body }}<p>{{ . }}</p>{{ end }}
  {{ if $img }}<img src="{{ $img }}" alt="{{ $alt }}">{{ end }}
</section>`,
		"theme.tengo": `nodes    := import("core/nodes")
menus    := import("core/menus")
settings := import("core/settings")

// Idempotent setting seed
seed_setting := func(key, value) {
    existing := settings.get(key)
    if existing == "" || is_error(existing) {
        settings.set(key, value)
    }
}
seed_setting("site.copyright_year", "2026")

// Idempotent page seed (existence check)
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
    settings.set("homepage_node_id", string(home.id))
}

// Slug-based menu (survives renames)
menus.upsert({
    slug: "main-nav",
    name: "Primary Navigation",
    items: [ { label: "Home", page: "home" } ]
})`,
	}
}

// themeStandardsSeedingPatterns returns the seeding-pattern playbook. Only
// included when verbose=true; the prose runs ~1.5 k tokens.
func themeStandardsSeedingPatterns() map[string]string {
	return map[string]string{
		"registration":      "Always register taxonomies BEFORE node types that use them.",
		"idempotency":       "Use existence checks (e.g. nodes.query({node_type, slug, limit:1}) and branch on .total == 0) to avoid duplicate data on script re-runs. theme.tengo runs again on every restart while the theme is active.",
		"menus":             "Use core/menus → menus.upsert({slug, name, items:[{label, page:'<slug>'}]}). The page:<slug> form resolves to NodeID at upsert so slug renames don't break menus.",
		"wellknown":         "Use core/wellknown to register /.well-known/* handlers (e.g. apple-app-site-association). Unregistered paths return instant 404 via WellKnownRegistry, mounted before the public catch-all.",
		"assets_module":     "Use core/assets to read files from the calling theme/extension's own root: assets.read('forms/trip-order.html') / assets.exists('data/regions.json'). Returns a string, or an error value (wrap with is_error) if missing or path escapes root. Path is relative to the theme/extension dir; absolute paths and ../ traversal are rejected.",
		"forms_seeding":     "Theme-bundled forms: emit core/events 'forms:upsert' with {slug, name, fields, layout, settings, force?}. Idempotent on slug — without force, an existing same-slug form is left alone (admin edits stick); with force:true, theme overwrites on every reload. Form HTML is server-rendered via {{event \"forms:render\" (dict \"form_id\" \"<slug>\" \"hidden\" (dict ...))}}. Public submit endpoint is /forms/submit/<slug>.",
		"assets_hot_reload": "Theme assets are served via an atomic-pointer resolver that swaps on theme.activated. Runtime theme switches serve the new theme's assets instantly with no restart.",
		"autoregistration":  "Themes in themes/ are autoregistered at startup. No DB seeding is required to surface a new theme.",
		"layout_seeding":    "The core default layout is seeded with source='seed' (migration 0036). Themes are free to install their own base.html / blank.html without colliding.",
		"terms_module":      "Seed taxonomy term ROWS via core/terms (terms.create / list / get / update / delete). Taxonomy DEFINITIONS go through core/taxonomies.register.",
		"schema_field_keys": "Field-schema key differs by surface: Tengo nodetypes.register / taxonomies.register use `name`; block.json field_schema uses `key`. Don't mix — wrong-key schemas silently produce empty admin inputs.",
	}
}

// extensionStandards returns the structured rule set every extension developer
// (human or AI) should follow. Compact mode (verbose=false) keeps the
// structured rulebook (manifest, capabilities, lifecycle, rules); verbose adds
// event_modes detail, sdui_reactivity, hot_deploy recipes.
func extensionStandards(verbose bool) map[string]any {
	out := extensionStandardsCore()
	if verbose {
		out["event_modes"] = extensionStandardsEventModes()
		out["sdui_reactivity"] = extensionStandardsSDUIReactivity()
		out["hot_deploy"] = extensionStandardsHotDeploy()
		return out
	}
	out["verbose_pointer"] = "Pass verbose:true for event_modes (fire-and-forget vs event-with-result detail), sdui_reactivity (SSE → TanStack invalidation), and hot_deploy recipes (admin_ui docker cp, plugin binary restart)."
	return out
}

func extensionStandardsCore() map[string]any {
	return map[string]any{
		"philosophy": "Extensions own their full stack: manifest, gRPC plugin, admin-UI micro-frontend, SQL migrations, blocks, public routes. Core is a kernel — if disabling/removing the extension would leave dead code in core, that code belongs in the extension.",
		"manifest": map[string]any{
			"required": []string{"name", "slug", "version"},
			"optional": []string{"author", "description", "priority", "provides", "capabilities", "plugins", "admin_ui", "settings_schema", "blocks", "templates", "layouts", "partials", "public_routes", "assets", "data_owned_tables"},
			"notes": []string{
				"slug MUST match the directory name (kebab-case).",
				"capabilities are enforced on every CoreAPI call. Declare exactly what's needed.",
				"public_routes are mounted on the public Fiber app without auth — proxied to HandleHTTPRequest with user_id=0.",
				"data_owned_tables[] declares which tables this extension may read/write through core.data.* — required for any extension that owns its own SQL tables (forms-extension declares forms/form_submissions/form_webhook_logs; email-manager declares email_*; media-manager declares media_files/media_image_sizes).",
				"`provides` tags are consumed by the kernel: `media-provider` makes the plugin the active backend for core.media.* (UploadMedia / GetMedia / QueryMedia / DeleteMedia); `email.provider` makes it the active backend for SendEmail dispatch. Highest-priority active plugin wins. Hot-swappable — operators can replace the bundled media-manager with an S3/R2/Cloudinary extension by activating it at higher priority.",
			},
		},
		"reserved_provider_tags": map[string]string{
			"media-provider": "Plugin handles core.media.* on behalf of the kernel. The bundled media-manager extension is the default. To hot-swap: ship a plugin that subscribes to the relevant CoreAPI calls AND declares provides:['media-provider'] at higher priority than 50.",
			"email.provider": "Plugin receives core.email.send dispatch from the email-manager dispatcher. smtp-provider and resend-provider both claim this tag; only the highest-priority active one is used.",
		},
		"capabilities": []string{
			"nodes:read", "nodes:write", "nodes:delete",
			"nodetypes:read", "nodetypes:write",
			"settings:read", "settings:write",
			"theme_settings:read", "theme_settings:write",
			"events:emit", "events:subscribe",
			"email:send",
			"menus:read", "menus:write", "menus:delete",
			"routes:register",
			"filters:register", "filters:apply",
			"media:read", "media:write", "media:delete",
			"users:read",
			"http:fetch",
			"log:write",
			"data:read", "data:write", "data:delete", "data:exec",
			"files:read", "files:write", "files:delete",
			"taxonomies:read", "taxonomies:write", "taxonomies:delete",
		},
		"admin_ui": map[string]any{
			"entry":          "Path to the built ES module, e.g. admin-ui/dist/index.js. The extension loader auto-injects a sibling <link rel='stylesheet'> for dist/index.css when present — DO NOT declare CSS in the manifest.",
			"menu.section":   "One of 'content' (default), 'design', 'development', 'settings'. Honored by the SDUI sidebar engine — items with no/unknown section land at the top level.",
			"settings_menu":  "Auto-spliced into the Settings sidebar group. Extensions that only contribute settings can omit menu entirely.",
			"icons":          "Any valid lucide-react icon name (e.g. 'ImageDown', 'Images'). Resolved dynamically; unknown names fall back to 'Puzzle'.",
			"slots":          "Named UI injection points. e.g. smtp-provider injects into email-manager's 'email-settings' slot.",
			"field_types":    "Custom field types registered for use in node type schemas, with shape supports/component metadata.",
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
			"window.__SQUILLA_SHARED__.ReactRouterDOM { useNavigate, useSearchParams, ... }",
			"window.__SQUILLA_SHARED__.Sonner { toast }",
			"window.__SQUILLA_SHARED__.ui (list-page primitives, AccordionRow, SectionHeader, CodeWindow, ...)",
			"@squilla/ui, @squilla/api, @squilla/icons resolve via vite externalize config — import normally.",
		},
		"plugin_interface": []string{
			"Initialize(hostConn) — get SquillaHost client; seed defaults idempotently.",
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
			"node.created":          "A node is created. Payload: node_id, node_title, node_type, slug, language_code.",
			"node.updated":          "A node's metadata or fields change. Payload: same as node.created.",
			"node.published":        "A node transitions to status=published. Payload: same as node.created.",
			"node.deleted":          "A node is deleted. Payload: node_id, node_title, node_type.",
			"node_type.created":     "Custom node type registered. Payload: slug.",
			"node_type.updated":     "Custom node type updated. Payload: slug.",
			"node_type.deleted":     "Custom node type removed. Payload: slug.",
			"taxonomy.created":      "Taxonomy registered. Payload: slug.",
			"taxonomy.updated":      "Taxonomy updated. Payload: slug.",
			"taxonomy.deleted":      "Taxonomy removed. Payload: slug.",
			"user.registered":       "A new user signs up. Payload: user_id, email, name.",
			"user.deleted":          "A user is removed. Payload: user_id, email.",
		},
		"http_routing": map[string]string{
			"admin_proxy":       "Auto-mounted at /admin/api/ext/<slug>/* for every active extension. Auth required (kernel session middleware). Anonymous = 401. Plugin sees req.Path with the wildcard tail (leading /) and req.UserId of the logged-in user (or 0 if anonymous).",
			"public_proxy":      "Mounted at the EXACT path declared in public_routes[]. NO auth. NO session. NOT under /api/... — the path you declare is the path users hit. /forms/submit/contact, /media/cache/medium/...jpg, etc. req.UserId is always 0.",
			"common_mistake":    "Assuming public routes live under /api/ext/<slug>/...They don't. The admin proxy uses /admin/api/ext/<slug>/*; the public proxy uses whatever path you declare verbatim. Test against the real URL.",
			"response_envelope": "Every error response (admin or public) uses {\"error\": \"<MACHINE_CODE>\", \"message\": \"<human text>\"}. For validation errors with per-field details, add \"fields\": {\"<field-id>\": \"<error>\"}. The admin UI and public scripts both read data.error and data.message — match the shape.",
		},
		"asset_references": map[string]string{
			"scheme":         "Extensions and themes both declare media in their manifest's assets[] entries. media-manager imports them on activation and tags them with the owner. Reference imported assets via the URI scheme `extension-asset:<slug>:<key>` (or `theme-asset:<key>` for themes).",
			"key_validation": "asset key must match ^[a-z0-9_-]+$. Path traversal in keys is rejected.",
			"resolution":     "The render pipeline walks the JSON tree and replaces extension-asset:<slug>:<key> strings with the resolved /media/extension/<slug>/<key>.<ext> URL before data hits view.html. Templates always see real URLs.",
			"why_use_it":     "Survives file moves (media-manager controls the path), survives WebP conversion, survives deactivation/reactivation (content-hash idempotent). Hardcoded /media/... paths in templates do NOT survive these transitions.",
			"deactivation":   "On extension.deactivated, media-manager removes every row tagged with this extension's slug along with the underlying file. Demo content goes with the extension — by design.",
		},
		"core_rules": []map[string]any{
			{"id": "1", "title": "The manifest is the contract.", "description": "Every binary, route, capability, block, custom field type, and admin UI route the extension wires up is declared in extension.json. If it's not there, the kernel can't see it."},
			{"id": "2", "title": "Capabilities are minimal.", "description": "Declare only what your code calls. Adding a capability is cheap; explaining data:write on a read-only extension is not."},
			{"id": "3", "title": "{\"error\": code, \"message\": text} is the public error envelope.", "description": "Every error response — admin or public — uses this shape. Clients read data.error and data.message; match the contract."},
			{"id": "4", "title": "Public routes are mounted at the path you declare.", "description": "Not under /api/... Not under /admin/... The path in public_routes[].path is the path users hit. Test the real URL."},
			{"id": "5", "title": "Tables are prefixed by the extension's domain.", "description": "forms, form_submissions, media_files, form_webhook_logs. Never collide with core tables (content_nodes, users, settings, menus)."},
			{"id": "6", "title": "JSONB columns come back as strings through the data store.", "description": "Always normalize before iterating. Use a normalizeJSONBFields helper that walks declared keys and json.Unmarshal()s strings to objects. The single biggest source of '[object Object]' bugs."},
			{"id": "7", "title": "Asset references survive theme/extension switches; hardcoded URLs don't.", "description": "Use extension-asset:<slug>:<key> and theme-asset:<key> everywhere. Never hardcode /media/extension/... paths in templates or seed data."},
			{"id": "8", "title": "Per-extension Tailwind builds are mandatory for Docker.", "description": "The admin shell's fallback @source works in dev but disappears in the Docker frontend stage. Every admin UI ships its own dist/index.css via @tailwindcss/vite + cssFileName: \"index\"."},
			{"id": "9", "title": "Use the design system primitives.", "description": "ListPageShell, ListHeader, ListSearch, ListFooter, EmptyState, Chip, StatusPill, RowActions, etc. Reach for them before rolling your own — visual consistency depends on it."},
			{"id": "10", "title": "All filter / sort / view / pagination state lives in URL params.", "description": "Refresh preserves it. Default values omit the param. Use replace:true for keystrokes; use resetPage:true when changing filters."},
			{"id": "11", "title": "Production code under 300 lines per file (500 hard limit).", "description": "Test files are exempt. Split early — forms/cmd/plugin/ is the model: handlers_<resource>.go, render.go, validation.go, etc."},
			{"id": "12", "title": "Don't reach for a slot pattern when an event-with-result will do.", "description": "Slots couple admin UIs at build time; events couple at runtime. Pick the looser coupling."},
			{"id": "13", "title": "Use provider tags for kernel-routed capabilities.", "description": "If your extension implements media or email backend behaviour, declare `provides:[\"media-provider\"]` or `provides:[\"email.provider\"]`. The plugin manager indexes plugins by tag; the kernel routes core.media.* / SendEmail to the highest-priority active claimant. Hot-swappable — never wire kernel calls to a hard-coded extension slug."},
			{"id": "14", "title": "Migration ownership transfer when extracting a kernel feature.", "description": "If you're moving a feature out of the kernel (the historical case for email-manager, media-manager, seo-extension): (1) add the table to data_owned_tables; (2) copy the CREATE TABLE + ALTERs into your migrations/, rewriting as idempotent (IF NOT EXISTS, DO $$ END $$ blocks); (3) delete the kernel migration file (the runner skips entries already in schema_migrations); (4) move seeds in idempotent form (ON CONFLICT DO NOTHING); (5) drop kernel models / services / coreImpl fields. The CoreAPI surface stays — its impl now routes through events / provider tags. Verify by deactivating the extension and confirming the kernel still boots."},
			{"id": "15", "title": "Per-extension settings register with the schema-driven settings registry.", "description": "Use settings_schema in extension.json — the kernel's settings registry consumes it, applies the same secret-key heuristic for at-rest encryption, and renders the form via SDUI. Translatable fields use the per-locale composite-PK lookup with default-language fallback automatically. No per-extension settings UI required."},
		},
		"testing": map[string]string{
			"pattern":     "Use a FakeHost test double that implements coreapi.CoreAPI with in-memory maps. Inject it instead of a real gRPC client. Reference: extensions/forms/cmd/plugin/fakehost_test.go.",
			"why":         "Don't spin up a real Postgres for unit tests — that's e2e territory. Mock the CoreAPI interface, not the gRPC layer; tests stay fast, deterministic, and runnable on every save.",
			"example":     "p := &MyPlugin{host: &FakeHost{...}}; resp, _ := p.handleSubmit(ctx, &pb.PluginHTTPRequest{...}); assert on resp.",
			"build_flags": "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/<slug> ./cmd/plugin/ — required for the Alpine runtime image.",
		},
		"existing_extensions": map[string]string{
			"media-manager":     "gRPC + React + Tengo. Declares provides:['media-provider'] — kernel core.media.* routes through this plugin. Reference for list-page primitives, drawer, upload modal, URL state, image optimizer settings, presigned-upload integration.",
			"email-manager":     "gRPC + React. OWNS the email dispatcher (subscribes to *, matches admin-defined rules, renders templates, retains logs). Owns email_* tables (migrated out of core in commit 7e49268). Owns 'email-settings' slot that providers inject into.",
			"seo-extension":     "gRPC. Owns /robots.txt (with modern AI-crawler controls), site-wide SEO defaults, OG/Twitter head_meta. Was part of internal/cms/ until commit 7e49268.",
			"sitemap-generator": "gRPC + Tengo. Yoast-style sitemaps; rebuild on node.published / node.deleted.",
			"smtp-provider":     "gRPC. Declares provides:['email.provider']. Subscribes to email.send; injects settings into email-manager.",
			"resend-provider":   "gRPC. Declares provides:['email.provider']. Resend API delivery; injects settings into email-manager.",
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
			"data_owned_tables[] declares every SQL table the extension reads or writes via core.data.* (forms-extension declares forms / form_submissions / form_webhook_logs; media-manager declares media_files / media_image_sizes; email-manager declares email_*).",
			"If the extension serves a kernel-routed capability (media or email backend), `provides:[\"media-provider\"]` or `provides:[\"email.provider\"]` is declared and migrated tests verify hot-swap behaviour.",
			"Settings groups are registered via the kernel's settings registry (settings_schema in extension.json) so per-language storage and at-rest encryption Just Work.",
		},
		"authoritative_resource": "squilla://guidelines/extensions",
	}
}

// extensionStandardsEventModes — long-form event semantics. Verbose only.
func extensionStandardsEventModes() map[string]string {
	return map[string]string{
		"fire_and_forget":   "Subscribe via GetSubscriptions(); HandleEvent processes the payload and returns {Handled: true}. No one waits for the response. Use for analytics, notifications, side effects.",
		"event_with_result": "Templates can call {{ event \"my-ext:render\" (dict ...) }} and use whatever your plugin returns. Plugin returns {Handled: true, Result: bytes} where Result is HTML; the kernel concatenates Result from every subscriber in priority order and injects the combined string into the template. Reference: forms-extension's forms:render event.",
		"priority":          "Lower number = earlier dispatch. For event-with-result, all Result bytes are concatenated in priority order. For fire-and-forget, priority controls dispatch order but kernel doesn't aggregate.",
		"opt_out":           "Return {Handled: false} to let the next plugin in the priority chain handle the event. Return {Handled: true, Result: nil} to claim the event but contribute nothing.",
	}
}

// extensionStandardsSDUIReactivity — SSE → TanStack invalidation. Verbose only.
func extensionStandardsSDUIReactivity() []string {
	return []string{
		"Typed SSE events route to specific TanStack query keys via a central qk factory (qk.boot, qk.layout, qk.list, qk.entity, qk.settings).",
		"useAuth subscribes through an sse-bus so sidebar user-info refreshes on user.updated without page reload.",
		"CONFIRM action uses shadcn AlertDialog; CORE_API writes toast success/error with per-action overrides.",
	}
}

// extensionStandardsHotDeploy — admin_ui docker cp + plugin restart recipes.
// Verbose only.
func extensionStandardsHotDeploy() map[string]string {
	return map[string]string{
		"admin_ui":      "After `npm run build`, copy dist into the running container: `docker cp dist/. squilla-app-1:/app/extensions/<slug>/admin-ui/dist/`. The Go binary serves these as static files — no container restart needed. Hard-refresh (Cmd+Shift+R) to bypass cached index.html.",
		"plugin_binary": "After `go build`, `docker cp bin/<slug> squilla-app-1:/app/extensions/<slug>/bin/<slug>` then `docker compose restart app` (required to bounce the plugin process).",
	}
}

func onboardingGuide() string {
	return `# Squilla Onboarding for AI Agents

Welcome! Your task is to build or modify a Squilla theme or extension. To
succeed without human intervention, follow the path that matches your task.

## A. Building a Theme

### 1. Discovery
- **Read the Guide**: 'read_resource' on 'squilla://guidelines/themes'.
- **Tool**: Call 'core.theme.standards' for the structured ruleset.

### 2. Implementation
- **Schema First**: Define 'block.json' before 'view.html'.
- **Test Data**: Every field MUST have on-brand 'test_data'. NO LOREM IPSUM.
- **Editor Experience**: Every field MUST have 'help' text.
- **Portability**: Use 'theme-asset:<key>' for images, slugs for node refs.

### 3. Seeding ('theme.tengo')
- Register taxonomies BEFORE node types that use them.
- Use existence checks for idempotency: 'r := nodes.query({node_type, slug, limit:1})' then branch on 'r.total == 0'.
- Seed navigation via 'core/menus' — 'menus.upsert({slug, name, items:
  [{label, page:"<slug>"}]})'. The 'page:<slug>' form resolves to a NodeID
  so renaming the target page does NOT break the menu.
- Register '/.well-known/*' handlers via 'core/wellknown' if needed.

### 4. Verification
- Cross-reference with 'core.theme.standards'.
- Ensure exactly one layout in 'theme.json' has 'is_default: true' (typically 'layouts/default.html'); content nodes without an explicit layout use that one.
- Themes are autoregistered from 'themes/' on startup — no DB seeding.

## B. Building an Extension

### 1. Discovery
- **Read the Guide**: 'read_resource' on 'squilla://guidelines/extensions'.
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

Everything you need is in 'squilla://guidelines/themes' and
'squilla://guidelines/extensions'.`
}
