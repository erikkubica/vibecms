package cms

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/auth"
	"vibecms/internal/events"
	"vibecms/internal/models"
	"vibecms/internal/secrets"
)

// SettingsHandler provides admin API endpoints for site settings management.
// It owns the secrets.Service so writes to credential-like keys land in
// the database as AES-GCM envelopes rather than plaintext.
type SettingsHandler struct {
	db       *gorm.DB
	eventBus *events.EventBus
	secrets  *secrets.Service
}

// NewSettingsHandler creates a SettingsHandler. Pass a non-nil
// *secrets.Service to opt into at-rest encryption for credential-like
// keys; passing nil keeps writes plaintext (legacy behaviour).
func NewSettingsHandler(db *gorm.DB, eventBus *events.EventBus, secretsSvc *secrets.Service) *SettingsHandler {
	return &SettingsHandler{db: db, eventBus: eventBus, secrets: secretsSvc}
}

// RegisterRoutes registers settings routes on the admin API router.
// Reads are open to any authenticated user (sidebar status, language list,
// site name reads, etc.); writes require manage_settings.
func (h *SettingsHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/settings", h.List)
	router.Put("/settings", auth.CapabilityRequired("manage_settings"), h.BulkUpdate)
}

// settingValue is the per-key shape returned by List. value is "***" for
// secret-looking keys; is_set indicates whether a real value is stored.
// Non-secret keys return the actual value with is_set unused.
type settingValue struct {
	Value string `json:"value"`
	IsSet bool   `json:"is_set,omitempty"`
}

// List handles GET /settings — returns all site settings as a key-value map.
// Secret-looking keys (those matching secretKeySuffixes) are redacted on
// the wire so an over-broad GET doesn't leak SMTP passwords, API keys,
// OAuth tokens, etc. The actual values stay in the DB and are still used
// by services that read settings directly via models.SiteSetting.
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

	result := make(map[string]any)
	for _, s := range settings {
		raw := ""
		if s.Value != nil {
			raw = *s.Value
		}
		if secrets.IsSecretKey(s.Key) {
			// `is_set` accounts for either a plaintext non-empty value or
			// a stored ciphertext envelope; both indicate "the operator
			// has provided this credential".
			result[s.Key] = settingValue{Value: "***", IsSet: raw != ""}
			continue
		}
		result[s.Key] = raw
	}

	return api.Success(c, result)
}

// BulkUpdate handles PUT /settings — updates multiple settings at once.
// Skips redaction-marker writes ("***") to avoid the common admin-UI
// pattern where the masked value is round-tripped from GET to PUT and
// would otherwise overwrite the real secret.
func (h *SettingsHandler) BulkUpdate(c *fiber.Ctx) error {
	var body map[string]string
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	for key, value := range body {
		stored := value
		if secrets.IsSecretKey(key) {
			// Skip the redaction marker — it means the admin UI fetched
			// the masked value and saved without editing. Overwriting
			// with "***" would destroy the real secret.
			if value == "***" {
				continue
			}
			if h.secrets != nil {
				enc, err := h.secrets.MaybeEncrypt(value)
				if err != nil {
					return api.Error(c, fiber.StatusInternalServerError, "ENCRYPT_FAILED", "Failed to encrypt secret setting")
				}
				stored = enc
			}
		}
		v := stored
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
