package cms

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"vibecms/internal/auth"
	"vibecms/internal/models"
	"vibecms/internal/rendering"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// AdminPageData holds all data passed to admin page templates.
type AdminPageData struct {
	Title    string
	User     *models.User
	Node     *models.ContentNode
	Nodes    []models.ContentNode
	Stats    map[string]int64
	AllNodes []models.ContentNode // for parent dropdown
	NodeType string              // "page" or "post" — controls which section is active
}

// AdminPageHandler renders admin HTML pages (not API endpoints).
type AdminPageHandler struct {
	db       *gorm.DB
	renderer *rendering.TemplateRenderer
}

// NewAdminPageHandler creates a new AdminPageHandler.
func NewAdminPageHandler(db *gorm.DB, renderer *rendering.TemplateRenderer) *AdminPageHandler {
	return &AdminPageHandler{
		db:       db,
		renderer: renderer,
	}
}

// RegisterRoutes registers admin page routes on the given router group.
func (h *AdminPageHandler) RegisterRoutes(app fiber.Router) {
	app.Get("/", h.RedirectToDashboard)
	app.Get("/dashboard", h.Dashboard)

	// Pages
	app.Get("/pages", h.listByType("page"))
	app.Get("/pages/new", h.newByType("page"))
	app.Get("/pages/:id/edit", h.editByType("page"))

	// Posts
	app.Get("/posts", h.listByType("post"))
	app.Get("/posts/new", h.newByType("post"))
	app.Get("/posts/:id/edit", h.editByType("post"))

	// Legacy redirect
	app.Get("/nodes", func(c *fiber.Ctx) error {
		return c.Redirect("/admin/pages", fiber.StatusFound)
	})
}

// renderAdmin renders an admin page using the admin layout.
func (h *AdminPageHandler) renderAdmin(c *fiber.Ctx, pageName string, data AdminPageData) error {
	var buf bytes.Buffer
	if err := h.renderer.Render(&buf, "layouts/admin.html", "admin/"+pageName, &data); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(
			fmt.Sprintf("Template render error: %v", err),
		)
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(buf.Bytes())
}

// RedirectToDashboard redirects /admin to /admin/dashboard.
func (h *AdminPageHandler) RedirectToDashboard(c *fiber.Ctx) error {
	return c.Redirect("/admin/dashboard", fiber.StatusFound)
}

// Dashboard renders the admin dashboard page with stats and recent nodes.
func (h *AdminPageHandler) Dashboard(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)

	var totalNodes, publishedNodes, draftNodes, totalUsers int64
	h.db.Model(&models.ContentNode{}).Count(&totalNodes)
	h.db.Model(&models.ContentNode{}).Where("status = ?", "published").Count(&publishedNodes)
	h.db.Model(&models.ContentNode{}).Where("status = ?", "draft").Count(&draftNodes)
	h.db.Model(&models.User{}).Count(&totalUsers)

	stats := map[string]int64{
		"total_nodes":     totalNodes,
		"published_nodes": publishedNodes,
		"draft_nodes":     draftNodes,
		"total_users":     totalUsers,
	}

	var recentNodes []models.ContentNode
	h.db.Order("updated_at DESC").Limit(10).Find(&recentNodes)

	return h.renderAdmin(c, "dashboard.html", AdminPageData{
		Title: "Dashboard",
		User:  user,
		Stats: stats,
		Nodes: recentNodes,
	})
}

// typeLabel returns the display label for a node type.
func typeLabel(nodeType string) string {
	switch nodeType {
	case "post":
		return "Post"
	default:
		return "Page"
	}
}

// typeLabelPlural returns the plural display label for a node type.
func typeLabelPlural(nodeType string) string {
	switch nodeType {
	case "post":
		return "Posts"
	default:
		return "Pages"
	}
}

// listByType returns a handler that lists nodes of a specific type.
func (h *AdminPageHandler) listByType(nodeType string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := auth.GetCurrentUser(c)

		query := h.db.Model(&models.ContentNode{}).
			Where("node_type = ?", nodeType).
			Order("updated_at DESC")

		if status := c.Query("status_filter"); status != "" {
			query = query.Where("status = ?", status)
		}
		if search := c.Query("search"); search != "" {
			like := "%" + search + "%"
			query = query.Where("title ILIKE ? OR slug ILIKE ?", like, like)
		}

		var nodes []models.ContentNode
		query.Find(&nodes)

		return h.renderAdmin(c, "nodes_list.html", AdminPageData{
			Title:    typeLabelPlural(nodeType),
			User:     user,
			Nodes:    nodes,
			NodeType: nodeType,
		})
	}
}

// newByType returns a handler that renders the new node form for a specific type.
func (h *AdminPageHandler) newByType(nodeType string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := auth.GetCurrentUser(c)

		var allNodes []models.ContentNode
		h.db.Select("id", "title").Where("node_type = ?", nodeType).Order("title ASC").Find(&allNodes)

		return h.renderAdmin(c, "node_edit.html", AdminPageData{
			Title:    "New " + typeLabel(nodeType),
			User:     user,
			AllNodes: allNodes,
			NodeType: nodeType,
		})
	}
}

// editByType returns a handler that renders the edit form for a node of a specific type.
func (h *AdminPageHandler) editByType(nodeType string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := auth.GetCurrentUser(c)

		id, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return c.Status(http.StatusBadRequest).SendString("Invalid ID")
		}

		var node models.ContentNode
		if err := h.db.First(&node, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(http.StatusNotFound).SendString(typeLabel(nodeType) + " not found")
			}
			return c.Status(http.StatusInternalServerError).SendString("Database error")
		}

		var allNodes []models.ContentNode
		h.db.Select("id", "title").
			Where("id != ? AND node_type = ?", id, nodeType).
			Order("title ASC").Find(&allNodes)

		return h.renderAdmin(c, "node_edit.html", AdminPageData{
			Title:    "Edit " + typeLabel(nodeType),
			User:     user,
			Node:     &node,
			AllNodes: allNodes,
			NodeType: nodeType,
		})
	}
}

// NodeTypeFromTitle extracts the node type from the page title for URL building.
func NodeTypeFromTitle(title string) string {
	lower := strings.ToLower(title)
	if strings.Contains(lower, "post") {
		return "post"
	}
	return "page"
}
