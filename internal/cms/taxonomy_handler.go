package cms

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/events"
	"vibecms/internal/models"
)

type TaxonomyHandler struct {
	db       *gorm.DB
	eventBus *events.EventBus
}

func NewTaxonomyHandler(db *gorm.DB, eventBus *events.EventBus) *TaxonomyHandler {
	return &TaxonomyHandler{db: db, eventBus: eventBus}
}

func (h *TaxonomyHandler) emit(action string, id int, slug string) {
	if h.eventBus == nil {
		return
	}
	h.eventBus.Publish(action, events.Payload{"id": id, "slug": slug})
}

func (h *TaxonomyHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/taxonomies", h.List)
	router.Get("/taxonomies/:slug", h.Get)
	router.Post("/taxonomies", h.Create)
	router.Patch("/taxonomies/:slug", h.Update)
	router.Delete("/taxonomies/:slug", h.Delete)
}

func (h *TaxonomyHandler) List(c *fiber.Ctx) error {
	var list []models.Taxonomy
	if err := h.db.Order("label ASC").Find(&list).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch taxonomies")
	}
	return api.Success(c, list)
}

func (h *TaxonomyHandler) Get(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var t models.Taxonomy
	if err := h.db.Where("slug = ?", slug).First(&t).Error; err != nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Taxonomy not found")
	}
	return api.Success(c, t)
}

func (h *TaxonomyHandler) Create(c *fiber.Ctx) error {
	var t models.Taxonomy
	if err := c.BodyParser(&t); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	if t.Slug == "" {
		t.Slug = slugify(t.Label)
	}

	if err := h.db.Create(&t).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", "Taxonomy slug already exists")
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create taxonomy")
	}

	h.emit("taxonomy.created", t.ID, t.Slug)
	return api.Created(c, t)
}

func (h *TaxonomyHandler) Update(c *fiber.Ctx) error {
	slug := c.Params("slug")

	var existing models.Taxonomy
	if err := h.db.Where("slug = ?", slug).First(&existing).Error; err != nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Taxonomy not found")
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Raw SQL — GORM cannot handle text[] updates properly
	label := existing.Label
	labelPlural := existing.LabelPlural
	description := existing.Description
	fieldSchema := string(existing.FieldSchema)
	nodeTypes := "{}"
	hierarchical := existing.Hierarchical
	showUI := existing.ShowUI

	if v, ok := updates["label"].(string); ok {
		label = v
	}
	if v, ok := updates["label_plural"].(string); ok {
		labelPlural = v
	}
	if v, ok := updates["description"].(string); ok {
		description = v
	}
	if v, ok := updates["hierarchical"].(bool); ok {
		hierarchical = v
	}
	if v, ok := updates["show_ui"].(bool); ok {
		showUI = v
	}
	if v, ok := updates["node_types"]; ok && v != nil {
		if arr, ok := v.([]interface{}); ok {
			strs := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					strs = append(strs, s)
				}
			}
			nodeTypes = "{" + strings.Join(strs, ",") + "}"
		}
	} else {
		// Keep existing
		parts := make([]string, len(existing.NodeTypes))
		for i, s := range existing.NodeTypes {
			parts[i] = s
		}
		nodeTypes = "{" + strings.Join(parts, ",") + "}"
	}
	if v, ok := updates["field_schema"]; ok && v != nil {
		b, _ := json.Marshal(v)
		fieldSchema = string(b)
	}

	err := h.db.Exec(
		`UPDATE taxonomies SET label=$1, label_plural=$2, description=$3, node_types=$4::text[], field_schema=$5::jsonb, hierarchical=$6, show_ui=$7, updated_at=NOW() WHERE slug=$8`,
		label, labelPlural, description, nodeTypes, fieldSchema, hierarchical, showUI, slug,
	).Error
	if err != nil {
		log.Printf("ERROR taxonomy update slug=%s: %v", slug, err)
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update taxonomy")
	}

	h.db.Where("slug = ?", slug).First(&existing)
	h.emit("taxonomy.updated", existing.ID, existing.Slug)
	return api.Success(c, existing)
}

func (h *TaxonomyHandler) Delete(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var existing models.Taxonomy
	if err := h.db.Where("slug = ?", slug).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Taxonomy not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch taxonomy")
	}
	if err := h.db.Delete(&existing).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete taxonomy")
	}
	h.emit("taxonomy.deleted", existing.ID, existing.Slug)
	return c.SendStatus(fiber.StatusNoContent)
}
