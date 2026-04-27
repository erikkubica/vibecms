package auth

import (
	"testing"
	"time"
)

// TestAbsorbBcryptTime_ApproximatelyMatchesHash burns a bcrypt
// compare and asserts it took at least a few milliseconds. We can't
// pin an exact duration (bcrypt cost varies by hardware), but a
// successful absorb should be measurably slower than zero — that's
// the whole point. Without this, the no-user login path returns in
// microseconds and an attacker can enumerate registered emails by
// timing.
func TestAbsorbBcryptTime_TakesRealTime(t *testing.T) {
	// Warm the once-init so the first measured call doesn't include
	// hash generation cost.
	AbsorbBcryptTime()

	start := time.Now()
	got := AbsorbBcryptTime()
	dur := time.Since(start)

	if got != false {
		t.Fatal("AbsorbBcryptTime must always return false")
	}
	// At cost=10 (the floor we accept), a single bcrypt compare is
	// typically 20-100ms on dev hardware. CI is sometimes noisy so we
	// pick a forgiving lower bound; anything sub-millisecond would
	// indicate the absorb is broken / no-op'd.
	if dur < 5*time.Millisecond {
		t.Fatalf("AbsorbBcryptTime returned in %v — expected real bcrypt cost", dur)
	}
}

func TestAbsorbBcryptTime_Idempotent(t *testing.T) {
	// Multiple calls must not panic / corrupt the dummy hash.
	for i := 0; i < 3; i++ {
		if AbsorbBcryptTime() {
			t.Fatal("must always return false")
		}
	}
}
