package cms

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"vibecms/internal/auth"
	"vibecms/internal/events"
	"vibecms/internal/models"
	"vibecms/internal/rendering"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// PageData holds all data passed to public page templates.
type PageData struct {
	Title          string
	User           *models.User
	Node           *models.ContentNode
	Nodes          []models.ContentNode
	Blocks         []map[string]interface{}
	RenderedBlocks []string
	FlashMsg       string
	FlashType      string // "success" or "error"
}

// PublicHandler serves the public-facing HTML pages.
type PublicHandler struct {
	db             *gorm.DB
	renderer       *rendering.TemplateRenderer
	sessions       *auth.SessionService
	layoutSvc      *LayoutService
	layoutBlockSvc *LayoutBlockService
	menuSvc        *MenuService
	renderCtx      *RenderContext
	eventBus       *events.EventBus

	cacheMu          sync.RWMutex
	siteSettings     map[string]string
	blockTypes       map[string]models.BlockType
	themeBlockCache  map[string]string
	activeLanguages  []models.Language
	blockOutputCache map[string]string
}

// NewPublicHandler creates a new PublicHandler.
func NewPublicHandler(
	db *gorm.DB,
	renderer *rendering.TemplateRenderer,
	sessions *auth.SessionService,
	layoutSvc *LayoutService,
	layoutBlockSvc *LayoutBlockService,
	menuSvc *MenuService,
	renderCtx *RenderContext,
	eventBus *events.EventBus,
) *PublicHandler {
	h := &PublicHandler{
		db:             db,
		renderer:       renderer,
		sessions:       sessions,
		layoutSvc:      layoutSvc,
		layoutBlockSvc: layoutBlockSvc,
		menuSvc:        menuSvc,
		renderCtx:      renderCtx,
		eventBus:       eventBus,
	}
	h.ClearCache()

	if eventBus != nil {
		eventBus.SubscribeAll(func(event string, payload events.Payload) {
			if strings.HasPrefix(event, "theme.") || strings.HasPrefix(event, "setting.") || strings.HasPrefix(event, "block_type.") || strings.HasPrefix(event, "language.") || strings.HasPrefix(event, "layout") {
				h.ClearCache()
			}
		})
	}

	return h
}

// ClearCache clears the in-memory caches.
func (h *PublicHandler) ClearCache() {
	h.renderer.ClearCache()

	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()
	h.siteSettings = nil
	h.blockTypes = nil
	h.activeLanguages = nil
	h.themeBlockCache = make(map[string]string)
	h.blockOutputCache = make(map[string]string)
}

// CacheStats returns statistics about the current cache state.
func (h *PublicHandler) CacheStats() map[string]interface{} {
	h.cacheMu.RLock()
	defer h.cacheMu.RUnlock()

	settingsCached := h.siteSettings != nil
	blockTypesCached := h.blockTypes != nil
	langsCached := h.activeLanguages != nil

	return map[string]interface{}{
		"site_settings":     settingsCached,
		"block_types":       blockTypesCached,
		"active_languages":  langsCached,
		"theme_block_files": len(h.themeBlockCache),
		"block_output":      len(h.blockOutputCache),
	}
}

// RegisterRoutes registers public page routes on the Fiber app.
func (h *PublicHandler) RegisterRoutes(app *fiber.App) {
	app.Get("/", h.HomePage)
	// Catch-all for public pages: match by full_url stored in DB
	app.Get("/*", h.PageByFullURL)
}

