package cms

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"squilla/internal/api"
	"squilla/internal/auth"
	"squilla/internal/events"
	"squilla/internal/models"
	"squilla/internal/secrets"
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

	// Resolve per-key with locale fallback: prefer the admin's current
	// language; fall back to the default language for keys that haven't
	// been overridden in that locale yet. The sentinel "*" means
	// language-agnostic — read only the empty-locale row, used by
	// settings that intentionally span every language (e.g. security).
	locale := string(c.Request().Header.Peek("X-Admin-Language"))
	def := h.defaultLanguageCode()
	if locale == "" {
		locale = def
	}

	q := h.db.Order("key ASC")
	if prefix != "" {
		q = q.Where("key LIKE ?", prefix+"%")
	}
	switch {
	case locale == "*":
		q = q.Where("language_code = ?", "")
	case locale == "" && def == "":
		// No languages seeded yet — return empty rather than dump every
		// row regardless of language.
		return api.Success(c, map[string]any{})
	case def == "" || locale == def:
		q = q.Where("language_code = ?", locale)
	default:
		q = q.Where("language_code IN ?", []string{locale, def})
	}

	var settings []models.SiteSetting
	if err := q.Find(&settings).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list settings")
	}

	type pick struct {
		raw    string
		locale string
	}
	chosen := make(map[string]pick, len(settings))
	for _, s := range settings {
		raw := ""
		if s.Value != nil {
			raw = *s.Value
		}
		existing, ok := chosen[s.Key]
		if !ok || (existing.locale == def && s.LanguageCode == locale) {
			chosen[s.Key] = pick{raw: raw, locale: s.LanguageCode}
		}
	}

	result := make(map[string]any, len(chosen))
	for key, p := range chosen {
		if secrets.IsSecretKey(key) {
			result[key] = settingValue{Value: "***", IsSet: p.raw != ""}
			continue
		}
		result[key] = p.raw
	}

	return api.Success(c, result)
}

// defaultLanguageCode returns the code of the language flagged is_default=true,
// or "" before any language has been seeded.
func (h *SettingsHandler) defaultLanguageCode() string {
	var code string
	_ = h.db.Table("languages").Select("code").Where("is_default = ?", true).Limit(1).Scan(&code).Error
	return code
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

	// Resolve write locale: admin's selected language, falling back to the
	// site default. Every settings row carries a real language code now, so
	// an empty locale is only possible on a fresh install before any
	// language has been seeded — in that case we skip the write rather than
	// stash data under "".
	locale := string(c.Request().Header.Peek("X-Admin-Language"))
	if locale == "*" {
		// Language-agnostic write — store under the empty-locale row so
		// reads with any locale see the same value. Used by settings whose
		// behaviour can't sensibly differ per language (security, etc.).
		locale = ""
	} else {
		if locale == "" {
			locale = h.defaultLanguageCode()
		}
		if locale == "" {
			return api.Error(c, fiber.StatusBadRequest, "NO_LANGUAGE",
				"No language is configured; create a default language before saving settings")
		}
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
		row := models.SiteSetting{Key: key, LanguageCode: locale, Value: &v}
		if err := h.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}, {Name: "language_code"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).Create(&row).Error; err != nil {
			return api.Error(c, fiber.StatusInternalServerError, "WRITE_FAILED", "Failed to persist setting")
		}
	}

	if h.eventBus != nil {
		go h.eventBus.Publish("setting.updated", events.Payload{
			"bulk": true,
		})
	}

	return api.Success(c, fiber.Map{"message": "Settings updated"})
}
