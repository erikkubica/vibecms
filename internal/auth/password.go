package auth

import (
	"os"
	"strconv"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// dummyBcryptHash is a fixed bcrypt hash used to absorb the cost of a
// "user not found" login attempt. Without it, the no-user path returns
// in microseconds while the user-found path runs a ~250ms bcrypt
// compare — leaking which emails are registered. The hash is computed
// once on first use against a constant plaintext that never matches a
// real password (the trailing colons are illegal in bcrypt input but
// CompareHashAndPassword always fails, so we never grant on it
// regardless).
var (
	dummyBcryptOnce sync.Once
	dummyBcryptHash []byte
)

func ensureDummyBcryptHash() {
	dummyBcryptOnce.Do(func() {
		// Cost is the same we use for real hashes so timing matches.
		// If hashing itself fails (out of memory, unlikely), fall back
		// to a static hash that's still bcrypt-shaped — the compare
		// will still take cost-bound time and always fail.
		h, err := bcrypt.GenerateFromPassword([]byte("::vibecms-dummy::"), passwordCost())
		if err != nil {
			h = []byte("$2a$12$" + "X" /* 53 chars total */)
		}
		dummyBcryptHash = h
	})
}

// AbsorbBcryptTime runs a bcrypt comparison against a fixed dummy hash
// so the caller's response time matches a successful user lookup that
// actually verifies a password. Always returns false — never use the
// return value for an authentication decision.
func AbsorbBcryptTime() bool {
	ensureDummyBcryptHash()
	_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte("absorb"))
	return false
}

// minBcryptCost is the lowest cost we accept. bcrypt's own minimum is 4
// (testing only); 10 is the historical default; OWASP recommends 12+ on
// modern hardware.
const (
	minBcryptCost     = 10
	maxBcryptCost     = 14
	defaultBcryptCost = 12
)

// passwordCost returns the bcrypt cost factor to use when hashing
// passwords, read from BCRYPT_COST. Out-of-range or unparseable values
// fall back to defaultBcryptCost so a typo in env config doesn't pin
// every deploy to the floor.
func passwordCost() int {
	raw := os.Getenv("BCRYPT_COST")
	if raw == "" {
		return defaultBcryptCost
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return defaultBcryptCost
	}
	if n < minBcryptCost {
		return minBcryptCost
	}
	if n > maxBcryptCost {
		return maxBcryptCost
	}
	return n
}

// HashPassword hashes a plaintext password with the configured bcrypt cost.
// Use this everywhere instead of calling bcrypt.GenerateFromPassword
// directly so the cost factor is centrally managed.
func HashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), passwordCost())
}

// trustForwardedProto returns true when the operator has opted in to
// honoring X-Forwarded-Proto from upstream proxies. Set
// TRUST_FORWARDED_PROTO=true behind any TLS-terminating proxy
// (nginx, Caddy, Coolify, fly.io, etc.) so cookies set by this app
// correctly get the Secure flag even though the proxy → app hop is
// plain HTTP.
func trustForwardedProto() bool {
	v := os.Getenv("TRUST_FORWARDED_PROTO")
	return v == "1" || v == "true" || v == "TRUE"
}
