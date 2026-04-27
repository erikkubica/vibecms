package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"vibecms/internal/coreapi"
	pb "vibecms/pkg/plugin/proto"
)

// fireWebhook sends an HTTP POST to the configured webhook URL and logs the result.
// It runs fire-and-log (no retry queue). Safe to call in a goroutine.
func (p *FormsPlugin) fireWebhook(ctx context.Context, form map[string]any, submissionID uint, data map[string]any, metadata map[string]any) {
	s := getFormSettings(form)
	enabled, _ := s["webhook_enabled"].(bool)
	if !enabled {
		return
	}
	url, _ := s["webhook_url"].(string)
	if url == "" {
		return
	}

	formID := uint(0)
	if v, ok := form["id"].(float64); ok {
		formID = uint(v)
	}

	headers := map[string]string{"Content-Type": "application/json"}
	if extra, _ := s["webhook_headers"].(string); extra != "" {
		var h map[string]string
		if err := json.Unmarshal([]byte(extra), &h); err == nil {
			for k, v := range h {
				headers[k] = v
			}
		}
	}

	body, _ := json.Marshal(map[string]any{
		"form_id":       formID,
		"form_slug":     form["slug"],
		"submission_id": submissionID,
		"data":          data,
		"metadata":      metadata,
		"fired_at":      time.Now().Format(time.RFC3339),
	})

	start := time.Now()
	res, fetchErr := p.host.Fetch(ctx, coreapi.FetchRequest{
		Method:  "POST",
		URL:     url,
		Headers: headers,
		Body:    string(body),
		Timeout: 5,
	})
	duration := int(time.Since(start).Milliseconds())

	logEntry := map[string]any{
		"form_id":      formID,
		"submission_id": submissionID,
		"url":          url,
		"request_body": string(body),
		"duration_ms":  duration,
	}
	if fetchErr != nil {
		logEntry["error"] = fetchErr.Error()
		logEntry["status_code"] = 0
		p.host.Log(ctx, "warn", fmt.Sprintf("webhook %s failed: %s", url, fetchErr), nil)
	} else {
		logEntry["status_code"] = res.StatusCode
		logEntry["response_body"] = webhookTruncate(res.Body, 4096)
	}
	if _, err := p.host.DataCreate(ctx, "form_webhook_logs", logEntry); err != nil {
		p.host.Log(ctx, "warn", "webhook log write failed: "+err.Error(), nil)
	}
}

// handleWebhookLogs returns the webhook log entries for a specific form.
func (p *FormsPlugin) handleWebhookLogs(ctx context.Context, formID uint) (*pb.PluginHTTPResponse, error) {
	result, err := p.host.DataQuery(ctx, "form_webhook_logs", coreapi.DataStoreQuery{
		Where:   map[string]any{"form_id": formID},
		OrderBy: "created_at DESC",
		Limit:   50,
	})
	if err != nil {
		return jsonError(500, "QUERY_FAILED", err.Error()), nil
	}
	return jsonResponse(200, result), nil
}

func webhookTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
