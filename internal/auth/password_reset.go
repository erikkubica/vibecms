package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"vibecms/internal/models"

	"gorm.io/gorm"
)

// passwordResetTokenLifetime is how long a reset link stays valid.
// Short windows reduce the impact of leaked emails.
const passwordResetTokenLifetime = time.Hour

var (
	ErrResetTokenNotFound = errors.New("password reset token not found")
	ErrResetTokenExpired  = errors.New("password reset token expired")
	ErrResetTokenUsed     = errors.New("password reset token already used")
)

// PasswordResetService creates and verifies password reset tokens.
// Tokens are 32-byte random values; only the SHA-256 hash is stored.
// Verification is constant-time on the hash to prevent timing oracles.
type PasswordResetService struct {
	db *gorm.DB
}

// NewPasswordResetService constructs a service backed by the given DB.
func NewPasswordResetService(db *gorm.DB) *PasswordResetService {
	return &PasswordResetService{db: db}
}

// IssueToken creates a new reset token for the given user and returns
// the raw token. The caller is responsible for sending it (typically
// via the event bus → email dispatcher path). Returns the row's
// expiry so the caller can include it in the email.
func (s *PasswordResetService) IssueToken(userID int, ip, userAgent string) (string, time.Time, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", time.Time{}, fmt.Errorf("generating reset token: %w", err)
	}
	rawToken := hex.EncodeToString(tokenBytes)
	tokenHash := hashResetToken(rawToken)

	expires := time.Now().Add(passwordResetTokenLifetime)
	row := models.PasswordResetToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expires,
		IPAddress: &ip,
		UserAgent: &userAgent,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return "", time.Time{}, fmt.Errorf("storing reset token: %w", err)
	}
	return rawToken, expires, nil
}

// VerifyAndConsume looks up a reset token, verifies expiry and single-use,
// marks it consumed, and returns the associated user ID. Constant-time
// comparison is implicit because the lookup hashes the input and compares
// to the unique-indexed hash column — no per-byte branching on the raw
// token. Returns ErrResetTokenNotFound for unknown/invalid tokens to
// avoid leaking which tokens exist.
func (s *PasswordResetService) VerifyAndConsume(rawToken string) (int, error) {
	if rawToken == "" {
		return 0, ErrResetTokenNotFound
	}
	tokenHash := hashResetToken(rawToken)

	var row models.PasswordResetToken
	err := s.db.Where("token_hash = ?", tokenHash).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrResetTokenNotFound
		}
		return 0, fmt.Errorf("looking up reset token: %w", err)
	}

	// Defence in depth — confirm the hash actually matches in
	// constant time before honouring the row.
	if subtle.ConstantTimeCompare([]byte(row.TokenHash), []byte(tokenHash)) != 1 {
		return 0, ErrResetTokenNotFound
	}

	if row.UsedAt != nil {
		return 0, ErrResetTokenUsed
	}
	if time.Now().After(row.ExpiresAt) {
		return 0, ErrResetTokenExpired
	}

	now := time.Now()
	if err := s.db.Model(&row).Update("used_at", &now).Error; err != nil {
		return 0, fmt.Errorf("consuming reset token: %w", err)
	}
	return row.UserID, nil
}

// InvalidateAllForUser marks every outstanding token for this user as
// used. Call on successful password reset (the consumed token is one
// of these, but other in-flight ones must be voided too) and on any
// security-relevant change to the account.
func (s *PasswordResetService) InvalidateAllForUser(userID int) error {
	now := time.Now()
	return s.db.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Update("used_at", &now).Error
}

// CleanExpired prunes tokens that are past expiry or have been used.
// Mirrors SessionService.CleanExpired and is hooked to the same retention loop.
func (s *PasswordResetService) CleanExpired() error {
	return s.db.Where("expires_at < ? OR used_at IS NOT NULL", time.Now()).
		Delete(&models.PasswordResetToken{}).Error
}

// StartCleanupLoop sweeps consumed/expired password reset tokens on a fixed
// interval until ctx is cancelled.
func (s *PasswordResetService) StartCleanupLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}
	go func() {
		if err := s.CleanExpired(); err != nil {
			log.Printf("password reset cleanup: initial sweep failed: %v", err)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.CleanExpired(); err != nil {
					log.Printf("password reset cleanup: %v", err)
				}
			}
		}
	}()
}

// hashResetToken returns the hex-encoded SHA-256 of the raw token.
// Same primitive as session token hashing — see hashToken in session_svc.go.
func hashResetToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
