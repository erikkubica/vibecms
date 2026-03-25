package cms

import (
	"encoding/json"
	"log"

	"vibecms/internal/auth"
	"vibecms/internal/models"
	"vibecms/internal/rendering"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// PageData holds all data passed to public page templates.
type PageData struct {
	Title     string
	User      *models.User
	Node      *models.ContentNode
	Nodes     []models.ContentNode
	Blocks    []map[string]interface{}
	FlashMsg  string
	FlashType string // "success" or "error"
}

// PublicHandler serves the public-facing HTML pages.
type PublicHandler struct {
	db       *gorm.DB
	renderer *rendering.TemplateRenderer
	sessions *auth.SessionService
}

// NewPublicHandler creates a new PublicHandler.
func NewPublicHandler(db *gorm.DB, renderer *rendering.TemplateRenderer, sessions *auth.SessionService) *PublicHandler {
	return &PublicHandler{
		db:       db,
		renderer: renderer,
		sessions: sessions,
	}
}

// RegisterRoutes registers public page routes on the Fiber app.
func (h *PublicHandler) RegisterRoutes(app *fiber.App) {
	app.Get("/", h.HomePage)
	app.Get("/:slug", h.PageBySlug)
	app.Get("/:lang/:slug", h.PageByLangSlug)
}

// HomePage renders the public homepage with recent published content nodes.
func (h *PublicHandler) HomePage(c *fiber.Ctx) error {
	user := h.currentUser(c)

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

// PageBySlug renders a content node page by its slug.
func (h *PublicHandler) PageBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	return h.renderNodePage(c, slug, "")
}

// PageByLangSlug renders a content node page by language code and slug.
func (h *PublicHandler) PageByLangSlug(c *fiber.Ctx) error {
	lang := c.Params("lang")
	slug := c.Params("slug")
	return h.renderNodePage(c, slug, lang)
}

// renderNodePage fetches and renders a content node by slug and optional language.
func (h *PublicHandler) renderNodePage(c *fiber.Ctx, slug, lang string) error {
	user := h.currentUser(c)

	query := h.db.Where("slug = ? AND status = ? AND deleted_at IS NULL", slug, "published")
	if lang != "" {
		query = query.Where("language_code = ?", lang)
	}

	var node models.ContentNode
	if err := query.First(&node).Error; err != nil {
		// Node not found — render page template without a node (shows 404).
		data := PageData{
			Title: "Page Not Found - VibeCMS",
			User:  user,
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		c.Status(fiber.StatusNotFound)
		return h.renderer.RenderPage(c, "public/page.html", data)
	}

	blocks := parseBlocks(node.BlocksData)

	data := PageData{
		Title:  node.Title + " - VibeCMS",
		User:   user,
		Node:   &node,
		Blocks: blocks,
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.RenderPage(c, "public/page.html", data)
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
