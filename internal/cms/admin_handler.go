package cms

import (
	"bytes"
	"encoding/json"
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
	Title      string
	User       *models.User
	Node       *models.ContentNode
	Nodes      []models.ContentNode
	Stats      map[string]int64
	AllNodes   []models.ContentNode // for parent dropdown
	NodeType   string               // "page" or "post" — controls which section is active
	IsHomepage bool
	FlashMsg   string
	FlashType  string
	HomepageID int // ID of the homepage node (0 if not set)
}

// AdminPageHandler renders admin HTML pages (not API endpoints).
type AdminPageHandler struct {
	db         *gorm.DB
	renderer   *rendering.TemplateRenderer
	contentSvc *ContentService
}

// NewAdminPageHandler creates a new AdminPageHandler.
func NewAdminPageHandler(db *gorm.DB, renderer *rendering.TemplateRenderer, contentSvc *ContentService) *AdminPageHandler {
	return &AdminPageHandler{
		db:         db,
		renderer:   renderer,
		contentSvc: contentSvc,
	}
}

// adminFlash sets flash message cookies for display after redirect.
func adminFlash(c *fiber.Ctx, msg, flashType string) {
	c.Cookie(&fiber.Cookie{Name: "flash_msg", Value: msg, Path: "/", MaxAge: 10, HTTPOnly: true})
	c.Cookie(&fiber.Cookie{Name: "flash_type", Value: flashType, Path: "/", MaxAge: 10, HTTPOnly: true})
}

// getHomepageID reads the homepage_node_id from site_settings.
func (h *AdminPageHandler) getHomepageID() int {
	var setting models.SiteSetting
	if err := h.db.Where("key = ?", "homepage_node_id").First(&setting).Error; err != nil {
		return 0
	}
	if setting.Value == nil {
		return 0
	}
	id, _ := strconv.Atoi(*setting.Value)
	return id
}

// RegisterRoutes registers admin page routes on the given router group.
func (h *AdminPageHandler) RegisterRoutes(app fiber.Router) {
	app.Get("/", h.RedirectToDashboard)
	app.Get("/dashboard", h.Dashboard)

	// Pages
	app.Get("/pages", h.listByType("page"))
	app.Get("/pages/new", h.newByType("page"))
	app.Get("/pages/:id/edit", h.editByType("page"))

	// Page form actions
	app.Post("/pages", h.createNode("page"))
	app.Post("/pages/:id", h.updateNode("page"))
	app.Post("/pages/:id/delete", h.deleteNode("page"))
	app.Post("/pages/:id/homepage", h.setHomepage)

	// Posts
	app.Get("/posts", h.listByType("post"))
	app.Get("/posts/new", h.newByType("post"))
	app.Get("/posts/:id/edit", h.editByType("post"))

	// Post form actions
	app.Post("/posts", h.createNode("post"))
	app.Post("/posts/:id", h.updateNode("post"))
	app.Post("/posts/:id/delete", h.deleteNode("post"))

	// Legacy redirect
	app.Get("/nodes", func(c *fiber.Ctx) error {
		return c.Redirect("/admin/pages", fiber.StatusFound)
	})
}

