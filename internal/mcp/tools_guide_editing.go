package mcp

import (
	"context"

	"squilla/internal/coreapi"
)

// editingPlaybook returns the structured authoring rulebook surfaced under
// core.guide.editing_playbook. It collects per-shape requirements (docs page
// fields_data, doc-* block schemas, SEO recommendations, layout selection
// rules) that AI authors otherwise reverse-engineer by querying live nodes.
func editingPlaybook() map[string]any {
	return map[string]any{
		"intent": "Reference data for authoring content via MCP. Pair with core.guide(topic='editing') recipes; pair with core.nodetype.get for the canonical schema of each node-type.",
		"docs_page": map[string]any{
			"node_type":   "documentation",
			"layout_slug": "docs (REQUIRED — node-type defaults are not auto-applied at write time).",
			"fields_data": map[string]string{
				"order":        "integer — sidebar sort order within the section, ascending.",
				"section.name": "string — display label for the sidebar group (e.g. 'Content Editing').",
				"section.slug": "string — kebab-case stable identifier; reuse existing slugs to keep grouping coherent.",
				"summary":      "string — one-paragraph TL;DR shown above the body and used as the meta_description fallback when seo_settings.meta_description is empty.",
			},
			"required_seo": []string{
				"seo_settings.meta_title (≤60 chars)",
				"seo_settings.meta_description (≤160 chars)",
				"excerpt (used in /docs index listings)",
			},
			"verification": "core.render.node_preview(id) → inspect HTML; or open response.full_url after status='published'.",
		},
		"docs_blocks": []map[string]any{
			{
				"slug":   "doc-content",
				"intent": "Plain prose body — paragraphs, headings, lists. Most docs pages use only this block.",
				"fields": map[string]string{
					"body": "HTML string. Trusted (no sanitization on render). Headings should start at h2 (the layout owns h1).",
				},
			},
			{
				"slug":   "doc-callout",
				"intent": "Aside / admonition. Use sparingly to flag tips, warnings, or notes.",
				"fields": map[string]string{
					"body":    "HTML string.",
					"label":   "string — short eyebrow above the body.",
					"variant": "enum: 'note' | 'tip' | 'heads-up' | 'danger'. Theme-specific; unknown values fall back to default styling.",
				},
			},
			{
				"slug":   "doc-codeblock",
				"intent": "Syntax-highlighted code window with optional traffic-lights and filename chrome.",
				"fields": map[string]string{
					"body":      "HTML string CONTAINING <pre><code>…</code></pre>. NOT a bare code string — the field name is `body` for symmetry with other blocks; treat it as HTML in / HTML out.",
					"file":      "string — optional filename shown in the chrome.",
					"language":  "string — 'go' | 'ts' | 'json' | 'bash' | … (theme highlighter map).",
					"show_dots": "boolean — render macOS traffic lights in the chrome.",
				},
			},
		},
		"shape_reminders": map[string]string{
			"top_level":      "Use fields_data on the node; use blocks_data:[{type, fields}] for content blocks. Misnaming silently drops data.",
			"featured_image": "Object {url, alt, width?, height?} — never a bare string and never an empty object {} (templates testing `{{ if .featured_image }}` would always pass).",
			"taxonomies":     "{<taxonomy>: [<term-slug>, …]} for real taxonomies. Term-typed fields go inside fields_data with shape {slug, name}.",
			"language_code":  "Always set explicitly (e.g. 'en'). Defaults to 'en' but per-language settings/terms only resolve correctly when this matches the user's locale.",
		},
		"verification_pattern": []string{
			"After every write: read response.warnings[] and resolve every entry before declaring success.",
			"After every write: call core.render.node_preview(id) — confirm the HTML is not empty and contains expected text.",
			"Before publishing UI claims in user docs: open the live admin (or playwright-cli) and inspect — never extrapolate from 'how every CMS works'.",
		},
		"open_gaps_known_to_kernel": []string{
			"No core.node.update_many — patch nodes one at a time when normalizing missing layout_slug across a node-type.",
			"node-type-level default layouts are not auto-applied; layout_slug must be set explicitly on each node.",
			"PNG compression level is not configurable in admin (image optimizer settings expose JPEG / WebP only).",
			"Media library uploads are images-only today; video / audio / document support is roadmap.",
			"Per-language admin switcher is in flux — verify reads/writes scope correctly when authoring across locales.",
			"Site-wide SEO settings page (/admin/settings/seo) is not implemented yet; per-node SEO is the only stable surface.",
			"Form builder is schema + custom HTML (no visual builder); see forms-extension docs.",
		},
	}
}

// collectNodeSections walks every node type and aggregates distinct
// fields_data.section.{slug,name} pairs across its nodes. Empty sections
// (node types whose schema doesn't use a section convention) are omitted.
// Returns map[node_type][]{slug, name}.
func (s *Server) collectNodeSections(ctx context.Context) (map[string][]map[string]string, error) {
	if s.deps.CoreAPI == nil {
		return nil, nil
	}
	types, err := s.deps.CoreAPI.ListNodeTypes(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]map[string]string{}
	for _, nt := range types {
		nodes, err := s.deps.CoreAPI.QueryNodes(ctx, coreapi.NodeQuery{NodeType: nt.Slug, Limit: 200})
		if err != nil || nodes == nil {
			continue
		}
		seen := map[string]map[string]string{}
		for _, n := range nodes.Nodes {
			if n == nil || n.FieldsData == nil {
				continue
			}
			raw, ok := n.FieldsData["section"]
			if !ok {
				continue
			}
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			slug, _ := m["slug"].(string)
			name, _ := m["name"].(string)
			if slug == "" {
				continue
			}
			if _, dup := seen[slug]; !dup {
				seen[slug] = map[string]string{"slug": slug, "name": name}
			}
		}
		if len(seen) == 0 {
			continue
		}
		entries := make([]map[string]string, 0, len(seen))
		for _, e := range seen {
			entries = append(entries, e)
		}
		out[nt.Slug] = entries
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
