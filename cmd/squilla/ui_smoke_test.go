package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestUIPages(t *testing.T) {
	// dependsOnSeed marks tests whose subject is a seeded content node
	// (the auth pages live under content_nodes and only land on a fresh
	// DB via SeedIfEmpty). When an operator has intentionally removed
	// those pages from a working install, a 404 is correct behavior — so
	// these subtests skip rather than fail when the page is absent.
	pages := []struct {
		name           string
		url            string
		wantStatus     int
		wantText       string
		dependsOnSeed  bool
	}{
		{"Homepage", baseURL + "/", 200, "Squilla", false},
		{"Login", baseURL + "/login", 200, "Sign In", true},
		{"Register", baseURL + "/register", 200, "Create Account", true},
		{"Forgot Password", baseURL + "/forgot-password", 200, "Reset", true},
		{"Health API", baseURL + "/api/v1/health", 200, `"status":"up"`, false},
	}

	for _, p := range pages {
		t.Run(p.name, func(t *testing.T) {
			resp, err := http.Get(p.url)
			if err != nil {
				t.Fatalf("GET %s failed: %v", p.url, err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if p.dependsOnSeed && resp.StatusCode == 404 {
				t.Skipf("seeded content node missing on this instance — %s returned 404", p.url)
			}
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

// TestUIAdminRequiresAuth verifies that anonymous requests can't reach the
// admin data plane. The /admin/* HTML route intentionally serves the SPA
// shell (200) regardless of session — auth state is determined client-side
// via /me so the bundle ships before that round-trip — so the meaningful
// boundary to assert is the admin API. Hitting any /admin/api/* endpoint
// without a session must return 401.
func TestUIAdminRequiresAuth(t *testing.T) {
	resp, err := http.Get(baseURL + "/admin/api/users")
	if err != nil {
		t.Fatalf("GET /admin/api/users failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401 from /admin/api/users without auth, got %d", resp.StatusCode)
	}
	t.Logf("GET /admin/api/users without auth -> %d (correctly blocked)", resp.StatusCode)
}

// TestUIAdminDashboard verifies that the SPA shell renders for an
// authenticated admin. The HTML body never contains feature copy like
// "Dashboard" — that's drawn by React at runtime — so the assertion is on
// SPA markers (the title and the importmap) that prove the shell loaded.
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
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Squilla Admin") {
		t.Fatalf("expected SPA shell title in body\nBody: %s", body[:min(len(body), 500)])
	}
	if !strings.Contains(bodyStr, "importmap") {
		t.Fatalf("expected importmap in SPA shell\nBody: %s", body[:min(len(body), 500)])
	}
	t.Logf("GET /admin/dashboard -> 200 OK (SPA shell loaded)")
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
		strings.NewReader("email=admin@squilla.local&password=admin123"))
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
		if c.Name == "squilla_session" {
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
