package main

import (
	"encoding/json"
	"strings"
	"testing"

	pb "vibecms/pkg/plugin/proto"
)

func TestHandlePreview_Success(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	input := map[string]any{
		"id":     float64(1),
		"name":   "Preview Form",
		"fields": []any{map[string]any{"id": "name", "label": "Name", "type": "text"}},
		"layout": `<form>{{range .fields_list}}<input name="{{.id}}">{{end}}</form>`,
	}
	body, _ := json.Marshal(input)
	req := &pb.PluginHTTPRequest{Method: "POST", Path: "preview", Body: body}

	resp, err := p.handlePreview(ctx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
	var result map[string]any
	decodeBody(t, resp.Body, &result)
	html, _ := result["html"].(string)
	if !strings.Contains(html, `name="name"`) {
		t.Errorf("preview HTML should contain name input, got: %s", html)
	}
}

func TestHandlePreview_InvalidJSON(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	req := &pb.PluginHTTPRequest{Method: "POST", Path: "preview", Body: []byte("not-json")}
	resp, err := p.handlePreview(ctx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHandlePreview_EmptyLayout(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	input := map[string]any{"id": float64(1), "name": "Test", "fields": []any{}}
	body, _ := json.Marshal(input)
	req := &pb.PluginHTTPRequest{Method: "POST", Path: "preview", Body: body}

	resp, err := p.handlePreview(ctx(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500 for empty layout, got %d", resp.StatusCode)
	}
}

// ---- handleRender ----

func TestHandleRender_BySlug(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.handleRender(ctx(), "contact-us")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
	var result map[string]any
	decodeBody(t, resp.Body, &result)
	if result["html"] == nil {
		t.Error("response should contain html")
	}
}

func TestHandleRender_ByID(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedForm(h)

	resp, err := p.handleRender(ctx(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
}

func TestHandleRender_NotFound(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp, err := p.handleRender(ctx(), "no-such-form")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
