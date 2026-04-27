package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"gorm.io/gorm"

	"vibecms/internal/models"
)

// defaultAuditRetentionDays caps audit log growth. 90 days is enough for
// most security investigations; operators wanting longer retention should
// stream to an external observability store rather than the OLTP DB.
const defaultAuditRetentionDays = 90

type auditEntry struct {
	tokenID    *int
	tool       string
	argsHash   string
	status     string // ok | error | denied
	errorCode  string
	durationMs int
}

// auditor buffers audit rows and flushes them off the request path.
type auditor struct {
	db *gorm.DB
	ch chan auditEntry
}

func newAuditor(db *gorm.DB) *auditor {
	a := &auditor{db: db, ch: make(chan auditEntry, 256)}
	go a.drain()
	return a
}

func (a *auditor) log(e auditEntry) {
	select {
	case a.ch <- e:
	default:
		// Audit channel full — drop to protect request path; log a warning.
		log.Printf("WARN: mcp audit channel full, dropping entry for tool=%s", e.tool)
	}
}

func (a *auditor) drain() {
	for e := range a.ch {
		row := &models.McpAuditLog{
			TokenID:    e.tokenID,
			Tool:       e.tool,
			ArgsHash:   e.argsHash,
			Status:     e.status,
			ErrorCode:  e.errorCode,
			DurationMs: e.durationMs,
		}
		if err := a.db.Create(row).Error; err != nil {
			log.Printf("WARN: mcp audit write failed: %v", err)
		}
	}
}

func hashArgs(args map[string]any) string {
	if args == nil {
		return ""
	}
	b, err := json.Marshal(args)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// sinceMs returns elapsed milliseconds since t.
func sinceMs(t time.Time) int {
	return int(time.Since(t) / time.Millisecond)
}

// CleanOldAudit deletes audit log rows older than the configured retention.
// Reads `mcp_audit_retention_days` from site_settings; falls back to
// defaultAuditRetentionDays.
func (a *auditor) CleanOldAudit() error {
	days := defaultAuditRetentionDays
	var setting models.SiteSetting
	if err := a.db.Where("key = ?", "mcp_audit_retention_days").First(&setting).Error; err == nil && setting.Value != nil {
		if n, err := strconv.Atoi(*setting.Value); err == nil && n > 0 {
			days = n
		}
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	return a.db.Where("created_at < ?", cutoff).Delete(&models.McpAuditLog{}).Error
}

// StartAuditCleanupLoop sweeps stale audit rows daily until ctx is cancelled.
// Public method on Server to expose the unexported auditor's loop.
func (s *Server) StartAuditCleanupLoop(ctx context.Context, interval time.Duration) {
	if s.auditor != nil {
		s.auditor.StartCleanupLoop(ctx, interval)
	}
}

// StartCleanupLoop sweeps stale audit rows daily until ctx is cancelled.
func (a *auditor) StartCleanupLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	go func() {
		if err := a.CleanOldAudit(); err != nil {
			log.Printf("mcp audit cleanup: initial sweep failed: %v", err)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := a.CleanOldAudit(); err != nil {
					log.Printf("mcp audit cleanup: %v", err)
				}
			}
		}
	}()
}
