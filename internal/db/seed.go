package db

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	"squilla/internal/auth"
	"squilla/internal/models"
)

// Primary navigation — renders "main-nav" menu with dropdown support.
const primaryNavTemplate = `{{- $menu := index .app.menus "main-nav" -}}
{{- if $menu -}}
<nav class="hidden md:flex items-center space-x-1">
    {{- range $menu.items -}}
    {{- if .children -}}
    <div class="relative" x-data="{ open: false }" @mouseenter="open = true" @mouseleave="open = false">
        <a href="{{.url}}" class="inline-flex items-center px-3 py-2 text-sm font-medium text-slate-700 hover:text-indigo-600 hover:bg-slate-50 rounded-md transition-colors" :class="{ 'text-indigo-600 bg-slate-50': open }">
            {{.title}}
            <svg class="ml-1 h-4 w-4 text-slate-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M5.22 8.22a.75.75 0 0 1 1.06 0L10 11.94l3.72-3.72a.75.75 0 1 1 1.06 1.06l-4.25 4.25a.75.75 0 0 1-1.06 0L5.22 9.28a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd"/></svg>
        </a>
        <div x-show="open" x-transition:enter="transition ease-out duration-100" x-transition:enter-start="opacity-0 scale-95" x-transition:enter-end="opacity-100 scale-100" x-transition:leave="transition ease-in duration-75" x-transition:leave-start="opacity-100 scale-100" x-transition:leave-end="opacity-0 scale-95" class="absolute left-0 mt-0 w-48 rounded-md bg-white shadow-lg ring-1 ring-black/5 py-1 z-50" style="display: none;">
            {{- range .children -}}
            <a href="{{.url}}" class="block px-4 py-2 text-sm text-slate-700 hover:bg-indigo-50 hover:text-indigo-600 transition-colors">{{.title}}</a>
            {{- end -}}
        </div>
    </div>
    {{- else -}}
    <a href="{{.url}}" class="px-3 py-2 text-sm font-medium text-slate-700 hover:text-indigo-600 hover:bg-slate-50 rounded-md transition-colors">{{.title}}</a>
    {{- end -}}
    {{- end -}}
</nav>
{{- end -}}`

// User menu — login/register when logged out, dashboard/logout when logged in.
const userMenuTemplate = `<div class="hidden md:flex items-center space-x-3">
    {{if .user.logged_in}}
    <span class="text-sm text-slate-500">{{.user.full_name}}</span>
    <a href="/admin" class="px-3 py-2 text-sm font-medium text-slate-700 hover:text-indigo-600 hover:bg-slate-50 rounded-md transition-colors">Dashboard</a>
    <a href="/logout" class="inline-flex items-center px-4 py-2 border border-slate-300 rounded-lg text-sm font-medium text-slate-700 bg-white hover:bg-slate-50 transition-colors">Logout</a>
    {{else}}
    <a href="/login" class="px-3 py-2 text-sm font-medium text-slate-700 hover:text-indigo-600 hover:bg-slate-50 rounded-md transition-colors">Login</a>
    <a href="/register" class="inline-flex items-center px-4 py-2 rounded-lg text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 transition-colors">Register</a>
    {{end}}
</div>`

