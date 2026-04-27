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

// MenuHandler provides HTTP handlers for menu CRUD and tree operations.
type MenuHandler struct {
	svc *MenuService
}

// NewMenuHandler creates a new MenuHandler with the given MenuService.
func NewMenuHandler(svc *MenuService) *MenuHandler {
	return &MenuHandler{svc: svc}
}

// RegisterRoutes registers all menu routes on the provided router group.
// Reads are open to authenticated users; mutations require manage_menus.
func (h *MenuHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/menus", h.List)
	router.Get("/menus/:id", h.Get)
	manage := auth.CapabilityRequired("manage_menus")
	router.Post("/menus", manage, h.Create)
	router.Patch("/menus/:id", manage, h.Update)
	router.Delete("/menus/:id", manage, h.Delete)
	router.Put("/menus/:id/items", manage, h.ReplaceItems)
}

// List handles GET /menus to retrieve all menus with optional language filter.
func (h *MenuHandler) List(c *fiber.Ctx) error {
	var languageID *int
	if langIDStr := c.Query("language_id"); langIDStr != "" {
		id, err := strconv.Atoi(langIDStr)
		if err != nil {
			return api.Error(c, fiber.StatusBadRequest, "INVALID_LANGUAGE_ID", "language_id must be a valid integer")
		}
		languageID = &id
	}

	menus, err := h.svc.List(languageID)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list menus")
	}

	return api.Success(c, menus)
}

// Get handles GET /menus/:id to retrieve a single menu with nested items.
func (h *MenuHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Menu ID must be a valid integer")
	}

	menu, err := h.svc.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Menu not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch menu")
	}

	return api.Success(c, menu)
}

// createMenuRequest represents the JSON body for creating a menu.
type createMenuRequest struct {
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	LanguageID *int   `json:"language_id"`
}

// Create handles POST /menus to create a new menu.
func (h *MenuHandler) Create(c *fiber.Ctx) error {
	var req createMenuRequest
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
	if len(fields) > 0 {
		return api.ValidationError(c, fields)
	}

	menu := models.Menu{
		Slug:       req.Slug,
		Name:       req.Name,
		LanguageID: req.LanguageID,
	}

	if err := h.svc.Create(&menu); err != nil {
		if strings.Contains(err.Error(), "SLUG_CONFLICT") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", "A menu with this slug and language already exists")
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create menu")
	}

	return api.Created(c, menu)
}

// Update handles PATCH /menus/:id to partially update menu metadata.
func (h *MenuHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Menu ID must be a valid integer")
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Remove fields that should not be directly updated.
	delete(body, "id")
	delete(body, "created_at")
	delete(body, "updated_at")
	delete(body, "version")
	delete(body, "items")

	if len(body) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "NO_UPDATES", "No valid fields to update")
	}

	updated, err := h.svc.Update(id, body)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Menu not found")
		}
		if strings.Contains(err.Error(), "SLUG_CONFLICT") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", "A menu with this slug and language already exists")
		}
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update menu")
	}

	return api.Success(c, updated)
}

// Delete handles DELETE /menus/:id to remove a menu and its items.
func (h *MenuHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Menu ID must be a valid integer")
	}

	if err := h.svc.Delete(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Menu not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete menu")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// replaceItemsRequest represents the JSON body for replacing menu items.
type replaceItemsRequest struct {
	Version int                   `json:"version"`
	Items   []models.MenuItemTree `json:"items"`
}

// ReplaceItems handles PUT /menus/:id/items to atomically replace all menu items.
func (h *MenuHandler) ReplaceItems(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Menu ID must be a valid integer")
	}

	var req replaceItemsRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	if err := h.svc.ReplaceItems(id, req.Version, req.Items); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Menu not found")
		}
		if strings.Contains(err.Error(), "VERSION_CONFLICT") {
			return api.Error(c, fiber.StatusConflict, "VERSION_CONFLICT", "Menu has been modified by another user; refresh and retry")
		}
		if strings.Contains(err.Error(), "DEPTH_EXCEEDED") {
			return api.Error(c, fiber.StatusBadRequest, "DEPTH_EXCEEDED", err.Error())
		}
		return api.Error(c, fiber.StatusInternalServerError, "REPLACE_FAILED", "Failed to replace menu items")
	}

	// Return updated menu with new tree.
	menu, err := h.svc.GetByID(id)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Items replaced but failed to fetch updated menu")
	}

	return api.Success(c, menu)
}
