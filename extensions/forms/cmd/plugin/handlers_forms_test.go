package main

import (
	"encoding/json"
	"strings"
	"testing"

	pb "vibecms/pkg/plugin/proto"
)

func makeReq(method, path string, body []byte) *pb.PluginHTTPRequest {
	return &pb.PluginHTTPRequest{
		Method: method,
		Path:   path,
		Body:   body,
	}
}

func makeReqWithQuery(method, path string, params map[string]string) *pb.PluginHTTPRequest {
	return &pb.PluginHTTPRequest{
		Method:      method,
		Path:        path,
		QueryParams: params,
	}
}

func decodeBody(t *testing.T, body []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("decode body: %v (body=%s)", err, body)
	}
}

func seedForm(h *FakeHost) map[string]any {
	row, _ := h.DataCreate(ctx(), formsTable, map[string]any{
		"name":   "Contact Us",
		"slug":   "contact-us",
		"fields": `[{"id":"name","label":"Name","type":"text"}]`,
		"layout": `<form>{{range .fields_list}}<input name="{{.id}}">{{end}}</form>`,
		"settings": `{}`,
		"notifications": `[]`,
	})
	return row
}

// ---- handleListForms ----

func TestHandleListForms(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.handleListForms(ctx(), makeReq("GET", "/", nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result struct {
		Rows []map[string]any `json:"rows"`
	}
	decodeBody(t, resp.Body, &result)
	if len(result.Rows) != 1 {
		t.Errorf("expected 1 form, got %d", len(result.Rows))
	}
}

// ---- handleCreateForm ----

func TestHandleCreateForm_Success(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{"name": "New Form", "slug": "new-form"})
	resp, err := p.handleCreateForm(ctx(), makeReq("POST", "/", body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
}

func TestHandleCreateForm_MissingName(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{"slug": "no-name"})
	resp, err := p.handleCreateForm(ctx(), makeReq("POST", "/", body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 422 {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
	var out map[string]any
	decodeBody(t, resp.Body, &out)
	fields, _ := out["fields"].(map[string]any)
	if _, ok := fields["name"]; !ok {
		t.Errorf("expected fields.name error, got %v", out)
	}
}

func TestHandleCreateForm_EmptyName(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{"name": "  ", "slug": "blank-name"})
	resp, err := p.handleCreateForm(ctx(), makeReq("POST", "/", body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 422 {
		t.Errorf("expected 422 for whitespace-only name, got %d", resp.StatusCode)
	}
}

func TestHandleCreateForm_MissingSlug(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{"name": "No Slug"})
	resp, err := p.handleCreateForm(ctx(), makeReq("POST", "/", body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 422 {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestHandleCreateForm_InvalidSlugFormat(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{"name": "Bad", "slug": "Bad Slug!"})
	resp, err := p.handleCreateForm(ctx(), makeReq("POST", "/", body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 422 {
		t.Errorf("expected 422 for invalid slug, got %d", resp.StatusCode)
	}
}

func TestHandleCreateForm_DuplicateSlug(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	body, _ := json.Marshal(map[string]any{"name": "Other", "slug": "contact-us"})
	resp, err := p.handleCreateForm(ctx(), makeReq("POST", "/", body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 422 {
		t.Errorf("expected 422 for duplicate slug, got %d", resp.StatusCode)
	}
}

func TestHandleCreateForm_InvalidJSON(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleCreateForm(ctx(), makeReq("POST", "/", []byte("not-json")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// ---- handleGetForm ----

func TestHandleGetForm_Found(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.handleGetForm(ctx(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var form map[string]any
	decodeBody(t, resp.Body, &form)
	if form["name"] != "Contact Us" {
		t.Errorf("name: got %v", form["name"])
	}
}

func TestHandleGetForm_NotFound(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleGetForm(ctx(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---- handleUpdateForm ----

func TestHandleUpdateForm(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	body, _ := json.Marshal(map[string]any{"name": "Updated", "slug": "contact-us"})
	resp, err := p.handleUpdateForm(ctx(), 1, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleUpdateForm_BlankNameRejected(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	body, _ := json.Marshal(map[string]any{"name": "", "slug": "contact-us"})
	resp, err := p.handleUpdateForm(ctx(), 1, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 422 {
		t.Errorf("expected 422 for blank name on update, got %d", resp.StatusCode)
	}
}

func TestHandleUpdateForm_InvalidJSON(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleUpdateForm(ctx(), 1, []byte("not-json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// ---- handleDeleteForm ----

func TestHandleDeleteForm(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.handleDeleteForm(ctx(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if len(h.Tables[formsTable]) != 0 {
		t.Error("form should be deleted")
	}
}

// ---- handleFormDuplicate ----

func TestHandleFormDuplicate_Success(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.handleFormDuplicate(ctx(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
	var form map[string]any
	decodeBody(t, resp.Body, &form)
	if !strings.Contains(form["name"].(string), "Copy") {
		t.Errorf("duplicated name should contain 'Copy', got %q", form["name"])
	}
	slug, _ := form["slug"].(string)
	if !strings.Contains(slug, "copy") {
		t.Errorf("duplicated slug should contain 'copy', got %q", slug)
	}
}

func TestHandleFormDuplicate_SlugCollision(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)
	// Create the slug that duplicate would generate
	h.DataCreate(ctx(), formsTable, map[string]any{"name": "X", "slug": "contact-us-copy"})

	resp, err := p.handleFormDuplicate(ctx(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
	var form map[string]any
	decodeBody(t, resp.Body, &form)
	slug, _ := form["slug"].(string)
	if slug == "contact-us-copy" {
		t.Error("slug should be suffixed on collision")
	}
}

func TestHandleFormDuplicate_NotFound(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleFormDuplicate(ctx(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---- handleFormExport ----

func TestHandleFormExport_Success(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.handleFormExport(ctx(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Headers["Content-Type"], "application/json") {
		t.Error("export should return JSON content type")
	}
	if !strings.Contains(resp.Headers["Content-Disposition"], "attachment") {
		t.Error("export should have attachment content disposition")
	}
	var exported map[string]any
	decodeBody(t, resp.Body, &exported)
	if _, ok := exported["id"]; ok {
		t.Error("export should not include id field")
	}
}

func TestHandleFormExport_NotFound(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleFormExport(ctx(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---- handleFormImport ----

func TestHandleFormImport_Success(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	importData := map[string]any{"name": "Imported Form", "slug": "imported"}
	body, _ := json.Marshal(importData)

	resp, err := p.handleFormImport(ctx(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
}

func TestHandleFormImport_SlugCollision(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h) // slug: "contact-us"

	importData := map[string]any{"name": "Contact Us Import", "slug": "contact-us"}
	body, _ := json.Marshal(importData)

	resp, err := p.handleFormImport(ctx(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
	var form map[string]any
	decodeBody(t, resp.Body, &form)
	if form["slug"] == "contact-us" {
		t.Error("imported slug should be suffixed on collision")
	}
}

func TestHandleFormImport_MissingName(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	body, _ := json.Marshal(map[string]any{"slug": "no-name"})
	resp, err := p.handleFormImport(ctx(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 422 {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

func TestHandleFormImport_InvalidJSON(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleFormImport(ctx(), []byte("not-json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// ---- enrichFormWithStats ----

func TestEnrichFormWithStats(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	form := map[string]any{"id": float64(1), "name": "Test"}
	h.DataCreate(ctx(), submissionsTable, map[string]any{"form_id": float64(1), "status": "unread"})
	h.DataCreate(ctx(), submissionsTable, map[string]any{"form_id": float64(1), "status": "read"})

	result := enrichFormWithStats(ctx(), p, form)
	if result["submission_count"] != int64(2) {
		t.Errorf("submission_count: got %v, want 2", result["submission_count"])
	}
}
