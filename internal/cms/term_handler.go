package cms

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/auth"
	"vibecms/internal/models"
)

// TermHandler provides HTTP handlers for taxonomy term CRUD.
type TermHandler struct {
	db *gorm.DB
}

// NewTermHandler creates a new TermHandler.
func NewTermHandler(db *gorm.DB) *TermHandler {
	return &TermHandler{db: db}
}

// RegisterRoutes registers all term routes on the provided router group.
// Reads are open; mutations require manage_settings (terms are scoped to
// taxonomy definitions, which require the same cap).
func (h *TermHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/terms/:nodeType/:taxonomy", h.List)
	router.Get("/terms/:id", h.Get)
	manage := auth.CapabilityRequired("manage_settings")
	router.Post("/terms/:nodeType/:taxonomy", manage, h.Create)
	router.Patch("/terms/:id", manage, h.Update)
	router.Delete("/terms/:id", manage, h.Delete)
}

// List returns all terms for a given node type and taxonomy.
func (h *TermHandler) List(c *fiber.Ctx) error {
	nodeType := c.Params("nodeType")
	taxonomy := c.Params("taxonomy")

	var terms []models.TaxonomyTerm
	if err := h.db.Where("node_type = ? AND taxonomy = ?", nodeType, taxonomy).
		Order("name ASC").Find(&terms).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch terms")
	}

	return api.Success(c, terms)
}

// Get returns a single term by ID.
func (h *TermHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Term ID must be a valid integer")
	}

	var term models.TaxonomyTerm
	if err := h.db.First(&term, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Term not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch term")
	}

	return api.Success(c, term)
}

// Create adds a new term for the given node type and taxonomy.
func (h *TermHandler) Create(c *fiber.Ctx) error {
	nodeType := c.Params("nodeType")
	taxonomy := c.Params("taxonomy")

	var term models.TaxonomyTerm
	if err := c.BodyParser(&term); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	term.NodeType = nodeType
	term.Taxonomy = taxonomy

	if term.Name == "" {
		return api.ValidationError(c, map[string]string{"name": "Name is required"})
	}

	if term.Slug == "" {
		term.Slug = slugify(term.Name)
	}

	if err := h.db.Create(&term).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", "A term with this slug already exists")
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create term")
	}

	return api.Created(c, term)
}

// Update modifies an existing term by ID.
func (h *TermHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Term ID must be a valid integer")
	}

	var existing models.TaxonomyTerm
	if err := h.db.First(&existing, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Term not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch term")
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	if v, ok := updates["name"].(string); ok && v != "" {
		existing.Name = v
	}
	if v, ok := updates["slug"].(string); ok && v != "" {
		existing.Slug = v
	}
	if v, ok := updates["description"].(string); ok {
		existing.Description = v
	}
	if v, ok := updates["fields_data"]; ok && v != nil {
		b, _ := json.Marshal(v)
		existing.FieldsData = models.JSONB(b)
	}

	if err := h.db.Save(&existing).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", "A term with this slug already exists")
		}
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update term")
	}

	return api.Success(c, existing)
}

// Delete removes a term by ID.
func (h *TermHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Term ID must be a valid integer")
	}

	result := h.db.Delete(&models.TaxonomyTerm{}, id)
	if result.Error != nil {
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete term")
	}
	if result.RowsAffected == 0 {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Term not found")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
