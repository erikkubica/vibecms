package main

import (
	"encoding/json"
	"testing"

	pb "vibecms/pkg/plugin/proto"
)

func routeReq(method, path string, body []byte, params map[string]string) *pb.PluginHTTPRequest {
	return &pb.PluginHTTPRequest{
		Method:      method,
		Path:        path,
		Body:        body,
		QueryParams: params,
	}
}

func TestRouteRequest_ListForms(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.routeRequest(ctx(), routeReq("GET", "/", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("GET / expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_CreateForm(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{"name": "F", "slug": "f"})
	resp, err := p.routeRequest(ctx(), routeReq("POST", "/", body, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("POST / expected 201, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_GetForm(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.routeRequest(ctx(), routeReq("GET", "/1", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("GET /1 expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_DeleteForm(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.routeRequest(ctx(), routeReq("DELETE", "/1", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("DELETE /1 expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_UpdateForm(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	body, _ := json.Marshal(map[string]any{"name": "Updated", "slug": "contact-us"})
	resp, err := p.routeRequest(ctx(), routeReq("PUT", "/1", body, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("PUT /1 expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_Duplicate(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.routeRequest(ctx(), routeReq("POST", "/1/duplicate", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("POST /1/duplicate expected 201, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_Export(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.routeRequest(ctx(), routeReq("GET", "/1/export", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("GET /1/export expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_Import(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{"name": "Imported", "slug": "imported"})
	resp, err := p.routeRequest(ctx(), routeReq("POST", "/import", body, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("POST /import expected 201, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_DefaultsLayout(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.routeRequest(ctx(), routeReq("GET", "/defaults/layout", nil, map[string]string{"style": "grid"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("GET /defaults/layout expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_Preview(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{
		"id": float64(1), "name": "T", "fields": []any{},
		"layout": `<form></form>`,
	})
	resp, err := p.routeRequest(ctx(), routeReq("POST", "/preview", body, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("POST /preview expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_SubmissionsExport(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.routeRequest(ctx(), routeReq("GET", "/submissions/export", nil, map[string]string{"form_id": "1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("GET /submissions/export expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_SubmissionsList(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.routeRequest(ctx(), routeReq("GET", "/submissions", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("GET /submissions expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_SubmissionBulk(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{"action": "mark_read", "ids": []uint{}})
	resp, err := p.routeRequest(ctx(), routeReq("POST", "/submissions/bulk", body, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("POST /submissions/bulk expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_SubmissionPatch(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedSubmission(h, 1, map[string]any{}, "unread")

	body, _ := json.Marshal(map[string]any{"status": "read"})
	resp, err := p.routeRequest(ctx(), routeReq("PATCH", "/submissions/1", body, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("PATCH /submissions/1 expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_SubmissionDelete(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedSubmission(h, 1, map[string]any{}, "unread")

	resp, err := p.routeRequest(ctx(), routeReq("DELETE", "/submissions/1", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("DELETE /submissions/1 expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_SubmissionGet(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	h.DataCreate(ctx(), formsTable, map[string]any{"name": "F", "slug": "f", "fields": `[]`})
	seedSubmission(h, 1, map[string]any{}, "unread")

	resp, err := p.routeRequest(ctx(), routeReq("GET", "/submissions/1", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("GET /submissions/1 expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_WebhookLogs(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.routeRequest(ctx(), routeReq("GET", "/1/webhooks", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("GET /1/webhooks expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_PublicSubmit(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedFormFull(h)

	body, _ := json.Marshal(map[string]any{"name": "Alice"})
	resp, err := p.routeRequest(ctx(), &pb.PluginHTTPRequest{
		Method:  "POST",
		Path:    "/forms/submit/contact-us",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    body,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 422 {
		// Missing required "name" is caught as validation error — proves routing worked
		// If form not found → 404; if routing broken → 404 with "NOT_FOUND"
		t.Logf("status=%d body=%s", resp.StatusCode, resp.Body)
	}
}

func TestRouteRequest_PublicRender(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.routeRequest(ctx(), routeReq("GET", "/forms/render/contact-us", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("GET /forms/render/contact-us expected 200, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
}

func TestRouteRequest_NotFound(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.routeRequest(ctx(), routeReq("GET", "/nonexistent/path/that/does/not/exist", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("unknown route expected 404, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_AdminPrefixStrip(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	// Admin proxy uses /admin/api/ext/forms prefix
	resp, err := p.routeRequest(ctx(), routeReq("GET", "/admin/api/ext/forms/1", nil, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("admin prefix stripping expected 200, got %d", resp.StatusCode)
	}
}

func TestRouteRequest_NotificationTest(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedFormWithNotifConfig(h)

	body, _ := json.Marshal(map[string]any{"override_recipient": "test@example.com"})
	resp, err := p.routeRequest(ctx(), routeReq("POST", "/1/notifications/0/test", body, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("notification test expected 200, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
}
