package cms

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"squilla/internal/api"
	"squilla/internal/auth"
	"squilla/internal/models"
)

// NodeTypeHandler provides HTTP handlers for node type CRUD operations.
type NodeTypeHandler struct {
	svc *NodeTypeService
}

// NewNodeTypeHandler creates a new NodeTypeHandler with the given NodeTypeService.
func NewNodeTypeHandler(svc *NodeTypeService) *NodeTypeHandler {
	return &NodeTypeHandler{svc: svc}
}

// RegisterRoutes registers all node type routes on the provided router group.
// Reads are open (lots of admin-UI flows enumerate types); mutations
// require manage_settings (node types are structural site config).
func (h *NodeTypeHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/node-types", h.List)
	router.Get("/node-types/:id", h.Get)
	manage := auth.CapabilityRequired("manage_settings")
	router.Post("/node-types", manage, h.Create)
	router.Patch("/node-types/:id", manage, h.Update)
	router.Delete("/node-types/:id", manage, h.Delete)
}

// List handles GET /node-types to retrieve paginated node types.
func (h *NodeTypeHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	nodeTypes, total, err := h.svc.List(page, perPage)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list node types")
	}

	// Same merge as Get — the node editor's taxonomy sidebar reads from
	// this list response, so theme/extension-registered taxonomies need to
	// land here too, not just on the single-node-type GET.
	for i := range nodeTypes {
		mergeRegisteredTaxonomies(h.svc.DB(), &nodeTypes[i])
	}

	return api.Paginated(c, nodeTypes, total, page, perPage)
}

// Get handles GET /node-types/:id to retrieve a single node type.
func (h *NodeTypeHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Node type ID must be a valid integer")
	}

	nt, err := h.svc.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Node type not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch node type")
	}

	// Merge taxonomies registered through the standalone `taxonomies` table
	// (theme/extension registrations land here with field_schema, hierarchy
	// flags, etc.) into the JSONB list the editor reads. Without this, a
	// theme-registered taxonomy like `doc_section` shows up in the sidebar
	// nav but is missing from the node-type's Taxonomies tab.
	mergeRegisteredTaxonomies(h.svc.DB(), nt)

	return api.Success(c, nt)
}

// mergeRegisteredTaxonomies pulls every row from the `taxonomies` table
// whose node_types array contains nt.Slug and folds it into nt.Taxonomies
// (the JSONB list the editor renders). Existing entries with the same slug
// are preserved so any node-type-local overrides win; only missing rows are
// appended.
func mergeRegisteredTaxonomies(db *gorm.DB, nt *models.NodeType) {
	if db == nil || nt == nil {
		return
	}
	var rows []models.Taxonomy
	if err := db.Where("? = ANY (node_types)", nt.Slug).Find(&rows).Error; err != nil {
		return
	}
	if len(rows) == 0 {
		return
	}

	type taxEntry struct {
		Slug        string `json:"slug"`
		Label       string `json:"label"`
		LabelPlural string `json:"label_plural,omitempty"`
		Multiple    bool   `json:"multiple,omitempty"`
	}
	var existing []taxEntry
	if len(nt.Taxonomies) > 0 {
		_ = json.Unmarshal([]byte(nt.Taxonomies), &existing)
	}
	have := make(map[string]struct{}, len(existing))
	for _, e := range existing {
		have[e.Slug] = struct{}{}
	}
	for _, r := range rows {
		if _, ok := have[r.Slug]; ok {
			continue
		}
		existing = append(existing, taxEntry{
			Slug:        r.Slug,
			Label:       r.Label,
			LabelPlural: r.LabelPlural,
			Multiple:    !r.Hierarchical,
		})
	}
	if b, err := json.Marshal(existing); err == nil {
		nt.Taxonomies = models.JSONB(b)
	}
}

// createNodeTypeRequest represents the JSON body for creating a node type.
type createNodeTypeRequest struct {
	Slug           string       `json:"slug"`
	Label          string       `json:"label"`
	LabelPlural    string       `json:"label_plural"`
	Icon           string       `json:"icon"`
	Description    string       `json:"description"`
	Taxonomies     models.JSONB `json:"taxonomies"`
	FieldSchema    models.JSONB `json:"field_schema"`
	URLPrefixes    models.JSONB `json:"url_prefixes"`
	SupportsBlocks *bool        `json:"supports_blocks"`
}

// Create handles POST /node-types to create a new node type.
func (h *NodeTypeHandler) Create(c *fiber.Ctx) error {
	var req createNodeTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	if req.Slug == "" {
		return api.ValidationError(c, map[string]string{
			"slug": "Slug is required",
		})
	}
	if req.Label == "" {
		return api.ValidationError(c, map[string]string{
			"label": "Label is required",
		})
	}

	nt := models.NodeType{
		Slug:           req.Slug,
		Label:          req.Label,
		LabelPlural:    req.LabelPlural,
		Icon:           req.Icon,
		Description:    req.Description,
		Taxonomies:     req.Taxonomies,
		FieldSchema:    req.FieldSchema,
		URLPrefixes:    req.URLPrefixes,
		SupportsBlocks: true,
	}
	if req.SupportsBlocks != nil {
		nt.SupportsBlocks = *req.SupportsBlocks
	}

	if nt.Icon == "" {
		nt.Icon = "file-text"
	}
	if len(nt.Taxonomies) == 0 {
		nt.Taxonomies = models.JSONB("[]")
	}
	if len(nt.FieldSchema) == 0 {
		nt.FieldSchema = models.JSONB("[]")
	}
	if len(nt.URLPrefixes) == 0 {
		nt.URLPrefixes = models.JSONB("{}")
	}

	if err := h.svc.Create(&nt); err != nil {
		if strings.Contains(err.Error(), "slug conflict") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", err.Error())
		}
		if strings.Contains(err.Error(), "validation error") {
			return api.ValidationError(c, map[string]string{
				"slug": err.Error(),
			})
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create node type")
	}

	return api.Created(c, nt)
}

// Update handles PATCH /node-types/:id to partially update a node type.
func (h *NodeTypeHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Node type ID must be a valid integer")
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Remove fields that should not be directly updated
	delete(body, "id")
	delete(body, "created_at")
	delete(body, "updated_at")

	if len(body) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "NO_UPDATES", "No valid fields to update")
	}

	updated, err := h.svc.Update(id, body)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Node type not found")
		}
		if strings.Contains(err.Error(), "slug conflict") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", err.Error())
		}
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update node type")
	}

	return api.Success(c, updated)
}

// Delete handles DELETE /node-types/:id to remove a node type.
func (h *NodeTypeHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Node type ID must be a valid integer")
	}

	if err := h.svc.Delete(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Node type not found")
		}
		if strings.Contains(err.Error(), "cannot delete built-in") {
			return api.Error(c, fiber.StatusForbidden, "BUILTIN_TYPE", err.Error())
		}
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete node type")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
