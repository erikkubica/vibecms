package main

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestRateLimiter_AllowUnlimited(t *testing.T) {
	rl := NewRateLimiter(100)

	// ip="" or limit<=0 → always allow
	if !rl.Allow("", 10) {
		t.Error("empty IP should always pass")
	}
	if !rl.Allow("1.2.3.4", 0) {
		t.Error("limit=0 should always pass")
	}
	if !rl.Allow("1.2.3.4", -1) {
		t.Error("limit<0 should always pass")
	}
}

func TestRateLimiter_AllowBurst(t *testing.T) {
	rl := NewRateLimiter(100)
	ip := "10.0.0.1"
	limit := 5

	// First 5 requests should be allowed (burst == limitPerHour)
	for i := 0; i < limit; i++ {
		if !rl.Allow(ip, limit) {
			t.Errorf("request %d should be allowed within burst", i+1)
		}
	}

	// Next request should be denied
	if rl.Allow(ip, limit) {
		t.Error("request beyond burst should be denied")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter(100)
	limit := 2

	// IP1: exhaust
	rl.Allow("1.1.1.1", limit)
	rl.Allow("1.1.1.1", limit)
	if rl.Allow("1.1.1.1", limit) {
		t.Error("IP1 should be rate-limited")
	}

	// IP2: should still work
	if !rl.Allow("2.2.2.2", limit) {
		t.Error("IP2 should not be rate-limited")
	}
}

func TestRateLimiter_LRUEviction(t *testing.T) {
	capacity := 3
	rl := NewRateLimiter(capacity)
	limit := 1

	// Fill cache: IP1, IP2, IP3
	rl.Allow("1.1.1.1", limit)
	rl.Allow("2.2.2.2", limit)
	rl.Allow("3.3.3.3", limit)

	// Add IP4 — should evict the least recently used (IP1 or IP2)
	rl.Allow("4.4.4.4", limit)

	// A new entry for an evicted IP should get a fresh limiter (allowed)
	// We can't know exactly which was evicted, but the cache should still work
	if rl.cache.Len() > capacity {
		t.Errorf("cache size %d exceeds capacity %d", rl.cache.Len(), capacity)
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(10000)
	ip := "concurrent-test"
	limit := 100

	var allowed int64
	var denied int64
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow(ip, limit) {
				atomic.AddInt64(&allowed, 1)
			} else {
				atomic.AddInt64(&denied, 1)
			}
		}()
	}
	wg.Wait()

	// Burst == limit == 100, so at most 100 should pass
	if allowed > int64(limit) {
		t.Errorf("expected at most %d allowed, got %d", limit, allowed)
	}
	if allowed+denied != 200 {
		t.Errorf("total requests should be 200, got %d", allowed+denied)
	}
}
