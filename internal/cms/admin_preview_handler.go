package cms

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"squilla/internal/api"
)

// RegisterAdminPreviewRoutes mounts the admin-side preview endpoint that the
// node editor's Preview button hits. Returns the rendered HTML for any node
// (drafts included) by reusing the same renderer the public path uses, so
// editors see exactly what visitors will see once published.
//
// GET /admin/api/nodes/:id/preview — text/html, 200 on success.
func (h *PublicHandler) RegisterAdminPreviewRoutes(router fiber.Router) {
	router.Get("/nodes/:id/preview", h.AdminNodePreview)
}

// AdminNodePreview renders one node by ID as full HTML. Auth is enforced by
// the parent /admin/api group's session middleware — there's no extra
// capability check because anyone with admin access can see any node anyway
// (drafts are only visible inside admin). Side effects are explicitly
// avoided: no view counts, no node.viewed events.
func (h *PublicHandler) AdminNodePreview(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Node id must be a positive integer")
	}
	html, err := h.RenderNodePreview(uint(id))
	if err != nil {
		return api.Error(c, fiber.StatusNotFound, "RENDER_FAILED", err.Error())
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	// Prevent caching — preview output reflects unsaved drafts and changes
	// frequently while editing.
	c.Set("Cache-Control", "no-store, max-age=0")
	return c.Status(fiber.StatusOK).SendString(html)
}
