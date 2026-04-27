package auth

import (
	"testing"
	"time"
)

func TestPerIPLimiter_AllowsBurstThenBlocks(t *testing.T) {
	// 60/min sustained = 1 per second; burst of 3 lets first 3 through fast.
	l := NewPerIPLimiter(60, time.Minute, 3)
	const ip = "1.2.3.4"
	for i := 0; i < 3; i++ {
		if !l.Allow(ip) {
			t.Fatalf("should allow burst request %d", i+1)
		}
	}
	// 4th in the same instant should hit the limiter — the burst is
	// drained and the refill is one token per second.
	if l.Allow(ip) {
		t.Fatal("should reject 4th immediate request (burst exhausted)")
	}
}

func TestPerIPLimiter_TracksPerIP(t *testing.T) {
	l := NewPerIPLimiter(60, time.Minute, 1)
	if !l.Allow("1.1.1.1") {
		t.Fatal("first IP should be allowed")
	}
	if !l.Allow("2.2.2.2") {
		t.Fatal("different IP should have its own bucket")
	}
	// Same IP again — burst exhausted.
	if l.Allow("1.1.1.1") {
		t.Fatal("same IP should be rejected after burst")
	}
}

func TestPerIPLimiter_CleanIdleRemovesStaleIPs(t *testing.T) {
	l := NewPerIPLimiter(60, time.Minute, 1)
	l.Allow("3.3.3.3")
	l.Allow("4.4.4.4")
	if len(l.limiters) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(l.limiters))
	}
	// Wait a moment, mark one IP as recently active, then clean entries
	// idle for >5ms.
	time.Sleep(10 * time.Millisecond)
	l.Allow("3.3.3.3") // refresh lastSeen
	l.CleanIdle(8 * time.Millisecond)

	if _, ok := l.limiters["3.3.3.3"]; !ok {
		t.Fatal("recently active IP was incorrectly cleaned")
	}
	if _, ok := l.limiters["4.4.4.4"]; ok {
		t.Fatal("idle IP should have been cleaned")
	}
}

// TestNormalizeEmail covers the normalizer used by the lockout tracker.
// Edge cases:
//   - mixed case folds to lower
//   - whitespace stripped
//   - tab/CR/LF stripped
//   - non-ascii left alone (we only fold A-Z, not Unicode capitals)
func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Alice@example.com", "alice@example.com"},
		{"  bob@x.com\n", "bob@x.com"},
		{"\tCAROL@X.COM ", "carol@x.com"},
		{"dän@example.com", "dän@example.com"}, // intentionally not folded
	}
	for _, c := range cases {
		if got := normalizeEmail(c.in); got != c.want {
			t.Errorf("normalizeEmail(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
