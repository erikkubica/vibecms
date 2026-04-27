package main

import (
	"encoding/csv"
	"bytes"
	"strings"
	"testing"
)

func seedFormAndSubmissions(h *FakeHost) {
	h.DataCreate(ctx(), formsTable, map[string]any{
		"name": "Export Form",
		"slug": "export-form",
		"fields": `[
			{"id":"name","label":"Full Name","type":"text"},
			{"id":"email","label":"Email","type":"email"}
		]`,
		"settings": `{}`,
	})
	h.DataCreate(ctx(), submissionsTable, map[string]any{
		"form_id":    float64(1),
		"data":       map[string]any{"name": "Alice", "email": "alice@example.com"},
		"metadata":   map[string]any{"ip": "1.2.3.4"},
		"status":     "read",
		"created_at": "2025-01-01T00:00:00Z",
	})
}

func TestHandleCSVExport_Success(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedFormAndSubmissions(h)

	resp, err := p.handleCSVExport(ctx(), makeReqWithQuery("GET", "submissions/export", map[string]string{
		"form_id": "1",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
	if !strings.Contains(resp.Headers["Content-Type"], "text/csv") {
		t.Errorf("expected text/csv content type, got %q", resp.Headers["Content-Type"])
	}
	if !strings.Contains(resp.Headers["Content-Disposition"], "attachment") {
		t.Error("should be attachment download")
	}

	// Parse CSV
	r := csv.NewReader(bytes.NewReader(resp.Body))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if len(records) < 2 {
		t.Fatalf("expected header + 1 data row, got %d rows", len(records))
	}
	header := records[0]
	if len(header) < 3 {
		t.Errorf("expected at least 3 headers, got %d: %v", len(header), header)
	}
}

func TestHandleCSVExport_MissingFormID(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleCSVExport(ctx(), makeReqWithQuery("GET", "submissions/export", map[string]string{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHandleCSVExport_InvalidFormID(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleCSVExport(ctx(), makeReqWithQuery("GET", "submissions/export", map[string]string{
		"form_id": "not-a-number",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHandleCSVExport_FormNotFound(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleCSVExport(ctx(), makeReqWithQuery("GET", "submissions/export", map[string]string{
		"form_id": "999",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestHandleCSVExport_StoreIPFalse(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	h.DataCreate(ctx(), formsTable, map[string]any{
		"name":     "Private Form",
		"slug":     "private-form",
		"fields":   `[{"id":"name","label":"Name","type":"text"}]`,
		"settings": `{"store_ip":false}`,
	})
	h.DataCreate(ctx(), submissionsTable, map[string]any{
		"form_id":  float64(1),
		"data":     map[string]any{"name": "Bob"},
		"metadata": map[string]any{"ip": "1.2.3.4"},
		"status":   "unread",
	})

	resp, err := p.handleCSVExport(ctx(), makeReqWithQuery("GET", "submissions/export", map[string]string{
		"form_id": "1",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	// IP Address column should NOT be in the CSV
	r := csv.NewReader(bytes.NewReader(resp.Body))
	records, _ := r.ReadAll()
	if len(records) > 0 {
		for _, h := range records[0] {
			if h == "IP Address" {
				t.Error("IP Address column should not be present when store_ip=false")
			}
		}
	}
}
