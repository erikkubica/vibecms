package main

import (
	"encoding/json"
	"testing"
)

func TestJSONResponse(t *testing.T) {
	resp := jsonResponse(200, map[string]string{"key": "val"})
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected JSON content type, got %q", resp.Headers["Content-Type"])
	}
}

// TestJSONError pins the public error envelope: {"error": "<CODE>", "message": "..."}.
// The vibe-form client script and the admin UI both read `error` for the
// machine-readable code and `message` for the human-readable text.
func TestJSONError(t *testing.T) {
	resp := jsonError(404, "NOT_FOUND", "not found")
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got := body["error"]; got != "NOT_FOUND" {
		t.Errorf("body.error = %q, want %q", got, "NOT_FOUND")
	}
	if got := body["message"]; got != "not found" {
		t.Errorf("body.message = %q, want %q", got, "not found")
	}
	if _, hasCode := body["code"]; hasCode {
		t.Errorf("body.code should not be present (legacy shape removed)")
	}
}

func TestCaptchaClientScript(t *testing.T) {
	script := captchaClientScript()
	if len(script) == 0 {
		t.Error("captchaClientScript should return non-empty string")
	}
}
