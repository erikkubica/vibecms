package cms

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"

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
	app.Get("/nodes", h.NodesList)
	app.Get("/nodes/new", h.NodeNew)
	app.Get("/nodes/:id/edit", h.NodeEdit)
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

// NodesList renders the content nodes list page with optional filtering.
func (h *AdminPageHandler) NodesList(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)

	query := h.db.Model(&models.ContentNode{}).Order("updated_at DESC")

	// Apply filters from query params.
	if status := c.Query("status_filter"); status != "" {
		query = query.Where("status = ?", status)
	}
	if nodeType := c.Query("type_filter"); nodeType != "" {
		query = query.Where("node_type = ?", nodeType)
	}
	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		query = query.Where("title ILIKE ? OR slug ILIKE ?", like, like)
	}

	var nodes []models.ContentNode
	query.Find(&nodes)

	return h.renderAdmin(c, "nodes_list.html", AdminPageData{
		Title: "Content",
		User:  user,
		Nodes: nodes,
	})
}

// NodeNew renders the create-new-node form.
func (h *AdminPageHandler) NodeNew(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)

	var allNodes []models.ContentNode
	h.db.Select("id", "title").Where("node_type = ?", "page").Order("title ASC").Find(&allNodes)

	return h.renderAdmin(c, "node_edit.html", AdminPageData{
		Title:    "New Page",
		User:     user,
		AllNodes: allNodes,
	})
}

// NodeEdit renders the edit form for an existing content node.
func (h *AdminPageHandler) NodeEdit(c *fiber.Ctx) error {
	user := auth.GetCurrentUser(c)

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString("Invalid node ID")
	}

	var node models.ContentNode
	if err := h.db.First(&node, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).SendString("Page not found")
		}
		return c.Status(http.StatusInternalServerError).SendString("Database error")
	}

	var allNodes []models.ContentNode
	h.db.Select("id", "title").Where("id != ? AND node_type = ?", id, "page").Order("title ASC").Find(&allNodes)

	return h.renderAdmin(c, "node_edit.html", AdminPageData{
		Title:    "Edit Page",
		User:     user,
		Node:     &node,
		AllNodes: allNodes,
	})
}
