package api

import (
	"crypto/subtle"
	"runtime"
	"time"

	"vibecms/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// HealthHandler provides health check and monitoring endpoints.
type HealthHandler struct {
	db        *gorm.DB
	startTime time.Time
}

// NewHealthHandler creates a new HealthHandler with a reference to the database
// and records the server start time.
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{
		db:        db,
		startTime: time.Now(),
	}
}

// HealthCheck returns a simple liveness response. This endpoint is public.
func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "up",
	})
}

// Stats returns detailed server statistics including uptime, goroutine count,
// database connectivity, and content counts. This endpoint requires bearer token auth.
func (h *HealthHandler) Stats(c *fiber.Ctx) error {
	// Check database connectivity.
	dbStatus := "connected"
	sqlDB, err := h.db.DB()
	if err != nil {
		dbStatus = "error"
	} else if err := sqlDB.Ping(); err != nil {
		dbStatus = "disconnected"
	}

	// Count content nodes.
	var totalNodes int64
	var publishedNodes int64
	h.db.Model(&models.ContentNode{}).Count(&totalNodes)
	h.db.Model(&models.ContentNode{}).Where("status = ?", "published").Count(&publishedNodes)

	return Success(c, fiber.Map{
		"performance": fiber.Map{
			"uptime_seconds": int(time.Since(h.startTime).Seconds()),
			"goroutines":     runtime.NumGoroutine(),
		},
		"health": fiber.Map{
			"database": dbStatus,
			"storage":  "ok",
		},
		"content": fiber.Map{
			"total_nodes":     totalNodes,
			"published_nodes": publishedNodes,
		},
	})
}

// BearerTokenRequired returns middleware that validates the Authorization header
// against the configured monitor bearer token.
func BearerTokenRequired(token string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if token == "" {
			return Error(c, fiber.StatusServiceUnavailable, "NOT_CONFIGURED", "Monitoring token not configured")
		}

		auth := c.Get("Authorization")
		if auth == "" {
			return Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Bearer token required")
		}

		// Expect "Bearer <token>" format.
		const prefix = "Bearer "
		if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix {
			return Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization format")
		}

		// Use constant-time compare to deny a timing-oracle attack on the
		// bearer token. Plain != would terminate at the first mismatching
		// byte, leaking the prefix one byte at a time.
		if subtle.ConstantTimeCompare([]byte(auth[len(prefix):]), []byte(token)) != 1 {
			return Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Invalid bearer token")
		}

		return c.Next()
	}
}
