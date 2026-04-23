package api

import (
	"vibecms/internal/models"
	"vibecms/internal/sdui"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// BootHandler provides the boot manifest and SDUI layout endpoints for the admin SPA.
type BootHandler struct {
	db     *gorm.DB
	engine *sdui.Engine
}

// NewBootHandler creates a new BootHandler with the given database and SDUI engine.
func NewBootHandler(db *gorm.DB, engine *sdui.Engine) *BootHandler {
	return &BootHandler{db: db, engine: engine}
}

// Boot returns the full boot manifest for the authenticated session.
// GET /admin/api/boot
func (h *BootHandler) Boot(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok || user == nil {
		return Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	manifest, err := h.engine.GenerateBootManifest(user)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "BOOT_ERROR", "Failed to generate boot manifest")
	}

	return Success(c, manifest)
}

// Layout returns the SDUI layout tree for a given page slug.
// GET /admin/api/layout/:page
func (h *BootHandler) Layout(c *fiber.Ctx) error {
	pageSlug := c.Params("page")
	if pageSlug == "" {
		return Error(c, fiber.StatusBadRequest, "BAD_REQUEST", "Page slug required")
	}

	// Extract query params as context for layout generation.
	params := make(map[string]string)
	c.Request().URI().QueryArgs().VisitAll(func(key, val []byte) {
		params[string(key)] = string(val)
	})

	user, _ := c.Locals("user").(*models.User)
	userName := ""
	if user != nil && user.FullName != nil {
		userName = *user.FullName
	}
	if userName == "" && user != nil {
		userName = user.Email
	}

	layout, err := h.engine.GenerateLayout(pageSlug, params, userName)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "LAYOUT_ERROR", "Failed to generate layout")
	}

	return Success(c, layout)
}

// RegisterRoutes registers the boot and layout API routes on the given router.
func (h *BootHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/boot", h.Boot)
	router.Get("/layout/:page", h.Layout)
}
