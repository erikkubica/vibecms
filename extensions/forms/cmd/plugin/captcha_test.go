package main

import (
	"fmt"
	"strings"
	"testing"

	"vibecms/internal/coreapi"
)

func TestVerifyCAPTCHA_NoProvider(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	// No provider → always succeeds
	if err := p.verifyCAPTCHA(ctx(), "none", "secret", "token", "1.2.3.4"); err != nil {
		t.Errorf("provider=none should succeed, got %v", err)
	}
	if err := p.verifyCAPTCHA(ctx(), "", "secret", "token", ""); err != nil {
		t.Errorf("provider=empty should succeed, got %v", err)
	}
}

func TestVerifyCAPTCHA_MissingSecret(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	err := p.verifyCAPTCHA(ctx(), "recaptcha", "", "token", "")
	if err == nil || !strings.Contains(err.Error(), "secret not configured") {
		t.Errorf("expected secret error, got %v", err)
	}
}

func TestVerifyCAPTCHA_MissingToken(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	err := p.verifyCAPTCHA(ctx(), "recaptcha", "my-secret", "", "")
	if err == nil || !strings.Contains(err.Error(), "token missing") {
		t.Errorf("expected token missing error, got %v", err)
	}
}

func TestVerifyCAPTCHA_UnknownProvider(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	err := p.verifyCAPTCHA(ctx(), "magic-captcha", "secret", "token", "")
	if err == nil || !strings.Contains(err.Error(), "unknown captcha provider") {
		t.Errorf("expected unknown provider error, got %v", err)
	}
}

func TestVerifyCAPTCHA_RecaptchaSuccess(t *testing.T) {
	h := NewFakeHost()
	h.FetchStub = func(req coreapi.FetchRequest) (*coreapi.FetchResponse, error) {
		if !strings.Contains(req.URL, "recaptcha") {
			t.Errorf("unexpected URL: %s", req.URL)
		}
		return &coreapi.FetchResponse{StatusCode: 200, Body: `{"success":true}`}, nil
	}
	p := newPlugin(h)

	if err := p.verifyCAPTCHA(ctx(), "recaptcha", "secret", "valid-token", "1.2.3.4"); err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestVerifyCAPTCHA_HcaptchaSuccess(t *testing.T) {
	h := NewFakeHost()
	h.FetchStub = func(req coreapi.FetchRequest) (*coreapi.FetchResponse, error) {
		if !strings.Contains(req.URL, "hcaptcha") {
			t.Errorf("unexpected URL: %s", req.URL)
		}
		return &coreapi.FetchResponse{StatusCode: 200, Body: `{"success":true}`}, nil
	}
	p := newPlugin(h)

	if err := p.verifyCAPTCHA(ctx(), "hcaptcha", "secret", "valid-token", ""); err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestVerifyCAPTCHA_TurnstileSuccess(t *testing.T) {
	h := NewFakeHost()
	h.FetchStub = func(req coreapi.FetchRequest) (*coreapi.FetchResponse, error) {
		if !strings.Contains(req.URL, "turnstile") {
			t.Errorf("unexpected URL: %s", req.URL)
		}
		return &coreapi.FetchResponse{StatusCode: 200, Body: `{"success":true}`}, nil
	}
	p := newPlugin(h)

	if err := p.verifyCAPTCHA(ctx(), "turnstile", "secret", "valid-token", ""); err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestVerifyCAPTCHA_VerificationFailed(t *testing.T) {
	h := NewFakeHost()
	h.FetchStub = func(_ coreapi.FetchRequest) (*coreapi.FetchResponse, error) {
		return &coreapi.FetchResponse{StatusCode: 200, Body: `{"success":false}`}, nil
	}
	p := newPlugin(h)

	err := p.verifyCAPTCHA(ctx(), "recaptcha", "secret", "bad-token", "")
	if err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Errorf("expected verification failed, got %v", err)
	}
}

func TestVerifyCAPTCHA_HTTP500(t *testing.T) {
	h := NewFakeHost()
	h.FetchStub = func(_ coreapi.FetchRequest) (*coreapi.FetchResponse, error) {
		return &coreapi.FetchResponse{StatusCode: 500, Body: ``}, nil
	}
	p := newPlugin(h)

	err := p.verifyCAPTCHA(ctx(), "recaptcha", "secret", "token", "")
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected HTTP 500 error, got %v", err)
	}
}

func TestVerifyCAPTCHA_FetchError(t *testing.T) {
	h := NewFakeHost()
	h.FetchStub = func(_ coreapi.FetchRequest) (*coreapi.FetchResponse, error) {
		return nil, fmt.Errorf("connection refused")
	}
	p := newPlugin(h)

	err := p.verifyCAPTCHA(ctx(), "recaptcha", "secret", "token", "")
	if err == nil || !strings.Contains(err.Error(), "captcha verify request") {
		t.Errorf("expected fetch error, got %v", err)
	}
}

func TestVerifyCAPTCHA_InvalidJSON(t *testing.T) {
	h := NewFakeHost()
	h.FetchStub = func(_ coreapi.FetchRequest) (*coreapi.FetchResponse, error) {
		return &coreapi.FetchResponse{StatusCode: 200, Body: `not-json`}, nil
	}
	p := newPlugin(h)

	err := p.verifyCAPTCHA(ctx(), "recaptcha", "secret", "token", "")
	if err == nil || !strings.Contains(err.Error(), "captcha verify parse") {
		t.Errorf("expected parse error, got %v", err)
	}
}

// ---- captchaScriptTag ----

func TestCaptchaScriptTag(t *testing.T) {
	cases := []struct {
		provider string
		siteKey  string
		wantTag  string
	}{
		{"recaptcha", "key123", "recaptcha/api.js"},
		{"hcaptcha", "key123", "hcaptcha.com"},
		{"turnstile", "key123", "turnstile"},
		{"none", "key123", ""},
		{"", "key123", ""},
		{"recaptcha", "", ""},
	}
	for _, tc := range cases {
		tag := captchaScriptTag(tc.provider, tc.siteKey)
		if tc.wantTag == "" && tag != "" {
			t.Errorf("provider=%q siteKey=%q: expected empty tag, got %q", tc.provider, tc.siteKey, tag)
		}
		if tc.wantTag != "" && !strings.Contains(tag, tc.wantTag) {
			t.Errorf("provider=%q siteKey=%q: expected tag containing %q, got %q", tc.provider, tc.siteKey, tc.wantTag, tag)
		}
	}
}

// ---- extractCaptchaToken ----

func TestExtractCaptchaToken(t *testing.T) {
	data := map[string]any{"name": "Erik", "_captcha_token": "tok123"}
	tok := extractCaptchaToken(data)
	if tok != "tok123" {
		t.Errorf("expected tok123, got %q", tok)
	}
	if _, ok := data["_captcha_token"]; ok {
		t.Error("_captcha_token should be removed from data")
	}
}
