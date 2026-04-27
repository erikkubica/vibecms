package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"vibecms/internal/coreapi"
	pb "vibecms/pkg/plugin/proto"
)

func seedFormFull(h *FakeHost) map[string]any {
	row, _ := h.DataCreate(ctx(), formsTable, map[string]any{
		"name": "Contact Us",
		"slug": "contact-us",
		"fields": `[
			{"id":"name","label":"Name","type":"text","required":true},
			{"id":"email","label":"Email","type":"email"}
		]`,
		"layout":        `<form>{{range .fields_list}}<input name="{{.id}}">{{end}}</form>`,
		"settings":      `{}`,
		"notifications": `[]`,
	})
	return row
}

func submitJSON(h *FakeHost, p *FormsPlugin, slug string, data map[string]any) *pb.PluginHTTPResponse {
	body, _ := json.Marshal(data)
	req := &pb.PluginHTTPRequest{
		Method:  "POST",
		Path:    "forms/submit/" + slug,
		Headers: map[string]string{"Content-Type": "application/json", "X-Forwarded-For": "1.2.3.4"},
		Body:    body,
	}
	resp, _ := p.handleSubmit(ctx(), slug, req)
	return resp
}

// ---- Happy path ----

func TestHandleSubmit_HappyPath(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedFormFull(h)

	resp := submitJSON(h, p, "contact-us", map[string]any{"name": "Alice", "email": "alice@example.com"})
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d (body: %s)", resp.StatusCode, resp.Body)
	}

	// Submission should be stored
	time.Sleep(10 * time.Millisecond) // let goroutines run
	if len(h.Tables[submissionsTable]) != 1 {
		t.Errorf("expected 1 submission, got %d", len(h.Tables[submissionsTable]))
	}

	// Event should be emitted
	emitted := false
	for _, e := range h.Emitted {
		if e.Action == "forms:submitted" {
			emitted = true
			break
		}
	}
	if !emitted {
		t.Error("forms:submitted event should be emitted")
	}
}

// ---- Form not found ----

