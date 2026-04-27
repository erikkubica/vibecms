package cms

import (
	"context"
	"log"
	"strconv"
	"time"

	"vibecms/internal/models"
)

// CleanOldRevisions prunes content_node_revisions, keeping the most
// recent N revisions per node (default defaultRevisionsPerNode, override
// via site_settings.content_revisions_per_node). Uses ROW_NUMBER over
// PARTITION BY content_node_id so each node keeps its own window.
//
// Without this, every PATCH /nodes/:id appends a row forever and
// blocks_data snapshots accumulate indefinitely.
func (s *ContentService) CleanOldRevisions() error {
	keep := defaultRevisionsPerNode
	var setting models.SiteSetting
	if err := s.db.Where("key = ?", "content_revisions_per_node").First(&setting).Error; err == nil && setting.Value != nil {
		if n, err := strconv.Atoi(*setting.Value); err == nil && n > 0 {
			keep = n
		}
	}
	// Window function trims older rows per node. Postgres-specific syntax
	// — fine because the kernel already requires Postgres.
	return s.db.Exec(`
		DELETE FROM content_node_revisions
		WHERE id IN (
			SELECT id FROM (
				SELECT id,
				       ROW_NUMBER() OVER (PARTITION BY content_node_id ORDER BY created_at DESC) AS rn
				FROM content_node_revisions
			) t WHERE rn > ?
		)`, keep).Error
}

// StartRevisionCleanupLoop sweeps stale revisions daily until ctx is cancelled.
func (s *ContentService) StartRevisionCleanupLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	go func() {
		if err := s.CleanOldRevisions(); err != nil {
			log.Printf("revision cleanup: initial sweep failed: %v", err)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.CleanOldRevisions(); err != nil {
					log.Printf("revision cleanup: %v", err)
				}
			}
		}
	}()
}
