package cms

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"squilla/internal/api"
	"squilla/internal/auth"
	"squilla/internal/models"
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
	manage := auth.CapabilityRequired("manage_settings")
	// Translations live at /terms/:id/translations. They must be registered
	// before the generic /terms/:nodeType/:taxonomy because Fiber matches by
	// registration order for routes of equal segment count, and a request
	// like POST /terms/278/translations would otherwise resolve to Create
	// with taxonomy="translations".
	router.Get("/terms/:id/translations", h.GetTranslations)
	router.Post("/terms/:id/translations", manage, h.CreateTranslation)

	router.Get("/terms/:id", h.Get)
	router.Patch("/terms/:id", manage, h.Update)
	router.Delete("/terms/:id", manage, h.Delete)

	router.Get("/terms/:nodeType/:taxonomy", h.List)
	router.Post("/terms/:nodeType/:taxonomy", manage, h.Create)
}

// termLocale resolves the language a term request is scoped to. Priority:
// explicit ?language_code= query param, then X-Admin-Language header, then
// the site default language. Returns "" only on a fresh install with no
// languages seeded — callers fall back to "no filter" in that case.
func (h *TermHandler) termLocale(c *fiber.Ctx) string {
	if v := c.Query("language_code"); v != "" {
		return v
	}
	if v := string(c.Request().Header.Peek("X-Admin-Language")); v != "" {
		return v
	}
	var def string
	_ = h.db.Table("languages").Select("code").Where("is_default = ?", true).Limit(1).Scan(&def).Error
	return def
}

// List returns all terms for a given node type and taxonomy. Scoped to the
// caller's current language unless ?language_code=all is passed.
func (h *TermHandler) List(c *fiber.Ctx) error {
	nodeType := c.Params("nodeType")
	taxonomy := c.Params("taxonomy")

	q := h.db.Where("node_type = ? AND taxonomy = ?", nodeType, taxonomy)
	if locale := h.termLocale(c); locale != "" && c.Query("language_code") != "all" {
		q = q.Where("language_code = ?", locale)
	}

	var terms []models.TaxonomyTerm
	if err := q.Order("name ASC").Find(&terms).Error; err != nil {
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

	// Default the language to the admin's current locale (or site default).
	// Body-supplied language_code wins so seeders/scripts can pin explicitly.
	if term.LanguageCode == "" {
		term.LanguageCode = h.termLocale(c)
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
	// Language is mutable: operators may relabel a term they originally
	// created in the wrong locale. Slug uniqueness is per-language, so the
	// DB will surface a SLUG_CONFLICT if another term in the target
	// language already owns this slug.
	if v, ok := updates["language_code"].(string); ok && v != "" {
		existing.LanguageCode = v
	}

	if err := h.db.Save(&existing).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", "A term with this slug already exists")
		}
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update term")
	}

	return api.Success(c, existing)
}

// GetTranslations returns the sibling translations of a term — every row
// sharing this term's translation_group_id, excluding the term itself. An
// ungrouped term (no translation_group_id yet) returns an empty slice.
func (h *TermHandler) GetTranslations(c *fiber.Ctx) error {
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
	if term.TranslationGroupID == nil || *term.TranslationGroupID == "" {
		return api.Success(c, []models.TaxonomyTerm{})
	}

	var siblings []models.TaxonomyTerm
	if err := h.db.Where("translation_group_id = ? AND id != ?", *term.TranslationGroupID, id).
		Order("language_code ASC").Find(&siblings).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch translations")
	}
	return api.Success(c, siblings)
}

type createTermTranslationRequest struct {
	LanguageCode string `json:"language_code"`
}

// CreateTranslation clones an existing term as a translation in another
// language. The source row gets a fresh translation_group_id if it didn't
// have one yet, and the new row is linked via the same group. Slug is
// reused (per-language uniqueness allows it); operators can rename in the
// editor afterwards.
func (h *TermHandler) CreateTranslation(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Term ID must be a valid integer")
	}

	var req createTermTranslationRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}
	if req.LanguageCode == "" {
		return api.ValidationError(c, map[string]string{"language_code": "Language code is required"})
	}

	var source models.TaxonomyTerm
	if err := h.db.First(&source, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Term not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch term")
	}
	if source.LanguageCode == req.LanguageCode {
		return api.Error(c, fiber.StatusConflict, "SAME_LANGUAGE",
			"Source term is already in the requested language")
	}

	// Don't create a duplicate sibling in the target language.
	if source.TranslationGroupID != nil && *source.TranslationGroupID != "" {
		var existing int64
		h.db.Model(&models.TaxonomyTerm{}).
			Where("translation_group_id = ? AND language_code = ?", *source.TranslationGroupID, req.LanguageCode).
			Count(&existing)
		if existing > 0 {
			return api.Error(c, fiber.StatusConflict, "TRANSLATION_EXISTS",
				fmt.Sprintf("Translation in %s already exists", req.LanguageCode))
		}
	}

	// Stamp the source with a translation group if it doesn't have one yet,
	// so the new row can join. We do this in the same TX so a partial
	// failure can't orphan the source.
	tx := h.db.Begin()
	if source.TranslationGroupID == nil || *source.TranslationGroupID == "" {
		gid := uuid.New().String()
		source.TranslationGroupID = &gid
		if err := tx.Model(&source).Update("translation_group_id", gid).Error; err != nil {
			tx.Rollback()
			return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to assign translation group")
		}
	}

	clone := models.TaxonomyTerm{
		NodeType:           source.NodeType,
		Taxonomy:           source.Taxonomy,
		LanguageCode:       req.LanguageCode,
		TranslationGroupID: source.TranslationGroupID,
		Slug:               source.Slug,
		Name:               source.Name,
		Description:        source.Description,
		ParentID:           source.ParentID,
		FieldsData:         source.FieldsData,
	}
	if err := tx.Create(&clone).Error; err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "duplicate key") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", "A term with this slug already exists in the target language")
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create translation")
	}
	if err := tx.Commit().Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to commit translation")
	}

	return api.Created(c, clone)
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