// Site header — shell that includes primary-nav and user-menu layout blocks.
const siteHeaderTemplate = `<header class="sticky top-0 z-50 bg-white border-b border-slate-200 shadow-sm" x-data="{ mobileOpen: false }">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-16">
            <div class="flex items-center space-x-8">
                <a href="/" class="text-xl font-bold text-indigo-600 hover:text-indigo-700 transition-colors">
                    {{.app.settings.site_name}}
                </a>
                {{renderLayoutBlock "primary-nav"}}
            </div>

            <div class="flex items-center">
                {{renderLayoutBlock "user-menu"}}

                {{/* Mobile Menu Button */}}
                <div class="md:hidden">
                    <button @click="mobileOpen = !mobileOpen" class="inline-flex items-center justify-center p-2 rounded-md text-slate-500 hover:text-indigo-600 hover:bg-slate-100 transition-colors" aria-label="Toggle menu">
                        <svg x-show="!mobileOpen" class="h-6 w-6" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5"/></svg>
                        <svg x-show="mobileOpen" class="h-6 w-6" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" style="display: none;"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12"/></svg>
                    </button>
                </div>
            </div>
        </div>
    </div>

    {{/* Mobile Navigation */}}
    {{- $menu := index .app.menus "main-nav" -}}
    {{- if $menu -}}
    <div x-show="mobileOpen" x-transition:enter="transition ease-out duration-200" x-transition:enter-start="opacity-0 -translate-y-1" x-transition:enter-end="opacity-100 translate-y-0" x-transition:leave="transition ease-in duration-150" x-transition:leave-start="opacity-100 translate-y-0" x-transition:leave-end="opacity-0 -translate-y-1" class="md:hidden border-t border-slate-200 bg-white" style="display: none;">
        <nav class="px-4 py-3 space-y-1">
            {{- range $menu.items -}}
            <a href="{{.url}}" class="block px-3 py-2 text-base font-medium text-slate-700 hover:text-indigo-600 hover:bg-slate-50 rounded-md transition-colors">{{.title}}</a>
            {{- if .children -}}
            {{- range .children -}}
            <a href="{{.url}}" class="block pl-8 pr-3 py-2 text-sm text-slate-500 hover:text-indigo-600 hover:bg-slate-50 rounded-md transition-colors">{{.title}}</a>
            {{- end -}}
            {{- end -}}
            {{- end -}}
            <hr class="my-2 border-slate-200">
            {{if .user.logged_in}}
            <a href="/admin" class="block px-3 py-2 text-base font-medium text-slate-700 hover:text-indigo-600 hover:bg-slate-50 rounded-md transition-colors">Dashboard</a>
            <a href="/logout" class="block px-3 py-2 text-base font-medium text-slate-700 hover:text-indigo-600 hover:bg-slate-50 rounded-md transition-colors">Logout</a>
            {{else}}
            <a href="/login" class="block px-3 py-2 text-base font-medium text-indigo-600 hover:bg-slate-50 rounded-md transition-colors">Login</a>
            {{end}}
        </nav>
    </div>
    {{- end -}}
</header>`

// Footer navigation — renders "footer-nav" menu.
const footerNavTemplate = `{{- $menu := index .app.menus "footer-nav" -}}
{{- if $menu -}}
<nav class="flex items-center space-x-6">
    {{- range $menu.items -}}
    <a href="{{.url}}" class="text-sm text-slate-500 hover:text-indigo-600 transition-colors">{{.title}}</a>
    {{- end -}}
</nav>
{{- end -}}`

// Site footer — uses footer-nav layout block for links.
const siteFooterTemplate = `<footer class="bg-white border-t border-slate-200 mt-auto">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div class="flex flex-col sm:flex-row items-center justify-between gap-4">
            <div class="text-sm text-slate-500">
                &copy; 2026 <span class="font-medium text-slate-700">{{.app.settings.site_name}}</span>. All rights reserved.
            </div>
            {{renderLayoutBlock "footer-nav"}}
        </div>
    </div>
</footer>`

// Default layout template — a complete HTML document referencing the layout blocks.
const defaultLayoutTemplate = `<!DOCTYPE html>
<html lang="{{.app.current_lang.code}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{or (index .node.seo "meta_title") .node.title "Squilla"}}</title>
    {{- with index .node.seo "meta_description" -}}
    <meta name="description" content="{{.}}">
    {{- end -}}
    <script src="https://cdn.tailwindcss.com"></script>
    <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
    {{- range .app.head_styles -}}
    <link rel="stylesheet" href="{{.}}">
    {{- end -}}
    {{.app.block_styles}}
</head>
<body class="min-h-screen flex flex-col bg-slate-50 text-slate-800 antialiased">
    {{renderLayoutBlock "site-header"}}

    <main class="flex-1">
        {{.node.blocks_html}}
    </main>

    {{renderLayoutBlock "site-footer"}}

    {{- range .app.foot_scripts -}}
    <script src="{{.}}" defer></script>
    {{- end -}}
    {{.app.block_scripts}}
</body>
</html>`

