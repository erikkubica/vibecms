package cms

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"squilla/internal/api"
	"squilla/internal/auth"
	"squilla/internal/models"
)

// RevisionHandler exposes browse + restore endpoints for content node
// revisions. Snapshots are written by ContentService.Update; this handler
// is the read/restore side. List/get are open to any authenticated admin
// (anyone editing nodes already has access to this data); restore is
// gated on the same capability as a regular node update.
type RevisionHandler struct {
	db          *gorm.DB
	contentSvc  *ContentService
}

// NewRevisionHandler creates a RevisionHandler.
func NewRevisionHandler(db *gorm.DB, contentSvc *ContentService) *RevisionHandler {
	return &RevisionHandler{db: db, contentSvc: contentSvc}
}

// RegisterRoutes mounts the revision routes on the admin API group.
func (h *RevisionHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/nodes/:id/revisions", h.List)
	router.Get("/nodes/:id/revisions/:revisionID", h.Get)
	// Restore reuses the same capability as a node update — restoring is
	// equivalent to a write that just happens to come from the past.
	router.Post(
		"/nodes/:id/revisions/:revisionID/restore",
		auth.CapabilityRequired("manage_content"),
		h.Restore,
	)
}

// revisionListItem is a slim shape returned by the list endpoint — full
// snapshots are heavy and only the editor's "browse" panel needs them on
// click.
type revisionListItem struct {
	ID            int64  `json:"id"`
	NodeID        int    `json:"node_id"`
	Title         string `json:"title"`
	Status        string `json:"status"`
	VersionNumber int    `json:"version_number"`
	CreatedBy     *int   `json:"created_by,omitempty"`
	CreatorName   string `json:"creator_name,omitempty"`
	CreatorEmail  string `json:"creator_email,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// List returns the revision history of a node, newest first. Limited to
// 100 rows — the retention sweep already caps growth, and an editor
// browsing back further than that should reach for diff tooling.
func (h *RevisionHandler) List(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid node id")
	}

	type row struct {
		ID            int64
		NodeID        int
		Title         string
		Status        string
		VersionNumber int
		CreatedBy     *int
		CreatorName   *string
		CreatorEmail  *string
		CreatedAt     string
	}
	var rows []row
	err = h.db.Table("content_node_revisions r").
		Select(`r.id, r.node_id, r.title, r.status, r.version_number,
		         r.created_by, u.name AS creator_name, u.email AS creator_email,
		         to_char(r.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at`).
		Joins("LEFT JOIN users u ON u.id = r.created_by").
		Where("r.node_id = ?", id).
		Order("r.created_at DESC").
		Limit(100).
		Scan(&rows).Error
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list revisions")
	}

	out := make([]revisionListItem, len(rows))
	for i, r := range rows {
		item := revisionListItem{
			ID:            r.ID,
			NodeID:        r.NodeID,
			Title:         r.Title,
			Status:        r.Status,
			VersionNumber: r.VersionNumber,
			CreatedBy:     r.CreatedBy,
			CreatedAt:     r.CreatedAt,
		}
		if r.CreatorName != nil {
			item.CreatorName = *r.CreatorName
		}
		if r.CreatorEmail != nil {
			item.CreatorEmail = *r.CreatorEmail
		}
		out[i] = item
	}
	return api.Success(c, out)
}

// Get returns the full snapshot for a single revision, including
// blocks_snapshot / fields_snapshot etc. so the admin UI can render a
// detail / diff view.
func (h *RevisionHandler) Get(c *fiber.Ctx) error {
	nodeID, err := strconv.Atoi(c.Params("id"))
	if err != nil || nodeID <= 0 {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid node id")
	}
	revID, err := strconv.ParseInt(c.Params("revisionID"), 10, 64)
	if err != nil || revID <= 0 {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_REVISION_ID", "Invalid revision id")
	}

	var rev models.ContentNodeRevision
	if err := h.db.
		Where("id = ? AND node_id = ?", revID, nodeID).
		First(&rev).Error; err != nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Revision not found")
	}
	return api.Success(c, rev)
}

// Restore writes a revision back onto the live node. The restore is
// itself a normal Update — meaning it lands as a NEW revision in the
// history (the prior live state becomes a recovery point of its own),
// never as a destructive overwrite.
func (h *RevisionHandler) Restore(c *fiber.Ctx) error {
	nodeID, err := strconv.Atoi(c.Params("id"))
	if err != nil || nodeID <= 0 {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid node id")
	}
	revID, err := strconv.ParseInt(c.Params("revisionID"), 10, 64)
	if err != nil || revID <= 0 {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_REVISION_ID", "Invalid revision id")
	}

	var rev models.ContentNodeRevision
	if err := h.db.
		Where("id = ? AND node_id = ?", revID, nodeID).
		First(&rev).Error; err != nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Revision not found")
	}

	updates := map[string]any{
		"title":          rev.Title,
		"status":         rev.Status,
		"language_code":  rev.LanguageCode,
		"excerpt":        rev.Excerpt,
		"featured_image": rev.FeaturedImage,
		"blocks_data":    rev.BlocksSnapshot,
		"fields_data":    rev.FieldsSnapshot,
		"seo_settings":   rev.SeoSnapshot,
		"taxonomies":     rev.TaxonomiesSnapshot,
	}
	if rev.LayoutSlug != nil {
		updates["layout_slug"] = *rev.LayoutSlug
	}
	if rev.Slug != "" {
		updates["slug"] = rev.Slug
	}

	userID := 0
	if u, ok := c.Locals("user_id").(int); ok {
		userID = u
	}
	node, err := h.contentSvc.Update(nodeID, updates, userID)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "RESTORE_FAILED", err.Error())
	}
	return api.Success(c, fiber.Map{
		"node":          node,
		"restored_from": rev.ID,
	})
}
