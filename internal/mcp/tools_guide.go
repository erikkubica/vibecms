package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"squilla/internal/coreapi"
)

// registerGuideTools exposes meta-tools that teach an AI client how to use the
// rest of the MCP surface. The goal is to collapse the cold-start problem: one
// call returns a goal→tool decision tree plus current CMS state, so the model
// does not burn 10 discovery calls before it does useful work.
func (s *Server) registerGuideTools() {
	s.addTool(mcp.NewTool("core.guide",
		mcp.WithDescription("META. Call this first when you're new to Squilla or unsure which tool to reach for. Returns a goal→tool decision tree (recipes for common journeys) plus a live snapshot of CMS state (active theme, counts, node types, recent nodes). Replaces ~10 discovery calls. Optional topic narrows the response: 'pages' | 'blocks' | 'themes' | 'taxonomies' | 'media' | 'extensions'."),
		mcp.WithString("topic", mcp.Description("Optional: narrow the response to one domain.")),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return s.buildGuide(ctx, stringArg(args, "topic"))
	})

	s.addTool(mcp.NewTool("core.theme.standards",
		mcp.WithDescription("Returns the official Squilla theme development standards. Use this to validate theme structure, block definitions (Rule 1.5), and field schemas (Rule 1.6). Always call this before creating or refactoring theme components."),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return themeStandards(), nil
	})

	s.addTool(mcp.NewTool("core.extension.standards",
		mcp.WithDescription("Returns the official Squilla extension development standards (manifest schema, capabilities, gRPC plugin lifecycle, admin-UI micro-frontend rules, list-page primitives, SDUI sidebar wiring, lifecycle events). Always call this before creating or refactoring an extension."),
	), "read", func(ctx context.Context, args map[string]any) (any, error) {
		return extensionStandards(), nil
	})
}

