package cms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"regexp"
	"strings"

	"squilla/internal/models"

	"github.com/gofiber/fiber/v2"
)

// This file owns the page-render flow: assembling the layout +
// partial-data + block-renderer chain that turns a (node, blocks)
// pair into final HTML, plus the bare 404 / error fallbacks. It's
// split out from public_handler.go because the layout-resolution
// logic is dense enough that keeping it adjacent to the request
// dispatch made the file unreadable.

// renderNodeWithLayout renders a content node through its resolved
// layout, building the per-partial data map (with node→layout_data
// fallbacks) and invoking the renderer with the page's block
// resolver. Returns (html, ok, err) — err only when the layout itself
// fails to render; ok=false with no error means we couldn't even
// resolve a layout (caller falls back to a barebones page).
func (h *PublicHandler) renderNodeWithLayout(c *fiber.Ctx, node *models.ContentNode, blocks []map[string]interface{}, renderedBlocks []string, user *models.User) (string, bool, error) {
	languages := h.loadActiveLanguages()
	var languageID *int
	var currentLang *models.Language

	for i := range languages {
		if languages[i].Code == node.LanguageCode {
			languageID = &languages[i].ID
			currentLang = &languages[i]
			break
		}
	}

	// Resolve layout for this node
	layout, err := h.layoutSvc.ResolveForNode(node, languageID)
	if err != nil || layout == nil {
		return "", false, fmt.Errorf("no layout resolved for node %q: %w", node.FullURL, err)
	}

	// Load site settings
	// Site settings are per-language (migration 0038). Resolve scoped to
	// the node's locale with default-language fallback so per-locale
	// overrides (e.g. seo_robots_index=false on a staging es subsite)
	// reach both the head_meta and the X-Robots-Tag header.
	settings := h.loadSiteSettingsForLocale(node.LanguageCode)
	c.Set("X-Robots-Tag", robotsDirective(settings))

	// Load menus
	menus := h.renderCtx.LoadMenus(languageID)

	// Combine rendered blocks into a single HTML string
	blocksHTML := strings.Join(renderedBlocks, "\n")

	// Build template data — only include CSS/JS for blocks used on this page
	usedSlugs := extractBlockSlugs(blocks)
	appData := h.renderCtx.BuildAppData(settings, languages, currentLang, usedSlugs)
	appData.Menus = menus

	nodeData := h.renderCtx.BuildNodeData(node, blocksHTML, languages)
	// Compose SEO meta tags into a single template.HTML string so the
	// theme's <head> can drop them in with one expression. Per-node SEO
	// wins; site_settings supply the fallbacks. See head_meta.go.
	appData.HeadMeta = BuildHeadMeta(node, nodeData.SEO, settings, nodeData.Translations, languages)

	templateData := TemplateData{
		App:           appData,
		Node:          nodeData,
		User:          buildUserData(user),
		ThemeSettings: h.loadThemeSettingsForRender(c.Context(), node.LanguageCode),
	}

	// Build block resolver
	blockResolver := func(slug string) (string, error) {
		lb, err := h.layoutBlockSvc.Resolve(slug, languageID)
		if err != nil {
			return "", err
		}
		return lb.TemplateCode, nil
	}

	// Build per-partial data from layout_data with fallback chain
	dataMap := templateData.ToMap()
	partialData := h.buildPartialData(node, layout, languageID, dataMap)

	// Render via the layout engine
	var buf bytes.Buffer
	if err := h.renderer.RenderLayout(&buf, layout.TemplateCode, dataMap, blockResolver, partialData); err != nil {
		renderErr := fmt.Errorf("layout %q render failed for %q: %w", layout.Slug, node.FullURL, err)
		log.Printf("WARN: %v", renderErr)
		return "", false, renderErr
	}

	return buf.String(), true, nil
}

