package cms

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"squilla/internal/auth"
	"squilla/internal/events"
	"squilla/internal/models"
	"squilla/internal/rendering"

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

	themeSettingsRegistry *ThemeSettingsRegistry
	themeSettingsAPI      settingsReader
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
	themeSettingsRegistry *ThemeSettingsRegistry,
	themeSettingsAPI settingsReader,
) *PublicHandler {
	h := &PublicHandler{
		db:                    db,
		renderer:              renderer,
		sessions:              sessions,
		layoutSvc:             layoutSvc,
		layoutBlockSvc:        layoutBlockSvc,
		menuSvc:               menuSvc,
		renderCtx:             renderCtx,
		eventBus:              eventBus,
		themeSettingsRegistry: themeSettingsRegistry,
		themeSettingsAPI:      themeSettingsAPI,
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
				h.resolveAssetRefsInBlocks(blocks)
				renderedBlocks := h.renderBlocksBatch(blocks, node.LanguageCode)

				// Layout-based rendering
				html, _, renderErr := h.renderNodeWithLayout(c, &node, blocks, renderedBlocks, user)
				if renderErr != nil {
					c.Set("Content-Type", "text/html; charset=utf-8")
					return c.SendString(h.layoutErrorPage(renderErr, node.FullURL))
				}
				if html != "" {
					c.Set("Content-Type", "text/html; charset=utf-8")
					return c.SendString(html)
				}
				// No layout resolved — show error page
				c.Set("Content-Type", "text/html; charset=utf-8")
				return c.SendString(h.layoutErrorPage(fmt.Errorf("no layout resolved for homepage"), "/"))
			}
		}
	}

	// No homepage node found — show error page
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(h.layoutErrorPage(fmt.Errorf("homepage content node not found"), "/"))
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
		// No 404 layout — show error page
		c.Set("Content-Type", "text/html; charset=utf-8")
		c.Status(fiber.StatusNotFound)
		return c.SendString(h.layoutErrorPage(fmt.Errorf("page not found and no 404 layout available"), c.Path()))
	}

	blocks := parseBlocks(node.BlocksData)
	h.resolveAssetRefsInBlocks(blocks)
	renderedBlocks := h.renderBlocksBatch(blocks, node.LanguageCode)

	// Layout-based rendering
	html, _, renderErr := h.renderNodeWithLayout(c, node, blocks, renderedBlocks, user)
	if renderErr != nil {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(h.layoutErrorPage(renderErr, node.FullURL))
	}
	if html != "" {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}

	// No layout resolved and no error — show error page (should not happen in normal operation)
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(h.layoutErrorPage(fmt.Errorf("no layout found for page %q", node.FullURL), node.FullURL))
}
