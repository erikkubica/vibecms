package db

import (
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"vibecms/internal/models"
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
    <title>{{or (index .node.seo "meta_title") .node.title "VibeCMS"}}</title>
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
	db.Exec(`INSERT INTO site_settings (key, value, updated_at) VALUES ('site_name', 'VibeCMS', NOW()) ON CONFLICT (key) DO NOTHING`)
	db.Exec(`INSERT INTO site_settings (key, value, updated_at) VALUES ('site_url', 'http://localhost:8099', NOW()) ON CONFLICT (key) DO NOTHING`)
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
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	var adminRole models.Role
	if err := db.Where("slug = ?", "admin").First(&adminRole).Error; err != nil {
		return fmt.Errorf("failed to find admin role: %w", err)
	}

	fullName := "Admin"
	admin := models.User{
		Email:        "admin@vibecms.local",
		PasswordHash: string(hash),
		RoleID:       adminRole.ID,
		FullName:     &fullName,
	}

	result := db.Where("email = ?", admin.Email).FirstOrCreate(&admin)
	if result.Error != nil {
		return fmt.Errorf("failed to seed admin user: %w", result.Error)
	}
	return nil
}

func seedContentNode(db *gorm.DB) error {
	seoSettings := json.RawMessage(`{"meta_title":"Welcome to VibeCMS","meta_description":"A high-performance, AI-native CMS."}`)
	now := time.Now()

	node := models.ContentNode{
		NodeType:     "page",
		Status:       "published",
		LanguageCode: "en",
		Slug:         "home",
		FullURL:      "/",
		Title:        "Welcome to VibeCMS",
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

func seedLayoutBlocks(db *gorm.DB) error {
	blocks := []models.LayoutBlock{
		{
			Slug:         "primary-nav",
			Name:         "Primary Navigation",
			Description:  "Main navigation menu with dropdown support (uses main-nav menu)",
			LanguageID:   nil,
			TemplateCode: primaryNavTemplate,
			Source:       "custom",
		},
		{
			Slug:         "user-menu",
			Name:         "User Menu",
			Description:  "Login/register or dashboard/logout based on auth state",
			LanguageID:   nil,
			TemplateCode: userMenuTemplate,
			Source:       "custom",
		},
		{
			Slug:         "site-header",
			Name:         "Site Header",
			Description:  "Full site header — includes primary-nav and user-menu blocks",
			LanguageID:   nil,
			TemplateCode: siteHeaderTemplate,
			Source:       "custom",
		},
		{
			Slug:         "footer-nav",
			Name:         "Footer Navigation",
			Description:  "Footer links from footer-nav menu",
			LanguageID:   nil,
			TemplateCode: footerNavTemplate,
			Source:       "custom",
		},
		{
			Slug:         "site-footer",
			Name:         "Site Footer",
			Description:  "Site footer with copyright and footer-nav block",
			LanguageID:   nil,
			TemplateCode: siteFooterTemplate,
			Source:       "custom",
		},
	}

	for _, block := range blocks {
		var existing models.LayoutBlock
		result := db.Where("slug = ? AND language_id IS NULL", block.Slug).First(&existing)
		if result.Error == nil {
			// If theme owns this block, don't overwrite
			if existing.Source == "theme" {
				continue
			}
			// Update existing custom block with the latest template
			db.Model(&existing).Updates(map[string]interface{}{
				"name":          block.Name,
				"description":   block.Description,
				"template_code": block.TemplateCode,
			})
		} else {
			if err := db.Create(&block).Error; err != nil {
				return fmt.Errorf("failed to seed layout block %q: %w", block.Slug, err)
			}
		}
	}
	return nil
}

func seedDefaultLayout(db *gorm.DB) error {
	layout := models.Layout{
		Slug:         "default",
		Name:         "Default Layout",
		Description:  "Default page layout with header, footer, and Tailwind CSS",
		LanguageID:   nil,
		TemplateCode: defaultLayoutTemplate,
		Source:       "custom",
		IsDefault:    true,
	}

	var existing models.Layout
	result := db.Where("slug = ? AND language_id IS NULL", layout.Slug).First(&existing)
	if result.Error == nil {
		// If theme owns this layout, don't overwrite
		if existing.Source == "theme" {
			return nil
		}
		// Update existing custom layout
		db.Model(&existing).Updates(map[string]interface{}{
			"name":          layout.Name,
			"description":   layout.Description,
			"template_code": layout.TemplateCode,
			"is_default":    layout.IsDefault,
		})
	} else {
		if err := db.Create(&layout).Error; err != nil {
			return fmt.Errorf("failed to seed default layout: %w", err)
		}
	}
	return nil
}

// --- Auth block HTML templates ---
// These use Alpine.js to read flash cookies client-side since content blocks
// don't have access to Go request context.

const flashAlpineSnippet = `x-data="{
    flash: '', flashType: '',
    init() {
        const msg = this.getCookie('flash_msg');
        const typ = this.getCookie('flash_type');
        if (msg) { this.flash = decodeURIComponent(msg); this.flashType = typ || 'error'; this.clearCookies(); }
    },
    getCookie(n) { const m = document.cookie.match('(^|;)\\\\s*'+n+'=([^;]+)'); return m ? m[2] : ''; },
    clearCookies() { document.cookie='flash_msg=;path=/;max-age=0'; document.cookie='flash_type=;path=/;max-age=0'; }
}"`

const loginFormTemplate = `<div class="flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full" ` + flashAlpineSnippet + `>
        <div class="text-center mb-8">
            <h1 class="text-3xl font-bold text-slate-900">Sign In</h1>
            <p class="mt-2 text-sm text-slate-600">Sign in to your account</p>
        </div>

        <template x-if="flash">
            <div class="mb-4 rounded-md p-4" :class="flashType === 'success' ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'">
                <p class="text-sm" :class="flashType === 'success' ? 'text-green-800' : 'text-red-800'" x-text="flash"></p>
            </div>
        </template>

        <div class="bg-white shadow-md rounded-lg px-8 py-8">
            <form method="POST" action="/auth/login-page" class="space-y-6">
                <div>
                    <label for="email" class="block text-sm font-medium text-slate-700">Email address</label>
                    <input id="email" name="email" type="email" autocomplete="email" required
                        class="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                        placeholder="you@example.com">
                </div>
                <div>
                    <label for="password" class="block text-sm font-medium text-slate-700">Password</label>
                    <input id="password" name="password" type="password" autocomplete="current-password" required
                        class="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                        placeholder="Enter your password">
                </div>
                <div class="flex items-center justify-between">
                    <div class="flex items-center">
                        <input id="remember" name="remember" type="checkbox" class="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500">
                        <label for="remember" class="ml-2 block text-sm text-slate-700">Remember me</label>
                    </div>
                    <a href="/forgot-password" class="text-sm font-medium text-indigo-600 hover:text-indigo-500">Forgot password?</a>
                </div>
                <button type="submit" class="w-full flex justify-center rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 transition-colors duration-200">Sign In</button>
            </form>
        </div>
        <p class="mt-6 text-center text-sm text-slate-600">Don't have an account? <a href="/register" class="font-medium text-indigo-600 hover:text-indigo-500">Register</a></p>
    </div>
</div>`

const registerFormTemplate = `<div class="flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full" ` + flashAlpineSnippet + `>
        <div class="text-center mb-8">
            <h1 class="text-3xl font-bold text-slate-900">Create Account</h1>
            <p class="mt-2 text-sm text-slate-600">Create your account</p>
        </div>

        <template x-if="flash">
            <div class="mb-4 rounded-md p-4" :class="flashType === 'success' ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'">
                <p class="text-sm" :class="flashType === 'success' ? 'text-green-800' : 'text-red-800'" x-text="flash"></p>
            </div>
        </template>

        <div class="bg-white shadow-md rounded-lg px-8 py-8">
            <form method="POST" action="/auth/register" class="space-y-6">
                <div>
                    <label for="full_name" class="block text-sm font-medium text-slate-700">Full Name</label>
                    <input id="full_name" name="full_name" type="text" autocomplete="name" required
                        class="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                        placeholder="Jane Doe">
                </div>
                <div>
                    <label for="email" class="block text-sm font-medium text-slate-700">Email address</label>
                    <input id="email" name="email" type="email" autocomplete="email" required
                        class="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                        placeholder="you@example.com">
                </div>
                <div>
                    <label for="password" class="block text-sm font-medium text-slate-700">Password</label>
                    <input id="password" name="password" type="password" autocomplete="new-password" required
                        class="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                        placeholder="Create a password">
                </div>
                <div>
                    <label for="password_confirm" class="block text-sm font-medium text-slate-700">Confirm Password</label>
                    <input id="password_confirm" name="password_confirm" type="password" autocomplete="new-password" required
                        class="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                        placeholder="Confirm your password">
                </div>
                <button type="submit" class="w-full flex justify-center rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 transition-colors duration-200">Create Account</button>
            </form>
        </div>
        <p class="mt-6 text-center text-sm text-slate-600">Already have an account? <a href="/login" class="font-medium text-indigo-600 hover:text-indigo-500">Sign In</a></p>
    </div>
</div>`

const forgotPasswordFormTemplate = `<div class="flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full" ` + flashAlpineSnippet + `>
        <div class="text-center mb-8">
            <h1 class="text-3xl font-bold text-slate-900">Forgot Password</h1>
            <p class="mt-2 text-sm text-slate-600">Reset your password</p>
        </div>

        <template x-if="flash">
            <div class="mb-4 rounded-md p-4" :class="flashType === 'success' ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'">
                <p class="text-sm" :class="flashType === 'success' ? 'text-green-800' : 'text-red-800'" x-text="flash"></p>
            </div>
        </template>

        <div class="bg-white shadow-md rounded-lg px-8 py-8">
            <p class="text-sm text-slate-600 mb-6">Enter your email address and we'll send you a link to reset your password.</p>
            <form method="POST" action="/auth/forgot-password" class="space-y-6">
                <div>
                    <label for="email" class="block text-sm font-medium text-slate-700">Email address</label>
                    <input id="email" name="email" type="email" autocomplete="email" required
                        class="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                        placeholder="you@example.com">
                </div>
                <button type="submit" class="w-full flex justify-center rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 transition-colors duration-200">Send Reset Link</button>
            </form>
        </div>
        <p class="mt-6 text-center text-sm text-slate-600"><a href="/login" class="font-medium text-indigo-600 hover:text-indigo-500">Back to Sign In</a></p>
    </div>
</div>`

const resetPasswordFormTemplate = `<div class="flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full" ` + flashAlpineSnippet + `>
        <div class="text-center mb-8">
            <h1 class="text-3xl font-bold text-slate-900">Reset Password</h1>
            <p class="mt-2 text-sm text-slate-600">Set a new password</p>
        </div>

        <template x-if="flash">
            <div class="mb-4 rounded-md p-4" :class="flashType === 'success' ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'">
                <p class="text-sm" :class="flashType === 'success' ? 'text-green-800' : 'text-red-800'" x-text="flash"></p>
            </div>
        </template>

        <div class="bg-white shadow-md rounded-lg px-8 py-8">
            <form method="POST" action="/auth/reset-password" class="space-y-6" x-data x-init="$el.querySelector('[name=token]').value = new URLSearchParams(location.search).get('token') || ''">
                <input type="hidden" name="token" value="">
                <div>
                    <label for="password" class="block text-sm font-medium text-slate-700">New Password</label>
                    <input id="password" name="password" type="password" autocomplete="new-password" required
                        class="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                        placeholder="Enter new password">
                </div>
                <div>
                    <label for="password_confirm" class="block text-sm font-medium text-slate-700">Confirm New Password</label>
                    <input id="password_confirm" name="password_confirm" type="password" autocomplete="new-password" required
                        class="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                        placeholder="Confirm new password">
                </div>
                <button type="submit" class="w-full flex justify-center rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 transition-colors duration-200">Reset Password</button>
            </form>
        </div>
    </div>
</div>`

func seedAuthBlockTypes(db *gorm.DB) error {
	blockTypes := []models.BlockType{
		{
			Slug:        "login-form",
			Label:       "Login Form",
			Icon:        "log-in",
			Description: "User login form with email and password fields",
			FieldSchema: models.JSONB(json.RawMessage(`[]`)),
			HTMLTemplate: loginFormTemplate,
			Source:      "system",
		},
		{
			Slug:        "register-form",
			Label:       "Registration Form",
			Icon:        "user-plus",
			Description: "User registration form with name, email, and password fields",
			FieldSchema: models.JSONB(json.RawMessage(`[]`)),
			HTMLTemplate: registerFormTemplate,
			Source:      "system",
		},
		{
			Slug:        "forgot-password-form",
			Label:       "Forgot Password Form",
			Icon:        "key",
			Description: "Password reset request form",
			FieldSchema: models.JSONB(json.RawMessage(`[]`)),
			HTMLTemplate: forgotPasswordFormTemplate,
			Source:      "system",
		},
		{
			Slug:        "reset-password-form",
			Label:       "Reset Password Form",
			Icon:        "lock",
			Description: "Set new password form (used with reset token)",
			FieldSchema: models.JSONB(json.RawMessage(`[]`)),
			HTMLTemplate: resetPasswordFormTemplate,
			Source:      "system",
		},
	}

	for _, bt := range blockTypes {
		var existing models.BlockType
		result := db.Where("slug = ?", bt.Slug).First(&existing)
		if result.Error == nil {
			db.Model(&existing).Updates(map[string]interface{}{
				"label":         bt.Label,
				"icon":          bt.Icon,
				"description":   bt.Description,
				"html_template": bt.HTMLTemplate,
				"source":        bt.Source,
			})
		} else {
			if err := db.Create(&bt).Error; err != nil {
				return fmt.Errorf("failed to seed auth block type %q: %w", bt.Slug, err)
			}
		}
	}
	return nil
}

func seedAuthPages(db *gorm.DB) error {
	now := time.Now()

	pages := []struct {
		slug      string
		fullURL   string
		title     string
		blockType string
		seoTitle  string
	}{
		{"login", "/login", "Sign In", "login-form", "Sign In"},
		{"register", "/register", "Create Account", "register-form", "Create Account"},
		{"forgot-password", "/forgot-password", "Forgot Password", "forgot-password-form", "Forgot Password"},
		{"reset-password", "/reset-password", "Reset Password", "reset-password-form", "Reset Password"},
	}

	for _, p := range pages {
		blocksData := json.RawMessage(fmt.Sprintf(`[{"type":"%s","fields":{}}]`, p.blockType))
		seoSettings := json.RawMessage(fmt.Sprintf(`{"meta_title":"%s"}`, p.seoTitle))

		node := models.ContentNode{
			NodeType:     "page",
			Status:       "published",
			LanguageCode: "en",
			Slug:         p.slug,
			FullURL:      p.fullURL,
			Title:        p.title,
			BlocksData:   models.JSONB(blocksData),
			SeoSettings:  models.JSONB(seoSettings),
			Version:      1,
			PublishedAt:  &now,
		}

		result := db.Where("full_url = ?", node.FullURL).FirstOrCreate(&node)
		if result.Error != nil {
			return fmt.Errorf("failed to seed auth page %q: %w", p.slug, result.Error)
		}
	}
	return nil
}

func seedEmailTemplates(db *gorm.DB) error {
	templates := []models.EmailTemplate{
		{
			Slug:            "welcome",
			Name:            "Welcome Email",
			SubjectTemplate: "Welcome to {{.site_name}}",
			BodyTemplate: `<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
<h2>Welcome, {{.user_full_name}}!</h2>
<p>Your account has been created on <strong>{{.site_name}}</strong>.</p>
<p>You can log in at: <a href="{{.site_url}}/login">{{.site_url}}/login</a></p>
</div>`,
			TestData: models.JSONB(`{"site_name":"VibeCMS","site_url":"http://localhost:8099","user_full_name":"Jane Doe","user_email":"jane@example.com"}`),
		},
		{
			Slug:            "user-registered-admin",
			Name:            "New User Registered (Admin)",
			SubjectTemplate: "New user registered: {{.user_full_name}}",
			BodyTemplate: `<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
<h2>New User Registration</h2>
<p>A new user has registered on <strong>{{.site_name}}</strong>.</p>
<table style="border-collapse: collapse; margin: 16px 0;">
<tr><td style="padding: 4px 12px 4px 0; color: #666;">Name:</td><td>{{.user_full_name}}</td></tr>
<tr><td style="padding: 4px 12px 4px 0; color: #666;">Email:</td><td>{{.user_email}}</td></tr>
</table>
</div>`,
			TestData: models.JSONB(`{"site_name":"VibeCMS","user_full_name":"Jane Doe","user_email":"jane@example.com"}`),
		},
		{
			Slug:            "password-reset",
			Name:            "Password Reset",
			SubjectTemplate: "Reset your password on {{.site_name}}",
			BodyTemplate: `<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
<h2>Password Reset</h2>
<p>Hi {{.user_full_name}},</p>
<p>You requested a password reset for your account on <strong>{{.site_name}}</strong>.</p>
<p><a href="{{.reset_url}}" style="display: inline-block; padding: 10px 24px; background: #4f46e5; color: #fff; text-decoration: none; border-radius: 6px;">Reset Password</a></p>
<p style="color: #666; font-size: 14px;">If you didn't request this, you can safely ignore this email.</p>
</div>`,
			TestData: models.JSONB(`{"site_name":"VibeCMS","user_full_name":"Jane Doe","reset_url":"http://localhost:8099/reset-password?token=abc123"}`),
		},
		{
			Slug:            "node-published",
			Name:            "Content Published",
			SubjectTemplate: "{{.node_title}} has been published",
			BodyTemplate: `<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
<h2>Content Published</h2>
<p>"<strong>{{.node_title}}</strong>" ({{.node_type}}) has been published on <strong>{{.site_name}}</strong>.</p>
<p><a href="{{.site_url}}{{.full_url}}">View it here</a></p>
</div>`,
			TestData: models.JSONB(`{"site_name":"VibeCMS","site_url":"http://localhost:8099","node_title":"Hello World","node_type":"post","full_url":"/hello-world"}`),
		},
		{
			Slug:            "node-created-admin",
			Name:            "New Content Created (Admin)",
			SubjectTemplate: "New {{.node_type}} created: {{.node_title}}",
			BodyTemplate: `<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
<h2>New Content Created</h2>
<p>A new <strong>{{.node_type}}</strong> has been created on <strong>{{.site_name}}</strong>.</p>
<table style="border-collapse: collapse; margin: 16px 0;">
<tr><td style="padding: 4px 12px 4px 0; color: #666;">Title:</td><td>{{.node_title}}</td></tr>
<tr><td style="padding: 4px 12px 4px 0; color: #666;">URL:</td><td>{{.site_url}}{{.full_url}}</td></tr>
</table>
</div>`,
			TestData: models.JSONB(`{"site_name":"VibeCMS","site_url":"http://localhost:8099","node_title":"Hello World","node_type":"post","full_url":"/hello-world"}`),
		},
	}

	for _, t := range templates {
		var existing models.EmailTemplate
		result := db.Where("slug = ?", t.Slug).First(&existing)
		if result.Error == nil {
			db.Model(&existing).Updates(map[string]interface{}{
				"name":             t.Name,
				"subject_template": t.SubjectTemplate,
				"body_template":    t.BodyTemplate,
				"test_data":        t.TestData,
			})
		} else {
			if err := db.Create(&t).Error; err != nil {
				return fmt.Errorf("failed to seed email template %q: %w", t.Slug, err)
			}
		}
	}
	return nil
}

func seedEmailRules(db *gorm.DB) error {
	// Look up template IDs
	var welcome, adminNotif, nodePublished, nodeCreatedAdmin models.EmailTemplate
	db.Where("slug = ?", "welcome").First(&welcome)
	db.Where("slug = ?", "user-registered-admin").First(&adminNotif)
	db.Where("slug = ?", "node-published").First(&nodePublished)
	db.Where("slug = ?", "node-created-admin").First(&nodeCreatedAdmin)

	if welcome.ID == 0 || adminNotif.ID == 0 || nodePublished.ID == 0 || nodeCreatedAdmin.ID == 0 {
		return nil // Templates not seeded yet, skip rules
	}

	rules := []models.EmailRule{
		{Action: "user.registered", TemplateID: welcome.ID, RecipientType: "actor", RecipientValue: "", Enabled: true},
		{Action: "user.registered", TemplateID: adminNotif.ID, RecipientType: "role", RecipientValue: "admin", Enabled: true},
		{Action: "node.published", TemplateID: nodePublished.ID, RecipientType: "node_author", RecipientValue: "", Enabled: true},
		{Action: "node.created", TemplateID: nodeCreatedAdmin.ID, RecipientType: "role", RecipientValue: "admin", Enabled: true},
	}

	for _, r := range rules {
		var count int64
		db.Model(&models.EmailRule{}).Where("action = ? AND template_id = ? AND recipient_type = ?", r.Action, r.TemplateID, r.RecipientType).Count(&count)
		if count == 0 {
			if err := db.Create(&r).Error; err != nil {
				return fmt.Errorf("failed to seed email rule for %q: %w", r.Action, err)
			}
		}
	}
	return nil
}

func seedMenus(db *gorm.DB) error {
	menus := []struct {
		slug  string
		name  string
		items []models.MenuItem
	}{
		{
			slug: "main-nav",
			name: "Main Navigation",
			items: []models.MenuItem{
				{Title: "Home", ItemType: "custom", URL: "/", Target: "_self", SortOrder: 0},
				{Title: "About", ItemType: "custom", URL: "/about", Target: "_self", SortOrder: 1},
			},
		},
		{
			slug: "footer-nav",
			name: "Footer Navigation",
			items: []models.MenuItem{
				{Title: "Home", ItemType: "custom", URL: "/", Target: "_self", SortOrder: 0},
				{Title: "About", ItemType: "custom", URL: "/about", Target: "_self", SortOrder: 1},
				{Title: "Privacy", ItemType: "custom", URL: "/privacy", Target: "_self", SortOrder: 2},
				{Title: "Terms", ItemType: "custom", URL: "/terms", Target: "_self", SortOrder: 3},
			},
		},
	}

	for _, m := range menus {
		var existing models.Menu
		result := db.Where("slug = ? AND language_id IS NULL", m.slug).First(&existing)
		if result.Error == nil {
			var count int64
			db.Model(&models.MenuItem{}).Where("menu_id = ?", existing.ID).Count(&count)
			if count > 0 {
				continue
			}
			for i := range m.items {
				m.items[i].MenuID = existing.ID
				db.Create(&m.items[i])
			}
			continue
		}

		menu := models.Menu{Slug: m.slug, Name: m.name, LanguageID: nil, Version: 1}
		if err := db.Create(&menu).Error; err != nil {
			return fmt.Errorf("failed to seed menu %q: %w", m.slug, err)
		}
		for i := range m.items {
			m.items[i].MenuID = menu.ID
			if err := db.Create(&m.items[i]).Error; err != nil {
				return fmt.Errorf("failed to seed menu item %q: %w", m.items[i].Title, err)
			}
		}
	}
	return nil
}
