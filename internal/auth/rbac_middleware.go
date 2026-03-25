package auth

import (
	"strings"

	"vibecms/internal/api"
	"vibecms/internal/models"

	"github.com/gofiber/fiber/v2"
)

const (
	// CookieName is the name of the session cookie.
	CookieName = "vibecms_session"

	// localsUserKey is the key used to store the authenticated user in Fiber locals.
	localsUserKey = "user"
)

// AuthRequired returns a Fiber middleware that validates the session token from
// the "vibecms_session" cookie and stores the authenticated user in c.Locals("user").
func AuthRequired(sessionSvc *SessionService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies(CookieName)
		if token == "" {
			return authFail(c, "Authentication required")
		}

		user, err := sessionSvc.ValidateSession(token)
		if err != nil {
			return authFail(c, "Invalid or expired session")
		}

		c.Locals(localsUserKey, user)
		return c.Next()
	}
}

// authFail redirects browsers to /login or returns JSON 401 for API requests.
func authFail(c *fiber.Ctx, msg string) error {
	accept := c.Get("Accept")
	if strings.Contains(accept, "text/html") || (!strings.Contains(accept, "application/json") && !strings.HasPrefix(c.Path(), "/admin/api")) {
		return c.Redirect("/login", fiber.StatusFound)
	}
	return api.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", msg)
}

// RoleRequired returns a Fiber middleware that checks whether the authenticated
// user has one of the specified roles. Returns 403 if the user's role is not allowed.
// This middleware must be used after AuthRequired.
func RoleRequired(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := GetCurrentUser(c)
		if user == nil {
			return api.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		}

		for _, role := range roles {
			if user.Role == role {
				return c.Next()
			}
		}

		return api.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
	}
}

// GetCurrentUser extracts the authenticated user from Fiber locals.
// Returns nil if no user is stored (i.e., the request is unauthenticated).
func GetCurrentUser(c *fiber.Ctx) *models.User {
	user, ok := c.Locals(localsUserKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}
