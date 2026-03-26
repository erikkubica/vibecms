package db

import (
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"vibecms/internal/models"
)

// Site header template with Tailwind CSS styling, Alpine.js mobile menu, and main-nav integration.
const siteHeaderTemplate = `<header class="sticky top-0 z-50 bg-white border-b border-slate-200 shadow-sm" x-data="{ mobileOpen: false }">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-16">
            {{/* Logo / Site Name */}}
            <div class="flex-shrink-0">
                <a href="/" class="text-xl font-bold text-indigo-600 hover:text-indigo-700 transition-colors">
                    {{.app.settings.site_name}}
                </a>
            </div>

            {{/* Desktop Navigation */}}
            {{- $menu := index .app.menus "main-nav" -}}
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
            {{- end -}}

            {{/* Mobile Menu Button */}}
            <div class="md:hidden">
                <button @click="mobileOpen = !mobileOpen" class="inline-flex items-center justify-center p-2 rounded-md text-slate-500 hover:text-indigo-600 hover:bg-slate-100 transition-colors" aria-label="Toggle menu">
                    <svg x-show="!mobileOpen" class="h-6 w-6" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5"/></svg>
                    <svg x-show="mobileOpen" class="h-6 w-6" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" style="display: none;"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12"/></svg>
                </button>
            </div>
        </div>
    </div>

    {{/* Mobile Navigation */}}
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
        </nav>
    </div>
    {{- end -}}
</header>`

// Site footer template with Tailwind CSS styling.
const siteFooterTemplate = `<footer class="bg-white border-t border-slate-200 mt-auto">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div class="flex flex-col sm:flex-row items-center justify-between gap-4">
            <div class="text-sm text-slate-500">
                &copy; 2026 <span class="font-medium text-slate-700">{{.app.settings.site_name}}</span>. All rights reserved.
            </div>
            <nav class="flex items-center space-x-6">
                <a href="/" class="text-sm text-slate-500 hover:text-indigo-600 transition-colors">Home</a>
                <a href="/about" class="text-sm text-slate-500 hover:text-indigo-600 transition-colors">About</a>
            </nav>
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
	if err := seedMainMenu(db); err != nil {
		return err
	}
	return nil
}

func seedAdminUser(db *gorm.DB) error {
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	fullName := "Admin"
	admin := models.User{
		Email:        "admin@vibecms.local",
		PasswordHash: string(hash),
		Role:         "admin",
		FullName:     &fullName,
	}

	result := db.Where("email = ?", admin.Email).FirstOrCreate(&admin)
	if result.Error != nil {
		return fmt.Errorf("failed to seed admin user: %w", result.Error)
	}
	return nil
}

func seedContentNode(db *gorm.DB) error {
	blocksData := json.RawMessage(`[{"type":"heading","data":{"text":"Welcome to VibeCMS","level":1}},{"type":"paragraph","data":{"text":"This is your first page. Edit it from the admin panel."}}]`)
	seoSettings := json.RawMessage(`{"meta_title":"Welcome to VibeCMS","meta_description":"A high-performance, AI-native CMS."}`)
	now := time.Now()

	node := models.ContentNode{
		NodeType:     "page",
		Status:       "published",
		LanguageCode: "en",
		Slug:         "home",
		FullURL:      "/",
		Title:        "Welcome to VibeCMS",
		BlocksData:   models.JSONB(blocksData),
		SeoSettings:  models.JSONB(seoSettings),
		Version:      1,
		PublishedAt:  &now,
	}

	result := db.Where("full_url = ?", node.FullURL).FirstOrCreate(&node)
	if result.Error != nil {
		return fmt.Errorf("failed to seed sample content node: %w", result.Error)
	}
	return nil
}

func seedLayoutBlocks(db *gorm.DB) error {
	blocks := []models.LayoutBlock{
		{
			Slug:         "site-header",
			Name:         "Site Header",
			Description:  "Primary site header with navigation and mobile menu",
			LanguageID:   nil,
			TemplateCode: siteHeaderTemplate,
			Source:       "custom",
		},
		{
			Slug:         "site-footer",
			Name:         "Site Footer",
			Description:  "Site footer with copyright and links",
			LanguageID:   nil,
			TemplateCode: siteFooterTemplate,
			Source:       "custom",
		},
	}

	for _, block := range blocks {
		var existing models.LayoutBlock
		result := db.Where("slug = ? AND language_id IS NULL", block.Slug).First(&existing)
		if result.Error == nil {
			// Update existing block with the latest template
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
		// Update existing layout (created by migration) with the full template
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

func seedMainMenu(db *gorm.DB) error {
	var existing models.Menu
	result := db.Where("slug = ? AND language_id IS NULL", "main-nav").First(&existing)
	if result.Error == nil {
		// Menu already exists; ensure it has items
		var count int64
		db.Model(&models.MenuItem{}).Where("menu_id = ?", existing.ID).Count(&count)
		if count > 0 {
			return nil // Already has items, skip
		}
		// Add default items to existing menu
		return seedMenuItems(db, existing.ID)
	}

	menu := models.Menu{
		Slug:       "main-nav",
		Name:       "Main Navigation",
		LanguageID: nil,
		Version:    1,
	}
	if err := db.Create(&menu).Error; err != nil {
		return fmt.Errorf("failed to seed main-nav menu: %w", err)
	}

	return seedMenuItems(db, menu.ID)
}

func seedMenuItems(db *gorm.DB, menuID int) error {
	items := []models.MenuItem{
		{
			MenuID:    menuID,
			Title:     "Home",
			ItemType:  "custom",
			URL:       "/",
			Target:    "_self",
			SortOrder: 0,
		},
		{
			MenuID:    menuID,
			Title:     "About",
			ItemType:  "custom",
			URL:       "/about",
			Target:    "_self",
			SortOrder: 1,
		},
	}

	for _, item := range items {
		if err := db.Create(&item).Error; err != nil {
			return fmt.Errorf("failed to seed menu item %q: %w", item.Title, err)
		}
	}
	return nil
}
