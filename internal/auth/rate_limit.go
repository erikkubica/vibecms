package auth

import (
	"sync"
	"time"

	"vibecms/internal/api"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/time/rate"
)

// PerIPLimiter throttles requests per source IP. Each IP gets its own
// rate.Limiter so a flood from one client can't starve everyone else.
// Process-local — fine for single-instance deploys; multi-instance
// deploys need a Redis-backed limiter (out of scope here).
type PerIPLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	limit    rate.Limit
	burst    int

	// lastSeen tracks when each IP was last active, so the cleanup loop
	// can drop entries for IPs that haven't hit the endpoint in a while.
	// Without this, the map grows linearly with unique attackers.
	lastSeen map[string]time.Time
}

// NewPerIPLimiter constructs a limiter at reqPerInterval / interval rate,
// allowing a short burst. For example NewPerIPLimiter(5, time.Minute, 5)
// = 5 req/min sustained, up to 5 in a burst.
func NewPerIPLimiter(reqPerInterval int, interval time.Duration, burst int) *PerIPLimiter {
	return &PerIPLimiter{
		limiters: make(map[string]*rate.Limiter),
		lastSeen: make(map[string]time.Time),
		limit:    rate.Limit(float64(reqPerInterval) / interval.Seconds()),
		burst:    burst,
	}
}

// Allow returns true if the IP is under its budget; false if it should
// be throttled.
func (p *PerIPLimiter) Allow(ip string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	lim, ok := p.limiters[ip]
	if !ok {
		lim = rate.NewLimiter(p.limit, p.burst)
		p.limiters[ip] = lim
	}
	p.lastSeen[ip] = time.Now()
	return lim.Allow()
}

// CleanIdle removes limiter entries for IPs that haven't been seen for
// the given duration. Call periodically (e.g. hourly) to bound memory.
func (p *PerIPLimiter) CleanIdle(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	p.mu.Lock()
	defer p.mu.Unlock()
	for ip, ts := range p.lastSeen {
		if ts.Before(cutoff) {
			delete(p.limiters, ip)
			delete(p.lastSeen, ip)
		}
	}
}

// Middleware returns a Fiber handler that consults this limiter and
// rejects with 429 when the budget is exhausted. Uses Fiber's c.IP()
// — which respects X-Forwarded-For only when the app is configured to
// trust proxies; otherwise it falls back to RemoteAddr. Operators
// behind a proxy must configure trusted proxies on the Fiber app.
func (p *PerIPLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !p.Allow(c.IP()) {
			return api.Error(c, fiber.StatusTooManyRequests, "RATE_LIMITED", "Too many requests — please try again in a minute")
		}
		return c.Next()
	}
}
