package cms

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"squilla/internal/models"
	"squilla/internal/testutil"
)

// robotsTestCase declares one input → expected-content combination.
// Each test seeds the supplied site_settings rows, hits /robots.txt,
// and asserts that every contains/excludes string is honoured. We
// match on substrings rather than full equality because the response
// is generated and includes a comment header that may legitimately
// drift; the substrings are the parts that actually matter.
type robotsTestCase struct {
	name     string
	settings map[string]string
	contains []string
	excludes []string
}

func TestRobotsHandler(t *testing.T) {
	cases := []robotsTestCase{
		{
			name:     "default allows everyone, advertises sitemap",
			settings: map[string]string{"site_url": "https://example.com"},
			contains: []string{
				"User-agent: *",
				"Allow: /",
				"Disallow: /admin",
				"Disallow: /auth/",
				"Sitemap: https://example.com/sitemap.xml",
			},
			excludes: []string{
				"GPTBot",
				"ClaudeBot",
				"ChatGPT-User",
			},
		},
		{
			name:     "indexing off → blanket Disallow, nothing else",
			settings: map[string]string{"seo_robots_index": "false"},
			contains: []string{
				"User-agent: *",
				"Disallow: /",
			},
			excludes: []string{
				"Allow: /",
				"GPTBot",
				"Sitemap:",
			},
		},
		{
			name: "AI training off blocks training crawlers, leaves search alone",
			settings: map[string]string{
				"robots_allow_ai_training": "false",
			},
			contains: []string{
				"User-agent: GPTBot",
				"User-agent: ClaudeBot",
				"User-agent: Google-Extended",
				"User-agent: CCBot",
			},
			excludes: []string{
				"User-agent: ChatGPT-User",
				"User-agent: PerplexityBot",
			},
		},
		{
			name: "AI search off blocks answer-engine crawlers, leaves training alone",
			settings: map[string]string{
				"robots_allow_ai_search": "false",
			},
			contains: []string{
				"User-agent: ChatGPT-User",
				"User-agent: PerplexityBot",
				"User-agent: OAI-SearchBot",
			},
			excludes: []string{
				"User-agent: GPTBot",
				"User-agent: ClaudeBot",
			},
		},
		{
			name: "both AI toggles off blocks every documented bot",
			settings: map[string]string{
				"robots_allow_ai_training": "false",
				"robots_allow_ai_search":   "false",
			},
			contains: []string{
				"User-agent: GPTBot",
				"User-agent: ClaudeBot",
				"User-agent: ChatGPT-User",
				"User-agent: PerplexityBot",
			},
		},
		{
			name: "extra disallow paths land in the User-agent: * block",
			settings: map[string]string{
				"robots_disallow_paths": "/private/\n/internal/\n",
			},
			contains: []string{
				"Disallow: /private/",
				"Disallow: /internal/",
			},
		},
		{
			name: "custom block is appended verbatim",
			settings: map[string]string{
				"robots_custom": "User-agent: SpecialBot\nCrawl-delay: 10",
			},
			contains: []string{
				"User-agent: SpecialBot",
				"Crawl-delay: 10",
			},
		},
		{
			name: "explicit sitemap URL override beats site_url derivation",
			settings: map[string]string{
				"site_url":           "https://example.com",
				"robots_sitemap_url": "https://cdn.example.com/seo/sitemap.xml",
			},
			contains: []string{
				"Sitemap: https://cdn.example.com/seo/sitemap.xml",
			},
			excludes: []string{
				"Sitemap: https://example.com/sitemap.xml",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewSQLiteDB(t)
			if err := db.AutoMigrate(&models.SiteSetting{}); err != nil {
				t.Fatalf("migrate: %v", err)
			}
			for k, v := range tc.settings {
				val := v
				if err := db.Create(&models.SiteSetting{Key: k, LanguageCode: "", Value: &val}).Error; err != nil {
					t.Fatalf("seed %q: %v", k, err)
				}
			}

			app := fiber.New()
			NewRobotsHandler(db).RegisterRoutes(app)

			req := httptest.NewRequest("GET", "/robots.txt", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("test request: %v", err)
			}
			if resp.StatusCode != 200 {
				t.Fatalf("status: got %d, want 200", resp.StatusCode)
			}
			if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
				t.Errorf("content-type: got %q, want text/plain*", ct)
			}
			body, _ := io.ReadAll(resp.Body)
			text := string(body)
			for _, want := range tc.contains {
				if !strings.Contains(text, want) {
					t.Errorf("missing %q from output\n--- output ---\n%s", want, text)
				}
			}
			for _, dontWant := range tc.excludes {
				if strings.Contains(text, dontWant) {
					t.Errorf("unexpected %q in output\n--- output ---\n%s", dontWant, text)
				}
			}
		})
	}
}
