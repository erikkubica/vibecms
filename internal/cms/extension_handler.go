package cms

import (
	"vibecms/internal/api"
	"vibecms/internal/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ExtensionHandler provides HTTP handlers for extension management.
type ExtensionHandler struct {
	db     *gorm.DB
	loader *ExtensionLoader
}

// NewExtensionHandler creates a new ExtensionHandler.
func NewExtensionHandler(db *gorm.DB, loader *ExtensionLoader) *ExtensionHandler {
	return &ExtensionHandler{db: db, loader: loader}
}

// RegisterRoutes registers all admin API extension routes on the provided router group.
func (h *ExtensionHandler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/extensions", auth.CapabilityRequired("manage_settings"))
	g.Get("/", h.List)
	g.Get("/:slug", h.Get)
	g.Post("/:slug/activate", h.Activate)
	g.Post("/:slug/deactivate", h.Deactivate)
}

// List handles GET /extensions — returns all extensions.
func (h *ExtensionHandler) List(c *fiber.Ctx) error {
	exts, err := h.loader.List()
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list extensions")
	}
	return api.Success(c, exts)
}

// Get handles GET /extensions/:slug — returns a single extension.
func (h *ExtensionHandler) Get(c *fiber.Ctx) error {
	slug := c.Params("slug")
	ext, err := h.loader.GetBySlug(slug)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch extension")
	}
	return api.Success(c, ext)
}

// Activate handles POST /extensions/:slug/activate.
func (h *ExtensionHandler) Activate(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if err := h.loader.Activate(slug); err != nil {
		if err.Error() == "extension not found: "+slug {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "ACTIVATE_FAILED", "Failed to activate extension")
	}
	return api.Success(c, fiber.Map{"message": "Extension activated"})
}

// Deactivate handles POST /extensions/:slug/deactivate.
func (h *ExtensionHandler) Deactivate(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if err := h.loader.Deactivate(slug); err != nil {
		if err.Error() == "extension not found: "+slug {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Extension not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DEACTIVATE_FAILED", "Failed to deactivate extension")
	}
	return api.Success(c, fiber.Map{"message": "Extension deactivated"})
}
