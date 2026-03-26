package rbac

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/auth"
	"vibecms/internal/models"
)

// RoleHandler provides admin API endpoints for role management.
// All endpoints require "manage_roles" capability.
type RoleHandler struct {
	db *gorm.DB
}

// NewRoleHandler creates a new RoleHandler with the given database connection.
func NewRoleHandler(db *gorm.DB) *RoleHandler {
	return &RoleHandler{db: db}
}

// RegisterRoutes registers all role and system-action routes on the provided router group.
func (h *RoleHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/system-actions", h.ListSystemActions)

	roles := router.Group("/roles", auth.CapabilityRequired("manage_roles"))
	roles.Get("/", h.List)
	roles.Get("/:id", h.Get)
	roles.Post("/", h.Create)
	roles.Patch("/:id", h.Update)
	roles.Delete("/:id", h.Delete)
}

// List handles GET /roles to retrieve all roles.
func (h *RoleHandler) List(c *fiber.Ctx) error {
	var roles []models.Role
	if err := h.db.Order("id ASC").Find(&roles).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list roles")
	}
	return api.Success(c, roles)
}

// Get handles GET /roles/:id to retrieve a single role.
func (h *RoleHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Role ID must be a valid integer")
	}

	var role models.Role
	if err := h.db.First(&role, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Role not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch role")
	}

	return api.Success(c, role)
}

// createRoleRequest represents the JSON body for creating a role.
type createRoleRequest struct {
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Capabilities json.RawMessage `json:"capabilities"`
}

// Create handles POST /roles to create a new role.
func (h *RoleHandler) Create(c *fiber.Ctx) error {
	var req createRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	fields := map[string]string{}
	if req.Slug == "" {
		fields["slug"] = "Slug is required"
	}
	if req.Name == "" {
		fields["name"] = "Name is required"
	}
	if len(fields) > 0 {
		return api.ValidationError(c, fields)
	}

	caps := models.JSONB("{}")
	if len(req.Capabilities) > 0 {
		caps = models.JSONB(req.Capabilities)
	}

	role := models.Role{
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		Capabilities: caps,
	}

	if err := h.db.Create(&role).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create role")
	}

	return api.Created(c, role)
}

// Update handles PATCH /roles/:id to partially update a role.
func (h *RoleHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Role ID must be a valid integer")
	}

	var role models.Role
	if err := h.db.First(&role, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Role not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch role")
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Prevent changing slug of system roles.
	if role.IsSystem {
		delete(body, "slug")
	}

	// Remove fields that should not be directly updated.
	delete(body, "id")
	delete(body, "is_system")
	delete(body, "created_at")
	delete(body, "updated_at")

	if len(body) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "NO_UPDATES", "No valid fields to update")
	}

	if err := h.db.Model(&role).Updates(body).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update role")
	}

	// Reload to return fresh data.
	h.db.First(&role, id)
	return api.Success(c, role)
}

// Delete handles DELETE /roles/:id to remove a role.
func (h *RoleHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Role ID must be a valid integer")
	}

	var role models.Role
	if err := h.db.First(&role, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Role not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch role")
	}

	if role.IsSystem {
		return api.Error(c, fiber.StatusForbidden, "SYSTEM_ROLE", "System roles cannot be deleted")
	}

	// Check if any users are assigned to this role.
	var count int64
	if err := h.db.Model(&models.User{}).Where("role_id = ?", id).Count(&count).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to check role usage")
	}
	if count > 0 {
		return api.Error(c, fiber.StatusConflict, "ROLE_IN_USE", "Cannot delete role that is assigned to users")
	}

	if err := h.db.Delete(&role).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete role")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ListSystemActions handles GET /system-actions to retrieve all system actions.
func (h *RoleHandler) ListSystemActions(c *fiber.Ctx) error {
	var actions []models.SystemAction
	if err := h.db.Order("category ASC, slug ASC").Find(&actions).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list system actions")
	}
	return api.Success(c, actions)
}
