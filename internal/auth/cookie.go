package auth

import "github.com/gofiber/fiber/v2"

// IsSecureRequest returns true when the request was made over TLS.
// It checks the underlying connection AND, when TRUST_FORWARDED_PROTO
// is set, the X-Forwarded-Proto header. Use this to decide whether to
// set the Secure flag on response cookies — without this check, deploys
// behind a TLS-terminating proxy strip the flag and the browser ends up
// with an insecure cookie that gets sent over HTTP if the user ever
// hits the domain via plain HTTP.
func IsSecureRequest(c *fiber.Ctx) bool {
	if c.Secure() {
		return true
	}
	if c.Protocol() == "https" {
		return true
	}
	if trustForwardedProto() && c.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	return false
}
