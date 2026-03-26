package cms

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/auth"
	"vibecms/internal/models"
)

// NodeHandler provides HTTP handlers for content node CRUD operations.
type NodeHandler struct {
	svc *ContentService
	db  *gorm.DB
}

// NewNodeHandler creates a new NodeHandler with the given ContentService.
func NewNodeHandler(svc *ContentService, db *gorm.DB) *NodeHandler {
	return &NodeHandler{svc: svc, db: db}
}

// RegisterRoutes registers all content node routes on the provided router group.
func (h *NodeHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/nodes", h.List)
	router.Get("/nodes/search", h.Search)
	router.Get("/nodes/:id", h.Get)
	router.Post("/nodes", h.Create)
	router.Patch("/nodes/:id", h.Update)
	router.Delete("/nodes/:id", h.Delete)
	router.Get("/nodes/:id/translations", h.GetTranslations)
	router.Post("/nodes/:id/translations", h.CreateTranslation)
	router.Post("/nodes/:id/homepage", h.SetHomepage)
	router.Get("/homepage", h.GetHomepage)
}

// GetTranslations handles GET /nodes/:id/translations.
func (h *NodeHandler) GetTranslations(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Node ID must be a valid integer")
	}

	translations, err := h.svc.GetTranslations(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Content node not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch translations")
	}

	return api.Success(c, translations)
}

// createTranslationRequest represents the JSON body for creating a translation.
type createTranslationRequest struct {
	LanguageCode string `json:"language_code"`
}

// CreateTranslation handles POST /nodes/:id/translations.
func (h *NodeHandler) CreateTranslation(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Node ID must be a valid integer")
	}

	var req createTranslationRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	if req.LanguageCode == "" {
		return api.ValidationError(c, map[string]string{
			"language_code": "Language code is required",
		})
	}

	node, err := h.svc.CreateTranslation(id, req.LanguageCode)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Content node not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create translation")
	}

	return api.Created(c, node)
}

// SetHomepage sets a node as the site homepage.
func (h *NodeHandler) SetHomepage(c *fiber.Ctx) error {
	id := c.Params("id")
	h.db.Exec(
		`INSERT INTO site_settings (key, value, updated_at) VALUES ('homepage_node_id', ?, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`, id,
	)
	return api.Success(c, fiber.Map{"message": "Homepage set", "node_id": id})
}

// GetHomepage returns the current homepage node ID.
func (h *NodeHandler) GetHomepage(c *fiber.Ctx) error {
	var value string
	h.db.Raw("SELECT value FROM site_settings WHERE key = 'homepage_node_id'").Scan(&value)
	id, _ := strconv.Atoi(value)
	return api.Success(c, fiber.Map{"homepage_node_id": id})
}

// Search handles GET /nodes/search for lightweight node lookup.
// Query params: q (search term), node_type (filter by type), limit (max results, default 20)
func (h *NodeHandler) Search(c *fiber.Ctx) error {
	q := c.Query("q", "")
	nodeType := c.Query("node_type", "")
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := h.db.Model(&models.ContentNode{}).
		Select("id, title, slug, node_type, status, language_code").
		Where("deleted_at IS NULL")

	if q != "" {
		searchTerm := "%" + q + "%"
		query = query.Where("title ILIKE ? OR slug ILIKE ?", searchTerm, searchTerm)
	}
	if nodeType != "" {
		query = query.Where("node_type = ?", nodeType)
	}

	var nodes []models.ContentNode
	if err := query.Order("title ASC").Limit(limit).Find(&nodes).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "SEARCH_FAILED", "Failed to search nodes")
	}

	// Return lightweight results
	type searchResult struct {
		ID           int    `json:"id"`
		Title        string `json:"title"`
		Slug         string `json:"slug"`
		NodeType     string `json:"node_type"`
		Status       string `json:"status"`
		LanguageCode string `json:"language_code"`
	}

	results := make([]searchResult, len(nodes))
	for i, n := range nodes {
		results[i] = searchResult{
			ID:           n.ID,
			Title:        n.Title,
			Slug:         n.Slug,
			NodeType:     n.NodeType,
			Status:       n.Status,
			LanguageCode: n.LanguageCode,
		}
	}

	return api.Success(c, results)
}

