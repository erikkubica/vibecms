package main

import (
	"encoding/json"
	"testing"

	pb "vibecms/pkg/plugin/proto"
)

func seedSubmission(h *FakeHost, formID uint, data map[string]any, status string) map[string]any {
	row, _ := h.DataCreate(ctx(), submissionsTable, map[string]any{
		"form_id": float64(formID),
		"data":    data,
		"status":  status,
	})
	return row
}

// ---- handleSubmissionPatch ----

func TestHandleSubmissionPatch_Success(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedSubmission(h, 1, map[string]any{}, "unread")

	body, _ := json.Marshal(map[string]any{"status": "read"})
	resp, err := p.handleSubmissionPatch(ctx(), 1, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
	// Check status updated
	if h.Tables[submissionsTable][1]["status"] != "read" {
		t.Errorf("status should be 'read', got %v", h.Tables[submissionsTable][1]["status"])
	}
}

func TestHandleSubmissionPatch_InvalidStatus(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedSubmission(h, 1, map[string]any{}, "unread")

	body, _ := json.Marshal(map[string]any{"status": "invalid-status"})
	resp, err := p.handleSubmissionPatch(ctx(), 1, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 422 {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestHandleSubmissionPatch_InvalidJSON(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleSubmissionPatch(ctx(), 1, []byte("not-json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// ---- handleSubmissionGet ----

func TestHandleSubmissionGet_Found(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	// Seed a form and submission
	h.DataCreate(ctx(), formsTable, map[string]any{
		"name":   "My Form",
		"slug":   "my-form",
		"fields": `[{"id":"name","label":"Name","type":"text"}]`,
	})
	seedSubmission(h, 1, map[string]any{"name": "Alice"}, "unread")

	resp, err := p.handleSubmissionGet(ctx(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var sub map[string]any
	decodeBody(t, resp.Body, &sub)
	if sub["form_name"] != "My Form" {
		t.Errorf("expected form_name 'My Form', got %v", sub["form_name"])
	}
}

func TestHandleSubmissionGet_NotFound(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleSubmissionGet(ctx(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---- handleSubmissionDelete ----

func TestHandleSubmissionDelete_Success(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedSubmission(h, 1, map[string]any{}, "unread")

	resp, err := p.handleSubmissionDelete(ctx(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if len(h.Tables[submissionsTable]) != 0 {
		t.Error("submission should be deleted")
	}
}

func TestHandleSubmissionDelete_WithFileCleanup(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	h.StoredFiles["forms/submissions/1/test.pdf"] = []byte("data")

	h.DataCreate(ctx(), submissionsTable, map[string]any{
		"form_id": float64(1),
		"status":  "unread",
		"data":    map[string]any{"doc": map[string]any{"url": "/forms/submissions/1/test.pdf"}},
	})

	resp, err := p.handleSubmissionDelete(ctx(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if len(h.StoredFiles) != 0 {
		t.Error("file should be deleted with the submission")
	}
}

func TestHandleSubmissionDelete_NotFound(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleSubmissionDelete(ctx(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---- handleSubmissions ----

func TestHandleSubmissions_List(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedSubmission(h, 1, map[string]any{}, "unread")
	seedSubmission(h, 1, map[string]any{}, "read")

	req := &pb.PluginHTTPRequest{
		Method:      "GET",
		Path:        "submissions",
		QueryParams: map[string]string{},
	}
	resp, err := p.handleSubmissions(ctx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]any
	decodeBody(t, resp.Body, &result)
	if result["total"] == nil {
		t.Error("response should have total")
	}
}

func TestHandleSubmissions_Pagination(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	for i := 0; i < 5; i++ {
		seedSubmission(h, 1, map[string]any{}, "unread")
	}

	req := &pb.PluginHTTPRequest{
		Method:      "GET",
		Path:        "submissions",
		QueryParams: map[string]string{"page": "1", "per_page": "2"},
	}
	resp, err := p.handleSubmissions(ctx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]any
	decodeBody(t, resp.Body, &result)
	rows, _ := result["rows"].([]any)
	if len(rows) > 2 {
		t.Errorf("expected at most 2 rows per page, got %d", len(rows))
	}
}

// ---- handleSubmissionsBulk ----

func TestHandleSubmissionsBulk_MarkRead(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedSubmission(h, 1, map[string]any{}, "unread")
	seedSubmission(h, 1, map[string]any{}, "unread")

	body, _ := json.Marshal(map[string]any{
		"action": "mark_read",
		"ids":    []uint{1, 2},
	})
	resp, err := p.handleSubmissionsBulk(ctx(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]any
	decodeBody(t, resp.Body, &result)
	if result["count"] == nil {
		t.Error("response should have count")
	}
}

func TestHandleSubmissionsBulk_Delete(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedSubmission(h, 1, map[string]any{}, "unread")
	seedSubmission(h, 1, map[string]any{}, "unread")

	body, _ := json.Marshal(map[string]any{
		"action": "delete",
		"ids":    []uint{1, 2},
	})
	resp, err := p.handleSubmissionsBulk(ctx(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if len(h.Tables[submissionsTable]) != 0 {
		t.Errorf("expected 0 submissions after delete, got %d", len(h.Tables[submissionsTable]))
	}
}

func TestHandleSubmissionsBulk_UnknownAction(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{
		"action": "unknown",
		"ids":    []uint{1},
	})
	resp, err := p.handleSubmissionsBulk(ctx(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 422 {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestHandleSubmissionsBulk_EmptyIDs(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{
		"action": "mark_read",
		"ids":    []uint{},
	})
	resp, err := p.handleSubmissionsBulk(ctx(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleSubmissionsBulk_TooManyIDs(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	ids := make([]uint, 1001)
	body, _ := json.Marshal(map[string]any{
		"action": "mark_read",
		"ids":    ids,
	})
	resp, err := p.handleSubmissionsBulk(ctx(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 422 {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestHandleSubmissionsBulk_Archive(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedSubmission(h, 1, map[string]any{}, "unread")

	body, _ := json.Marshal(map[string]any{
		"action": "archive",
		"ids":    []uint{1},
	})
	resp, err := p.handleSubmissionsBulk(ctx(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