// HomePage renders the public homepage. If a homepage node is configured in
// site_settings, that node is rendered. Otherwise, recent published content is shown.
func (h *PublicHandler) HomePage(c *fiber.Ctx) error {
	user := h.currentUser(c)

	// Check if a homepage node is configured
	settings := h.loadSiteSettings()
	if val, ok := settings["homepage_node_id"]; ok && val != "" {
		if homepageID, err := strconv.Atoi(val); err == nil && homepageID > 0 {
			var node models.ContentNode
			if err := h.db.Where("id = ? AND status = ?", homepageID, "published").First(&node).Error; err == nil {
				blocks := parseBlocks(node.BlocksData)
				renderedBlocks := h.renderBlocksBatch(blocks)

				// Try layout-based rendering first
				if html, ok := h.renderNodeWithLayout(c, &node, blocks, renderedBlocks, user); ok {
					c.Set("Content-Type", "text/html; charset=utf-8")
					return c.SendString(html)
				}

				// Fall back to file-based template rendering
				data := PageData{
					Title:          node.Title + " - VibeCMS",
					User:           user,
					Node:           &node,
					Blocks:         blocks,
					RenderedBlocks: renderedBlocks,
				}
				c.Set("Content-Type", "text/html; charset=utf-8")
				return h.renderer.RenderPage(c, "public/page.html", data)
			}
		}
	}

	// Fallback: show recent published content
	var nodes []models.ContentNode
	h.db.Where("status = ? AND deleted_at IS NULL", "published").
		Order("published_at DESC").
		Limit(9).
		Find(&nodes)

	data := PageData{
		Title: "VibeCMS - High-Performance AI-Native CMS",
		User:  user,
		Nodes: nodes,
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.RenderPage(c, "public/home.html", data)
}

// PageByFullURL looks up a content node by matching the request path against full_url.
// This handles all URL patterns: /{lang-slug}/{slug}, /{lang-slug}/{type-prefix}/{slug}, etc.
// For languages with hide_prefix, both /test and /en/test resolve to the same page.
func (h *PublicHandler) PageByFullURL(c *fiber.Ctx) error {
	user := h.currentUser(c)
	path := "/" + c.Params("*")

	// Strip trailing slash for consistency (but keep "/" as-is)
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	node, found := h.findNodeByURL(path)

	if !found {
		// Check if path is a language slug (e.g. /de) — serve that language's homepage
		node, found = h.findLanguageHomepage(path)
	}

	if !found {
		// Try alternate forms for hide_prefix languages:
		// 1. If path looks like /en/slug, try /slug (hide_prefix match)
		// 2. If path looks like /slug, try /{lang-slug}/slug for each hide_prefix language
		node, found = h.findNodeWithPrefixFallback(path)
	}

	if !found {
		// Try rendering 404 with the default layout
		if html, ok := h.render404WithLayout(c); ok {
			c.Set("Content-Type", "text/html; charset=utf-8")
			c.Status(fiber.StatusNotFound)
			return c.SendString(html)
		}
		// Fall back to file-based 404
		data := PageData{
			Title: "Page Not Found - VibeCMS",
			User:  user,
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		c.Status(fiber.StatusNotFound)
		return h.renderer.RenderPage(c, "public/page.html", data)
	}

	blocks := parseBlocks(node.BlocksData)
	renderedBlocks := h.renderBlocksBatch(blocks)

	// Try layout-based rendering first
	if html, ok := h.renderNodeWithLayout(c, node, blocks, renderedBlocks, user); ok {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}

	// Fall back to file-based template rendering
	data := PageData{
		Title:          node.Title + " - VibeCMS",
		User:           user,
		Node:           node,
		Blocks:         blocks,
		RenderedBlocks: renderedBlocks,
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.RenderPage(c, "public/page.html", data)
}

// renderNodeWithLayout attempts to render a content node using the layout system.
// If a layout is resolved, it returns the fully rendered HTML and true.
// If no layout is found or an error occurs, it returns "" and false so the
// caller can fall back to the legacy file-based rendering.
func (h *PublicHandler) renderNodeWithLayout(c *fiber.Ctx, node *models.ContentNode, blocks []map[string]interface{}, renderedBlocks []string, user *models.User) (string, bool) {
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
		return "", false
	}

	// Load site settings
	settings := h.loadSiteSettings()

	// Load menus
	menus := h.renderCtx.LoadMenus(languageID)

	// Combine rendered blocks into a single HTML string
	blocksHTML := strings.Join(renderedBlocks, "\n")

	// Build template data — only include CSS/JS for blocks used on this page
	usedSlugs := extractBlockSlugs(blocks)
	appData := h.renderCtx.BuildAppData(settings, languages, currentLang, usedSlugs)
	appData.Menus = menus

	nodeData := h.renderCtx.BuildNodeData(node, blocksHTML, languages)

	templateData := TemplateData{
		App:  appData,
		Node: nodeData,
		User: buildUserData(user),
	}

	// Build block resolver
	blockResolver := func(slug string) (string, error) {
		lb, err := h.layoutBlockSvc.Resolve(slug, languageID)
		if err != nil {
			return "", err
		}
		return lb.TemplateCode, nil
	}

	// Render via the layout engine
	var buf bytes.Buffer
	if err := h.renderer.RenderLayout(&buf, layout.TemplateCode, templateData.ToMap(), blockResolver); err != nil {
		log.Printf("WARN: layout render failed, falling back: %v", err)
		return "", false
	}

	return buf.String(), true
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

	// Find default layout — try language-specific first, then universal (NULL)
	layout, err := h.layoutSvc.ResolveDefault(defaultLangID)
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

	user := h.currentUser(c)
	templateData := TemplateData{App: appData, Node: nodeData, User: buildUserData(user)}

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
		Title:      title,
		Slug:       "",
		FullURL:    c.Path(),
		BlocksHTML: innerHTML,
		Fields:     make(map[string]interface{}),
		SEO:        map[string]interface{}{"title": title},
		NodeType:   "page",
	}

	templateData := TemplateData{App: appData, Node: nodeData, User: buildUserData(user)}

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

// loadSiteSettings loads all site settings into a map keyed by setting key.
func (h *PublicHandler) loadSiteSettings() map[string]string {
	h.cacheMu.RLock()
	settings := h.siteSettings
	h.cacheMu.RUnlock()

	if settings != nil {
		return settings
	}

	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()
	if h.siteSettings != nil {
		return h.siteSettings
	}

	settings = make(map[string]string)
	var allSettings []models.SiteSetting
	h.db.Find(&allSettings)
	for _, s := range allSettings {
		if s.Value != nil {
			settings[s.Key] = *s.Value
		}
	}

	h.siteSettings = settings
	return settings
}

// loadActiveLanguages loads all active languages as a slice.
func (h *PublicHandler) loadActiveLanguages() []models.Language {
	h.cacheMu.RLock()
	languages := h.activeLanguages
	h.cacheMu.RUnlock()

	if languages != nil {
		return languages
	}

	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()
	if h.activeLanguages != nil {
		return h.activeLanguages
	}

	var langs []models.Language
	h.db.Where("is_active = ?", true).Order("sort_order ASC").Find(&langs)

	h.activeLanguages = langs
	return langs
}

func (h *PublicHandler) getBlockType(slug string) (models.BlockType, bool) {
	h.cacheMu.RLock()
	blocks := h.blockTypes
	h.cacheMu.RUnlock()

	if blocks == nil {
		h.cacheMu.Lock()
		if h.blockTypes == nil {
			var dt []models.BlockType
			h.db.Find(&dt)
			blocks = make(map[string]models.BlockType)
			for _, b := range dt {
				blocks[b.Slug] = b
			}
			h.blockTypes = blocks
		} else {
			blocks = h.blockTypes
		}
		h.cacheMu.Unlock()
	}

	bt, ok := blocks[slug]
	return bt, ok
}

// renderBlocks renders each block's HTML template with its field values.
func (h *PublicHandler) renderBlocks(blocks []map[string]interface{}) []string {
	var rendered []string
	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		fields, _ := block["fields"].(map[string]interface{})
		if fields == nil {
			fields = block // fallback for old format
		}

		bt, ok := h.getBlockType(blockType)
		if !ok || bt.HTMLTemplate == "" {
			// Fallback: render raw JSON debug block
			jsonBytes, _ := json.MarshalIndent(fields, "", "  ")
			rendered = append(rendered, fmt.Sprintf(
				`<div class="mb-8 bg-slate-50 rounded-xl p-6 border border-slate-200">
					<div class="flex items-center gap-2 mb-3">
						<span class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-semibold bg-slate-200 text-slate-700">%s</span>
						<span class="text-xs text-slate-400">No template defined</span>
					</div>
					<pre class="text-sm text-slate-600 overflow-x-auto bg-white rounded-lg p-4 border border-slate-200"><code>%s</code></pre>
				</div>`, blockType, string(jsonBytes)))
			continue
		}

		// Check for theme file override with caching
		tmplContent := bt.HTMLTemplate
		themeFile := fmt.Sprintf("themes/default/blocks/%s.html", blockType)
		
		h.cacheMu.RLock()
		cachedContent, hasCache := h.themeBlockCache[themeFile]
		h.cacheMu.RUnlock()

		if hasCache {
			if cachedContent != "" {
				tmplContent = cachedContent
			}
		} else {
			if fileContent, err := os.ReadFile(themeFile); err == nil {
				tmplContent = string(fileContent)
				h.cacheMu.Lock()
				h.themeBlockCache[themeFile] = tmplContent
				h.cacheMu.Unlock()
			} else {
				h.cacheMu.Lock()
				h.themeBlockCache[themeFile] = "" // Cache the miss
				h.cacheMu.Unlock()
			}
		}

		// Hydrate node references — resolve node selector fields to full node data
		markRichTextFields(fields, bt.FieldSchema)
		
		// Use the new RenderParsed method for cached template execution
		cacheKey := "block:" + blockType + ":" + tmplContent
		var buf bytes.Buffer
		err := h.renderer.RenderParsed(&buf, cacheKey, tmplContent, fields, template.FuncMap{
			"safeHTML": func(s interface{}) template.HTML {
				return template.HTML(fmt.Sprintf("%v", s))
			},
			"safeURL": func(s interface{}) template.URL {
				return template.URL(fmt.Sprintf("%v", s))
			},
		})
		if err != nil {
			rendered = append(rendered, fmt.Sprintf(`<div class="mb-4 text-red-500 text-sm">Template error in %s: %v</div>`, blockType, err))
			continue
		}

		rendered = append(rendered, buf.String())
	}
	return rendered
}

// renderBlocksBatch is the optimized version that performs batch node hydration.
func (h *PublicHandler) renderBlocksBatch(blocks []map[string]interface{}) []string {
	var rendered []string
	if len(blocks) == 0 {
		return rendered
	}

	// Step 1: Collect all node IDs across all blocks
	allNodeIDs := make(map[int]bool)
	for _, block := range blocks {
		fields, _ := block["fields"].(map[string]interface{})
		if fields == nil {
			fields = block
		}
		collectNodeIDs(fields, allNodeIDs)
	}

	// Step 2: Batch fetch nodes if any IDs were found
	nodeMap := make(map[int]map[string]interface{})
	if len(allNodeIDs) > 0 {
		var ids []int
		for id := range allNodeIDs {
			ids = append(ids, id)
		}
		var nodes []models.ContentNode
		if err := h.db.Where("id IN ?", ids).Find(&nodes).Error; err == nil {
			// Get node types for schema info
			typeSlugs := make(map[string]bool)
			for _, n := range nodes {
				if n.NodeType != "" {
					typeSlugs[n.NodeType] = true
				}
			}
			var slugs []string
			for s := range typeSlugs {
				slugs = append(slugs, s)
			}
			var nodeTypes []models.NodeType
			if len(slugs) > 0 {
				h.db.Where("slug IN ?", slugs).Find(&nodeTypes)
			}
			typeSchemaMap := make(map[string]models.JSONB)
			for _, nt := range nodeTypes {
				typeSchemaMap[nt.Slug] = nt.FieldSchema
			}

			for _, n := range nodes {
				fields := make(map[string]interface{})
				json.Unmarshal(n.FieldsData, &fields)
				schema := typeSchemaMap[n.NodeType]
				markRichTextFields(fields, schema)
				nodeMap[n.ID] = map[string]interface{}{
					"id":            n.ID,
					"title":         n.Title,
					"slug":          n.Slug,
					"full_url":      n.FullURL,
					"fields":        fields,
					"node_type":     n.NodeType,
					"language_code": n.LanguageCode,
					"status":        n.Status,
				}
			}
		}
	}

	// Step 3: Render each block with pre-hydrated nodeMap
	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		fields, _ := block["fields"].(map[string]interface{})
		if fields == nil {
			fields = block
		}

		bt, ok := h.getBlockType(blockType)
		if !ok || bt.HTMLTemplate == "" {
			jsonBytes, _ := json.MarshalIndent(fields, "", "  ")
			rendered = append(rendered, fmt.Sprintf(`<div class="mb-8 bg-slate-50 rounded-xl p-6 border border-slate-200">
				<div class="flex items-center gap-2 mb-3">
					<span class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-semibold bg-slate-200 text-slate-700">%s</span>
				</div>
				<pre class="text-sm text-slate-600 bg-white rounded-lg p-4 border border-slate-200"><code>%s</code></pre>
			</div>`, blockType, string(jsonBytes)))
			continue
		}

		// Apply batch-hydrated nodes
		applyHydratedNodes(fields, nodeMap)

		// Check block output cache (only for blocks with cache_output enabled)
		if bt.CacheOutput {
			outputKey := blockOutputKey(blockType, fields)
			h.cacheMu.RLock()
			cached, hit := h.blockOutputCache[outputKey]
			h.cacheMu.RUnlock()
			if hit {
				rendered = append(rendered, cached)
				continue
			}
		}

		tmplContent := bt.HTMLTemplate
		themeFile := fmt.Sprintf("themes/default/blocks/%s.html", blockType)
		h.cacheMu.RLock()
		cachedContent, hasCache := h.themeBlockCache[themeFile]
		h.cacheMu.RUnlock()
		if hasCache && cachedContent != "" {
			tmplContent = cachedContent
		} else if !hasCache {
			if fileContent, err := os.ReadFile(themeFile); err == nil {
				tmplContent = string(fileContent)
				h.cacheMu.Lock()
				h.themeBlockCache[themeFile] = tmplContent
				h.cacheMu.Unlock()
			} else {
				h.cacheMu.Lock()
				h.themeBlockCache[themeFile] = ""
				h.cacheMu.Unlock()
			}
		}

		markRichTextFields(fields, bt.FieldSchema)

		tmplCacheKey := "block:" + blockType + ":" + tmplContent
		var buf bytes.Buffer
		err := h.renderer.RenderParsed(&buf, tmplCacheKey, tmplContent, fields, template.FuncMap{
			"safeHTML": func(s interface{}) template.HTML {
				return template.HTML(fmt.Sprintf("%v", s))
			},
			"safeURL": func(s interface{}) template.URL {
				return template.URL(fmt.Sprintf("%v", s))
			},
		})
		if err != nil {
			log.Printf("WARN: block template render error [%s]: %v", blockType, err)
			continue
		}

		output := buf.String()

		// Store in block output cache if enabled
		if bt.CacheOutput {
			outputKey := blockOutputKey(blockType, fields)
			h.cacheMu.Lock()
			h.blockOutputCache[outputKey] = output
			h.cacheMu.Unlock()
		}

		rendered = append(rendered, output)
	}
	return rendered
}

