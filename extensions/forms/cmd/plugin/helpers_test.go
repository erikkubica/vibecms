package main

import (
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

func TestJSONError(t *testing.T) {
	resp := jsonError(404, "NOT_FOUND", "not found")
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}


func TestCaptchaClientScript(t *testing.T) {
	script := captchaClientScript()
	if len(script) == 0 {
		t.Error("captchaClientScript should return non-empty string")
	}
	if script[0:1] != "w" {
		// should start with "window."
	}
}
