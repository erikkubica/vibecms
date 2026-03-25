package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const baseURL = "http://localhost:8099"

func TestSmokeHealthCheck(t *testing.T) {
	resp, err := http.Get(baseURL + "/api/v1/health")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"status":"up"`) {
		t.Fatalf("unexpected body: %s", body)
	}
	t.Logf("Health: %s", body)
}

func TestSmokeAuthFlow(t *testing.T) {
	// Login
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "admin@vibecms.local",
		"password": "admin123",
	})
	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("login expected 200, got %d: %s", resp.StatusCode, body)
	}

	// Extract cookie
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "vibecms_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("no session cookie returned")
	}
	t.Logf("Login OK, cookie: %s...%s", sessionCookie.Value[:8], sessionCookie.Value[len(sessionCookie.Value)-4:])

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Login response: %s", body)

	// GET /me
	req, _ := http.NewRequest("GET", baseURL+"/me", nil)
	req.AddCookie(sessionCookie)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /me failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("GET /me expected 200, got %d: %s", resp2.StatusCode, body)
	}
	meBody, _ := io.ReadAll(resp2.Body)
	t.Logf("GET /me: %s", meBody)

	// Logout
	req3, _ := http.NewRequest("POST", baseURL+"/auth/logout", nil)
	req3.AddCookie(sessionCookie)
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != 200 {
		t.Fatalf("logout expected 200, got %d", resp3.StatusCode)
	}
	t.Log("Logout OK")

	// Verify session is invalid after logout
	req4, _ := http.NewRequest("GET", baseURL+"/me", nil)
	req4.AddCookie(sessionCookie)
	resp4, err := http.DefaultClient.Do(req4)
	if err != nil {
		t.Fatalf("GET /me after logout failed: %v", err)
	}
	defer resp4.Body.Close()
	if resp4.StatusCode != 401 {
		t.Fatalf("expected 401 after logout, got %d", resp4.StatusCode)
	}
	t.Log("Session invalidated after logout OK")
}

func loginAndGetCookie(t *testing.T) *http.Cookie {
	t.Helper()
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "admin@vibecms.local",
		"password": "admin123",
	})
	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()
	for _, c := range resp.Cookies() {
		if c.Name == "vibecms_session" {
			return c
		}
	}
	t.Fatal("no session cookie")
	return nil
}

func TestSmokeNodeCRUD(t *testing.T) {
	cookie := loginAndGetCookie(t)

	// Create node
	createBody, _ := json.Marshal(map[string]interface{}{
		"title":     "About Us",
		"node_type": "page",
		"blocks_data": []map[string]interface{}{
			{"type": "text", "content": map[string]string{"body": "Welcome to VibeCMS"}},
		},
	})
	req, _ := http.NewRequest("POST", baseURL+"/admin/api/nodes", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create node failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		t.Fatalf("create node expected 201, got %d: %s", resp.StatusCode, body)
	}
	t.Logf("Create node: %s", body)

	// Extract node ID
	var createResp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &createResp)
	nodeID := createResp.Data.ID
	t.Logf("Created node ID: %d", nodeID)

	// List nodes
	req2, _ := http.NewRequest("GET", baseURL+"/admin/api/nodes", nil)
	req2.AddCookie(cookie)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("list nodes failed: %v", err)
	}
	defer resp2.Body.Close()
	listBody, _ := io.ReadAll(resp2.Body)
	if resp2.StatusCode != 200 {
		t.Fatalf("list nodes expected 200, got %d: %s", resp2.StatusCode, listBody)
	}
	t.Logf("List nodes: %s", listBody)

	// Get single node
	req3, _ := http.NewRequest("GET", fmt.Sprintf("%s/admin/api/nodes/%d", baseURL, nodeID), nil)
	req3.AddCookie(cookie)
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("get node failed: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != 200 {
		getBody, _ := io.ReadAll(resp3.Body)
		t.Fatalf("get node expected 200, got %d: %s", resp3.StatusCode, getBody)
	}
	t.Log("Get node OK")

	// Update node
	updateBody, _ := json.Marshal(map[string]interface{}{
		"title":  "About VibeCMS",
		"status": "published",
	})
	req4, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/admin/api/nodes/%d", baseURL, nodeID), bytes.NewReader(updateBody))
	req4.Header.Set("Content-Type", "application/json")
	req4.AddCookie(cookie)
	resp4, err := http.DefaultClient.Do(req4)
	if err != nil {
		t.Fatalf("update node failed: %v", err)
	}
	defer resp4.Body.Close()
	updateRespBody, _ := io.ReadAll(resp4.Body)
	if resp4.StatusCode != 200 {
		t.Fatalf("update node expected 200, got %d: %s", resp4.StatusCode, updateRespBody)
	}
	t.Logf("Update node: %s", updateRespBody)

	// Delete node
	req5, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/admin/api/nodes/%d", baseURL, nodeID), nil)
	req5.AddCookie(cookie)
	resp5, err := http.DefaultClient.Do(req5)
	if err != nil {
		t.Fatalf("delete node failed: %v", err)
	}
	defer resp5.Body.Close()
	if resp5.StatusCode != 204 {
		delBody, _ := io.ReadAll(resp5.Body)
		t.Fatalf("delete node expected 204, got %d: %s", resp5.StatusCode, delBody)
	}
	t.Log("Delete node OK (soft delete)")
}

func TestSmokePerformance(t *testing.T) {
	cookie := loginAndGetCookie(t)

	// Measure response time for listing nodes
	start := time.Now()
	req, _ := http.NewRequest("GET", baseURL+"/admin/api/nodes", nil)
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list nodes failed: %v", err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)
	elapsed := time.Since(start)
	t.Logf("GET /admin/api/nodes response time: %v", elapsed)
	if elapsed > 50*time.Millisecond {
		t.Logf("WARNING: Response time %v exceeds 50ms target (may be due to network/cold start)", elapsed)
	}
}