// hydrateFields walks through field values and hydrates node references.
// Note: Preferred way is to use renderBlocksBatch for superior performance.
func (h *PublicHandler) hydrateFields(fields map[string]interface{}) {
	nodeIDs := make(map[int]bool)
	collectNodeIDs(fields, nodeIDs)
	if len(nodeIDs) == 0 {
		return
	}

	var ids []int
	for id := range nodeIDs {
		ids = append(ids, id)
	}

	var nodes []models.ContentNode
	if err := h.db.Where("id IN ?", ids).Find(&nodes).Error; err != nil {
		return
	}

	nodeTypeSlugs := make(map[string]bool)
	for _, n := range nodes {
		if n.NodeType != "" {
			nodeTypeSlugs[n.NodeType] = true
		}
	}
	var types []string
	for nt := range nodeTypeSlugs {
		types = append(types, nt)
	}

	var nodeTypes []models.NodeType
	if len(types) > 0 {
		h.db.Where("slug IN ?", types).Find(&nodeTypes)
	}
	typeSchemaMap := make(map[string]models.JSONB)
	for _, nt := range nodeTypes {
		typeSchemaMap[nt.Slug] = nt.FieldSchema
	}

	nodeMap := make(map[int]map[string]interface{})
	for _, node := range nodes {
		hydrated := map[string]interface{}{
			"id":            node.ID,
			"uuid":          node.UUID,
			"title":         node.Title,
			"slug":          node.Slug,
			"full_url":      node.FullURL,
			"node_type":     node.NodeType,
			"status":        node.Status,
			"language_code": node.LanguageCode,
			"version":       node.Version,
			"created_at":    node.CreatedAt,
			"updated_at":    node.UpdatedAt,
		}
		if node.PublishedAt != nil {
			hydrated["published_at"] = *node.PublishedAt
		}

		if len(node.FieldsData) > 0 {
			var fieldsData map[string]interface{}
			if err := json.Unmarshal([]byte(node.FieldsData), &fieldsData); err == nil {
				if schema, ok := typeSchemaMap[node.NodeType]; ok {
					markRichTextFields(fieldsData, schema)
				}
				for k, v := range fieldsData {
					hydrated[k] = v
				}
			}
		}

		if len(node.BlocksData) > 0 {
			var blocksData []map[string]interface{}
			if err := json.Unmarshal([]byte(node.BlocksData), &blocksData); err == nil {
				hydrated["blocks"] = blocksData
			}
		}

		nodeMap[int(node.ID)] = hydrated
	}

	applyHydratedNodes(fields, nodeMap)
}

