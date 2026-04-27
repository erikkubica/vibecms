package auth

import (
	"sync"
	"time"
)

// LockoutTracker counts failed login attempts per (lowercased) email and
// locks the account for a fixed window after too many failures. Backed
// by an in-memory map — process-local, not cluster-aware. Multi-instance
// deploys should layer this on a shared store (Redis) so an attacker
// can't bounce between replicas.
type LockoutTracker struct {
	mu          sync.Mutex
	failures    map[string]*lockoutEntry
	maxFailures int
	window      time.Duration
	lockFor     time.Duration
}

type lockoutEntry struct {
	count     int
	firstSeen time.Time
	lockUntil time.Time
}

// NewLockoutTracker — sensible defaults: 5 failed attempts within a
// 15-minute window lock the account for 15 minutes. The "window" is
// the rolling period over which failures count; passing it without
// hitting maxFailures resets the counter.
func NewLockoutTracker(maxFailures int, window, lockFor time.Duration) *LockoutTracker {
	return &LockoutTracker{
		failures:    make(map[string]*lockoutEntry),
		maxFailures: maxFailures,
		window:      window,
		lockFor:     lockFor,
	}
}

// IsLocked returns true if the account is currently locked out. The
// caller should treat a locked account exactly the same as a wrong
// password to avoid leaking which accounts exist.
func (l *LockoutTracker) IsLocked(email string) bool {
	key := normalizeEmail(email)
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.failures[key]
	if !ok {
		return false
	}
	return time.Now().Before(e.lockUntil)
}

// RecordFailure increments the failure counter and locks the account
// once maxFailures is reached within the rolling window.
func (l *LockoutTracker) RecordFailure(email string) {
	key := normalizeEmail(email)
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.failures[key]
	if !ok || now.Sub(e.firstSeen) > l.window {
		l.failures[key] = &lockoutEntry{count: 1, firstSeen: now}
		return
	}
	e.count++
	if e.count >= l.maxFailures {
		e.lockUntil = now.Add(l.lockFor)
	}
}

// RecordSuccess clears the failure count (called on successful login).
func (l *LockoutTracker) RecordSuccess(email string) {
	key := normalizeEmail(email)
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, key)
}

// CleanIdle drops entries whose lockUntil and firstSeen+window have both
// passed. Called periodically to bound memory.
func (l *LockoutTracker) CleanIdle() {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, e := range l.failures {
		if now.After(e.lockUntil) && now.Sub(e.firstSeen) > l.window {
			delete(l.failures, k)
		}
	}
}

// normalizeEmail lowercases and trims so "Alice@example.com" and
// "alice@example.com  " can't share a single attacker's failure count
// across two records.
func normalizeEmail(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out = append(out, c)
	}
	return string(out)
}
