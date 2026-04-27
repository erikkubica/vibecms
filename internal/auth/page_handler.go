package auth

import (
	"fmt"
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
	resetSvc   *PasswordResetService
	eventBus   *events.EventBus

	// Defenses against credential-stuffing and brute force. Per-IP rate
	// limiters cap the request volume from any one source; the lockout
	// tracker freezes individual accounts after repeated failures so an
	// attacker rotating IPs against one user still hits a wall.
	loginLimiter   *PerIPLimiter
	signupLimiter  *PerIPLimiter
	forgotLimiter  *PerIPLimiter
	resetLimiter   *PerIPLimiter
	lockout        *LockoutTracker
}

// NewPageAuthHandler creates a new PageAuthHandler.
func NewPageAuthHandler(db *gorm.DB, sessionSvc *SessionService, eventBus *events.EventBus) *PageAuthHandler {
	return &PageAuthHandler{
		db:            db,
		sessionSvc:    sessionSvc,
		resetSvc:      NewPasswordResetService(db),
		eventBus:      eventBus,
		loginLimiter:  NewPerIPLimiter(10, time.Minute, 10),       // 10/min sustained, burst 10 — covers normal sign-in retries
		signupLimiter: NewPerIPLimiter(3, time.Hour, 3),            // 3/hr — tight, registration is rare
		forgotLimiter: NewPerIPLimiter(3, time.Hour, 3),            // 3/hr — even tighter; forgot-password is the leaking oracle
		resetLimiter:  NewPerIPLimiter(10, time.Hour, 10),          // 10/hr — reset attempts may bounce around mistypes
		lockout:       NewLockoutTracker(5, 15*time.Minute, 15*time.Minute),
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

// registrationAllowed returns true when site_settings.allow_registration
// is exactly "true". Default is closed: any other value (missing row,
// "false", "0", "") returns false.
func registrationAllowed(db *gorm.DB) bool {
	var value string
	row := db.Raw("SELECT value FROM site_settings WHERE key = ?", "allow_registration").Row()
	if err := row.Scan(&value); err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

// --- POST handlers ---

func (h *PageAuthHandler) ProcessLogin(c *fiber.Ctx) error {
	if !h.loginLimiter.Allow(c.IP()) {
		setFlash(c, "Too many sign-in attempts. Please wait a minute and try again.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")

	if email == "" || password == "" {
		setFlash(c, "Email and password are required.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

	// Locked accounts get the same generic error as wrong-password to
	// avoid leaking which accounts exist or are under attack.
	if h.lockout.IsLocked(email) {
		setFlash(c, "Invalid email or password.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

	var user models.User
	if err := h.db.Preload("Role").Where("email = ?", email).First(&user).Error; err != nil {
		// Record the failure even when the email doesn't exist — otherwise
		// an attacker can probe by timing and lockout signal. Also burn
		// an equivalent bcrypt comparison so the response time matches
		// the user-found path; without that the no-user response is
		// hundreds of ms faster, leaking which emails are registered.
		AbsorbBcryptTime()
		h.lockout.RecordFailure(email)
		setFlash(c, "Invalid email or password.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		h.lockout.RecordFailure(email)
		setFlash(c, "Invalid email or password.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

	// Successful login — clear any pending lockout state for this account.
	h.lockout.RecordSuccess(email)

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
		Secure:   IsSecureRequest(c),
		SameSite: "Strict",
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
	if !h.signupLimiter.Allow(c.IP()) {
		setFlash(c, "Too many registration attempts. Please try again later.", "error")
		return c.Redirect("/register", fiber.StatusFound)
	}

	// Public registration is disabled by default. Operators enable it by
	// setting site_settings.allow_registration = "true". This stops the
	// open-admin-enrollment vulnerability where any visitor could register
	// and (formerly) get an editor role with admin_access:true.
	if !registrationAllowed(h.db) {
		setFlash(c, "Registration is currently disabled.", "error")
		return c.Redirect("/register", fiber.StatusFound)
	}

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

	hash, err := HashPassword(password)
	if err != nil {
		log.Printf("failed to hash password: %v", err)
		setFlash(c, "An unexpected error occurred.", "error")
		return c.Redirect("/register", fiber.StatusFound)
	}

	// Public registration creates `member` role users — no admin_access,
	// read-only on content. Operators who want richer roles for new
	// signups can change the role afterwards via the admin UI.
	var memberRole models.Role
	if err := h.db.Where("slug = ?", "member").First(&memberRole).Error; err != nil {
		log.Printf("failed to find member role: %v", err)
		setFlash(c, "An unexpected error occurred.", "error")
		return c.Redirect("/register", fiber.StatusFound)
	}

	user := models.User{
		Email:        email,
		PasswordHash: string(hash),
		RoleID:       memberRole.ID,
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
		Secure:   IsSecureRequest(c),
		SameSite: "Strict",
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

// ProcessForgotPassword issues a password reset token for a real account
// and publishes user.password_reset_requested so the email dispatcher
// (or whichever provider is wired) can deliver the reset link. Always
// returns the same generic success flash regardless of whether the
// email exists — never confirm or deny account existence to the
// requester (that's a username enumeration oracle).
func (h *PageAuthHandler) ProcessForgotPassword(c *fiber.Ctx) error {
	if !h.forgotLimiter.Allow(c.IP()) {
		// Same generic message — don't tell attackers they're rate-limited
		// (still an oracle that the endpoint is reachable).
		setFlash(c, "If an account exists with that email, a reset link has been sent.", "success")
		return c.Redirect("/forgot-password", fiber.StatusFound)
	}

	email := strings.TrimSpace(c.FormValue("email"))
	if email == "" {
		setFlash(c, "Email is required.", "error")
		return c.Redirect("/forgot-password", fiber.StatusFound)
	}

	const successMsg = "If an account exists with that email, a reset link has been sent."

	var user models.User
	err := h.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		// Don't leak whether the email exists. Same flash either way.
		setFlash(c, successMsg, "success")
		return c.Redirect("/forgot-password", fiber.StatusFound)
	}

	rawToken, expires, err := h.resetSvc.IssueToken(user.ID, c.IP(), c.Get("User-Agent"))
	if err != nil {
		log.Printf("issuing password reset token for user %d: %v", user.ID, err)
		setFlash(c, successMsg, "success")
		return c.Redirect("/forgot-password", fiber.StatusFound)
	}

	resetURL := buildResetURL(c, rawToken)

	if h.eventBus != nil {
		// Sync publish so the email-sending rule chain runs before we
		// redirect — otherwise the user could refresh the success page
		// and re-submit before the email is dispatched. Failure to
		// dispatch isn't surfaced to the user (oracle).
		h.eventBus.PublishSync("user.password_reset_requested", events.Payload{
			"user_id":      user.ID,
			"user_email":   user.Email,
			"actor_email":  user.Email,
			"reset_url":    resetURL,
			"reset_token":  rawToken, // template variable; persisted in email log if logging is enabled
			"expires_at":   expires.Format(time.RFC3339),
			"ip_address":   c.IP(),
		})
	}

	setFlash(c, successMsg, "success")
	return c.Redirect("/forgot-password", fiber.StatusFound)
}

// ProcessResetPassword consumes a password reset token, updates the user's
// password, invalidates remaining tokens, and clears all of that user's
// sessions so an attacker who already had a session is kicked out.
func (h *PageAuthHandler) ProcessResetPassword(c *fiber.Ctx) error {
	if !h.resetLimiter.Allow(c.IP()) {
		setFlash(c, "Too many reset attempts. Please try again later.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

	token := strings.TrimSpace(c.FormValue("token"))
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}
	password := c.FormValue("password")
	passwordConfirm := c.FormValue("password_confirm")

	if token == "" || password == "" || passwordConfirm == "" {
		setFlash(c, "All fields are required.", "error")
		return c.Redirect("/reset-password?token="+token, fiber.StatusFound)
	}
	if password != passwordConfirm {
		setFlash(c, "Passwords do not match.", "error")
		return c.Redirect("/reset-password?token="+token, fiber.StatusFound)
	}

	userID, err := h.resetSvc.VerifyAndConsume(token)
	if err != nil {
		// Generic error — don't reveal whether the token was unknown,
		// expired, or already used.
		setFlash(c, "This password reset link is invalid or has expired.", "error")
		return c.Redirect("/forgot-password", fiber.StatusFound)
	}

	hash, err := HashPassword(password)
	if err != nil {
		log.Printf("hashing new password for user %d: %v", userID, err)
		setFlash(c, "An unexpected error occurred.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

	if err := h.db.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", string(hash)).Error; err != nil {
		log.Printf("updating password for user %d: %v", userID, err)
		setFlash(c, "An unexpected error occurred.", "error")
		return c.Redirect("/login", fiber.StatusFound)
	}

	// Invalidate any other in-flight reset tokens.
	if err := h.resetSvc.InvalidateAllForUser(userID); err != nil {
		log.Printf("invalidating reset tokens for user %d: %v", userID, err)
	}

	// Wipe all sessions for the user — anyone (including the legitimate
	// user) holding a session needs to re-authenticate with the new
	// password. This is the kill-switch behaviour every password reset
	// flow should have.
	if err := h.db.Where("user_id = ?", userID).Delete(&models.Session{}).Error; err != nil {
		log.Printf("clearing sessions after reset for user %d: %v", userID, err)
	}

	if h.eventBus != nil {
		go h.eventBus.Publish("user.password_reset_completed", events.Payload{
			"user_id":    userID,
			"ip_address": c.IP(),
		})
	}

	setFlash(c, "Password reset successfully. Please log in.", "success")
	return c.Redirect("/login", fiber.StatusFound)
}

// buildResetURL composes an absolute reset URL using site_url when set,
// falling back to the request's host. Falling back via the request
// preserves dev-time correctness (no env config required).
func buildResetURL(c *fiber.Ctx, token string) string {
	scheme := "http"
	if IsSecureRequest(c) {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/reset-password?token=%s", scheme, c.Hostname(), token)
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
		Secure:   IsSecureRequest(c),
		SameSite: "Strict",
		Expires:  time.Now().Add(-1 * time.Hour),
	})

	setFlash(c, "You have been logged out.", "success")
	return c.Redirect("/", fiber.StatusFound)
}