// fieldSchemaDef represents a field definition from the block type's field_schema.
type fieldSchemaDef struct {
	Key       string           `json:"key"`
	Type      string           `json:"type"`
	SubFields []fieldSchemaDef `json:"sub_fields"`
}

// markRichTextFields walks field values and converts richtext/textarea HTML strings
// to template.HTML so Go's html/template does not escape them.
// It uses the block type's field_schema to identify which fields are richtext.
func markRichTextFields(fields map[string]interface{}, schema models.JSONB) {
	var defs []fieldSchemaDef
	if err := json.Unmarshal(schema, &defs); err != nil {
		return
	}
	applyRichTextMarking(fields, defs)
}

// applyRichTextMarking recursively walks fields and marks richtext values as template.HTML.
func applyRichTextMarking(fields map[string]interface{}, defs []fieldSchemaDef) {
	for _, def := range defs {
		val, ok := fields[def.Key]
		if !ok || val == nil {
			continue
		}
		switch def.Type {
		case "richtext":
			// Convert string HTML to template.HTML to prevent escaping
			if s, ok := val.(string); ok {
				fields[def.Key] = template.HTML(s)
			}
		case "image":
			// Normalize image fields to always be a map with at least "url".
			// Media picker stores full objects (url, alt, width, height, etc.).
			// Plain strings (from test_data) get wrapped as {"url": "..."}.
			// Templates access via .field.url, .field.alt, .field.width, etc.
			switch v := val.(type) {
			case string:
				if v != "" {
					fields[def.Key] = map[string]interface{}{"url": v}
				}
			case map[string]interface{}:
				// Already a proper object — leave as-is.
			}
		case "link":
			// Normalize link fields to always be a map with text, url, alt, target.
			// Templates access via .field.url, .field.text, .field.target, etc.
			switch v := val.(type) {
			case string:
				if v != "" {
					fields[def.Key] = map[string]interface{}{"url": v, "text": v}
				}
			case map[string]interface{}:
				// Already a proper object — leave as-is.
			}
		case "group":
			// Recurse into group sub-fields
			if m, ok := val.(map[string]interface{}); ok && len(def.SubFields) > 0 {
				applyRichTextMarking(m, def.SubFields)
			}
		case "repeater":
			// Recurse into each repeater row
			if arr, ok := val.([]interface{}); ok && len(def.SubFields) > 0 {
				for _, item := range arr {
					if m, ok := item.(map[string]interface{}); ok {
						applyRichTextMarking(m, def.SubFields)
					}
				}
			}
		}
	}
}

