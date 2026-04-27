package auth

import (
	"strings"

	"vibecms/internal/api"

	"github.com/gofiber/fiber/v2"
)

// JSONOnlyMutations is CSRF defense for the admin API: any state-changing
// request (POST/PUT/PATCH/DELETE) must carry Content-Type: application/json.
// Combined with SameSite=Strict on the session cookie this stops the
// classic form-submit-from-attacker-site CSRF: a hostile page can submit
// a form with multipart/form-data or application/x-www-form-urlencoded
// without preflight, but the browser blocks application/json POST until
// it has done a CORS preflight that the server can reject.
//
// Mounted on the /admin/api group only; public form-based auth endpoints
// (/auth/login-page, /auth/register, etc.) are handled separately and
// must keep accepting form encoding.
func JSONOnlyMutations() fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}
		ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
		// Strip any charset param: "application/json; charset=utf-8" → "application/json".
		if i := strings.Index(ct, ";"); i >= 0 {
			ct = strings.TrimSpace(ct[:i])
		}
		if ct == "application/json" || ct == "" && c.Method() == fiber.MethodDelete {
			// DELETE often has no body and no Content-Type — accept that.
			return c.Next()
		}
		// Allow multipart/form-data only for the file upload endpoints
		// that genuinely need it. Anything else gets rejected.
		if strings.HasPrefix(ct, "multipart/form-data") && allowsMultipart(c.Path()) {
			return c.Next()
		}
		return api.Error(c, fiber.StatusUnsupportedMediaType, "INVALID_CONTENT_TYPE", "Mutations require Content-Type: application/json")
	}
}

// allowsMultipart returns true for admin API paths that legitimately
// accept file uploads via multipart/form-data. Keep this list explicit
// so adding a new upload endpoint is a deliberate change.
func allowsMultipart(path string) bool {
	multipartPaths := []string{
		"/admin/api/themes/upload",
		"/admin/api/extensions/upload",
		"/admin/api/media/upload",
	}
	for _, p := range multipartPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	// Extension-mounted multipart endpoints under /admin/api/ext/<slug>/...
	// are extension-controlled — bypass and let the extension's handler
	// validate. (Capability gates already constrain who can reach this.)
	if strings.HasPrefix(path, "/admin/api/ext/") {
		return true
	}
	return false
}
