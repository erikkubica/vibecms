package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"vibecms/internal/coreapi"
)

// startRetentionWorker spawns a goroutine that runs every hour.
// Cancelled when ctx is cancelled (i.e. on Shutdown).
func (p *FormsPlugin) startRetentionWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		defer ticker.Stop()
		// Run once at startup so cleanup doesn't wait an hour after restart.
		p.runRetention(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.runRetention(ctx)
			}
		}
	}()
}

func (p *FormsPlugin) runRetention(ctx context.Context) {
	forms, err := p.host.DataQuery(ctx, formsTable, coreapi.DataStoreQuery{})
	if err != nil {
		p.host.Log(ctx, "error", "retention: list forms failed: "+err.Error(), nil)
		return
	}

	for _, row := range forms.Rows {
		f := normalizeForm(row)
		s := getFormSettings(f)
		period, _ := s["retention_period"].(string)
		days, _ := strconv.Atoi(period)
		if days <= 0 {
			continue
		}

		cutoff := time.Now().AddDate(0, 0, -days)
		formID := uint(0)
		if v, ok := f["id"].(float64); ok {
			formID = uint(v)
		}

		// Find expired submissions
		subs, err := p.host.DataQuery(ctx, submissionsTable, coreapi.DataStoreQuery{
			Where: map[string]any{"form_id": formID},
			Raw:   "created_at < ?",
			Args:  []any{cutoff},
		})
		if err != nil {
			continue
		}

		for _, sub := range subs.Rows {
			id, _ := sub["id"].(float64)
			// Delete files
			data := parseJSONMap(sub["data"])
			for _, v := range data {
				deleteFileValueIfPresent(ctx, p.host, v)
			}
			// Delete row
			_ = p.host.DataDelete(ctx, submissionsTable, uint(id))
		}
		if len(subs.Rows) > 0 {
			p.host.Log(ctx, "info", fmt.Sprintf("retention: form %d deleted %d submissions", formID, len(subs.Rows)), nil)
		}
	}
}

// deleteFileValueIfPresent walks a submission value and deletes any file URLs found.
func deleteFileValueIfPresent(ctx context.Context, host coreapi.CoreAPI, v any) {
	switch val := v.(type) {
	case map[string]any:
		if u, ok := val["url"].(string); ok && strings.HasPrefix(u, "/forms/submissions/") {
			_ = host.DeleteFile(ctx, strings.TrimPrefix(u, "/"))
		}
	case []any:
		for _, item := range val {
			deleteFileValueIfPresent(ctx, host, item)
		}
	}
}