func parseNodeID(idVal interface{}) int {
	switch id := idVal.(type) {
	case float64:
		return int(id)
	case int:
		return id
	case json.Number:
		n, _ := id.Int64()
		return int(n)
	}
	return 0
}

func collectNodeIDs(fields map[string]interface{}, nodeIDs map[int]bool) {
	for _, val := range fields {
		switch v := val.(type) {
		case map[string]interface{}:
			if idVal, hasID := v["id"]; hasID {
				if id := parseNodeID(idVal); id > 0 {
					nodeIDs[id] = true
				}
			}
		case []interface{}:
			if len(v) > 0 {
				if first, ok := v[0].(map[string]interface{}); ok {
					if _, hasID := first["id"]; hasID {
						for _, item := range v {
							if m, ok := item.(map[string]interface{}); ok {
								if idVal, hasID := m["id"]; hasID {
									if id := parseNodeID(idVal); id > 0 {
										nodeIDs[id] = true
									}
								}
							}
						}
					} else {
						for _, item := range v {
							if m, ok := item.(map[string]interface{}); ok {
								collectNodeIDs(m, nodeIDs)
							}
						}
					}
				}
			}
		}
	}
}

func applyHydratedNodes(fields map[string]interface{}, nodeMap map[int]map[string]interface{}) {
	for key, val := range fields {
		switch v := val.(type) {
		case map[string]interface{}:
			if idVal, hasID := v["id"]; hasID {
				if id := parseNodeID(idVal); id > 0 {
					if hydrated, ok := nodeMap[id]; ok {
						fields[key] = hydrated
					}
				}
			}
		case []interface{}:
			if len(v) > 0 {
				if first, ok := v[0].(map[string]interface{}); ok {
					if _, hasID := first["id"]; hasID {
						for i, item := range v {
							if m, ok := item.(map[string]interface{}); ok {
								if idVal, hasID := m["id"]; hasID {
									if id := parseNodeID(idVal); id > 0 {
										if hydrated, ok := nodeMap[id]; ok {
											v[i] = hydrated
										}
									}
								}
							}
						}
						fields[key] = v
					} else {
						for _, item := range v {
							if m, ok := item.(map[string]interface{}); ok {
								applyHydratedNodes(m, nodeMap)
							}
						}
					}
				}
			}
		}
	}
}

