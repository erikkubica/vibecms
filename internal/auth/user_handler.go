package auth

import (
	"strconv"

	"vibecms/internal/api"
	"vibecms/internal/events"
	"vibecms/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// createUserRequest represents the expected JSON body for creating a user.
type createUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	RoleID   int    `json:"role_id"`
}

// updateUserRequest represents the expected JSON body for updating a user.
type updateUserRequest struct {
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
	FullName *string `json:"full_name,omitempty"`
	RoleID   *int    `json:"role_id,omitempty"`
}

// userResponse represents the sanitized user data returned in API responses.
type userResponse struct {
	ID          int         `json:"id"`
	Email       string      `json:"email"`
	FullName    *string     `json:"full_name"`
	RoleID      int         `json:"role_id"`
	Role        models.Role `json:"role"`
	LastLoginAt interface{} `json:"last_login_at"`
	CreatedAt   interface{} `json:"created_at"`
	UpdatedAt   interface{} `json:"updated_at"`
}

func toUserResponse(u models.User) userResponse {
	return userResponse{
		ID:          u.ID,
		Email:       u.Email,
		FullName:    u.FullName,
		RoleID:      u.RoleID,
		Role:        u.Role,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// UserHandler handles user management HTTP endpoints.
type UserHandler struct {
	db       *gorm.DB
	eventBus *events.EventBus
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(db *gorm.DB, eventBus *events.EventBus) *UserHandler {
	return &UserHandler{db: db, eventBus: eventBus}
}

// RegisterRoutes registers user management routes on the given router group.
func (h *UserHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/users", h.ListUsers)
	router.Get("/users/:id", h.GetUser)
	router.Post("/users", h.CreateUser)
	router.Patch("/users/:id", h.UpdateUser)
	router.Delete("/users/:id", h.DeleteUser)
}

// ListUsers returns a paginated list of users. Requires manage_users capability.
func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
	currentUser := GetCurrentUser(c)
	if !HasCapability(currentUser, "manage_users") {
		return api.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	var total int64
	h.db.Model(&models.User{}).Count(&total)

	var users []models.User
	offset := (page - 1) * perPage
	if err := h.db.Preload("Role").Order("id ASC").Offset(offset).Limit(perPage).Find(&users).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch users")
	}

	responses := make([]userResponse, len(users))
	for i, u := range users {
		responses[i] = toUserResponse(u)
	}

	return api.Paginated(c, responses, total, page, perPage)
}

// GetUser returns a single user by ID.
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "BAD_REQUEST", "Invalid user ID")
	}

	var user models.User
	if err := h.db.Preload("Role").First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "User not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch user")
	}

	return api.Success(c, toUserResponse(user))
}

// CreateUser creates a new user. Requires manage_users capability.
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	currentUser := GetCurrentUser(c)
	if !HasCapability(currentUser, "manage_users") {
		return api.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
	}

	var req createUserRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
	}

	fields := make(map[string]string)
	if req.Email == "" {
		fields["email"] = "Email is required"
	}
	if req.Password == "" {
		fields["password"] = "Password is required"
	}
	if len(fields) > 0 {
		return api.ValidationError(c, fields)
	}

	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash password")
	}

	roleID := req.RoleID
	if roleID == 0 {
		var editorRole models.Role
		if err := h.db.Where("slug = ?", "editor").First(&editorRole).Error; err == nil {
			roleID = editorRole.ID
		}
	}

	user := models.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FullName:     &req.FullName,
		RoleID:       roleID,
	}

	if err := h.db.Create(&user).Error; err != nil {
		return api.Error(c, fiber.StatusConflict, "CONFLICT", "A user with this email already exists")
	}

	h.db.Preload("Role").First(&user, user.ID)

	if h.eventBus != nil {
		go h.eventBus.Publish("user.registered", events.Payload{
			"user_id":    user.ID,
			"user_email": user.Email,
		})
	}

	return api.Created(c, toUserResponse(user))
}

// UpdateUser updates an existing user.
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "BAD_REQUEST", "Invalid user ID")
	}

	currentUser := GetCurrentUser(c)
	if currentUser == nil {
		return api.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	if !HasCapability(currentUser, "manage_users") && currentUser.ID != id {
		return api.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only update your own profile")
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "User not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch user")
	}

	var req updateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
	}

	updates := make(map[string]interface{})

	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.FullName != nil {
		updates["full_name"] = *req.FullName
	}
	if req.RoleID != nil {
		if !HasCapability(currentUser, "manage_users") {
			return api.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Only admins can change user roles")
		}
		// Self-promotion guard: a user with manage_users could otherwise
		// PATCH /users/<their_id> {"role_id": <admin_id>} to grant
		// themselves admin. Always require an outside actor for role
		// elevation (or demotion — same path).
		if currentUser.ID == id {
			return api.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You cannot change your own role")
		}
		updates["role_id"] = *req.RoleID
	}
	if req.Password != nil {
		hashedPassword, err := HashPassword(*req.Password)
		if err != nil {
			return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash password")
		}
		updates["password_hash"] = string(hashedPassword)
	}

	if len(updates) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "BAD_REQUEST", "No fields to update")
	}

	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update user")
	}

	h.db.Preload("Role").First(&user, id)

	if h.eventBus != nil {
		go h.eventBus.Publish("user.updated", events.Payload{
			"user_id":    user.ID,
			"user_email": user.Email,
		})
	}

	return api.Success(c, toUserResponse(user))
}

// DeleteUser deletes a user by ID. Requires manage_users capability.
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	currentUser := GetCurrentUser(c)
	if !HasCapability(currentUser, "manage_users") {
		return api.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "BAD_REQUEST", "Invalid user ID")
	}

	// Operators must not delete their own account — that locks them out
	// of the very session this request is running under.
	if currentUser != nil && currentUser.ID == id {
		return api.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You cannot delete your own account")
	}

	result := h.db.Delete(&models.User{}, id)
	if result.Error != nil {
		return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete user")
	}
	if result.RowsAffected == 0 {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "User not found")
	}

	if h.eventBus != nil {
		go h.eventBus.Publish("user.deleted", events.Payload{
			"user_id": id,
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
