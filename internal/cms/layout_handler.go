package cms

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/models"
)

// LayoutHandler provides HTTP handlers for layout CRUD operations.
type LayoutHandler struct {
	svc *LayoutService
}

// NewLayoutHandler creates a new LayoutHandler with the given LayoutService.
func NewLayoutHandler(svc *LayoutService) *LayoutHandler {
	return &LayoutHandler{svc: svc}
}

// RegisterRoutes registers all layout routes on the provided router group.
func (h *LayoutHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/layouts", h.List)
	router.Get("/layouts/:id", h.Get)
	router.Post("/layouts", h.Create)
	router.Patch("/layouts/:id", h.Update)
	router.Delete("/layouts/:id", h.Delete)
	router.Post("/layouts/:id/detach", h.Detach)
	router.Post("/layouts/:id/reattach", h.Reattach)
}

// List handles GET /layouts to retrieve all layouts with optional filters.
func (h *LayoutHandler) List(c *fiber.Ctx) error {
	var languageID *int
	if langIDStr := c.Query("language_id"); langIDStr != "" {
		id, err := strconv.Atoi(langIDStr)
		if err != nil {
			return api.Error(c, fiber.StatusBadRequest, "INVALID_LANGUAGE_ID", "language_id must be a valid integer")
		}
		languageID = &id
	}
	source := c.Query("source")

	layouts, err := h.svc.List(languageID, source)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list layouts")
	}

	return api.Success(c, layouts)
}

// Get handles GET /layouts/:id to retrieve a single layout.
func (h *LayoutHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Layout ID must be a valid integer")
	}

	layout, err := h.svc.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Layout not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch layout")
	}

	return api.Success(c, layout)
}

// createLayoutRequest represents the JSON body for creating a layout.
type createLayoutRequest struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	LanguageID   *int   `json:"language_id"`
	TemplateCode string `json:"template_code"`
	IsDefault    bool   `json:"is_default"`
}

// Create handles POST /layouts to create a new layout.
func (h *LayoutHandler) Create(c *fiber.Ctx) error {
	var req createLayoutRequest
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

	layout := models.Layout{
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		LanguageID:   req.LanguageID,
		TemplateCode: req.TemplateCode,
		Source:       "custom",
		IsDefault:    req.IsDefault,
	}

	if err := h.svc.Create(&layout); err != nil {
		if strings.Contains(err.Error(), "SLUG_CONFLICT") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", "A layout with this slug and language already exists")
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create layout")
	}

	return api.Created(c, layout)
}

// Update handles PATCH /layouts/:id to partially update a layout.
func (h *LayoutHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Layout ID must be a valid integer")
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
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Layout not found")
		}
		if strings.Contains(err.Error(), "THEME_READONLY") {
			return api.Error(c, fiber.StatusForbidden, "THEME_READONLY", "Theme layouts cannot be edited directly; detach first")
		}
		if strings.Contains(err.Error(), "SLUG_CONFLICT") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", "A layout with this slug and language already exists")
		}
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update layout")
	}

	return api.Success(c, updated)
}

// Delete handles DELETE /layouts/:id to remove a layout.
func (h *LayoutHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Layout ID must be a valid integer")
	}

	if err := h.svc.Delete(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Layout not found")
		}
		if strings.Contains(err.Error(), "THEME_READONLY") {
			return api.Error(c, fiber.StatusForbidden, "THEME_READONLY", "Theme layouts cannot be deleted; detach first")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete layout")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Detach handles POST /layouts/:id/detach to convert a theme layout to custom.
func (h *LayoutHandler) Detach(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Layout ID must be a valid integer")
	}

	layout, err := h.svc.Detach(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Layout not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DETACH_FAILED", "Failed to detach layout")
	}

	return api.Success(c, layout)
}

// Reattach handles POST /layouts/:id/reattach to restore a layout to its theme version.
func (h *LayoutHandler) Reattach(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Layout ID must be a valid integer")
	}

	layout, err := h.svc.Reattach(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Layout not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "REATTACH_FAILED", "Failed to reattach layout")
	}

	return api.Success(c, layout)
}
