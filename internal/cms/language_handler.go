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

// LanguageHandler provides HTTP handlers for language CRUD operations.
type LanguageHandler struct {
	svc *LanguageService
}

// NewLanguageHandler creates a new LanguageHandler with the given LanguageService.
func NewLanguageHandler(svc *LanguageService) *LanguageHandler {
	return &LanguageHandler{svc: svc}
}

// RegisterRoutes registers all language routes on the provided router group.
// Reads are open (theme code reads the active language list); mutations
// require manage_settings.
func (h *LanguageHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/languages", h.List)
	router.Get("/languages/:id", h.Get)
	manage := auth.CapabilityRequired("manage_settings")
	router.Post("/languages", manage, h.Create)
	router.Patch("/languages/:id", manage, h.Update)
	router.Delete("/languages/:id", manage, h.Delete)
}

// List handles GET /languages to retrieve all languages.
// Supports ?active=true query param to return only active languages.
func (h *LanguageHandler) List(c *fiber.Ctx) error {
	activeOnly := c.Query("active")

	var err error
	if activeOnly == "true" {
		languages, err := h.svc.ListActive()
		if err != nil {
			return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list languages")
		}
		return api.Success(c, languages)
	}

	languages, err := h.svc.List()
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list languages")
	}

	return api.Success(c, languages)
}

// Get handles GET /languages/:id to retrieve a single language.
func (h *LanguageHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Language ID must be a valid integer")
	}

	lang, err := h.svc.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Language not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch language")
	}

	return api.Success(c, lang)
}

// createLanguageRequest represents the JSON body for creating a language.
type createLanguageRequest struct {
	Code       string `json:"code"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	NativeName string `json:"native_name"`
	Flag       string `json:"flag"`
	IsDefault  bool   `json:"is_default"`
	IsActive   *bool  `json:"is_active"`
	SortOrder  int    `json:"sort_order"`
}

// Create handles POST /languages to create a new language.
func (h *LanguageHandler) Create(c *fiber.Ctx) error {
	var req createLanguageRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	if req.Code == "" {
		return api.ValidationError(c, map[string]string{
			"code": "Code is required",
		})
	}
	if req.Name == "" {
		return api.ValidationError(c, map[string]string{
			"name": "Name is required",
		})
	}

	lang := models.Language{
		Code:       req.Code,
		Slug:       req.Slug,
		Name:       req.Name,
		NativeName: req.NativeName,
		Flag:       req.Flag,
		IsDefault:  req.IsDefault,
		SortOrder:  req.SortOrder,
	}

	// Default is_active to true if not explicitly set
	if req.IsActive != nil {
		lang.IsActive = *req.IsActive
	} else {
		lang.IsActive = true
	}

	if err := h.svc.Create(&lang); err != nil {
		if strings.Contains(err.Error(), "code conflict") {
			return api.Error(c, fiber.StatusConflict, "CODE_CONFLICT", err.Error())
		}
		if strings.Contains(err.Error(), "validation error") {
			return api.ValidationError(c, map[string]string{
				"code": err.Error(),
			})
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create language")
	}

	return api.Created(c, lang)
}

// Update handles PATCH /languages/:id to partially update a language.
func (h *LanguageHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Language ID must be a valid integer")
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Remove fields that should not be directly updated
	delete(body, "id")
	delete(body, "created_at")
	delete(body, "updated_at")

	if len(body) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "NO_UPDATES", "No valid fields to update")
	}

	updated, err := h.svc.Update(id, body)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Language not found")
		}
		if strings.Contains(err.Error(), "code conflict") {
			return api.Error(c, fiber.StatusConflict, "CODE_CONFLICT", err.Error())
		}
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update language")
	}

	return api.Success(c, updated)
}

// Delete handles DELETE /languages/:id to remove a language.
func (h *LanguageHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Language ID must be a valid integer")
	}

	if err := h.svc.Delete(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Language not found")
		}
		if strings.Contains(err.Error(), "cannot delete default") {
			return api.Error(c, fiber.StatusForbidden, "DEFAULT_LANGUAGE", err.Error())
		}
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete language")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