// SeedIfEmpty runs Seed only when the users table has no rows. Used to
// bootstrap zero-config deploys on first boot without re-seeding on
// every restart.
func SeedIfEmpty(db *gorm.DB) error {
	var count int64
	if err := db.Table("users").Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}
	log.Printf("[seed] users table contains %d row(s)", count)
	if count > 0 {
		return nil
	}
	log.Println("[seed] no users found — running first-boot seed")
	return Seed(db)
}

// Seed populates the database with initial data including a default admin
// user, a sample content node, layout blocks, a default layout, and a main navigation menu.
func Seed(db *gorm.DB) error {
	if err := seedRoles(db); err != nil {
		return err
	}
	if err := seedAdminUser(db); err != nil {
		return err
	}
	if err := seedContentNode(db); err != nil {
		return err
	}
	if err := seedLayoutBlocks(db); err != nil {
		return err
	}
	if err := seedDefaultLayout(db); err != nil {
		return err
	}
	if err := seedMenus(db); err != nil {
		return err
	}
	if err := seedAuthBlockTypes(db); err != nil {
		return err
	}
	if err := seedAuthPages(db); err != nil {
		return err
	}
	if err := seedEmailTemplates(db); err != nil {
		return err
	}
	if err := seedEmailRules(db); err != nil {
		return err
	}
	// Seed default site name setting
	db.Exec(`INSERT INTO site_settings (key, value, updated_at) VALUES ('site_name', 'Squilla', NOW()) ON CONFLICT (key) DO NOTHING`)
	db.Exec(`INSERT INTO site_settings (key, value, updated_at) VALUES ('site_url', 'http://localhost:8099', NOW()) ON CONFLICT (key) DO NOTHING`)
	// Public registration is closed by default. Operators flip this to "true"
	// in Admin → Security → Settings to allow self-registration. Stored under
	// language_code='' — registration is a global capability, not per-locale.
	db.Exec(`INSERT INTO site_settings (key, language_code, value, updated_at) VALUES ('allow_registration', '', 'false', NOW()) ON CONFLICT (key, language_code) DO NOTHING`)
	db.Exec(`INSERT INTO site_settings (key, language_code, value, updated_at) VALUES ('default_registration_role', '', 'member', NOW()) ON CONFLICT (key, language_code) DO NOTHING`)

	// Assign default layout_slug to all seeded nodes that don't have one.
	// This avoids extra fallback lookups on every render and ensures nodes
	// survive theme deactivate/reactivate cycles via the slug reference.
	db.Model(&models.ContentNode{}).
		Where("layout_slug IS NULL AND deleted_at IS NULL").
		Update("layout_slug", "default")

	return nil
}

