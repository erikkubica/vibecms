package cms

import (
	"github.com/gofiber/fiber/v2"

	"vibecms/internal/api"
	"vibecms/internal/events"
)

// CacheHandler provides admin API endpoints for cache management.
type CacheHandler struct {
	publicHandler *PublicHandler
	eventBus      *events.EventBus
}

// NewCacheHandler creates a new CacheHandler.
func NewCacheHandler(ph *PublicHandler, eventBus *events.EventBus) *CacheHandler {
	return &CacheHandler{publicHandler: ph, eventBus: eventBus}
}

// RegisterRoutes registers cache management routes.
func (h *CacheHandler) RegisterRoutes(router fiber.Router) {
	router.Post("/cache/clear", h.ClearAll)
	router.Get("/cache/stats", h.Stats)
}

// ClearAll handles POST /cache/clear — clears all template and data caches.
func (h *CacheHandler) ClearAll(c *fiber.Ctx) error {
	h.publicHandler.ClearCache()
	go h.eventBus.Publish("sitemap.rebuild", events.Payload{})
	return api.Success(c, fiber.Map{"message": "All caches cleared"})
}

// Stats handles GET /cache/stats — returns cache statistics.
func (h *CacheHandler) Stats(c *fiber.Ctx) error {
	stats := h.publicHandler.CacheStats()
	return api.Success(c, stats)
}
