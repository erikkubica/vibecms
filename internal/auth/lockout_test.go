package auth

import (
	"testing"
	"time"
)

func TestLockoutTracker_LocksAfterMaxFailures(t *testing.T) {
	l := NewLockoutTracker(3, time.Hour, time.Hour)
	email := "alice@example.com"

	for i := 0; i < 2; i++ {
		l.RecordFailure(email)
		if l.IsLocked(email) {
			t.Fatalf("locked after %d failures, want >=3", i+1)
		}
	}
	l.RecordFailure(email)
	if !l.IsLocked(email) {
		t.Fatal("should be locked after 3 failures")
	}
}

func TestLockoutTracker_RecordSuccessClears(t *testing.T) {
	l := NewLockoutTracker(3, time.Hour, time.Hour)
	email := "bob@example.com"

	l.RecordFailure(email)
	l.RecordFailure(email)
	l.RecordSuccess(email)
	// Counter reset — three more failures should be needed to lock.
	l.RecordFailure(email)
	l.RecordFailure(email)
	if l.IsLocked(email) {
		t.Fatal("should not be locked — RecordSuccess must reset counter")
	}
	l.RecordFailure(email)
	if !l.IsLocked(email) {
		t.Fatal("should be locked after 3 fresh failures")
	}
}

func TestLockoutTracker_NormalizesEmail(t *testing.T) {
	l := NewLockoutTracker(2, time.Hour, time.Hour)

	// Mix of casing and whitespace must collapse to one record.
	l.RecordFailure("Alice@example.com")
	l.RecordFailure("alice@example.com  ")
	if !l.IsLocked("ALICE@EXAMPLE.COM") {
		t.Fatal("normalization failed — same logical account counted as different")
	}
}

func TestLockoutTracker_LockExpires(t *testing.T) {
	l := NewLockoutTracker(2, time.Hour, 10*time.Millisecond)
	email := "carol@example.com"

	l.RecordFailure(email)
	l.RecordFailure(email)
	if !l.IsLocked(email) {
		t.Fatal("should be locked")
	}
	time.Sleep(15 * time.Millisecond)
	if l.IsLocked(email) {
		t.Fatal("lock should have expired")
	}
}

func TestLockoutTracker_WindowResetsCounter(t *testing.T) {
	l := NewLockoutTracker(3, 10*time.Millisecond, time.Hour)
	email := "dan@example.com"

	l.RecordFailure(email)
	l.RecordFailure(email)
	time.Sleep(15 * time.Millisecond)
	// Window expired — failure count should reset.
	l.RecordFailure(email)
	if l.IsLocked(email) {
		t.Fatal("window expired; counter should have reset and account should not be locked")
	}
}

func TestLockoutTracker_CleanIdleDropsExpired(t *testing.T) {
	l := NewLockoutTracker(2, 10*time.Millisecond, 10*time.Millisecond)
	for _, email := range []string{"a@x", "b@x", "c@x"} {
		l.RecordFailure(email)
		l.RecordFailure(email)
	}
	if len(l.failures) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(l.failures))
	}
	time.Sleep(20 * time.Millisecond)
	l.CleanIdle()
	if len(l.failures) != 0 {
		t.Fatalf("CleanIdle should have removed expired entries, got %d", len(l.failures))
	}
}
