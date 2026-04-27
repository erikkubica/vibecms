package main

import (
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/time/rate"
)

// RateLimiter is a bounded LRU-backed per-IP token-bucket rate limiter.
type RateLimiter struct {
	cache *lru.Cache[string, *rate.Limiter]
	mu    sync.Mutex
}

// NewRateLimiter creates a RateLimiter that tracks at most capacity unique IPs.
func NewRateLimiter(capacity int) *RateLimiter {
	c, _ := lru.New[string, *rate.Limiter](capacity)
	return &RateLimiter{cache: c}
}

// Allow returns false if ip has exceeded limitPerHour in the rolling window.
// Returns true when ip is empty or limitPerHour <= 0 (no limiting).
func (r *RateLimiter) Allow(ip string, limitPerHour int) bool {
	if ip == "" || limitPerHour <= 0 {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	lim, ok := r.cache.Get(ip)
	if !ok {
		// Tokens refill at limitPerHour/hour; burst = limitPerHour.
		lim = rate.NewLimiter(rate.Limit(float64(limitPerHour)/3600.0), limitPerHour)
		r.cache.Add(ip, lim)
	}
	return lim.Allow()
}
