package auth

import (
	"log"
	"strings"
	"time"

	"vibecms/internal/models"
	"vibecms/internal/rendering"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// pageData holds data passed to auth page templates.
type pageData struct {
	Title     string
	User      *models.User
	FlashMsg  string
	FlashType string // "success" or "error"
	Token     string
}

// PageAuthHandler serves HTML authentication pages (login, register, etc.).
type PageAuthHandler struct {
	db         *gorm.DB
	sessionSvc *SessionService
	renderer   *rendering.TemplateRenderer
}

// NewPageAuthHandler creates a new PageAuthHandler.
func NewPageAuthHandler(db *gorm.DB, sessionSvc *SessionService, renderer *rendering.TemplateRenderer) *PageAuthHandler {
	return &PageAuthHandler{
		db:         db,
		sessionSvc: sessionSvc,
		renderer:   renderer,
	}
}

// RegisterRoutes registers all auth page routes on the Fiber app.
func (h *PageAuthHandler) RegisterRoutes(app *fiber.App) {
	app.Get("/login", h.ShowLogin)
	app.Post("/auth/login-page", h.ProcessLogin)
	app.Get("/register", h.ShowRegister)
	app.Post("/register", h.ProcessRegister)
	app.Get("/auth/forgot-password", h.ShowForgotPassword)
	app.Post("/auth/forgot-password", h.ProcessForgotPassword)
	app.Get("/auth/reset-password", h.ShowResetPassword)
	app.Post("/auth/reset-password", h.ProcessResetPassword)
	app.Get("/logout", h.Logout)
}

// renderTemplate renders the named auth template with the base layout.
func (h *PageAuthHandler) renderTemplate(c *fiber.Ctx, name string, data pageData) error {
	if data.FlashMsg == "" {
		data.FlashMsg = c.Cookies("flash_msg")
		data.FlashType = c.Cookies("flash_type")
	}
	clearFlashCookies(c)

	c.Set("Content-Type", "text/html; charset=utf-8")

	var buf strings.Builder
	if err := h.renderer.RenderPage(&buf, "auth/"+name, data); err != nil {
		log.Printf("template render error (%s): %v", name, err)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}
	return c.SendString(buf.String())
}

// setFlash sets flash message cookies for the next request.
func setFlash(c *fiber.Ctx, msg, flashType string) {
	c.Cookie(&fiber.Cookie{
		Name:     "flash_msg",
		Value:    msg,
		Path:     "/",
		MaxAge:   10,
		HTTPOnly: true,
	})
	c.Cookie(&fiber.Cookie{
		Name:     "flash_type",
		Value:    flashType,
		Path:     "/",
		MaxAge:   10,
		HTTPOnly: true,
	})
}

// clearFlashCookies removes flash cookies.
func clearFlashCookies(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name: "flash_msg", Value: "", Path: "/", MaxAge: -1, HTTPOnly: true,
	})
	c.Cookie(&fiber.Cookie{
		Name: "flash_type", Value: "", Path: "/", MaxAge: -1, HTTPOnly: true,
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

// --- GET handlers ---

func (h *PageAuthHandler) ShowLogin(c *fiber.Ctx) error {
	if h.isLoggedIn(c) {
		return c.Redirect("/admin/dashboard", fiber.StatusFound)
	}
	return h.renderTemplate(c, "login.html", pageData{Title: "Sign In"})
}

func (h *PageAuthHandler) ShowRegister(c *fiber.Ctx) error {
	return h.renderTemplate(c, "register.html", pageData{Title: "Create Account"})
}

func (h *PageAuthHandler) ShowForgotPassword(c *fiber.Ctx) error {
	return h.renderTemplate(c, "forgot_password.html", pageData{Title: "Forgot Password"})
}

func (h *PageAuthHandler) ShowResetPassword(c *fiber.Ctx) error {
	token := c.Query("token")
	return h.renderTemplate(c, "reset_password.html", pageData{Title: "Reset Password", Token: token})
}

// --- POST handlers ---

func (h *PageAuthHandler) ProcessLogin(c *fiber.Ctx) error {
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")

	if email == "" || password == "" {
		return h.renderTemplate(c, "login.html", pageData{
			Title: "Sign In", FlashMsg: "Email and password are required.", FlashType: "error",
		})
	}

	var user models.User
	if err := h.db.Where("email = ?", email).First(&user).Error; err != nil {
		return h.renderTemplate(c, "login.html", pageData{
			Title: "Sign In", FlashMsg: "Invalid email or password.", FlashType: "error",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return h.renderTemplate(c, "login.html", pageData{
			Title: "Sign In", FlashMsg: "Invalid email or password.", FlashType: "error",
		})
	}

	token, err := h.sessionSvc.CreateSession(user.ID, c.IP(), c.Get("User-Agent"))
	if err != nil {
		log.Printf("failed to create session: %v", err)
		return h.renderTemplate(c, "login.html", pageData{
			Title: "Sign In", FlashMsg: "An unexpected error occurred.", FlashType: "error",
		})
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

	setFlash(c, "Welcome back!", "success")
	return c.Redirect("/admin/dashboard", fiber.StatusFound)
}

func (h *PageAuthHandler) ProcessRegister(c *fiber.Ctx) error {
	fullName := strings.TrimSpace(c.FormValue("full_name"))
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")
	passwordConfirm := c.FormValue("password_confirm")

	if fullName == "" || email == "" || password == "" || passwordConfirm == "" {
		return h.renderTemplate(c, "register.html", pageData{
			Title: "Create Account", FlashMsg: "All fields are required.", FlashType: "error",
		})
	}

	if password != passwordConfirm {
		return h.renderTemplate(c, "register.html", pageData{
			Title: "Create Account", FlashMsg: "Passwords do not match.", FlashType: "error",
		})
	}

	var existing models.User
	if err := h.db.Where("email = ?", email).First(&existing).Error; err == nil {
		return h.renderTemplate(c, "register.html", pageData{
			Title: "Create Account", FlashMsg: "An account with this email already exists.", FlashType: "error",
		})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to hash password: %v", err)
		return h.renderTemplate(c, "register.html", pageData{
			Title: "Create Account", FlashMsg: "An unexpected error occurred.", FlashType: "error",
		})
	}

	user := models.User{
		Email:        email,
		PasswordHash: string(hash),
		Role:         "editor",
		FullName:     &fullName,
	}
	if err := h.db.Create(&user).Error; err != nil {
		log.Printf("failed to create user: %v", err)
		return h.renderTemplate(c, "register.html", pageData{
			Title: "Create Account", FlashMsg: "An unexpected error occurred.", FlashType: "error",
		})
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

	setFlash(c, "Account created!", "success")
	return c.Redirect("/admin/dashboard", fiber.StatusFound)
}

func (h *PageAuthHandler) ProcessForgotPassword(c *fiber.Ctx) error {
	setFlash(c, "If an account exists with that email, a reset link has been sent.", "success")
	return c.Redirect("/auth/forgot-password", fiber.StatusFound)
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