// blockOutputKey generates a cache key for a rendered block based on its type and field content.
func blockOutputKey(blockType string, fields map[string]interface{}) string {
	b, _ := json.Marshal(fields)
	h := sha256.Sum256(b)
	return blockType + ":" + hex.EncodeToString(h[:16])
}

// findNodeByURL does a direct full_url lookup.
// findLanguageHomepage checks if the path is just a language slug (e.g. /de)
// and returns the homepage translation for that language.
func (h *PublicHandler) findLanguageHomepage(path string) (*models.ContentNode, bool) {
	// Path must be /{something} with no further segments
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) != 1 || segments[0] == "" {
		return nil, false
	}
	langSlug := segments[0]

	// Check if this matches any active language slug
	var lang models.Language
	foundLang := false
	for _, l := range h.loadActiveLanguages() {
		if l.Slug == langSlug {
			lang = l
			foundLang = true
			break
		}
	}
	if !foundLang {
		return nil, false
	}

	// Find the configured homepage
	settings := h.loadSiteSettings()
	val, ok := settings["homepage_node_id"]
	if !ok || val == "" {
		return nil, false
	}
	homepageID, err := strconv.Atoi(val)
	if err != nil || homepageID <= 0 {
		return nil, false
	}

	var homepage models.ContentNode
	if err := h.db.First(&homepage, homepageID).Error; err != nil {
		return nil, false
	}

	// If homepage language matches, return it directly
	if homepage.LanguageCode == lang.Code {
		if homepage.Status == "published" {
			return &homepage, true
		}
		return nil, false
	}

	// Find translation of homepage in the target language
	if homepage.TranslationGroupID == nil || *homepage.TranslationGroupID == "" {
		return nil, false
	}
	var translated models.ContentNode
	if err := h.db.Where("translation_group_id = ? AND language_code = ? AND status = ? AND deleted_at IS NULL",
		*homepage.TranslationGroupID, lang.Code, "published").First(&translated).Error; err != nil {
		return nil, false
	}
	return &translated, true
}

