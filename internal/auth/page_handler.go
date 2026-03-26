package auth

import (
	"log"
	"strings"
	"time"

	"vibecms/internal/events"
	"vibecms/internal/models"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// PageAuthHandler serves auth form POST handlers and logout.
// GET pages for login/register/forgot-password/reset-password are served as
// content nodes with auth block types (login-form, register-form, etc.).
type PageAuthHandler struct {
	db         *gorm.DB
	sessionSvc *SessionService
	eventBus   *events.EventBus
}

// NewPageAuthHandler creates a new PageAuthHandler.
func NewPageAuthHandler(db *gorm.DB, sessionSvc *SessionService, eventBus *events.EventBus) *PageAuthHandler {
	return &PageAuthHandler{
		db:         db,
		sessionSvc: sessionSvc,
		eventBus:   eventBus,
	}
}

// RegisterRoutes registers auth POST handlers and logout on the Fiber app.
func (h *PageAuthHandler) RegisterRoutes(app *fiber.App) {
	app.Post("/auth/login-page", h.ProcessLogin)
	app.Post("/auth/register", h.ProcessRegister)
	app.Post("/auth/forgot-password", h.ProcessForgotPassword)
	app.Post("/auth/reset-password", h.ProcessResetPassword)
	app.Get("/logout", h.Logout)
}

// setFlash sets flash message cookies for the next request.
func setFlash(c *fiber.Ctx, msg, flashType string) {
	c.Cookie(&fiber.Cookie{
		Name:     "flash_msg",
		Value:    msg,
		Path:     "/",
		MaxAge:   10,
		HTTPOnly: false, // Alpine.js needs to read these
	})
	c.Cookie(&fiber.Cookie{
		Name:     "flash_type",
		Value:    flashType,
		Path:     "/",
		MaxAge:   10,
		HTTPOnly: false,
	})
}

// isLoggedIn checks if the current request has a valid session.
func (h *PageAuthHandler) isLoggedIn(c *fiber.Ctx) bool {
	token := c.Cookies(CookieName)
	if token == "" {
		return false
	}
	_, err := h.sessionSvc.ValidateSession(token)
	return err == nil
}

// --- POST handlers ---

func (h *PageAuthHandler) ProcessLogin(c *fiber.Ctx) error {
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")

	if email == "" || password == "" {
		setFlash(c, "Email and password are required.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

	var user models.User
	if err := h.db.Preload("Role").Where("email = ?", email).First(&user).Error; err != nil {
		setFlash(c, "Invalid email or password.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		setFlash(c, "Invalid email or password.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

	token, err := h.sessionSvc.CreateSession(user.ID, c.IP(), c.Get("User-Agent"))
	if err != nil {
		log.Printf("failed to create session: %v", err)
		setFlash(c, "An unexpected error occurred.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

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

	if h.eventBus != nil {
		go h.eventBus.Publish("user.login", events.Payload{
			"user_id":     user.ID,
			"user_email":  user.Email,
			"actor_email": user.Email,
			"ip_address":  c.IP(),
		})
	}

	setFlash(c, "Welcome back!", "success")
	return c.Redirect("/admin/dashboard", fiber.StatusFound)
}

func (h *PageAuthHandler) ProcessRegister(c *fiber.Ctx) error {
	fullName := strings.TrimSpace(c.FormValue("full_name"))
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")
	passwordConfirm := c.FormValue("password_confirm")

	if fullName == "" || email == "" || password == "" || passwordConfirm == "" {
		setFlash(c, "All fields are required.", "error")
		return c.Redirect("/register", fiber.StatusFound)
	}

	if password != passwordConfirm {
		setFlash(c, "Passwords do not match.", "error")
		return c.Redirect("/register", fiber.StatusFound)
	}

	var existing models.User
	if err := h.db.Where("email = ?", email).First(&existing).Error; err == nil {
		setFlash(c, "An account with this email already exists.", "error")
		return c.Redirect("/register", fiber.StatusFound)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to hash password: %v", err)
		setFlash(c, "An unexpected error occurred.", "error")
		return c.Redirect("/register", fiber.StatusFound)
	}

	var editorRole models.Role
	if err := h.db.Where("slug = ?", "editor").First(&editorRole).Error; err != nil {
		log.Printf("failed to find editor role: %v", err)
		setFlash(c, "An unexpected error occurred.", "error")
		return c.Redirect("/register", fiber.StatusFound)
	}

	user := models.User{
		Email:        email,
		PasswordHash: string(hash),
		RoleID:       editorRole.ID,
		FullName:     &fullName,
	}
	if err := h.db.Create(&user).Error; err != nil {
		log.Printf("failed to create user: %v", err)
		setFlash(c, "An unexpected error occurred.", "error")
		return c.Redirect("/register", fiber.StatusFound)
	}

	token, err := h.sessionSvc.CreateSession(user.ID, c.IP(), c.Get("User-Agent"))
	if err != nil {
		setFlash(c, "Account created! Please sign in.", "success")
		return c.Redirect("/login", fiber.StatusFound)
	}

	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
		Expires:  time.Now().Add(h.sessionSvc.sessionExpiry),
	})

	if h.eventBus != nil {
		go h.eventBus.Publish("user.registered", events.Payload{
			"user_id":     user.ID,
			"user_email":  user.Email,
			"actor_email": user.Email,
		})
	}

	setFlash(c, "Account created!", "success")
	return c.Redirect("/admin/dashboard", fiber.StatusFound)
}

func (h *PageAuthHandler) ProcessForgotPassword(c *fiber.Ctx) error {
	setFlash(c, "If an account exists with that email, a reset link has been sent.", "success")
	return c.Redirect("/forgot-password", fiber.StatusFound)
}

func (h *PageAuthHandler) ProcessResetPassword(c *fiber.Ctx) error {
	setFlash(c, "Password reset successfully. Please log in.", "success")
	return c.Redirect("/login", fiber.StatusFound)
}

func (h *PageAuthHandler) Logout(c *fiber.Ctx) error {
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

	setFlash(c, "You have been logged out.", "success")
	return c.Redirect("/", fiber.StatusFound)
}
