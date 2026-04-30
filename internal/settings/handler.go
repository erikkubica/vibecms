package settings

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"squilla/internal/api"
	"squilla/internal/auth"
	"squilla/internal/events"
)

// Handler exposes the schema registry and per-schema values to the admin
// SPA. The legacy /admin/api/settings endpoint stays mounted (sidebar,
// public render context, and pre-registry callers still rely on its
// flat key-value shape); this handler is the schema-aware layer added
// alongside it under /admin/api/settings/schemas.
type Handler struct {
	registry *Registry
	store    *Store
	db       *gorm.DB
	eventBus *events.EventBus
}

// NewHandler constructs the schema HTTP handler.
func NewHandler(registry *Registry, store *Store, db *gorm.DB, eventBus *events.EventBus) *Handler {
	return &Handler{registry: registry, store: store, db: db, eventBus: eventBus}
}

// RegisterRoutes mounts the schema endpoints on the given admin API
// router. Reads require any authenticated session (capability is
// enforced per-schema below); writes additionally require the
// schema's declared capability, falling back to manage_settings.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/settings/schemas", h.List)
	router.Get("/settings/schemas/:id", h.Get)
	router.Put("/settings/schemas/:id", auth.CapabilityRequired("manage_settings"), h.Save)
}

// listEntry is the slim shape returned by GET /settings/schemas — just
// enough for the admin UI to render a directory of available surfaces
// without paying the cost of every field definition.
type listEntry struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	Description       string `json:"description,omitempty"`
	Capability        string `json:"capability,omitempty"`
	HasTranslatable   bool   `json:"has_translatable"`
}

// List handles GET /settings/schemas.
func (h *Handler) List(c *fiber.Ctx) error {
	schemas := h.registry.List()
	out := make([]listEntry, 0, len(schemas))
	for _, s := range schemas {
		out = append(out, listEntry{
			ID:              s.ID,
			Title:           s.Title,
			Description:     s.Description,
			Capability:      s.Capability,
			HasTranslatable: s.HasTranslatable(),
		})
	}
	return api.Success(c, out)
}

// schemaEnvelope is the response shape for GET /settings/schemas/:id —
// the schema itself plus current values resolved against the requested
// locale (with non-translatable fields read from the empty-locale row).
type schemaEnvelope struct {
	Schema Schema            `json:"schema"`
	Values map[string]string `json:"values"`
	Locale string            `json:"locale"`
}

// Get handles GET /settings/schemas/:id?locale=...
//
// locale resolution:
//   - X-Admin-Language header wins (mirrors the legacy handler so the
//     admin shell's locale picker keeps working without per-endpoint
//     plumbing).
//   - Falls back to ?locale= query param.
//   - Falls back to the site's default language.
//   - Empty string is allowed only when the schema has no translatable
//     fields; otherwise the response uses the default and the UI
//     surfaces a "set up languages" hint.
func (h *Handler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	schema, ok := h.registry.Get(id)
	if !ok {
		return api.Error(c, fiber.StatusNotFound, "SCHEMA_NOT_FOUND", "Settings schema not found")
	}
	if schema.Capability != "" {
		if err := requireCapability(c, schema.Capability); err != nil {
			return err
		}
	}

	locale := resolveLocale(c, h.db)
	values, err := h.store.Load(schema, locale)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LOAD_FAILED", err.Error())
	}
	return api.Success(c, schemaEnvelope{Schema: schema, Values: values, Locale: locale})
}

// saveBody is the payload accepted by PUT /settings/schemas/:id.
type saveBody struct {
	Values map[string]string `json:"values"`
}

// Save handles PUT /settings/schemas/:id.
//
// Translatable fields require a locale. We reject the write rather
// than silently routing to the default language because the admin UI
// is expected to send X-Admin-Language alongside the body.
func (h *Handler) Save(c *fiber.Ctx) error {
	id := c.Params("id")
	schema, ok := h.registry.Get(id)
	if !ok {
		return api.Error(c, fiber.StatusNotFound, "SCHEMA_NOT_FOUND", "Settings schema not found")
	}
	if schema.Capability != "" {
		if err := requireCapability(c, schema.Capability); err != nil {
			return err
		}
	}

	var body saveBody
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	locale := resolveLocale(c, h.db)
	if locale == "" && schema.HasTranslatable() {
		return api.Error(c, fiber.StatusBadRequest, "NO_LANGUAGE",
			"No language is configured; create a default language before saving translatable settings")
	}

	if err := h.store.Save(schema, locale, body.Values); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "SAVE_FAILED", err.Error())
	}

	if h.eventBus != nil {
		go h.eventBus.Publish("setting.updated", events.Payload{
			"schema_id": schema.ID,
			"keys":      keysOf(body.Values),
			"locale":    locale,
		})
	}
	return api.Success(c, fiber.Map{"message": "Settings updated"})
}

// resolveLocale picks the request's effective locale: header wins, then
// query, then the site's default language. Returns "" when nothing is
// configured (fresh install before language seed).
func resolveLocale(c *fiber.Ctx, db *gorm.DB) string {
	if h := string(c.Request().Header.Peek("X-Admin-Language")); h != "" && h != "*" {
		return h
	}
	if q := c.Query("locale"); q != "" && q != "*" {
		return q
	}
	var code string
	_ = db.Table("languages").Select("code").Where("is_default = ?", true).Limit(1).Scan(&code).Error
	return code
}

// requireCapability is a per-schema gate. We don't use the auth middleware
// directly because the required capability is data-driven (declared by
// the schema) rather than known at route-registration time.
func requireCapability(c *fiber.Ctx, cap string) error {
	user := auth.GetCurrentUser(c)
	if user == nil {
		return api.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	if !auth.HasCapability(user, cap) {
		return api.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Capability "+cap+" required")
	}
	return nil
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