func (h *PublicHandler) findNodeByURL(path string) (*models.ContentNode, bool) {
	var node models.ContentNode
	if err := h.db.Where("full_url = ? AND status = ? AND deleted_at IS NULL", path, "published").First(&node).Error; err != nil {
		return nil, false
	}
	return &node, true
}

// findNodeWithPrefixFallback handles both directions for hide_prefix languages:
// - /en/test -> tries /test (for when hide_prefix is ON but user typed with prefix)
// - /test -> tries /en/test (for when hide_prefix is OFF but URL has no prefix)
func (h *PublicHandler) findNodeWithPrefixFallback(path string) (*models.ContentNode, bool) {
	// Get all active languages
	allLangs := h.loadActiveLanguages()

	// Get all languages with hide_prefix enabled
	var hiddenLangs []models.Language
	for _, l := range allLangs {
		if l.HidePrefix {
			hiddenLangs = append(hiddenLangs, l)
		}
	}

	// Case 1: path is /en/something — check if "en" is a language slug with hide_prefix,
	// meaning the stored URL is just /something
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
	if len(parts) >= 2 {
		firstSegment := parts[0]
		rest := "/" + parts[1]
		for _, lang := range hiddenLangs {
			if lang.Slug == firstSegment || lang.Code == firstSegment {
				if node, found := h.findNodeByURL(rest); found {
					return node, true
				}
			}
		}
	}

	// Case 2: path is /something — try prepending each hide_prefix language slug
	for _, lang := range hiddenLangs {
		prefixed := "/" + lang.Slug + path
		if node, found := h.findNodeByURL(prefixed); found {
			return node, true
		}
	}

	// Case 3: path is /something — try prepending any active language slug
	// (handles case where URL was built without prefix but hide_prefix is now off)
	for _, lang := range allLangs {
		if lang.HidePrefix {
			continue // already tried above
		}
		prefixed := "/" + lang.Slug + path
		if node, found := h.findNodeByURL(prefixed); found {
			return node, true
		}
	}

	return nil, false
}

