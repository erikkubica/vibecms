package cms

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// verifyWebhookAuth checks the request against any of the supported
// webhook authentication mechanisms. Returns true on the first match.
//
// Methods checked, in order:
//   - X-Hub-Signature-256 (GitHub) — HMAC-SHA256 over the raw body.
//     Format: "sha256=<hex-digest>".
//   - X-Gitlab-Token (GitLab) — plain shared secret, constant-time compare.
//   - X-Webhook-Secret — generic shared secret, constant-time compare.
//
// HMAC is preferred because it binds the secret to the request body — an
// attacker who replays a signed request can't substitute their own body.
func verifyWebhookAuth(c *fiber.Ctx, secret string, body []byte) bool {
	if sig := c.Get("X-Hub-Signature-256"); sig != "" {
		return verifyHMACSignature(sig, secret, body)
	}
	if token := c.Get("X-Gitlab-Token"); token != "" {
		return constantTimeEquals(token, secret)
	}
	if header := c.Get("X-Webhook-Secret"); header != "" {
		return constantTimeEquals(header, secret)
	}
	return false
}

// verifyHMACSignature verifies a GitHub-style sha256=<hex> signature
// against secret + body. Defends against:
//   - non-sha256 schemes (rejected),
//   - malformed hex (rejected),
//   - tampered body (HMAC mismatch),
//   - constant-time leak (uses hmac.Equal on raw bytes).
func verifyHMACSignature(sig, secret string, body []byte) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(sig, prefix) {
		return false
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(sig, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(provided, expected)
}

// constantTimeEquals does a length-safe constant-time string compare.
// subtle.ConstantTimeCompare panics on different lengths in some Go
// versions and returns 0 in others — we hedge by length-checking first.
func constantTimeEquals(a, b string) bool {
	if len(a) != len(b) {
		// Still consume time on length mismatch by hashing one side, so
		// the timing oracle can't trivially distinguish empty vs short.
		_ = subtle.ConstantTimeCompare([]byte(a), []byte(a))
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