// renderAdmin renders an admin page using the admin layout.
func (h *AdminPageHandler) renderAdmin(c *fiber.Ctx, pageName string, data AdminPageData) error {
	// Read and clear flash cookies
	if msg := c.Cookies("flash_msg"); msg != "" {
		data.FlashMsg = msg
		data.FlashType = c.Cookies("flash_type")
		// Clear the cookies
		c.Cookie(&fiber.Cookie{Name: "flash_msg", Value: "", Path: "/", MaxAge: -1, HTTPOnly: true})
		c.Cookie(&fiber.Cookie{Name: "flash_type", Value: "", Path: "/", MaxAge: -1, HTTPOnly: true})
	}

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
		Title:      "Dashboard",
		User:       user,
		Stats:      stats,
		Nodes:      recentNodes,
		HomepageID: h.getHomepageID(),
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
			Title:      typeLabelPlural(nodeType),
			User:       user,
			Nodes:      nodes,
			NodeType:   nodeType,
			HomepageID: h.getHomepageID(),
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

		homepageID := h.getHomepageID()

		return h.renderAdmin(c, "node_edit.html", AdminPageData{
			Title:      "Edit " + typeLabel(nodeType),
			User:       user,
			Node:       &node,
			AllNodes:   allNodes,
			NodeType:   nodeType,
			IsHomepage: homepageID == node.ID,
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

// createNode returns a handler that creates a new content node from form POST.
func (h *AdminPageHandler) createNode(nodeType string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := auth.GetCurrentUser(c)
		userID := 0
		if user != nil {
			userID = user.ID
		}

		title := strings.TrimSpace(c.FormValue("title"))
		if title == "" {
			adminFlash(c, "Title is required", "error")
			return c.Redirect(fmt.Sprintf("/admin/%ss/new", nodeType), fiber.StatusFound)
		}

		slug := strings.TrimSpace(c.FormValue("slug"))
		if slug == "" {
			slug = Slugify(title)
		}

		status := c.FormValue("status")
		if status == "" {
			status = "draft"
		}

		langCode := c.FormValue("language_code")
		if langCode == "" {
			langCode = "en"
		}

		blocksData := c.FormValue("blocks_data")
		if blocksData == "" {
			blocksData = "[]"
		}

		// Validate JSON
		if !json.Valid([]byte(blocksData)) {
			adminFlash(c, "Invalid JSON in blocks data", "error")
			return c.Redirect(fmt.Sprintf("/admin/%ss/new", nodeType), fiber.StatusFound)
		}

		// Parse parent_id
		var parentID *int
		if pidStr := c.FormValue("parent_id"); pidStr != "" {
			pid, err := strconv.Atoi(pidStr)
			if err == nil {
				parentID = &pid
			}
		}

		node := &models.ContentNode{
			Title:        title,
			Slug:         slug,
			Status:       status,
			LanguageCode: langCode,
			NodeType:     nodeType,
			BlocksData:   models.JSONB(blocksData),
			ParentID:     parentID,
		}

		if err := h.contentSvc.Create(node, userID); err != nil {
			adminFlash(c, fmt.Sprintf("Error creating %s: %s", typeLabel(nodeType), err.Error()), "error")
			return c.Redirect(fmt.Sprintf("/admin/%ss/new", nodeType), fiber.StatusFound)
		}

		adminFlash(c, fmt.Sprintf("%s created successfully", typeLabel(nodeType)), "success")
		return c.Redirect(fmt.Sprintf("/admin/%ss/%d/edit", nodeType, node.ID), fiber.StatusFound)
	}
}

// updateNode returns a handler that updates an existing content node from form POST.
func (h *AdminPageHandler) updateNode(nodeType string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := auth.GetCurrentUser(c)
		userID := 0
		if user != nil {
			userID = user.ID
		}

		id, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			adminFlash(c, "Invalid ID", "error")
			return c.Redirect(fmt.Sprintf("/admin/%ss", nodeType), fiber.StatusFound)
		}

		title := strings.TrimSpace(c.FormValue("title"))
		if title == "" {
			adminFlash(c, "Title is required", "error")
			return c.Redirect(fmt.Sprintf("/admin/%ss/%d/edit", nodeType, id), fiber.StatusFound)
		}

		slug := strings.TrimSpace(c.FormValue("slug"))
		if slug == "" {
			slug = Slugify(title)
		}

		blocksData := c.FormValue("blocks_data")
		if blocksData == "" {
			blocksData = "[]"
		}

		// Validate JSON
		if !json.Valid([]byte(blocksData)) {
			adminFlash(c, "Invalid JSON in blocks data", "error")
			return c.Redirect(fmt.Sprintf("/admin/%ss/%d/edit", nodeType, id), fiber.StatusFound)
		}

		updates := map[string]interface{}{
			"title":         title,
			"slug":          slug,
			"status":        c.FormValue("status"),
			"language_code": c.FormValue("language_code"),
			"blocks_data":   models.JSONB(blocksData),
		}

		// Handle parent_id
		if pidStr := c.FormValue("parent_id"); pidStr != "" {
			pid, err := strconv.Atoi(pidStr)
			if err == nil {
				updates["parent_id"] = &pid
			}
		} else {
			updates["parent_id"] = nil
		}

		if _, err := h.contentSvc.Update(id, updates, userID); err != nil {
			adminFlash(c, fmt.Sprintf("Error updating %s: %s", typeLabel(nodeType), err.Error()), "error")
			return c.Redirect(fmt.Sprintf("/admin/%ss/%d/edit", nodeType, id), fiber.StatusFound)
		}

		adminFlash(c, fmt.Sprintf("%s updated successfully", typeLabel(nodeType)), "success")
		return c.Redirect(fmt.Sprintf("/admin/%ss/%d/edit", nodeType, id), fiber.StatusFound)
	}
}

// deleteNode returns a handler that deletes a content node.
func (h *AdminPageHandler) deleteNode(nodeType string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			adminFlash(c, "Invalid ID", "error")
			return c.Redirect(fmt.Sprintf("/admin/%ss", nodeType), fiber.StatusFound)
		}

		if err := h.contentSvc.Delete(id); err != nil {
			adminFlash(c, fmt.Sprintf("Error deleting %s: %s", typeLabel(nodeType), err.Error()), "error")
			return c.Redirect(fmt.Sprintf("/admin/%ss", nodeType), fiber.StatusFound)
		}

		adminFlash(c, fmt.Sprintf("%s deleted successfully", typeLabel(nodeType)), "success")
		return c.Redirect(fmt.Sprintf("/admin/%ss", nodeType), fiber.StatusFound)
	}
}

// setHomepage sets a page as the site homepage by storing its ID in site_settings.
func (h *AdminPageHandler) setHomepage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		adminFlash(c, "Invalid ID", "error")
		return c.Redirect("/admin/pages", fiber.StatusFound)
	}

	// Verify the node exists
	if _, err := h.contentSvc.GetByID(id); err != nil {
		adminFlash(c, "Page not found", "error")
		return c.Redirect("/admin/pages", fiber.StatusFound)
	}

	// Upsert into site_settings
	idStr := strconv.Itoa(id)
	setting := models.SiteSetting{
		Key:   "homepage_node_id",
		Value: &idStr,
	}
	result := h.db.Where("key = ?", "homepage_node_id").First(&models.SiteSetting{})
	if result.Error == gorm.ErrRecordNotFound {
		h.db.Create(&setting)
	} else {
		h.db.Model(&models.SiteSetting{}).Where("key = ?", "homepage_node_id").Update("value", &idStr)
	}

	adminFlash(c, "Page set as homepage", "success")
	return c.Redirect(fmt.Sprintf("/admin/pages/%d/edit", id), fiber.StatusFound)
}