func seedRoles(db *gorm.DB) error {
	roles := []models.Role{
		{Slug: "admin", Name: "Administrator", Description: "Full system access", IsSystem: true, Capabilities: models.JSONB(`{"admin_access":true,"manage_users":true,"manage_roles":true,"manage_settings":true,"manage_menus":true,"manage_layouts":true,"manage_email":true,"default_node_access":{"access":"write","scope":"all"},"email_subscriptions":["user.registered","user.deleted","node.created","node.updated","node.published","node.deleted"]}`)},
		{Slug: "editor", Name: "Editor", Description: "Can manage all content", IsSystem: true, Capabilities: models.JSONB(`{"admin_access":true,"manage_users":false,"manage_roles":false,"manage_settings":false,"manage_menus":true,"manage_layouts":false,"manage_email":false,"default_node_access":{"access":"write","scope":"all"},"email_subscriptions":["node.created","node.published"]}`)},
		{Slug: "author", Name: "Author", Description: "Can manage own content", IsSystem: true, Capabilities: models.JSONB(`{"admin_access":true,"manage_users":false,"manage_roles":false,"manage_settings":false,"manage_menus":false,"manage_layouts":false,"manage_email":false,"default_node_access":{"access":"write","scope":"own"},"email_subscriptions":["node.published"]}`)},
		{Slug: "member", Name: "Member", Description: "Public member, no admin access", IsSystem: true, Capabilities: models.JSONB(`{"admin_access":false,"manage_users":false,"manage_roles":false,"manage_settings":false,"manage_menus":false,"manage_layouts":false,"manage_email":false,"default_node_access":{"access":"read","scope":"all"},"email_subscriptions":[]}`)},
	}
	for _, role := range roles {
		var existing models.Role
		result := db.Where("slug = ?", role.Slug).First(&existing)
		if result.Error == nil {
			db.Model(&existing).Updates(map[string]interface{}{
				"name":         role.Name,
				"description":  role.Description,
				"capabilities": role.Capabilities,
			})
		} else {
			if err := db.Create(&role).Error; err != nil {
				return fmt.Errorf("failed to seed role %q: %w", role.Slug, err)
			}
		}
	}
	return nil
}

func seedAdminUser(db *gorm.DB) error {
	var adminRole models.Role
	if err := db.Where("slug = ?", "admin").First(&adminRole).Error; err != nil {
		return fmt.Errorf("failed to find admin role: %w", err)
	}

	email := envOr("ADMIN_EMAIL", "admin@squilla.local")

	var existing models.User
	err := db.Where("email = ?", email).First(&existing).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing admin: %w", err)
	}

	password := os.Getenv("ADMIN_PASSWORD")
	generated := false
	if password == "" {
		password, err = generateRandomPassword(24)
		if err != nil {
			return fmt.Errorf("failed to generate admin password: %w", err)
		}
		generated = true
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	fullName := "Admin"
	admin := models.User{
		Email:        email,
		PasswordHash: string(hash),
		RoleID:       adminRole.ID,
		FullName:     &fullName,
	}
	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	if generated {
		banner := strings.Repeat("=", 72)
		log.Printf("\n%s\n Squilla first-boot admin credentials (shown ONCE — change immediately)\n   Email:    %s\n   Password: %s\n%s\n", banner, email, password, banner)
	}
	return nil
}

// generateRandomPassword returns a URL-safe base64 string of approximately the
// requested length, derived from crypto/rand.
func generateRandomPassword(length int) (string, error) {
	n := length * 3 / 4
	if n < 12 {
		n = 12
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf)[:length], nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func seedContentNode(db *gorm.DB) error {
	seoSettings := json.RawMessage(`{"meta_title":"Welcome to Squilla","meta_description":"A high-performance, AI-native CMS."}`)
	now := time.Now()

	node := models.ContentNode{
		NodeType:     "page",
		Status:       "published",
		LanguageCode: "en",
		Slug:         "home",
		FullURL:      "/",
		Title:        "Welcome to Squilla",
		BlocksData:   models.JSONB(json.RawMessage(`[]`)),
		SeoSettings:  models.JSONB(seoSettings),
		Version:      1,
		PublishedAt:  &now,
	}

	result := db.Where("full_url = ?", node.FullURL).FirstOrCreate(&node)
	if result.Error != nil {
		return fmt.Errorf("failed to seed sample content node: %w", result.Error)
	}

	// Set as homepage
	db.Exec(`INSERT INTO site_settings (key, value, updated_at) VALUES ('homepage_node_id', ?, NOW()) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		fmt.Sprintf("%d", node.ID))

	return nil
}