// currentUser attempts to retrieve the logged-in user from the session cookie.
// Returns nil if no valid session exists (does not require auth).
func (h *PublicHandler) currentUser(c *fiber.Ctx) *models.User {
	token := c.Cookies(auth.CookieName)
	if token == "" {
		return nil
	}
	user, err := h.sessions.ValidateSession(token)
	if err != nil {
		return nil
	}
	return user
}

// buildUserData converts a User model (possibly nil) to template-friendly UserData.
func buildUserData(user *models.User) UserData {
	if user == nil {
		return UserData{LoggedIn: false}
	}
	fullName := ""
	if user.FullName != nil {
		fullName = *user.FullName
	}
	return UserData{
		LoggedIn: true,
		ID:       user.ID,
		Email:    user.Email,
		Role:     user.Role.Slug,
		FullName: fullName,
	}
}

// parseBlocks unmarshals JSONB blocks_data into a slice of maps.
func parseBlocks(data models.JSONB) []map[string]interface{} {
	if len(data) == 0 {
		return nil
	}

	var blocks []map[string]interface{}
	if err := json.Unmarshal([]byte(data), &blocks); err != nil {
		log.Printf("warning: failed to parse blocks_data: %v", err)
		return nil
	}
	return blocks
}

// extractBlockSlugs returns the unique block type slugs used in a parsed blocks list.
func extractBlockSlugs(blocks []map[string]interface{}) []string {
	seen := make(map[string]bool, len(blocks))
	slugs := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if t, ok := b["type"].(string); ok && t != "" && !seen[t] {
			seen[t] = true
			slugs = append(slugs, t)
		}
	}
	return slugs
}