// List handles GET /nodes with pagination and filtering.
func (h *NodeHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	status := c.Query("status")
	nodeType := c.Query("node_type")
	langCode := c.Query("language_code")
	search := c.Query("search")

	nodes, total, err := h.svc.List(page, perPage, status, nodeType, langCode, search)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list content nodes")
	}

	return api.Paginated(c, nodes, total, page, perPage)
}

// Get handles GET /nodes/:id to retrieve a single content node.
func (h *NodeHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Node ID must be a valid integer")
	}

	node, err := h.svc.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Content node not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch content node")
	}

	return api.Success(c, node)
}

// createNodeRequest represents the JSON body for creating a content node.
type createNodeRequest struct {
	Title        string          `json:"title"`
	NodeType     string          `json:"node_type"`
	LanguageCode string          `json:"language_code"`
	ParentID     *int            `json:"parent_id"`
	Slug         string          `json:"slug"`
	BlocksData   json.RawMessage `json:"blocks_data"`
	SeoSettings  json.RawMessage `json:"seo_settings"`
}

// Create handles POST /nodes to create a new content node.
func (h *NodeHandler) Create(c *fiber.Ctx) error {
	var req createNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	if req.Title == "" {
		return api.ValidationError(c, map[string]string{
			"title": "Title is required",
		})
	}

	node := models.ContentNode{
		Title:        req.Title,
		NodeType:     req.NodeType,
		LanguageCode: req.LanguageCode,
		ParentID:     req.ParentID,
		Slug:         req.Slug,
		BlocksData:   models.JSONB(req.BlocksData),
		SeoSettings:  models.JSONB(req.SeoSettings),
	}

	if node.NodeType == "" {
		node.NodeType = "page"
	}
	if node.LanguageCode == "" {
		node.LanguageCode = "en"
	}

	user := auth.GetCurrentUser(c)
	userID := 0
	if user != nil {
		userID = user.ID
	}

	if err := h.svc.Create(&node, userID); err != nil {
		if isSlugConflict(err) {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", err.Error())
		}
		if isValidationError(err) {
			return api.ValidationError(c, map[string]string{
				"slug": err.Error(),
			})
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create content node")
	}

	return api.Created(c, node)
}

// Update handles PATCH /nodes/:id to partially update a content node.
func (h *NodeHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Node ID must be a valid integer")
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Remove fields that should not be directly updated
	delete(body, "id")
	delete(body, "uuid")
	delete(body, "version")
	delete(body, "created_at")
	delete(body, "updated_at")
	delete(body, "deleted_at")

	if len(body) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "NO_UPDATES", "No valid fields to update")
	}

	user := auth.GetCurrentUser(c)
	userID := 0
	if user != nil {
		userID = user.ID
	}

	updated, err := h.svc.Update(id, body, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Content node not found")
		}
		if isSlugConflict(err) {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", err.Error())
		}
		if isValidationError(err) {
			return api.ValidationError(c, map[string]string{
				"slug": err.Error(),
			})
		}
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update content node")
	}

	return api.Success(c, updated)
}

// Delete handles DELETE /nodes/:id to soft-delete a content node.
func (h *NodeHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Node ID must be a valid integer")
	}

	if err := h.svc.Delete(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Content node not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete content node")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// isSlugConflict checks if an error is a slug/full_url conflict.
func isSlugConflict(err error) bool {
	return err != nil && contains(err.Error(), "slug conflict")
}

// isValidationError checks if an error is a validation error.
func isValidationError(err error) bool {
	return err != nil && contains(err.Error(), "invalid slug")
}

// contains checks if s contains substr (case-insensitive not needed here,
// since our error messages are controlled).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
