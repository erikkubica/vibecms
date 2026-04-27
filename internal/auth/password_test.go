package auth

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordCost_DefaultsTo12(t *testing.T) {
	t.Setenv("BCRYPT_COST", "")
	if got := passwordCost(); got != defaultBcryptCost {
		t.Errorf("default cost = %d, want %d", got, defaultBcryptCost)
	}
}

func TestPasswordCost_HonoursValidEnv(t *testing.T) {
	t.Setenv("BCRYPT_COST", "13")
	if got := passwordCost(); got != 13 {
		t.Errorf("cost = %d, want 13", got)
	}
}

func TestPasswordCost_ClampsToBounds(t *testing.T) {
	t.Setenv("BCRYPT_COST", "4")
	if got := passwordCost(); got != minBcryptCost {
		t.Errorf("cost = %d, want clamp to %d", got, minBcryptCost)
	}
	t.Setenv("BCRYPT_COST", "20")
	if got := passwordCost(); got != maxBcryptCost {
		t.Errorf("cost = %d, want clamp to %d", got, maxBcryptCost)
	}
}

func TestPasswordCost_FallbackOnNonNumeric(t *testing.T) {
	t.Setenv("BCRYPT_COST", "not-a-number")
	if got := passwordCost(); got != defaultBcryptCost {
		t.Errorf("cost = %d, want fallback %d", got, defaultBcryptCost)
	}
}

func TestHashPassword_VerifiesAgainstBcrypt(t *testing.T) {
	// Use the minimum cost so the test is fast.
	os.Setenv("BCRYPT_COST", "10")
	t.Cleanup(func() { os.Unsetenv("BCRYPT_COST") })

	hash, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte("hunter2")); err != nil {
		t.Fatalf("bcrypt verify failed: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte("wrong")); err == nil {
		t.Fatal("wrong password should not verify")
	}
}

func TestTrustForwardedProto(t *testing.T) {
	t.Setenv("TRUST_FORWARDED_PROTO", "")
	if trustForwardedProto() {
		t.Error("default should be false")
	}
	t.Setenv("TRUST_FORWARDED_PROTO", "true")
	if !trustForwardedProto() {
		t.Error("'true' should opt in")
	}
	t.Setenv("TRUST_FORWARDED_PROTO", "1")
	if !trustForwardedProto() {
		t.Error("'1' should opt in")
	}
	t.Setenv("TRUST_FORWARDED_PROTO", "false")
	if trustForwardedProto() {
		t.Error("'false' should not opt in")
	}
}
