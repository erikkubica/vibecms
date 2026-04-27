package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"vibecms/internal/models"

	"gorm.io/gorm"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
	ErrInvalidToken    = errors.New("invalid session token")
)

// SessionService manages user sessions backed by the database.
type SessionService struct {
	db            *gorm.DB
	sessionExpiry time.Duration
}

// NewSessionService creates a new SessionService with the given database connection
// and session expiry duration in hours.
func NewSessionService(db *gorm.DB, expiryHours int) *SessionService {
	return &SessionService{
		db:            db,
		sessionExpiry: time.Duration(expiryHours) * time.Hour,
	}
}

// CreateSession generates a new session for the given user and returns the raw token.
// The token is a random 32-byte value; only its SHA-256 hash is stored in the database.
func (s *SessionService) CreateSession(userID int, ip, userAgent string) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}

	rawToken := hex.EncodeToString(tokenBytes)
	tokenHash := hashToken(rawToken)

	session := models.Session{
		UserID:    userID,
		TokenHash: tokenHash,
		IPAddress: &ip,
		UserAgent: &userAgent,
		ExpiresAt: time.Now().Add(s.sessionExpiry),
	}

	if err := s.db.Create(&session).Error; err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return rawToken, nil
}

// ValidateSession verifies the given raw token against stored session hashes.
// It checks for expiry and returns the associated user with preloaded data.
func (s *SessionService) ValidateSession(token string) (*models.User, error) {
	if token == "" {
		return nil, ErrInvalidToken
	}

	tokenHash := hashToken(token)

	var session models.Session
	err := s.db.Preload("User").Preload("User.Role").Where("token_hash = ?", tokenHash).First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		// Clean up the expired session.
		s.db.Delete(&session)
		return nil, ErrSessionExpired
	}

	return &session.User, nil
}

// DeleteSession removes the session identified by the given raw token.
func (s *SessionService) DeleteSession(token string) error {
	if token == "" {
		return ErrInvalidToken
	}

	tokenHash := hashToken(token)

	result := s.db.Where("token_hash = ?", tokenHash).Delete(&models.Session{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// CleanExpired removes all sessions that have passed their expiry time.
// This is intended to be called periodically by a background cron job.
func (s *SessionService) CleanExpired() error {
	result := s.db.Where("expires_at < ?", time.Now()).Delete(&models.Session{})
	if result.Error != nil {
		return fmt.Errorf("failed to clean expired sessions: %w", result.Error)
	}
	return nil
}

// StartCleanupLoop runs CleanExpired on a fixed interval until ctx is
// cancelled. Spawn this once at startup; it logs failures but never
// returns an error so the goroutine doesn't leak silently. Call
// CleanExpired once before the loop body so first cleanup runs at
// boot rather than after the first interval.
func (s *SessionService) StartCleanupLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}
	go func() {
		if err := s.CleanExpired(); err != nil {
			log.Printf("session cleanup: initial sweep failed: %v", err)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.CleanExpired(); err != nil {
					log.Printf("session cleanup: %v", err)
				}
			}
		}
	}()
}

// hashToken returns the hex-encoded SHA-256 hash of the given token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
