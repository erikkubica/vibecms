package cms

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/events"
	"vibecms/internal/models"
)

// SettingsHandler provides admin API endpoints for site settings management.
type SettingsHandler struct {
	db       *gorm.DB
	eventBus *events.EventBus
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(db *gorm.DB, eventBus *events.EventBus) *SettingsHandler {
	return &SettingsHandler{db: db, eventBus: eventBus}
}

// RegisterRoutes registers settings routes on the admin API router.
func (h *SettingsHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/settings", h.List)
	router.Put("/settings", h.BulkUpdate)
}

// List handles GET /settings — returns all site settings as a key-value map.
func (h *SettingsHandler) List(c *fiber.Ctx) error {
	prefix := c.Query("prefix", "")

	var settings []models.SiteSetting
	q := h.db.Order("key ASC")
	if prefix != "" {
		q = q.Where("key LIKE ?", prefix+"%")
	}
	if err := q.Find(&settings).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list settings")
	}

	result := make(map[string]string)
	for _, s := range settings {
		if s.Value != nil {
			result[s.Key] = *s.Value
		} else {
			result[s.Key] = ""
		}
	}

	return api.Success(c, result)
}

// BulkUpdate handles PUT /settings — updates multiple settings at once.
func (h *SettingsHandler) BulkUpdate(c *fiber.Ctx) error {
	var body map[string]string
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	for key, value := range body {
		v := value
		setting := models.SiteSetting{
			Key:   key,
			Value: &v,
		}
		h.db.Where("key = ?", key).Assign(setting).FirstOrCreate(&setting)
	}

	if h.eventBus != nil {
		go h.eventBus.Publish("setting.updated", events.Payload{
			"bulk": true,
		})
	}

	return api.Success(c, fiber.Map{"message": "Settings updated"})
}
