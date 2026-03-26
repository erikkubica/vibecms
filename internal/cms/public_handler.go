package cms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"strconv"
	"strings"

	"vibecms/internal/auth"
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
) *PublicHandler {
	return &PublicHandler{
		db:             db,
		renderer:       renderer,
		sessions:       sessions,
		layoutSvc:      layoutSvc,
		layoutBlockSvc: layoutBlockSvc,
		menuSvc:        menuSvc,
		renderCtx:      renderCtx,
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
	var setting models.SiteSetting
	if err := h.db.Where("key = ?", "homepage_node_id").First(&setting).Error; err == nil && setting.Value != nil {
		if homepageID, err := strconv.Atoi(*setting.Value); err == nil && homepageID > 0 {
			var node models.ContentNode
			if err := h.db.Where("id = ? AND status = ?", homepageID, "published").First(&node).Error; err == nil {
				blocks := parseBlocks(node.BlocksData)
				renderedBlocks := h.renderBlocks(blocks)

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
	renderedBlocks := h.renderBlocks(blocks)

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
	// Resolve the node's language to get its ID
	var nodeLang models.Language
	var languageID *int
	if err := h.db.Where("code = ?", node.LanguageCode).First(&nodeLang).Error; err == nil {
		languageID = &nodeLang.ID
	}

	// Resolve layout for this node
	layout, err := h.layoutSvc.ResolveForNode(node, languageID)
	if err != nil || layout == nil {
		return "", false
	}

	// Get all active languages
	var languages []models.Language
	h.db.Where("is_active = ?", true).Order("sort_order ASC").Find(&languages)

	// Find the current language
	var currentLang *models.Language
	for i := range languages {
		if languages[i].Code == node.LanguageCode {
			currentLang = &languages[i]
			break
		}
	}

	// Load site settings
	settings := h.loadSiteSettings()

	// Load menus
	menus := h.renderCtx.LoadMenus(languageID)

	// Combine rendered blocks into a single HTML string
	blocksHTML := strings.Join(renderedBlocks, "\n")

	// Build template data
	appData := h.renderCtx.BuildAppData(settings, languages, currentLang)
	appData.Menus = menus

	nodeData := h.renderCtx.BuildNodeData(node, blocksHTML)

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
	// Get default language
	var defaultLang models.Language
	if err := h.db.Where("is_default = ?", true).First(&defaultLang).Error; err != nil {
		return "", false
	}
	defaultLangID := &defaultLang.ID

	// Find default layout — try language-specific first, then universal (NULL)
	var layout models.Layout
	err := h.db.Where("is_default = ? AND language_id = ?", true, *defaultLangID).First(&layout).Error
	if err != nil {
		// Fall back to universal default layout
		if err2 := h.db.Where("is_default = ? AND language_id IS NULL", true).First(&layout).Error; err2 != nil {
			return "", false
		}
	}

	// Get languages, settings, menus
	var languages []models.Language
	h.db.Where("is_active = ?", true).Order("sort_order ASC").Find(&languages)

	var currentLang *models.Language
	for i := range languages {
		if languages[i].Code == defaultLang.Code {
			currentLang = &languages[i]
			break
		}
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

	appData := h.renderCtx.BuildAppData(settings, languages, currentLang)
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
	var defaultLang models.Language
	if err := h.db.Where("is_default = ?", true).First(&defaultLang).Error; err != nil {
		return "", false
	}
	defaultLangID := &defaultLang.ID

	var layout models.Layout
	if err := h.db.Where("is_default = ? AND language_id = ?", true, defaultLangID).First(&layout).Error; err != nil {
		if err2 := h.db.Where("is_default = ? AND language_id IS NULL", true).First(&layout).Error; err2 != nil {
			return "", false
		}
	}

	var languages []models.Language
	h.db.Where("is_active = ?", true).Order("sort_order ASC").Find(&languages)

	var currentLang *models.Language
	for i := range languages {
		if languages[i].Code == defaultLang.Code {
			currentLang = &languages[i]
			break
		}
	}

	settings := h.loadSiteSettings()
	menus := h.renderCtx.LoadMenus(defaultLangID)
	user := h.currentUser(c)

	appData := h.renderCtx.BuildAppData(settings, languages, currentLang)
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
	settings := make(map[string]string)
	var allSettings []models.SiteSetting
	h.db.Find(&allSettings)
	for _, s := range allSettings {
		if s.Value != nil {
			settings[s.Key] = *s.Value
		}
	}
	return settings
}

// renderBlocks renders each block's HTML template with its field values.
func (h *PublicHandler) renderBlocks(blocks []map[string]interface{}) []string {
	// Fetch all block types
	var blockTypes []models.BlockType
	h.db.Find(&blockTypes)
	btMap := make(map[string]models.BlockType)
	for _, bt := range blockTypes {
		btMap[bt.Slug] = bt
	}

	var rendered []string
	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		fields, _ := block["fields"].(map[string]interface{})
		if fields == nil {
			fields = block // fallback for old format
		}

		bt, ok := btMap[blockType]
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

		// Check for theme file override
		tmplContent := bt.HTMLTemplate
		themeFile := fmt.Sprintf("themes/default/blocks/%s.html", blockType)
		if fileContent, err := os.ReadFile(themeFile); err == nil {
			tmplContent = string(fileContent)
		}

		// Hydrate node references — resolve node selector fields to full node data
		h.hydrateFields(fields)

		// Auto-mark richtext fields as template.HTML so they render unescaped.
		// Also walk repeater rows to mark nested richtext fields.
		markRichTextFields(fields, bt.FieldSchema)

		// Execute template with field values and custom functions
		tmpl, err := template.New(blockType).Funcs(template.FuncMap{
			"safeHTML": func(s interface{}) template.HTML {
				return template.HTML(fmt.Sprintf("%v", s))
			},
		}).Parse(tmplContent)
		if err != nil {
			rendered = append(rendered, fmt.Sprintf(`<div class="mb-4 text-red-500 text-sm">Template error in %s: %v</div>`, blockType, err))
			continue
		}

		var buf strings.Builder
		if err := tmpl.Execute(&buf, fields); err != nil {
			rendered = append(rendered, fmt.Sprintf(`<div class="mb-4 text-red-500 text-sm">Render error in %s: %v</div>`, blockType, err))
			continue
		}

		rendered = append(rendered, buf.String())
	}
	return rendered
}

// hydrateFields walks through field values and hydrates node references.
// A node reference is a map with an "id" key (single node selector) or
// an array of such maps (multi node selector). Each reference is replaced
// with full node data including flattened fields_data.
func (h *PublicHandler) hydrateFields(fields map[string]interface{}) {
	for key, val := range fields {
		switch v := val.(type) {
		case map[string]interface{}:
			// Single node reference: {"id": 5, "title": "..."}
			if _, hasID := v["id"]; hasID {
				fields[key] = h.hydrateNodeRef(v)
			}
		case []interface{}:
			// Array: could be multi node selector or repeater rows
			if len(v) > 0 {
				if first, ok := v[0].(map[string]interface{}); ok {
					if _, hasID := first["id"]; hasID {
						// Multi node selector: hydrate each
						for i, item := range v {
							if m, ok := item.(map[string]interface{}); ok {
								v[i] = h.hydrateNodeRef(m)
							}
						}
						fields[key] = v
					} else {
						// Repeater rows: recursively hydrate each row
						for _, item := range v {
							if m, ok := item.(map[string]interface{}); ok {
								h.hydrateFields(m)
							}
						}
					}
				}
			}
		}
	}
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

// hydrateNodeRef takes a lightweight node reference {"id": 5, "title": "..."}
// and returns a fully hydrated map with all node fields + flattened fields_data.
// Template can then use {{.title}}, {{.slug}}, {{.position}}, {{.bio}}, etc.
func (h *PublicHandler) hydrateNodeRef(ref map[string]interface{}) map[string]interface{} {
	idVal, ok := ref["id"]
	if !ok {
		return ref
	}

	// Parse ID (could be float64 from JSON)
	var nodeID int
	switch id := idVal.(type) {
	case float64:
		nodeID = int(id)
	case int:
		nodeID = id
	case json.Number:
		n, _ := id.Int64()
		nodeID = int(n)
	default:
		return ref
	}

	if nodeID == 0 {
		return ref
	}

	var node models.ContentNode
	if err := h.db.First(&node, nodeID).Error; err != nil {
		return ref
	}

	// Build hydrated map with core node fields
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

	// Flatten fields_data into the hydrated map so custom fields are directly accessible
	if len(node.FieldsData) > 0 {
		var fieldsData map[string]interface{}
		if err := json.Unmarshal([]byte(node.FieldsData), &fieldsData); err == nil {
			// Look up node type schema to mark richtext fields
			var nodeType models.NodeType
			if err := h.db.Where("slug = ?", node.NodeType).First(&nodeType).Error; err == nil {
				markRichTextFields(fieldsData, nodeType.FieldSchema)
			}
			for k, v := range fieldsData {
				hydrated[k] = v
			}
		}
	}

	// Also parse and flatten blocks_data for advanced use
	if len(node.BlocksData) > 0 {
		var blocksData []map[string]interface{}
		if err := json.Unmarshal([]byte(node.BlocksData), &blocksData); err == nil {
			hydrated["blocks"] = blocksData
		}
	}

	return hydrated
}

// findNodeByURL does a direct full_url lookup.
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
	// Get all languages with hide_prefix enabled
	var hiddenLangs []models.Language
	h.db.Where("hide_prefix = ?", true).Find(&hiddenLangs)

	// Get all active languages for the reverse lookup
	var allLangs []models.Language
	h.db.Where("is_active = ?", true).Find(&allLangs)

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
