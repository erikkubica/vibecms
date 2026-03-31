package cms

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/models"
)

// TemplateHandler provides HTTP handlers for template CRUD operations.
type TemplateHandler struct {
	svc *TemplateService
}

// NewTemplateHandler creates a new TemplateHandler with the given TemplateService.
func NewTemplateHandler(svc *TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: svc}
}

// RegisterRoutes registers all template routes on the provided router group.
func (h *TemplateHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/templates", h.List)
	router.Get("/templates/:id", h.Get)
	router.Post("/templates", h.Create)
	router.Patch("/templates/:id", h.Update)
	router.Delete("/templates/:id", h.Delete)
	router.Post("/templates/:id/detach", h.Detach)
	router.Post("/templates/:id/reattach", h.Reattach)
}

// List handles GET /templates to retrieve all templates with pagination.
func (h *TemplateHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	templates, total, err := h.svc.List(page, perPage)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list templates")
	}

	return api.Paginated(c, templates, total, page, perPage)
}

// Get handles GET /templates/:id to retrieve a single template.
func (h *TemplateHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Template ID must be a valid integer")
	}

	t, err := h.svc.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Template not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch template")
	}

	return api.Success(c, t)
}

// createTemplateRequest represents the JSON body for creating a template.
type createTemplateRequest struct {
	Slug        string       `json:"slug"`
	Label       string       `json:"label"`
	Description string       `json:"description"`
	BlockConfig models.JSONB `json:"block_config"`
}

// Create handles POST /templates to create a new template.
func (h *TemplateHandler) Create(c *fiber.Ctx) error {
	var req createTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	if req.Slug == "" {
		return api.ValidationError(c, map[string]string{
			"slug": "Slug is required",
		})
	}
	if req.Label == "" {
		return api.ValidationError(c, map[string]string{
			"label": "Label is required",
		})
	}

	t := models.Template{
		Slug:        req.Slug,
		Label:       req.Label,
		Description: req.Description,
		BlockConfig: req.BlockConfig,
	}

	if len(t.BlockConfig) == 0 {
		t.BlockConfig = models.JSONB("[]")
	}

	if err := h.svc.Create(&t); err != nil {
		if strings.Contains(err.Error(), "slug conflict") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", err.Error())
		}
		if strings.Contains(err.Error(), "validation error") {
			return api.ValidationError(c, map[string]string{
				"slug": err.Error(),
			})
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create template")
	}

	return api.Created(c, t)
}

// Update handles PATCH /templates/:id to partially update a template.
func (h *TemplateHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Template ID must be a valid integer")
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Remove fields that should not be directly updated
	delete(body, "id")
	delete(body, "created_at")
	delete(body, "updated_at")

	// Check for theme-readonly
	existing, err := h.svc.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Template not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch template")
	}
	if existing.Source != "custom" {
		return api.Error(c, fiber.StatusForbidden, "THEME_READONLY", "Theme/extension templates cannot be edited directly; detach first")
	}

	if len(body) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "NO_UPDATES", "No valid fields to update")
	}

	updated, err := h.svc.Update(id, body)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Template not found")
		}
		if strings.Contains(err.Error(), "slug conflict") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", err.Error())
		}
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update template")
	}

	return api.Success(c, updated)
}

// Delete handles DELETE /templates/:id to remove a template.
func (h *TemplateHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Template ID must be a valid integer")
	}

	existing, err := h.svc.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Template not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch template")
	}
	if existing.Source != "custom" {
		return api.Error(c, fiber.StatusForbidden, "THEME_READONLY", "Theme/extension templates cannot be deleted; detach first")
	}

	if err := h.svc.Delete(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Template not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete template")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Detach handles POST /templates/:id/detach to convert a theme template to custom.
func (h *TemplateHandler) Detach(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Template ID must be a valid integer")
	}

	t, err := h.svc.Detach(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Template not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DETACH_FAILED", "Failed to detach template")
	}

	return api.Success(c, t)
}

// Reattach handles POST /templates/:id/reattach to restore a template to its theme version.
func (h *TemplateHandler) Reattach(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Template ID must be a valid integer")
	}

	t, err := h.svc.Reattach(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Template not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "REATTACH_FAILED", "Failed to reattach template")
	}

	return api.Success(c, t)
}
