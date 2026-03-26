package auth

import (
	"time"

	"vibecms/internal/api"
	"vibecms/internal/models"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// loginRequest represents the expected JSON body for login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthHandler handles authentication-related HTTP endpoints.
type AuthHandler struct {
	db         *gorm.DB
	sessionSvc *SessionService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(db *gorm.DB, sessionSvc *SessionService) *AuthHandler {
	return &AuthHandler{
		db:         db,
		sessionSvc: sessionSvc,
	}
}

// RegisterRoutes registers authentication routes under /auth and /me.
func (h *AuthHandler) RegisterRoutes(app *fiber.App) {
	app.Post("/auth/login", h.Login)
	app.Post("/auth/logout", AuthRequired(h.sessionSvc), h.Logout)
	app.Get("/me", AuthRequired(h.sessionSvc), h.Me)
}

// Login authenticates a user with email and password, creates a session,
// and sets an HTTP-only session cookie.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return api.ValidationError(c, map[string]string{
			"email":    "Email is required",
			"password": "Password is required",
		})
	}

	var user models.User
	if err := h.db.Preload("Role").Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		}
		return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return api.Error(c, fiber.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
	}

	token, err := h.sessionSvc.CreateSession(user.ID, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create session")
	}

	// Update last login timestamp.
	now := time.Now()
	h.db.Model(&user).Update("last_login_at", now)

	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
		Expires:  time.Now().Add(h.sessionSvc.sessionExpiry),
	})

	return api.Success(c, fiber.Map{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role.Slug,
	})
}

// Logout invalidates the current session and clears the session cookie.
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	token := c.Cookies(CookieName)
	if token != "" {
		_ = h.sessionSvc.DeleteSession(token)
	}

	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
		Expires:  time.Now().Add(-1 * time.Hour),
	})

	return api.Success(c, fiber.Map{
		"message": "Logged out successfully",
	})
}

// Me returns the profile of the currently authenticated user.
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	user := GetCurrentUser(c)
	if user == nil {
		return api.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	return api.Success(c, fiber.Map{
		"id":            user.ID,
		"email":         user.Email,
		"full_name":     user.FullName,
		"role":          user.Role.Slug,
		"last_login_at": user.LastLoginAt,
	})
}