// layoutErrorPage returns a user-friendly error page when layout rendering fails.
// This is always shown (not just in dev) — silently falling back to a bare
// "Squilla" branded page would be worse for the site owner's visitors.
func (h *PublicHandler) layoutErrorPage(err error, url string) string {
	siteName := "This site"
	if settings := h.loadSiteSettings(); settings != nil {
		if name, ok := settings["site_name"]; ok && name != "" {
			siteName = name
		}
	}
	// HTML-escape every interpolated value: `url` arrives from
	// c.Path() in the 404 path (untrusted), `err.Error()` can include
	// node-stored content via template parse errors, and even
	// `siteName` is admin-editable. Without escaping, a request to
	// /<script>alert(1)</script> would reflect script tags into the
	// 404 response — classic reflected XSS.
	safeSite := template.HTMLEscapeString(siteName)
	safeURL := template.HTMLEscapeString(url)
	safeErr := template.HTMLEscapeString(err.Error())
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Rendering Error – %s</title>
<style>
body{font-family:system-ui,-apple-system,sans-serif;margin:0;padding:0;background:#f8f9fa;color:#1a1a1a;display:flex;min-height:100vh;align-items:center;justify-content:center}
.card{background:#fff;border:1px solid #e5e7eb;border-radius:12px;max-width:640px;width:90%%;padding:2rem;box-shadow:0 4px 12px rgba(0,0,0,.06)}
h1{font-size:1.1rem;margin:0 0 .5rem;display:flex;align-items:center;gap:.5rem}
.badge{background:#fef2f2;color:#b91c1c;font-size:.7rem;font-weight:600;padding:3px 8px;border-radius:4px;text-transform:uppercase;letter-spacing:.03em}
p{margin:.5rem 0;color:#6b7280;font-size:.9rem;line-height:1.5}
.url{font-family:monospace;background:#f3f4f6;padding:2px 6px;border-radius:4px;font-size:.8rem}
details{margin-top:1.25rem;border-top:1px solid #f3f4f6;padding-top:1rem}
summary{cursor:pointer;font-size:.8rem;color:#9ca3af;font-weight:500}
pre{background:#fef2f2;border:1px solid #fca5a5;padding:.75rem 1rem;border-radius:8px;font-size:.75rem;white-space:pre-wrap;word-break:break-all;color:#991b1b;margin:.5rem 0 0;overflow:auto}
.footer{margin-top:1.25rem;font-size:.7rem;color:#d1d5db;text-align:center}
</style></head><body>
<div class="card">
<h1><span class="badge">Error</span>Page failed to render</h1>
<p>%s encountered a problem while rendering this page.</p>
<p>Page: <span class="url">%s</span></p>
<details><summary>Technical details</summary><pre>%s</pre></details>
<div class="footer">The site administrator has been notified.</div>
</div></body></html>`, safeSite, safeSite, safeURL, safeErr)
}

// buildPartialData constructs the per-partial field values map by:
// 1. Reading explicit values from node.layout_data[partial_slug]
// 2. Resolving default_from fallbacks from the template data context
func (h *PublicHandler) buildPartialData(node *models.ContentNode, layout *models.Layout, languageID *int, dataMap map[string]interface{}) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})

	// Parse node's layout_data
	var layoutData map[string]map[string]interface{}
	if len(node.LayoutData) > 0 {
		if err := json.Unmarshal(node.LayoutData, &layoutData); err != nil {
			log.Printf("WARN: failed to parse layout_data: %v", err)
			layoutData = make(map[string]map[string]interface{})
		}
	}
	if layoutData == nil {
		layoutData = make(map[string]map[string]interface{})
	}

	// Discover partials used by this layout
	partialSlugs := extractPartialSlugs(layout.TemplateCode)

	for _, slug := range partialSlugs {
		fields := make(map[string]interface{})

		// Get the partial's field_schema for default_from resolution
		lb, err := h.layoutBlockSvc.Resolve(slug, languageID)
		if err != nil {
			result[slug] = fields
			continue
		}

		var schema []map[string]interface{}
		if len(lb.FieldSchema) > 0 {
			json.Unmarshal(lb.FieldSchema, &schema)
		}

		// Get explicit values from layout_data
		explicit := layoutData[slug]
		if explicit == nil {
			explicit = make(map[string]interface{})
		}

		// For each field in schema, resolve value with fallback
		for _, fieldDef := range schema {
			fieldSlug, _ := fieldDef["key"].(string)
			if fieldSlug == "" {
				fieldSlug, _ = fieldDef["slug"].(string)
			}
			if fieldSlug == "" {
				continue
			}

			// 1. Explicit value from layout_data
			if val, ok := explicit[fieldSlug]; ok && val != nil && !isEmptyValue(val) {
				fields[fieldSlug] = val
				continue
			}

			// 2. default_from fallback
			if defaultFrom, ok := fieldDef["default_from"].(string); ok && defaultFrom != "" {
				if val := resolveDataPath(dataMap, defaultFrom); val != nil {
					fields[fieldSlug] = val
					continue
				}
			}

			// 3. Default value from schema
			if def, ok := fieldDef["default"]; ok {
				fields[fieldSlug] = def
				continue
			}

			// 4. Empty
			fields[fieldSlug] = nil
		}

		// Also include any extra explicit values not in schema
		for k, v := range explicit {
			if _, exists := fields[k]; !exists {
				fields[k] = v
			}
		}

		result[slug] = fields
	}

	return result
}

// extractPartialSlugs parses renderLayoutBlock calls from a layout template.
func extractPartialSlugs(templateCode string) []string {
	re := regexp.MustCompile(`renderLayoutBlock\s+"([^"]+)"`)
	matches := re.FindAllStringSubmatch(templateCode, -1)
	seen := make(map[string]bool)
	var slugs []string
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			seen[m[1]] = true
			slugs = append(slugs, m[1])
		}
	}
	return slugs
}

// isEmptyValue checks if a value is effectively empty (empty string, empty map, nil).
func isEmptyValue(val interface{}) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string:
		return v == ""
	case map[string]interface{}:
		return len(v) == 0
	}
	return false
}

// resolveDataPath resolves a dot-separated path like "node.title" from the data map.
func resolveDataPath(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = data
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = m[part]
		if !ok {
			return nil
		}
	}
	return current
}

// render404WithLayout renders a 404 page using the default layout.
func (h *PublicHandler) render404WithLayout(c *fiber.Ctx) (string, bool) {
	languages := h.loadActiveLanguages()
	var defaultLang *models.Language
	for i := range languages {
		if languages[i].IsDefault {
			defaultLang = &languages[i]
			break
		}
	}
	if defaultLang == nil {
		return "", false
	}
	defaultLangID := &defaultLang.ID

	// Prefer a theme-supplied 404/error layout, otherwise fall back to
	// the default. Themes opt in by adding `{ "slug": "404", "file":
	// "404.html" }` (or "error" for legacy compatibility) to their
	// theme.json layouts array — see LayoutService.Resolve404.
	layout, err := h.layoutSvc.Resolve404(defaultLangID)
	if err != nil {
		return "", false
	}

	settings := h.loadSiteSettings()
	menus := h.renderCtx.LoadMenus(defaultLangID)

	// Build 404 content
	notFoundHTML := `<div class="text-center py-24">
		<h1 class="text-6xl font-extrabold text-slate-200 mb-4">404</h1>
		<h2 class="text-2xl font-bold text-slate-700 mb-2">Page Not Found</h2>
		<p class="text-slate-500 mb-8">The page you are looking for does not exist or has been removed.</p>
		<a href="/" class="inline-flex items-center px-6 py-3 border border-transparent rounded-lg text-sm font-semibold text-white bg-indigo-600 hover:bg-indigo-700 transition-colors">Back to Home</a>
	</div>`

	appData := h.renderCtx.BuildAppData(settings, languages, defaultLang, []string{})
	appData.Menus = menus

	nodeData := NodeData{
		Title:        "Page Not Found",
		Slug:         "404",
		FullURL:      c.Path(),
		BlocksHTML:   template.HTML(notFoundHTML),
		Fields:       make(map[string]interface{}),
		SEO:          map[string]interface{}{"title": "Page Not Found"},
		NodeType:     "page",
		LanguageCode: defaultLang.Code,
	}
	// Synthesize a ContentNode for BuildHeadMeta so 404s emit the same
	// canonical/og/twitter scaffolding as real pages — no preview-tab
	// surprises, and shared infra browsers picking the page up still
	// see a sensible site_name. Force noindex on 404 regardless of
	// site-wide robots toggle (404s should never be indexed).
	notFoundSettings := mapClone(settings)
	notFoundSettings["seo_robots_index"] = "false"
	syntheticNode := &models.ContentNode{
		Title:        nodeData.Title,
		Slug:         nodeData.Slug,
		FullURL:      nodeData.FullURL,
		LanguageCode: nodeData.LanguageCode,
		NodeType:     nodeData.NodeType,
	}
	appData.HeadMeta = BuildHeadMeta(syntheticNode, nodeData.SEO, notFoundSettings, nil, languages)

	user := h.currentUser(c)
	templateData := TemplateData{
		App:           appData,
		Node:          nodeData,
		User:          buildUserData(user),
		ThemeSettings: h.loadThemeSettingsForRender(c.Context(), defaultLang.Code),
	}

	blockResolver := func(slug string) (string, error) {
		lb, err := h.layoutBlockSvc.Resolve(slug, defaultLangID)
		if err != nil {
			return "", err
		}
		return lb.TemplateCode, nil
	}

	var buf bytes.Buffer
	if err := h.renderer.RenderLayout(&buf, layout.TemplateCode, templateData.ToMap(), blockResolver); err != nil {
		log.Printf("WARN: 404 layout render failed: %v", err)
		return "", false
	}

	return buf.String(), true
}

// RenderWithLayout renders arbitrary HTML content inside the default site layout.
// Used by auth pages, error pages, etc. that need the full site chrome.
func (h *PublicHandler) RenderWithLayout(c *fiber.Ctx, title string, innerHTML template.HTML) (string, bool) {
	languages := h.loadActiveLanguages()
	var defaultLang *models.Language
	for i := range languages {
		if languages[i].IsDefault {
			defaultLang = &languages[i]
			break
		}
	}
	if defaultLang == nil {
		return "", false
	}
	defaultLangID := &defaultLang.ID

	// Find default layout — try language-specific first, then universal (NULL)
	layout, err := h.layoutSvc.ResolveDefault(defaultLangID)
	if err != nil {
		return "", false
	}

	settings := h.loadSiteSettings()
	menus := h.renderCtx.LoadMenus(defaultLangID)
	user := h.currentUser(c)

	appData := h.renderCtx.BuildAppData(settings, languages, defaultLang, []string{})
	appData.Menus = menus

	nodeData := NodeData{
		Title:        title,
		Slug:         "",
		FullURL:      c.Path(),
		BlocksHTML:   innerHTML,
		Fields:       make(map[string]interface{}),
		SEO:          map[string]interface{}{"title": title},
		NodeType:     "page",
		LanguageCode: defaultLang.Code,
	}
	// Synthesize a node for the head_meta builder. RenderWithLayout
	// covers auth pages and operator-facing chrome that should still
	// emit canonical + og/twitter scaffolding so social previews and
	// search engines see something coherent on link sharing.
	syntheticNode := &models.ContentNode{
		Title:        nodeData.Title,
		FullURL:      nodeData.FullURL,
		LanguageCode: nodeData.LanguageCode,
		NodeType:     nodeData.NodeType,
	}
	appData.HeadMeta = BuildHeadMeta(syntheticNode, nodeData.SEO, settings, nil, languages)

	templateData := TemplateData{
		App:           appData,
		Node:          nodeData,
		User:          buildUserData(user),
		ThemeSettings: h.loadThemeSettingsForRender(c.Context(), nodeData.LanguageCode),
	}

	blockResolver := func(slug string) (string, error) {
		lb, err := h.layoutBlockSvc.Resolve(slug, defaultLangID)
		if err != nil {
			return "", err
		}
		return lb.TemplateCode, nil
	}

	var buf bytes.Buffer
	if err := h.renderer.RenderLayout(&buf, layout.TemplateCode, templateData.ToMap(), blockResolver); err != nil {
		log.Printf("WARN: layout render failed for page %q: %v", title, err)
		return "", false
	}

	return buf.String(), true
}
