package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestUIPages(t *testing.T) {
	pages := []struct {
		name       string
		url        string
		wantStatus int
		wantText   string
	}{
		{"Homepage", baseURL + "/", 200, "VibeCMS"},
		{"Login", baseURL + "/login", 200, "Sign In"},
		{"Register", baseURL + "/register", 200, "Create Account"},
		{"Forgot Password", baseURL + "/auth/forgot-password", 200, "Reset"},
		{"Health API", baseURL + "/api/v1/health", 200, `"status":"up"`},
	}

	for _, p := range pages {
		t.Run(p.name, func(t *testing.T) {
			resp, err := http.Get(p.url)
			if err != nil {
				t.Fatalf("GET %s failed: %v", p.url, err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != p.wantStatus {
				t.Fatalf("GET %s: expected %d, got %d\nBody: %s", p.url, p.wantStatus, resp.StatusCode, body[:min(len(body), 500)])
			}
			if !strings.Contains(string(body), p.wantText) {
				t.Fatalf("GET %s: expected %q in body\nBody: %s", p.url, p.wantText, body[:min(len(body), 500)])
			}
			t.Logf("GET %s -> %d OK (contains %q)", p.url, resp.StatusCode, p.wantText)
		})
	}
}

func TestUIAdminRequiresAuth(t *testing.T) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(baseURL + "/admin/dashboard")
	if err != nil {
		t.Fatalf("GET /admin/dashboard failed: %v", err)
	}
	defer resp.Body.Close()
	// Should get 401 or redirect to login
	if resp.StatusCode != 401 && resp.StatusCode != 302 && resp.StatusCode != 303 {
		t.Fatalf("expected 401 or redirect, got %d", resp.StatusCode)
	}
	t.Logf("GET /admin/dashboard without auth -> %d (correctly blocked)", resp.StatusCode)
}

func TestUIAdminDashboard(t *testing.T) {
	cookie := loginAndGetCookie(t)

	req, _ := http.NewRequest("GET", baseURL+"/admin/dashboard", nil)
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /admin/dashboard failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d\nBody: %s", resp.StatusCode, body[:min(len(body), 500)])
	}
	if !strings.Contains(string(body), "Dashboard") {
		t.Fatalf("expected 'Dashboard' in body\nBody: %s", body[:min(len(body), 500)])
	}
	t.Logf("GET /admin/dashboard -> 200 OK (contains 'Dashboard')")
}

func TestUILoginFlow(t *testing.T) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// POST login form
	resp, err := client.Post(baseURL+"/auth/login-page",
		"application/x-www-form-urlencoded",
		strings.NewReader("email=admin@vibecms.local&password=admin123"))
	if err != nil {
		t.Fatalf("POST /auth/login-page failed: %v", err)
	}
	defer resp.Body.Close()
	// Should redirect to /admin
	if resp.StatusCode != 302 && resp.StatusCode != 303 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected redirect, got %d\nBody: %s", resp.StatusCode, body[:min(len(body), 500)])
	}
	loc := resp.Header.Get("Location")
	t.Logf("POST /auth/login-page -> %d redirect to %s", resp.StatusCode, loc)

	// Check session cookie was set
	var hasCookie bool
	for _, c := range resp.Cookies() {
		if c.Name == "vibecms_session" {
			hasCookie = true
			break
		}
	}
	if !hasCookie {
		t.Fatal("no session cookie set after login")
	}
	t.Log("Session cookie set OK")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