func TestHandleSubmit_FormNotFound(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	resp := submitJSON(h, p, "no-such-form", map[string]any{"name": "Alice"})
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---- Honeypot ----

func TestHandleSubmit_HoneypotTriggered(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedFormFull(h)

	// website_url is filled → bot detected
	resp := submitJSON(h, p, "contact-us", map[string]any{
		"name":        "Bot",
		"website_url": "http://spam.com",
	})
	// Returns 200 silently
	if resp.StatusCode != 200 {
		t.Errorf("honeypot triggered should return 200, got %d", resp.StatusCode)
	}
	// No submission stored
	if len(h.Tables[submissionsTable]) != 0 {
		t.Error("honeypot submission should not be stored")
	}
}

// ---- Rate limiting ----

func TestHandleSubmit_RateLimited(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	// Create form with rate_limit=1
	h.DataCreate(ctx(), formsTable, map[string]any{
		"name":     "Limited",
		"slug":     "limited",
		"fields":   `[]`,
		"settings": `{"rate_limit":1}`,
	})

	ip := "10.0.0.1"
	body, _ := json.Marshal(map[string]any{})
	req1 := &pb.PluginHTTPRequest{
		Method:  "POST",
		Path:    "forms/submit/limited",
		Headers: map[string]string{"Content-Type": "application/json", "X-Forwarded-For": ip},
		Body:    body,
	}
	req2 := &pb.PluginHTTPRequest{
		Method:  "POST",
		Path:    "forms/submit/limited",
		Headers: map[string]string{"Content-Type": "application/json", "X-Forwarded-For": ip},
		Body:    body,
	}

	resp1, _ := p.handleSubmit(ctx(), "limited", req1)
	resp2, _ := p.handleSubmit(ctx(), "limited", req2)

	if resp1.StatusCode != 200 {
		t.Errorf("first request should succeed, got %d", resp1.StatusCode)
	}
	if resp2.StatusCode != 429 {
		t.Errorf("second request should be rate limited (429), got %d", resp2.StatusCode)
	}
}

// ---- Validation failure ----

func TestHandleSubmit_ValidationFailed(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedFormFull(h)

	// Missing required "name" field
	resp := submitJSON(h, p, "contact-us", map[string]any{"email": "test@example.com"})
	if resp.StatusCode != 422 {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
	var body map[string]any
	decodeBody(t, resp.Body, &body)
	if body["error"] != "VALIDATION_FAILED" {
		t.Errorf("expected VALIDATION_FAILED error, got %v", body["error"])
	}
	fields, ok := body["fields"].(map[string]any)
	if !ok || fields["name"] == "" {
		t.Errorf("expected name validation error, got %v", body["fields"])
	}
}

// ---- Invalid body ----

func TestHandleSubmit_InvalidBody(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedFormFull(h)

	req := &pb.PluginHTTPRequest{
		Method:  "POST",
		Path:    "forms/submit/contact-us",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte("not-json"),
	}
	resp, _ := p.handleSubmit(ctx(), "contact-us", req)
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// ---- CAPTCHA failure ----

func TestHandleSubmit_CaptchaFailed(t *testing.T) {
	h := NewFakeHost()
	h.FetchStub = func(_ coreapi.FetchRequest) (*coreapi.FetchResponse, error) {
		return &coreapi.FetchResponse{StatusCode: 200, Body: `{"success":false}`}, nil
	}
	p := newPlugin(h)

	h.DataCreate(ctx(), formsTable, map[string]any{
		"name":     "Protected",
		"slug":     "protected",
		"fields":   `[]`,
		"settings": `{"captcha_provider":"recaptcha","captcha_secret_key":"secret"}`,
	})

	body, _ := json.Marshal(map[string]any{"_captcha_token": "bad-token"})
	req := &pb.PluginHTTPRequest{
		Method:  "POST",
		Path:    "forms/submit/protected",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    body,
	}
	resp, _ := p.handleSubmit(ctx(), "protected", req)
	if resp.StatusCode != 422 {
		t.Errorf("expected 422 for captcha failure, got %d", resp.StatusCode)
	}
}

// ---- Multipart submission ----

func TestHandleSubmit_MultipartTextFields(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)
	seedFormFull(h)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("name", "Bob")
	mw.WriteField("email", "bob@example.com")
	mw.Close()

	req := &pb.PluginHTTPRequest{
		Method:  "POST",
		Path:    "forms/submit/contact-us",
		Headers: map[string]string{"Content-Type": mw.FormDataContentType()},
		Body:    buf.Bytes(),
	}
	resp, _ := p.handleSubmit(ctx(), "contact-us", req)
	if resp.StatusCode != 200 {
		t.Errorf("multipart submission: expected 200, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
}

// ---- parseSubmissionBody ----

func TestParseSubmissionBody_JSON(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"name": "Alice"})
	req := &pb.PluginHTTPRequest{
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    body,
	}
	data, files, err := parseSubmissionBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["name"] != "Alice" {
		t.Errorf("name: got %v", data["name"])
	}
	if len(files) != 0 {
		t.Errorf("expected no files, got %d", len(files))
	}
}

func TestParseSubmissionBody_Multipart(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("subject", "Hello")

	fw, _ := mw.CreateFormFile("attachment", "doc.pdf")
	fw.Write([]byte("pdf data"))
	mw.Close()

	req := &pb.PluginHTTPRequest{
		Headers: map[string]string{"Content-Type": mw.FormDataContentType()},
		Body:    buf.Bytes(),
	}
	data, files, err := parseSubmissionBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["subject"] != "Hello" {
		t.Errorf("subject: got %v", data["subject"])
	}
	if len(files) != 1 || files[0].FileName != "doc.pdf" {
		t.Errorf("expected 1 file doc.pdf, got %v", files)
	}
}

func TestParseSubmissionBody_InvalidJSON(t *testing.T) {
	req := &pb.PluginHTTPRequest{
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte("not-json"),
	}
	_, _, err := parseSubmissionBody(req)
	if err == nil {
		t.Error("invalid JSON should return error")
	}
}

func TestParseSubmissionBody_MultipartMissingBoundary(t *testing.T) {
	req := &pb.PluginHTTPRequest{
		Headers: map[string]string{"Content-Type": "multipart/form-data"},
		Body:    []byte("some data"),
	}
	_, _, err := parseSubmissionBody(req)
	if err == nil || !strings.Contains(err.Error(), "boundary") {
		t.Errorf("expected boundary error, got %v", err)
	}
}

// ---- File upload + store in submission ----

func TestHandleSubmit_FileUploadStored(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	h.DataCreate(ctx(), formsTable, map[string]any{
		"name":  "File Form",
		"slug":  "file-form",
		"fields": `[{"id":"doc","label":"Document","type":"file"}]`,
		"settings": `{}`,
	})

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("doc", "test.pdf")
	fmt.Fprint(fw, "pdf content")
	mw.Close()

	req := &pb.PluginHTTPRequest{
		Method:  "POST",
		Path:    "forms/submit/file-form",
		Headers: map[string]string{"Content-Type": mw.FormDataContentType(), "X-Forwarded-For": "1.2.3.4"},
		Body:    buf.Bytes(),
	}
	resp, _ := p.handleSubmit(ctx(), "file-form", req)
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d (body: %s)", resp.StatusCode, resp.Body)
	}
	// File should be stored
	if len(h.StoredFiles) == 0 {
		t.Error("file should have been stored")
	}
}

// ---- IP not stored when store_ip=false ----

func TestHandleSubmit_StoreIPFalse(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	h.DataCreate(ctx(), formsTable, map[string]any{
		"name":     "Private",
		"slug":     "private",
		"fields":   `[]`,
		"settings": `{"store_ip":false}`,
	})

	req := &pb.PluginHTTPRequest{
		Method:  "POST",
		Path:    "forms/submit/private",
		Headers: map[string]string{"Content-Type": "application/json", "X-Forwarded-For": "1.2.3.4"},
		Body:    []byte(`{}`),
	}
	resp, _ := p.handleSubmit(ctx(), "private", req)
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Check no IP in submission metadata
	for _, row := range h.Tables[submissionsTable] {
		if meta, ok := row["metadata"].(map[string]any); ok {
			if meta["ip"] != nil {
				t.Errorf("IP should not be stored when store_ip=false, got %v", meta["ip"])
			}
		}
	}
}