// buildGuide assembles the static decision tree alongside a cheap live snapshot
// of the CMS so the AI client has both "what can I do" and "what exists right
// now" in a single payload.
func (s *Server) buildGuide(ctx context.Context, topic string) (map[string]any, error) {
	out := map[string]any{
		"version":     "1",
		"namespace":   "core.<domain>.<verb>",
		"recipes":     guideRecipes(topic),
		"data_shapes": guideShapes(),
		"conventions": guideConventions(),
		"gotchas":     guideGotchas(),
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
			"notes": "term field values are objects {slug, name}, not bare strings. Terms are per-language (rows carry language_code; default = site's default language when omitted). To localize an existing term across languages call POST /admin/api/terms/<id>/translations with {language_code:'<code>'} — source and clone share a translation_group_id UUID. Slug uniqueness is per (node_type, taxonomy, slug, language_code), so the same slug can exist across languages.",
		},
		{
			"goal":  "Wire up an extension (media manager, email, etc.)",
			"topic": "extensions",
			"steps": []string{"core.extension.list", "core.extension.activate"},
			"notes": "Activation is hot: HotActivate spawns the plugin gRPC subprocess, runs migrations, loads scripts, registers blocks. No app/container restart. Dropping a new extension directory onto disk (docker cp, volume mount, git pull) is picked up automatically by the fs watcher; call core.extension.rescan from CI/ops if you want an explicit trigger. restart_required is always false today; the flag is reserved.",
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
			"notes": "ASYMMETRIES TO INTERNALIZE: (1) node level uses fields_data, blocks inside blocks_data use fields. (2) block.json uses key:, nodetypes.register uses name:. (3) select options are PLAIN STRINGS, never {value,label}. (4) term-typed fields need term_node_type in schema and store as {slug, name} object. (5) Settings keys keep their dots — templates use `index $s \"squilla.brand.version\"` or `mustSetting`. (6) Theme HTTP routes mount at /api/theme/<path>, not /<path>. (7) `dev_mode` global in seeds (true when SQUILLA_DEV_MODE=true) — branch to overwrite-on-reseed for fast iteration.",
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
			"notes": "Field shapes follow the standard field-types reference (text/number/toggle/image/select/repeater/...). Type mismatches after a saved value never auto-mutate the DB — render falls back to the field's declared default and the admin form surfaces a 'previous value' hint. Tengo callers need the `theme_settings:read` capability (themes bypass; extensions list it in extension.json). Bad pages are soft-failed at activation (logged & skipped). Run core.theme.checklist({slug}) to verify schema files parse before activating.",
		},
		{
			"goal":  "Deploy a theme that lives outside the primary repo",
			"topic": "themes",
			"steps": []string{"core.theme.standards", "core.theme.deploy", "core.render.node_preview"},
			"notes": "Build the theme directory locally, zip it (theme.json at root or one level deep), base64-encode the bytes, call core.theme.deploy({body_base64, activate:true}). The archive is unpacked into data/themes/<slug>/ (persistent volume — survives container restarts) via an atomic dir swap, the row is upserted, and — when activate=true — the theme is activated immediately. Image-bundled themes in themes/ are read-only; deploying a same-slug theme overrides the bundled copy on the next scan (data wins on collision). 50 MB cap. Slug must match [A-Za-z0-9_-]+. Re-deploying the same slug refreshes the existing row and overwrites files in place.",
		},
		{
			"goal":  "Deploy an extension from a local build (no docker cp, no git push)",
			"topic": "extensions",
			"steps": []string{"core.extension.standards", "core.extension.deploy", "core.extension.get"},
			"notes": "Zip the extension directory (extension.json + admin-ui/dist + scripts + bin/<plugin> if any), base64-encode, call core.extension.deploy({body_base64, activate:true}). The archive lands in data/extensions/<slug>/ (persistent volume); image-bundled extensions in extensions/ stay untouched. Plugin binaries declared in manifest.plugins[].binary are chmod'd to 0755 automatically. Pre-build them for the host OS/arch — Squilla does not cross-compile. activate=true runs HotActivate (migrations, plugin spawn, script load, block load) without a server restart. 50 MB cap.",
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
			"summary": "site_settings rows carry a language_code. Reads scope to the caller's locale (X-Admin-Language for admin, request locale for public render) and FALL BACK to the default-language row when no per-locale value exists. The legacy `''` shared sentinel is gone — every row has a real language code (migration 0040 backfilled). Theme settings inherit the same model; there is no `translatable` flag on the schema, every field is implicitly per-locale.",
		},
		{
			"topic":   "terms_per_language",
			"summary": "taxonomy_terms rows are per-language. Slug uniqueness is (node_type, taxonomy, slug, language_code). To create a translation of an existing term: POST /admin/api/terms/<id>/translations with {language_code:'<code>'} — source row gets a fresh translation_group_id UUID if it didn't have one, clone joins the same group. Routes /terms/:id/translations are registered before the generic /terms/:nodeType/:taxonomy because they share a 3-segment shape and Fiber matches by registration order.",
		},
		{
			"topic":   "themes_persistent_data_dir",
			"summary": "Themes and extensions live in two parallel dirs: image-bundled (themes/, extensions/, read-only) + operator-installed (data/themes/, data/extensions/, persistent volume). Both scanned at boot; data wins on slug collision. Writes (theme.deploy, extension.deploy, git install, zip upload) all target data/. Delete refuses to rmdir under the bundled root so a same-slug bundled theme stays available as fallback. docker-compose mounts ./data:/app/data; coolify-compose mounts the squilla-data named volume. Without this volume, theme.deploy unpacks into the container's writable layer and gets wiped on restart.",
		},
		{
			"topic":   "theme_activate_pre_flight",
			"summary": "core.theme.activate stat()s theme.json on disk before destroying the previous theme's registration. If the manifest is missing (e.g. files wiped from a non-persistent layer), Activate returns an error and the previous theme stays intact — preventing the 'reactivate to fix it' flow from wiping all blocks/layouts/templates with nothing to replace them.",
		},
		{
			"topic":   "lost_admin_password",
			"summary": "Recover via CLI: `docker exec -it <app> ./squilla reset-password <email> <new-password>`. Hashes via the same auth.HashPassword used at signup, writes directly to users.password_hash. Idempotent. Works without SMTP. Setting ADMIN_PASSWORD env var only takes effect during first-boot seed (when no admin user exists yet); it does NOT reset an existing user's password.",
		},
		{
			"topic":   "git_install_https_only",
			"summary": "core.theme.git_install (and InstallFromGit) only accepts https:// URLs. SSH-style git@github.com:owner/repo.git is rejected upfront with a clear message — the kernel SSH key would be overprivileged for cloning arbitrary themes. For private repos use a https URL + a personal access token in the token field.",
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
	}
}

// toolIndex returns a flat index of every currently-registered tool with its
// one-line description, captured at registration time by addTool.
func (s *Server) toolIndex() []toolCatalogEntry {
	return s.toolCatalog
}

