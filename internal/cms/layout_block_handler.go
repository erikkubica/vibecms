package cms

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/auth"
	"vibecms/internal/models"
)

// LayoutBlockHandler provides HTTP handlers for layout block CRUD operations.
type LayoutBlockHandler struct {
	svc *LayoutBlockService
}

// NewLayoutBlockHandler creates a new LayoutBlockHandler with the given LayoutBlockService.
func NewLayoutBlockHandler(svc *LayoutBlockService) *LayoutBlockHandler {
	return &LayoutBlockHandler{svc: svc}
}

// RegisterRoutes registers all layout block routes on the provided router group.
// Reads are open to authenticated users; mutations require manage_layouts.
func (h *LayoutBlockHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/layout-blocks", h.List)
	router.Get("/layout-blocks/:id", h.Get)
	manage := auth.CapabilityRequired("manage_layouts")
	router.Post("/layout-blocks", manage, h.Create)
	router.Patch("/layout-blocks/:id", manage, h.Update)
	router.Delete("/layout-blocks/:id", manage, h.Delete)
	router.Post("/layout-blocks/:id/detach", manage, h.Detach)
	router.Post("/layout-blocks/:id/reattach", manage, h.Reattach)
}

// List handles GET /layout-blocks to retrieve all layout blocks with optional filters and pagination.
func (h *LayoutBlockHandler) List(c *fiber.Ctx) error {
	var languageID *int
	if langIDStr := c.Query("language_id"); langIDStr != "" {
		id, err := strconv.Atoi(langIDStr)
		if err != nil {
			return api.Error(c, fiber.StatusBadRequest, "INVALID_LANGUAGE_ID", "language_id must be a valid integer")
		}
		languageID = &id
	}
	source := c.Query("source")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	blocks, total, err := h.svc.List(languageID, source, page, perPage)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list layout blocks")
	}

	return api.Paginated(c, blocks, total, page, perPage)
}

// Get handles GET /layout-blocks/:id to retrieve a single layout block.
func (h *LayoutBlockHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Layout block ID must be a valid integer")
	}

	block, err := h.svc.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Layout block not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch layout block")
	}

	return api.Success(c, block)
}

// createLayoutBlockRequest represents the JSON body for creating a layout block.
type createLayoutBlockRequest struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	LanguageID   *int   `json:"language_id"`
	TemplateCode string `json:"template_code"`
}

// Create handles POST /layout-blocks to create a new layout block.
func (h *LayoutBlockHandler) Create(c *fiber.Ctx) error {
	var req createLayoutBlockRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Validate required fields.
	fields := map[string]string{}
	if req.Slug == "" {
		fields["slug"] = "Slug is required"
	}
	if req.Name == "" {
		fields["name"] = "Name is required"
	}
	if req.TemplateCode == "" {
		fields["template_code"] = "Template code is required"
	}
	if len(fields) > 0 {
		return api.ValidationError(c, fields)
	}

	block := models.LayoutBlock{
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		LanguageID:   req.LanguageID,
		TemplateCode: req.TemplateCode,
		Source:       "custom",
	}

	if err := h.svc.Create(&block); err != nil {
		if strings.Contains(err.Error(), "SLUG_CONFLICT") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", "A layout block with this slug and language already exists")
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create layout block")
	}

	return api.Created(c, block)
}

// Update handles PATCH /layout-blocks/:id to partially update a layout block.
func (h *LayoutBlockHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Layout block ID must be a valid integer")
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Remove fields that should not be directly updated.
	delete(body, "id")
	delete(body, "created_at")
	delete(body, "updated_at")
	delete(body, "source")
	delete(body, "theme_name")

	if len(body) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "NO_UPDATES", "No valid fields to update")
	}

	updated, err := h.svc.Update(id, body)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Layout block not found")
		}
		if strings.Contains(err.Error(), "THEME_READONLY") {
			return api.Error(c, fiber.StatusForbidden, "THEME_READONLY", "Theme layout blocks cannot be edited directly; detach first")
		}
		if strings.Contains(err.Error(), "SLUG_CONFLICT") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", "A layout block with this slug and language already exists")
		}
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update layout block")
	}

	return api.Success(c, updated)
}

// Delete handles DELETE /layout-blocks/:id to remove a layout block.
func (h *LayoutBlockHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Layout block ID must be a valid integer")
	}

	if err := h.svc.Delete(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Layout block not found")
		}
		if strings.Contains(err.Error(), "THEME_READONLY") {
			return api.Error(c, fiber.StatusForbidden, "THEME_READONLY", "Theme layout blocks cannot be deleted; detach first")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete layout block")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Detach handles POST /layout-blocks/:id/detach to convert a theme layout block to custom.
func (h *LayoutBlockHandler) Detach(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Layout block ID must be a valid integer")
	}

	block, err := h.svc.Detach(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Layout block not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DETACH_FAILED", "Failed to detach layout block")
	}

	return api.Success(c, block)
}

// Reattach handles POST /layout-blocks/:id/reattach to restore a layout block to its theme version.
func (h *LayoutBlockHandler) Reattach(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Layout block ID must be a valid integer")
	}

	block, err := h.svc.Reattach(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Layout block not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "REATTACH_FAILED", "Failed to reattach layout block")
	}

	return api.Success(c, block)
}