func themeStandards() map[string]any {
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
			"layout":  "Sees full .node (id, title, slug, fields, blocks_html, taxonomies, ...), .app (head_styles, foot_scripts, settings, menus, current_lang, theme_url, ...), .user (logged_in, role, ...).",
			"partial": "Same as layout, plus .partial map populated from any partial-level field_schema.",
			"block":   "Sees ONLY the block's own field values at root, e.g. {{ .heading }}, {{ .items }}. Cannot reach .app or .node. To pull node data into a block, use {{ filter \"list_nodes\" ... }} or {{ filter \"get_node\" ... }}.",
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
		},
		"examples": map[string]string{
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
		"seeding_patterns": map[string]string{
			"registration":      "Always register taxonomies BEFORE node types that use them.",
			"idempotency":       "Use existence checks (e.g. nodes.query({node_type, slug, limit:1}) and branch on .total == 0) to avoid duplicate data on script re-runs. theme.tengo runs again on every restart while the theme is active.",
			"menus":             "Use core/menus → menus.upsert({slug, name, items:[{label, page:'<slug>'}]}). The page:<slug> form resolves to NodeID at upsert so slug renames don't break menus.",
			"wellknown":         "Use core/wellknown to register /.well-known/* handlers (e.g. apple-app-site-association). Unregistered paths return instant 404 via WellKnownRegistry, mounted before the public catch-all.",
			"assets_module":     "Use core/assets to read files from the calling theme/extension's own root: assets.read('forms/trip-order.html') / assets.exists('data/regions.json'). Returns a string, or an error value (wrap with is_error) if missing or path escapes root. Ideal for shipping form layouts, JSON fixtures, or default content as plain files instead of inlining multi-line strings in theme.tengo. Path is relative to the theme/extension dir; absolute paths and ../ traversal are rejected.",
			"forms_seeding":     "Theme-bundled forms: emit core/events 'forms:upsert' with {slug, name, fields, layout, settings, force?}. Idempotent on slug — without force, an existing same-slug form is left alone (admin edits stick); with force:true, theme overwrites on every reload. For theme-styled forms, ship the layout as themes/<theme>/forms/<slug>.html using forms-ext template syntax ({{.field_id.label}}, {{range .options}}…) and theme CSS classes; load it with core/assets.read and pass as the layout field. Form HTML is server-rendered via {{event \"forms:render\" (dict \"form_id\" \"<slug>\" \"hidden\" (dict ...))}} from a layout/block — hidden injects <input type=hidden> before </form> for per-page context (trip_slug, price). Public submit endpoint is /forms/submit/<slug>; the AJAX runtime is /extensions/<slug>/blocks/vibe-form/script.js.",
			"assets_hot_reload": "Theme assets are served via an atomic-pointer resolver that swaps on theme.activated. Runtime theme switches serve the new theme's assets instantly with no restart.",
			"autoregistration":  "Themes in themes/ are autoregistered at startup (mirroring the extension scan). No DB seeding is required to surface a new theme.",
			"layout_seeding":    "The core default layout is seeded with source='seed' (migration 0036). Themes are free to install their own base.html / blank.html without colliding.",
			"terms_module":      "Seed actual taxonomy terms (e.g. the 'Foodie' / 'Adventure' rows under trip_tag) via core/terms: terms.create({node_type, taxonomy, slug, name, description?, parent_id?, fields_data?}). Also exposes terms.list(node_type, taxonomy) / .get(id) / .update(id, updates) / .delete(id). The taxonomy DEFINITION is registered with core/taxonomies.register; the actual term ROWS go through core/terms.",
			"schema_field_keys": "Field-schema key naming differs by surface: in Tengo (core/nodetypes.register, core/taxonomies.register) the field key is `name`; in block.json `field_schema` it is `key`. Both go through the same field types. Don't mix — Tengo schemas with `key` (or block.json with `name`) silently produce empty schemas.",
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
		"event_modes": map[string]string{
			"fire_and_forget":   "Subscribe via GetSubscriptions(); HandleEvent processes the payload and returns {Handled: true}. No one waits for the response. Use for analytics, notifications, side effects.",
			"event_with_result": "Templates can call {{ event \"my-ext:render\" (dict ...) }} and use whatever your plugin returns. Plugin returns {Handled: true, Result: bytes} where Result is HTML; the kernel concatenates Result from every subscriber in priority order and injects the combined string into the template. Reference: forms-extension's forms:render event.",
			"priority":          "Lower number = earlier dispatch. For event-with-result, all Result bytes are concatenated in priority order. For fire-and-forget, priority controls dispatch order but kernel doesn't aggregate.",
			"opt_out":           "Return {Handled: false} to let the next plugin in the priority chain handle the event. Return {Handled: true, Result: nil} to claim the event but contribute nothing.",
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
		},
		"testing": map[string]string{
			"pattern":     "Use a FakeHost test double that implements coreapi.CoreAPI with in-memory maps. Inject it instead of a real gRPC client. Reference: extensions/forms/cmd/plugin/fakehost_test.go.",
			"why":         "Don't spin up a real Postgres for unit tests — that's e2e territory. Mock the CoreAPI interface, not the gRPC layer; tests stay fast, deterministic, and runnable on every save.",
			"example":     "p := &MyPlugin{host: &FakeHost{...}}; resp, _ := p.handleSubmit(ctx, &pb.PluginHTTPRequest{...}); assert on resp.",
			"build_flags": "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/<slug> ./cmd/plugin/ — required for the Alpine runtime image.",
		},
		"hot_deploy": map[string]string{
			"admin_ui":      "After `npm run build`, copy dist into the running container: `docker cp dist/. squilla-app-1:/app/extensions/<slug>/admin-ui/dist/`. The Go binary serves these as static files — no container restart needed. Hard-refresh (Cmd+Shift+R) to bypass cached index.html.",
			"plugin_binary": "After `go build`, `docker cp bin/<slug> squilla-app-1:/app/extensions/<slug>/bin/<slug>` then `docker compose restart app` (required to bounce the plugin process).",
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
		"authoritative_resource": "squilla://guidelines/extensions",
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
