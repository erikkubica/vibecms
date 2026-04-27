# Layout, Template & Menu System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add layout management (full page templates from `<head>` to `</footer>`), reusable layout blocks (partials), a hierarchical menu system with 3-level submenu support, and a theme asset pipeline to VibeCMS.

**Architecture:** Separate tables for layouts, layout_blocks, menus/menu_items. Convention-based template resolution cascade with per-node override. Theme registration via `theme.json` manifest. Language-aware with default-language fallback. In-process caching for resolved layouts/menus.

**Tech Stack:** Go 1.22+, Fiber, GORM, PostgreSQL (SERIAL PKs), Go html/template, React + TypeScript (admin SPA), Monaco editor

**Spec:** `docs/superpowers/specs/2026-03-26-layout-template-menu-system-design.md`

---

## File Structure

### New Files — Backend

| File | Responsibility |
|------|---------------|
| `internal/models/layout.go` | Layout GORM model |
| `internal/models/layout_block.go` | LayoutBlock (partial) GORM model |
| `internal/models/menu.go` | Menu + MenuItem GORM models |
| `internal/cms/layout_svc.go` | Layout CRUD + cascade resolution + caching |
| `internal/cms/layout_handler.go` | Layout REST API endpoints |
| `internal/cms/layout_block_svc.go` | LayoutBlock CRUD + language fallback |
| `internal/cms/layout_block_handler.go` | LayoutBlock REST API endpoints |
| `internal/cms/menu_svc.go` | Menu + MenuItem CRUD + tree operations + optimistic locking |
| `internal/cms/menu_handler.go` | Menu REST API endpoints |
| `internal/cms/theme_loader.go` | Theme manifest parsing + asset registry + DB upsert |
| `internal/cms/render_context.go` | Build `.app` and `.node` template context |
| `internal/db/migrations/0008_layouts_menus.sql` | DDL for all new tables + alterations |

### New Files — Frontend

| File | Responsibility |
|------|---------------|
| `admin-ui/src/pages/layouts-list.tsx` | Layout list page |
| `admin-ui/src/pages/layout-editor.tsx` | Layout code editor page |
| `admin-ui/src/pages/layout-blocks-list.tsx` | Layout block list page |
| `admin-ui/src/pages/layout-block-editor.tsx` | Layout block code editor page |
| `admin-ui/src/pages/menus-list.tsx` | Menu list page |
| `admin-ui/src/pages/menu-editor.tsx` | Menu drag-and-drop tree editor |
| `admin-ui/src/components/code-editor.tsx` | Shared Monaco/CodeMirror editor component (if not existing) |
| `admin-ui/src/components/menu-tree.tsx` | Nested sortable menu tree component |

### Modified Files

| File | Change |
|------|--------|
| `internal/models/content_node.go` | Add `LayoutID *int` field |
| `internal/cms/public_handler.go` | Replace fixed template with layout resolution + new render context |
| `internal/rendering/template_renderer.go` | Add `renderLayoutBlock` FuncMap, layout rendering from DB code |
| `cmd/vibecms/main.go` | Wire new services/handlers, theme loader, asset route |
| `admin-ui/src/api/client.ts` | Add Layout, LayoutBlock, Menu API functions |
| `admin-ui/src/App.tsx` (or router file) | Add routes for new pages |
| `internal/cms/node_handler.go` | Accept `layout_id` in create/update payloads |

---

## Task 1: Database Migration

**Files:**
- Create: `internal/db/migrations/0008_layouts_menus.sql`

- [ ] **Step 1: Write migration SQL**

```sql
-- Layouts
CREATE TABLE IF NOT EXISTS layouts (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    language_code VARCHAR(10) NOT NULL REFERENCES languages(code),
    template_code TEXT NOT NULL DEFAULT '',
    source VARCHAR(20) NOT NULL DEFAULT 'custom',
    theme_name VARCHAR(100),
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(slug, language_code)
);

CREATE INDEX IF NOT EXISTS idx_layouts_source_theme ON layouts(source, theme_name);
CREATE UNIQUE INDEX IF NOT EXISTS layouts_one_default_per_lang ON layouts(language_code) WHERE is_default = true;

-- Layout Blocks (partials)
CREATE TABLE IF NOT EXISTS layout_blocks (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    language_code VARCHAR(10) NOT NULL REFERENCES languages(code),
    template_code TEXT NOT NULL DEFAULT '',
    source VARCHAR(20) NOT NULL DEFAULT 'custom',
    theme_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(slug, language_code)
);

CREATE INDEX IF NOT EXISTS idx_layout_blocks_source_theme ON layout_blocks(source, theme_name);

-- Menus
CREATE TABLE IF NOT EXISTS menus (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    language_code VARCHAR(10) NOT NULL REFERENCES languages(code),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(slug, language_code)
);

-- Menu Items
CREATE TABLE IF NOT EXISTS menu_items (
    id SERIAL PRIMARY KEY,
    menu_id INT NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    parent_id INT REFERENCES menu_items(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    item_type VARCHAR(20) NOT NULL DEFAULT 'url',
    node_id INT REFERENCES content_nodes(id),
    url VARCHAR(2048),
    target VARCHAR(20) NOT NULL DEFAULT '_self',
    css_class VARCHAR(255),
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_menu_items_menu_sort ON menu_items(menu_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_menu_items_menu_parent ON menu_items(menu_id, parent_id);
CREATE INDEX IF NOT EXISTS idx_menu_items_node ON menu_items(node_id);

-- Add layout_id to content_nodes
ALTER TABLE content_nodes ADD COLUMN IF NOT EXISTS layout_id INT REFERENCES layouts(id) ON DELETE SET NULL;

-- Add theme fields to block_types
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS theme_name VARCHAR(100);
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS view_file VARCHAR(255);
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS block_css TEXT;
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS block_js TEXT;

-- Seed default layout for default language
INSERT INTO layouts (slug, name, description, language_code, template_code, source, is_default)
SELECT 'default', 'Default Layout', 'Default page layout', code,
'<!DOCTYPE html>
<html lang="{{.app.currentLang.Code}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.node.seo.title}}</title>
    {{range .app.headStyles}}<link rel="stylesheet" href="{{.}}">{{end}}
    {{range .app.headScripts}}<script src="{{.}}"></script>{{end}}
    {{.app.blockStyles}}
</head>
<body>
    <main>{{.node.blocks_html}}</main>
    {{range .app.footScripts}}<script src="{{.}}" defer></script>{{end}}
    {{.app.blockScripts}}
</body>
</html>',
'custom', true
FROM languages WHERE is_default = true
ON CONFLICT (slug, language_code) DO NOTHING;
```

- [ ] **Step 2: Verify migration runs**

Run: `cd /root/projects/vibecms && go run cmd/vibecms/main.go` (start and check logs for migration success)
Expected: No migration errors, tables created

- [ ] **Step 3: Commit**

```bash
git add internal/db/migrations/0008_layouts_menus.sql
git commit -m "feat: add migration for layouts, layout_blocks, menus tables"
```

---

## Task 2: Layout Model + Service + Handler

**Files:**
- Create: `internal/models/layout.go`
- Create: `internal/cms/layout_svc.go`
- Create: `internal/cms/layout_handler.go`
- Modify: `cmd/vibecms/main.go`

- [ ] **Step 1: Create Layout model**

Create `internal/models/layout.go`:

```go
package models

import "time"

type Layout struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Slug         string    `gorm:"column:slug;type:varchar(255);not null" json:"slug"`
	Name         string    `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description  string    `gorm:"column:description;type:text" json:"description"`
	LanguageCode string    `gorm:"column:language_code;type:varchar(10);not null" json:"language_code"`
	TemplateCode string    `gorm:"column:template_code;type:text;not null" json:"template_code"`
	Source       string    `gorm:"column:source;type:varchar(20);not null;default:'custom'" json:"source"`
	ThemeName    *string   `gorm:"column:theme_name;type:varchar(100)" json:"theme_name"`
	IsDefault    bool      `gorm:"column:is_default;type:boolean;not null;default:false" json:"is_default"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Layout) TableName() string {
	return "layouts"
}
```

- [ ] **Step 2: Create Layout service**

Create `internal/cms/layout_svc.go`. Follow the `block_type_svc.go` pattern exactly:

```go
package cms

import (
	"fmt"
	"sync"

	"vibecms/internal/models"

	"gorm.io/gorm"
)

type LayoutService struct {
	db    *gorm.DB
	cache sync.Map // key: "slug:lang" or "default:lang", value: *models.Layout
}

func NewLayoutService(db *gorm.DB) *LayoutService {
	return &LayoutService{db: db}
}

func (s *LayoutService) List(languageCode, source string) ([]models.Layout, error) {
	var layouts []models.Layout
	q := s.db.Order("name ASC")
	if languageCode != "" {
		q = q.Where("language_code = ?", languageCode)
	}
	if source != "" {
		q = q.Where("source = ?", source)
	}
	if err := q.Find(&layouts).Error; err != nil {
		return nil, fmt.Errorf("failed to list layouts: %w", err)
	}
	return layouts, nil
}

func (s *LayoutService) GetByID(id int) (*models.Layout, error) {
	var layout models.Layout
	if err := s.db.First(&layout, id).Error; err != nil {
		return nil, fmt.Errorf("layout not found: %w", err)
	}
	return &layout, nil
}

func (s *LayoutService) Create(layout *models.Layout) error {
	if layout.Slug == "" || layout.Name == "" {
		return fmt.Errorf("slug and name are required")
	}
	if layout.LanguageCode == "" {
		return fmt.Errorf("language_code is required")
	}
	// Check slug+lang uniqueness
	var count int64
	s.db.Model(&models.Layout{}).Where("slug = ? AND language_code = ?", layout.Slug, layout.LanguageCode).Count(&count)
	if count > 0 {
		return fmt.Errorf("SLUG_CONFLICT")
	}
	if err := s.db.Create(layout).Error; err != nil {
		return fmt.Errorf("failed to create layout: %w", err)
	}
	s.InvalidateCache()
	return nil
}

func (s *LayoutService) Update(id int, updates map[string]interface{}) (*models.Layout, error) {
	var layout models.Layout
	if err := s.db.First(&layout, id).Error; err != nil {
		return nil, fmt.Errorf("layout not found: %w", err)
	}
	if layout.Source == "theme" {
		return nil, fmt.Errorf("THEME_READONLY")
	}
	if slug, ok := updates["slug"]; ok {
		var count int64
		s.db.Model(&models.Layout{}).Where("slug = ? AND language_code = ? AND id != ?", slug, layout.LanguageCode, id).Count(&count)
		if count > 0 {
			return nil, fmt.Errorf("SLUG_CONFLICT")
		}
	}
	if err := s.db.Model(&layout).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update layout: %w", err)
	}
	s.InvalidateCache()
	// Re-fetch
	if err := s.db.First(&layout, id).Error; err != nil {
		return nil, fmt.Errorf("failed to re-fetch layout: %w", err)
	}
	return &layout, nil
}

func (s *LayoutService) Delete(id int) error {
	var layout models.Layout
	if err := s.db.First(&layout, id).Error; err != nil {
		return fmt.Errorf("layout not found: %w", err)
	}
	if layout.Source == "theme" {
		return fmt.Errorf("THEME_READONLY")
	}
	result := s.db.Delete(&layout)
	if result.RowsAffected == 0 {
		return fmt.Errorf("layout not found")
	}
	s.InvalidateCache()
	return nil
}

func (s *LayoutService) Detach(id int) (*models.Layout, error) {
	var layout models.Layout
	if err := s.db.First(&layout, id).Error; err != nil {
		return nil, fmt.Errorf("layout not found: %w", err)
	}
	if layout.Source != "theme" {
		return nil, fmt.Errorf("layout is already custom")
	}
	if err := s.db.Model(&layout).Updates(map[string]interface{}{
		"source":     "custom",
		"theme_name": nil,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to detach layout: %w", err)
	}
	s.InvalidateCache()
	if err := s.db.First(&layout, id).Error; err != nil {
		return nil, fmt.Errorf("failed to re-fetch layout: %w", err)
	}
	return &layout, nil
}

// ResolveForNode implements the template resolution cascade.
// 1. node.layout_id (explicit) → 2. layout-{type}-{slug} per lang → 3. layout-{type} per lang → 4. default per lang → 5. hardcoded
func (s *LayoutService) ResolveForNode(node *models.ContentNode, defaultLang string) (*models.Layout, error) {
	// Step 1: explicit layout_id
	if node.LayoutID != nil {
		layout, err := s.GetByID(*node.LayoutID)
		if err == nil {
			return layout, nil
		}
		// Fall through if not found
	}

	lang := node.LanguageCode
	nodeType := node.NodeType
	slug := node.Slug

	// Steps 2-7: cascade with language fallback
	candidates := []struct {
		slug      string
		isDefault bool
	}{
		{fmt.Sprintf("layout-%s-%s", nodeType, slug), false},
		{fmt.Sprintf("layout-%s", nodeType), false},
		{"", true}, // is_default=true
	}

	langs := []string{lang}
	if lang != defaultLang {
		langs = append(langs, defaultLang)
	}

	for _, c := range candidates {
		for _, l := range langs {
			cacheKey := fmt.Sprintf("%s:%s:%v", c.slug, l, c.isDefault)
			if cached, ok := s.cache.Load(cacheKey); ok {
				if cached == nil {
					continue // cached miss
				}
				layout := cached.(*models.Layout)
				return layout, nil
			}

			var layout models.Layout
			var err error
			if c.isDefault {
				err = s.db.Where("is_default = true AND language_code = ?", l).First(&layout).Error
			} else {
				err = s.db.Where("slug = ? AND language_code = ?", c.slug, l).First(&layout).Error
			}
			if err == nil {
				s.cache.Store(cacheKey, &layout)
				return &layout, nil
			}
			s.cache.Store(cacheKey, nil) // cache the miss
		}
	}

	return nil, fmt.Errorf("no layout found")
}

func (s *LayoutService) InvalidateCache() {
	s.cache = sync.Map{}
}
```

- [ ] **Step 3: Create Layout handler**

Create `internal/cms/layout_handler.go`. Follow `block_type_handler.go` pattern:

```go
package cms

import (
	"strconv"

	"vibecms/internal/api"

	"github.com/gofiber/fiber/v2"
)

type LayoutHandler struct {
	svc *LayoutService
}

func NewLayoutHandler(svc *LayoutService) *LayoutHandler {
	return &LayoutHandler{svc: svc}
}

func (h *LayoutHandler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/layouts")
	g.Get("/", h.List)
	g.Get("/:id", h.GetByID)
	g.Post("/", h.Create)
	g.Patch("/:id", h.Update)
	g.Delete("/:id", h.Delete)
	g.Post("/:id/detach", h.Detach)
}

type createLayoutRequest struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	LanguageCode string `json:"language_code"`
	TemplateCode string `json:"template_code"`
	IsDefault    bool   `json:"is_default"`
}

func (h *LayoutHandler) List(c *fiber.Ctx) error {
	lang := c.Query("language_code")
	source := c.Query("source")
	layouts, err := h.svc.List(lang, source)
	if err != nil {
		return api.Error(c, 500, "LIST_FAILED", err.Error())
	}
	return api.Success(c, layouts)
}

func (h *LayoutHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid layout ID")
	}
	layout, err := h.svc.GetByID(id)
	if err != nil {
		return api.Error(c, 404, "NOT_FOUND", "Layout not found")
	}
	return api.Success(c, layout)
}

func (h *LayoutHandler) Create(c *fiber.Ctx) error {
	var req createLayoutRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, 400, "INVALID_BODY", "Invalid request body")
	}
	if req.Slug == "" || req.Name == "" || req.LanguageCode == "" {
		return api.Error(c, 400, "VALIDATION_ERROR", "slug, name, and language_code are required")
	}
	layout := &models.Layout{
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		LanguageCode: req.LanguageCode,
		TemplateCode: req.TemplateCode,
		Source:       "custom",
		IsDefault:    req.IsDefault,
	}
	if err := h.svc.Create(layout); err != nil {
		if err.Error() == "SLUG_CONFLICT" {
			return api.Error(c, 409, "SLUG_CONFLICT", "A layout with this slug and language already exists")
		}
		return api.Error(c, 500, "CREATE_FAILED", err.Error())
	}
	return api.Created(c, layout)
}

func (h *LayoutHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid layout ID")
	}
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, 400, "INVALID_BODY", "Invalid request body")
	}
	layout, err := h.svc.Update(id, body)
	if err != nil {
		if err.Error() == "THEME_READONLY" {
			return api.Error(c, 403, "THEME_READONLY", "Theme layouts cannot be edited directly. Detach first.")
		}
		if err.Error() == "SLUG_CONFLICT" {
			return api.Error(c, 409, "SLUG_CONFLICT", "Slug already exists for this language")
		}
		return api.Error(c, 500, "UPDATE_FAILED", err.Error())
	}
	return api.Success(c, layout)
}

func (h *LayoutHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid layout ID")
	}
	if err := h.svc.Delete(id); err != nil {
		if err.Error() == "THEME_READONLY" {
			return api.Error(c, 403, "THEME_READONLY", "Theme layouts cannot be deleted")
		}
		return api.Error(c, 404, "NOT_FOUND", err.Error())
	}
	return api.Success(c, fiber.Map{"deleted": true})
}

func (h *LayoutHandler) Detach(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid layout ID")
	}
	layout, err := h.svc.Detach(id)
	if err != nil {
		return api.Error(c, 400, "DETACH_FAILED", err.Error())
	}
	return api.Success(c, layout)
}
```

Note: The import for `models` needs the full module path — check `go.mod` for the module name (likely `vibecms`). Adjust `vibecms/internal/models` and `vibecms/internal/api` to match.

- [ ] **Step 4: Wire into main.go**

In `cmd/vibecms/main.go`, add after existing service/handler creation:

```go
// Layout service + handler
layoutSvc := cms.NewLayoutService(database)
layoutHandler := cms.NewLayoutHandler(layoutSvc)

// In route registration (inside adminAPI group):
layoutHandler.RegisterRoutes(adminAPI)
```

- [ ] **Step 5: Verify compilation and test endpoints**

Run: `cd /root/projects/vibecms && go build ./cmd/vibecms/`
Expected: Compiles without errors

Start server and test:
```bash
# Create layout
curl -s -X POST http://localhost:8099/admin/api/layouts \
  -H "Content-Type: application/json" \
  -b "session=<token>" \
  -d '{"slug":"test","name":"Test Layout","language_code":"en","template_code":"<html>{{.node.blocks_html}}</html>"}' | jq .

# List layouts
curl -s http://localhost:8099/admin/api/layouts -b "session=<token>" | jq .
```

- [ ] **Step 6: Commit**

```bash
git add internal/models/layout.go internal/cms/layout_svc.go internal/cms/layout_handler.go cmd/vibecms/main.go
git commit -m "feat: add layout model, service, handler with cascade resolution and caching"
```

---

## Task 3: LayoutBlock Model + Service + Handler

**Files:**
- Create: `internal/models/layout_block.go`
- Create: `internal/cms/layout_block_svc.go`
- Create: `internal/cms/layout_block_handler.go`
- Modify: `cmd/vibecms/main.go`

- [ ] **Step 1: Create LayoutBlock model**

Create `internal/models/layout_block.go`:

```go
package models

import "time"

type LayoutBlock struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Slug         string    `gorm:"column:slug;type:varchar(255);not null" json:"slug"`
	Name         string    `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description  string    `gorm:"column:description;type:text" json:"description"`
	LanguageCode string    `gorm:"column:language_code;type:varchar(10);not null" json:"language_code"`
	TemplateCode string    `gorm:"column:template_code;type:text;not null" json:"template_code"`
	Source       string    `gorm:"column:source;type:varchar(20);not null;default:'custom'" json:"source"`
	ThemeName    *string   `gorm:"column:theme_name;type:varchar(100)" json:"theme_name"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (LayoutBlock) TableName() string {
	return "layout_blocks"
}
```

- [ ] **Step 2: Create LayoutBlock service**

Create `internal/cms/layout_block_svc.go`. Same pattern as `layout_svc.go` but simpler — no cascade resolution, just slug+language lookup with fallback:

```go
package cms

import (
	"fmt"
	"sync"

	"vibecms/internal/models"

	"gorm.io/gorm"
)

type LayoutBlockService struct {
	db    *gorm.DB
	cache sync.Map // key: "slug:lang", value: *models.LayoutBlock
}

func NewLayoutBlockService(db *gorm.DB) *LayoutBlockService {
	return &LayoutBlockService{db: db}
}

func (s *LayoutBlockService) List(languageCode, source string) ([]models.LayoutBlock, error) {
	var blocks []models.LayoutBlock
	q := s.db.Order("name ASC")
	if languageCode != "" {
		q = q.Where("language_code = ?", languageCode)
	}
	if source != "" {
		q = q.Where("source = ?", source)
	}
	if err := q.Find(&blocks).Error; err != nil {
		return nil, fmt.Errorf("failed to list layout blocks: %w", err)
	}
	return blocks, nil
}

func (s *LayoutBlockService) GetByID(id int) (*models.LayoutBlock, error) {
	var block models.LayoutBlock
	if err := s.db.First(&block, id).Error; err != nil {
		return nil, fmt.Errorf("layout block not found: %w", err)
	}
	return &block, nil
}

// Resolve finds a layout block by slug with language fallback.
func (s *LayoutBlockService) Resolve(slug, lang, defaultLang string) (*models.LayoutBlock, error) {
	langs := []string{lang}
	if lang != defaultLang {
		langs = append(langs, defaultLang)
	}
	for _, l := range langs {
		cacheKey := fmt.Sprintf("%s:%s", slug, l)
		if cached, ok := s.cache.Load(cacheKey); ok {
			if cached == nil {
				continue
			}
			return cached.(*models.LayoutBlock), nil
		}
		var block models.LayoutBlock
		if err := s.db.Where("slug = ? AND language_code = ?", slug, l).First(&block).Error; err == nil {
			s.cache.Store(cacheKey, &block)
			return &block, nil
		}
		s.cache.Store(cacheKey, nil)
	}
	return nil, fmt.Errorf("layout block '%s' not found", slug)
}

func (s *LayoutBlockService) Create(block *models.LayoutBlock) error {
	if block.Slug == "" || block.Name == "" || block.LanguageCode == "" {
		return fmt.Errorf("slug, name, and language_code are required")
	}
	var count int64
	s.db.Model(&models.LayoutBlock{}).Where("slug = ? AND language_code = ?", block.Slug, block.LanguageCode).Count(&count)
	if count > 0 {
		return fmt.Errorf("SLUG_CONFLICT")
	}
	if err := s.db.Create(block).Error; err != nil {
		return fmt.Errorf("failed to create layout block: %w", err)
	}
	s.InvalidateCache()
	return nil
}

func (s *LayoutBlockService) Update(id int, updates map[string]interface{}) (*models.LayoutBlock, error) {
	var block models.LayoutBlock
	if err := s.db.First(&block, id).Error; err != nil {
		return nil, fmt.Errorf("layout block not found: %w", err)
	}
	if block.Source == "theme" {
		return nil, fmt.Errorf("THEME_READONLY")
	}
	if slug, ok := updates["slug"]; ok {
		var count int64
		s.db.Model(&models.LayoutBlock{}).Where("slug = ? AND language_code = ? AND id != ?", slug, block.LanguageCode, id).Count(&count)
		if count > 0 {
			return nil, fmt.Errorf("SLUG_CONFLICT")
		}
	}
	if err := s.db.Model(&block).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update layout block: %w", err)
	}
	s.InvalidateCache()
	if err := s.db.First(&block, id).Error; err != nil {
		return nil, fmt.Errorf("failed to re-fetch: %w", err)
	}
	return &block, nil
}

func (s *LayoutBlockService) Delete(id int) error {
	var block models.LayoutBlock
	if err := s.db.First(&block, id).Error; err != nil {
		return fmt.Errorf("layout block not found: %w", err)
	}
	if block.Source == "theme" {
		return fmt.Errorf("THEME_READONLY")
	}
	s.db.Delete(&block)
	s.InvalidateCache()
	return nil
}

func (s *LayoutBlockService) Detach(id int) (*models.LayoutBlock, error) {
	var block models.LayoutBlock
	if err := s.db.First(&block, id).Error; err != nil {
		return nil, fmt.Errorf("layout block not found: %w", err)
	}
	if block.Source != "theme" {
		return nil, fmt.Errorf("already custom")
	}
	s.db.Model(&block).Updates(map[string]interface{}{"source": "custom", "theme_name": nil})
	s.InvalidateCache()
	s.db.First(&block, id)
	return &block, nil
}

func (s *LayoutBlockService) InvalidateCache() {
	s.cache = sync.Map{}
}
```

- [ ] **Step 3: Create LayoutBlock handler**

Create `internal/cms/layout_block_handler.go`. Same structure as `layout_handler.go` but routes under `/layout-blocks`:

```go
package cms

import (
	"strconv"

	"vibecms/internal/api"

	"github.com/gofiber/fiber/v2"
)

type LayoutBlockHandler struct {
	svc *LayoutBlockService
}

func NewLayoutBlockHandler(svc *LayoutBlockService) *LayoutBlockHandler {
	return &LayoutBlockHandler{svc: svc}
}

func (h *LayoutBlockHandler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/layout-blocks")
	g.Get("/", h.List)
	g.Get("/:id", h.GetByID)
	g.Post("/", h.Create)
	g.Patch("/:id", h.Update)
	g.Delete("/:id", h.Delete)
	g.Post("/:id/detach", h.Detach)
}

type createLayoutBlockRequest struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	LanguageCode string `json:"language_code"`
	TemplateCode string `json:"template_code"`
}

func (h *LayoutBlockHandler) List(c *fiber.Ctx) error {
	lang := c.Query("language_code")
	source := c.Query("source")
	blocks, err := h.svc.List(lang, source)
	if err != nil {
		return api.Error(c, 500, "LIST_FAILED", err.Error())
	}
	return api.Success(c, blocks)
}

func (h *LayoutBlockHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid ID")
	}
	block, err := h.svc.GetByID(id)
	if err != nil {
		return api.Error(c, 404, "NOT_FOUND", "Layout block not found")
	}
	return api.Success(c, block)
}

func (h *LayoutBlockHandler) Create(c *fiber.Ctx) error {
	var req createLayoutBlockRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, 400, "INVALID_BODY", "Invalid request body")
	}
	if req.Slug == "" || req.Name == "" || req.LanguageCode == "" {
		return api.Error(c, 400, "VALIDATION_ERROR", "slug, name, and language_code are required")
	}
	block := &models.LayoutBlock{
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		LanguageCode: req.LanguageCode,
		TemplateCode: req.TemplateCode,
		Source:       "custom",
	}
	if err := h.svc.Create(block); err != nil {
		if err.Error() == "SLUG_CONFLICT" {
			return api.Error(c, 409, "SLUG_CONFLICT", "Slug already exists for this language")
		}
		return api.Error(c, 500, "CREATE_FAILED", err.Error())
	}
	return api.Created(c, block)
}

func (h *LayoutBlockHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid ID")
	}
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, 400, "INVALID_BODY", "Invalid request body")
	}
	block, err := h.svc.Update(id, body)
	if err != nil {
		if err.Error() == "THEME_READONLY" {
			return api.Error(c, 403, "THEME_READONLY", "Theme layout blocks cannot be edited. Detach first.")
		}
		if err.Error() == "SLUG_CONFLICT" {
			return api.Error(c, 409, "SLUG_CONFLICT", "Slug already exists")
		}
		return api.Error(c, 500, "UPDATE_FAILED", err.Error())
	}
	return api.Success(c, block)
}

func (h *LayoutBlockHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid ID")
	}
	if err := h.svc.Delete(id); err != nil {
		if err.Error() == "THEME_READONLY" {
			return api.Error(c, 403, "THEME_READONLY", "Theme layout blocks cannot be deleted")
		}
		return api.Error(c, 404, "NOT_FOUND", err.Error())
	}
	return api.Success(c, fiber.Map{"deleted": true})
}

func (h *LayoutBlockHandler) Detach(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid ID")
	}
	block, err := h.svc.Detach(id)
	if err != nil {
		return api.Error(c, 400, "DETACH_FAILED", err.Error())
	}
	return api.Success(c, block)
}
```

- [ ] **Step 4: Wire into main.go**

Add to `cmd/vibecms/main.go`:

```go
layoutBlockSvc := cms.NewLayoutBlockService(database)
layoutBlockHandler := cms.NewLayoutBlockHandler(layoutBlockSvc)
layoutBlockHandler.RegisterRoutes(adminAPI)
```

- [ ] **Step 5: Verify compilation**

Run: `cd /root/projects/vibecms && go build ./cmd/vibecms/`
Expected: Compiles without errors

- [ ] **Step 6: Commit**

```bash
git add internal/models/layout_block.go internal/cms/layout_block_svc.go internal/cms/layout_block_handler.go cmd/vibecms/main.go
git commit -m "feat: add layout block (partial) model, service, handler"
```

---

## Task 4: Menu + MenuItem Model, Service, Handler

**Files:**
- Create: `internal/models/menu.go`
- Create: `internal/cms/menu_svc.go`
- Create: `internal/cms/menu_handler.go`
- Modify: `cmd/vibecms/main.go`

- [ ] **Step 1: Create Menu + MenuItem models**

Create `internal/models/menu.go`:

```go
package models

import "time"

type Menu struct {
	ID           int        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Slug         string     `gorm:"column:slug;type:varchar(255);not null" json:"slug"`
	Name         string     `gorm:"column:name;type:varchar(255);not null" json:"name"`
	LanguageCode string     `gorm:"column:language_code;type:varchar(10);not null" json:"language_code"`
	Version      int        `gorm:"column:version;type:int;not null;default:1" json:"version"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Items        []MenuItem `gorm:"-" json:"items,omitempty"`
}

func (Menu) TableName() string {
	return "menus"
}

type MenuItem struct {
	ID        int        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MenuID    int        `gorm:"column:menu_id;not null" json:"menu_id"`
	ParentID  *int       `gorm:"column:parent_id" json:"parent_id"`
	Title     string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	ItemType  string     `gorm:"column:item_type;type:varchar(20);not null;default:'url'" json:"item_type"`
	NodeID    *int       `gorm:"column:node_id" json:"node_id"`
	URL       string     `gorm:"column:url;type:varchar(2048)" json:"url"`
	Target    string     `gorm:"column:target;type:varchar(20);not null;default:'_self'" json:"target"`
	CSSClass  string     `gorm:"column:css_class;type:varchar(255)" json:"css_class"`
	SortOrder int        `gorm:"column:sort_order;type:int;not null;default:0" json:"sort_order"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Children  []MenuItem `gorm:"-" json:"children,omitempty"`
}

func (MenuItem) TableName() string {
	return "menu_items"
}

// MenuItemTree is the API input format for bulk saving menu items.
type MenuItemTree struct {
	Title    string         `json:"title"`
	ItemType string         `json:"item_type"`
	NodeID   *int           `json:"node_id"`
	URL      string         `json:"url"`
	Target   string         `json:"target"`
	CSSClass string         `json:"css_class"`
	Children []MenuItemTree `json:"children,omitempty"`
}
```

- [ ] **Step 2: Create Menu service**

Create `internal/cms/menu_svc.go`:

```go
package cms

import (
	"fmt"
	"sync"

	"vibecms/internal/models"

	"gorm.io/gorm"
)

type MenuService struct {
	db    *gorm.DB
	cache sync.Map // key: "slug:lang", value: *models.Menu (with nested items)
}

func NewMenuService(db *gorm.DB) *MenuService {
	return &MenuService{db: db}
}

func (s *MenuService) List(languageCode string) ([]models.Menu, error) {
	var menus []models.Menu
	q := s.db.Order("name ASC")
	if languageCode != "" {
		q = q.Where("language_code = ?", languageCode)
	}
	if err := q.Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("failed to list menus: %w", err)
	}
	return menus, nil
}

func (s *MenuService) GetByID(id int) (*models.Menu, error) {
	var menu models.Menu
	if err := s.db.First(&menu, id).Error; err != nil {
		return nil, fmt.Errorf("menu not found: %w", err)
	}
	// Load items as tree
	items, err := s.getItemsTree(menu.ID)
	if err != nil {
		return nil, err
	}
	menu.Items = items
	return &menu, nil
}

func (s *MenuService) getItemsTree(menuID int) ([]MenuItem, error) {
	var flat []models.MenuItem
	if err := s.db.Where("menu_id = ?", menuID).Order("sort_order ASC").Find(&flat).Error; err != nil {
		return nil, fmt.Errorf("failed to load menu items: %w", err)
	}
	return buildTree(flat), nil
}

// buildTree converts flat list with parent_id into nested tree.
func buildTree(flat []models.MenuItem) []models.MenuItem {
	byID := make(map[int]*models.MenuItem)
	var roots []models.MenuItem

	for i := range flat {
		flat[i].Children = []models.MenuItem{}
		byID[flat[i].ID] = &flat[i]
	}

	for i := range flat {
		if flat[i].ParentID != nil {
			if parent, ok := byID[*flat[i].ParentID]; ok {
				parent.Children = append(parent.Children, flat[i])
				continue
			}
		}
		roots = append(roots, flat[i])
	}

	// Update root children from byID (since we modified parent pointers)
	for i := range roots {
		if updated, ok := byID[roots[i].ID]; ok {
			roots[i].Children = updated.Children
		}
	}

	return roots
}

func (s *MenuService) Create(menu *models.Menu) error {
	if menu.Slug == "" || menu.Name == "" || menu.LanguageCode == "" {
		return fmt.Errorf("slug, name, and language_code are required")
	}
	var count int64
	s.db.Model(&models.Menu{}).Where("slug = ? AND language_code = ?", menu.Slug, menu.LanguageCode).Count(&count)
	if count > 0 {
		return fmt.Errorf("SLUG_CONFLICT")
	}
	if err := s.db.Create(menu).Error; err != nil {
		return fmt.Errorf("failed to create menu: %w", err)
	}
	s.InvalidateCache()
	return nil
}

func (s *MenuService) Update(id int, updates map[string]interface{}) (*models.Menu, error) {
	var menu models.Menu
	if err := s.db.First(&menu, id).Error; err != nil {
		return nil, fmt.Errorf("menu not found: %w", err)
	}
	if err := s.db.Model(&menu).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update menu: %w", err)
	}
	s.InvalidateCache()
	return s.GetByID(id)
}

func (s *MenuService) Delete(id int) error {
	result := s.db.Delete(&models.Menu{}, id)
	if result.RowsAffected == 0 {
		return fmt.Errorf("menu not found")
	}
	s.InvalidateCache()
	return nil
}

// ReplaceItems replaces the entire item tree with optimistic locking.
// clientVersion must match current DB version; returns 409-style error if not.
func (s *MenuService) ReplaceItems(menuID int, clientVersion int, tree []models.MenuItemTree) error {
	var menu models.Menu
	if err := s.db.First(&menu, menuID).Error; err != nil {
		return fmt.Errorf("menu not found: %w", err)
	}
	if menu.Version != clientVersion {
		return fmt.Errorf("VERSION_CONFLICT")
	}

	// Validate depth (max 3 levels: 0, 1, 2)
	if err := validateTreeDepth(tree, 0); err != nil {
		return err
	}

	// Transaction: delete old items, insert new, bump version
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete all existing items
		if err := tx.Where("menu_id = ?", menuID).Delete(&models.MenuItem{}).Error; err != nil {
			return fmt.Errorf("failed to delete old items: %w", err)
		}

		// Insert new items
		if err := insertItems(tx, menuID, nil, tree, 0); err != nil {
			return err
		}

		// Bump version
		if err := tx.Model(&menu).Update("version", menu.Version+1).Error; err != nil {
			return fmt.Errorf("failed to bump version: %w", err)
		}

		return nil
	})
}

func validateTreeDepth(items []models.MenuItemTree, depth int) error {
	if depth > 2 {
		return fmt.Errorf("DEPTH_EXCEEDED: max 3 levels (0-2)")
	}
	for _, item := range items {
		if err := validateTreeDepth(item.Children, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func insertItems(tx *gorm.DB, menuID int, parentID *int, items []models.MenuItemTree, sortStart int) error {
	for i, item := range items {
		mi := models.MenuItem{
			MenuID:    menuID,
			ParentID:  parentID,
			Title:     item.Title,
			ItemType:  item.ItemType,
			NodeID:    item.NodeID,
			URL:       item.URL,
			Target:    item.Target,
			CSSClass:  item.CSSClass,
			SortOrder: sortStart + i,
		}
		if mi.Target == "" {
			mi.Target = "_self"
		}
		if mi.ItemType == "" {
			mi.ItemType = "url"
		}
		if err := tx.Create(&mi).Error; err != nil {
			return fmt.Errorf("failed to insert menu item: %w", err)
		}
		if len(item.Children) > 0 {
			if err := insertItems(tx, menuID, &mi.ID, item.Children, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

// Resolve finds a menu by slug with language fallback and returns it with nested items.
func (s *MenuService) Resolve(slug, lang, defaultLang string) (*models.Menu, error) {
	langs := []string{lang}
	if lang != defaultLang {
		langs = append(langs, defaultLang)
	}
	for _, l := range langs {
		cacheKey := fmt.Sprintf("%s:%s", slug, l)
		if cached, ok := s.cache.Load(cacheKey); ok {
			if cached == nil {
				continue
			}
			return cached.(*models.Menu), nil
		}
		var menu models.Menu
		if err := s.db.Where("slug = ? AND language_code = ?", slug, l).First(&menu).Error; err == nil {
			items, _ := s.getItemsTree(menu.ID)
			menu.Items = items
			s.cache.Store(cacheKey, &menu)
			return &menu, nil
		}
		s.cache.Store(cacheKey, nil)
	}
	return nil, fmt.Errorf("menu '%s' not found", slug)
}

func (s *MenuService) InvalidateCache() {
	s.cache = sync.Map{}
}
```

- [ ] **Step 3: Create Menu handler**

Create `internal/cms/menu_handler.go`:

```go
package cms

import (
	"strconv"

	"vibecms/internal/api"
	"vibecms/internal/models"

	"github.com/gofiber/fiber/v2"
)

type MenuHandler struct {
	svc *MenuService
}

func NewMenuHandler(svc *MenuService) *MenuHandler {
	return &MenuHandler{svc: svc}
}

func (h *MenuHandler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/menus")
	g.Get("/", h.List)
	g.Get("/:id", h.GetByID)
	g.Post("/", h.Create)
	g.Patch("/:id", h.Update)
	g.Delete("/:id", h.Delete)
	g.Put("/:id/items", h.ReplaceItems)
}

type createMenuRequest struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	LanguageCode string `json:"language_code"`
}

type replaceItemsRequest struct {
	Version int                    `json:"version"`
	Items   []models.MenuItemTree  `json:"items"`
}

func (h *MenuHandler) List(c *fiber.Ctx) error {
	lang := c.Query("language_code")
	menus, err := h.svc.List(lang)
	if err != nil {
		return api.Error(c, 500, "LIST_FAILED", err.Error())
	}
	return api.Success(c, menus)
}

func (h *MenuHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid menu ID")
	}
	menu, err := h.svc.GetByID(id)
	if err != nil {
		return api.Error(c, 404, "NOT_FOUND", "Menu not found")
	}
	return api.Success(c, menu)
}

func (h *MenuHandler) Create(c *fiber.Ctx) error {
	var req createMenuRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, 400, "INVALID_BODY", "Invalid request body")
	}
	if req.Slug == "" || req.Name == "" || req.LanguageCode == "" {
		return api.Error(c, 400, "VALIDATION_ERROR", "slug, name, and language_code are required")
	}
	menu := &models.Menu{
		Slug:         req.Slug,
		Name:         req.Name,
		LanguageCode: req.LanguageCode,
	}
	if err := h.svc.Create(menu); err != nil {
		if err.Error() == "SLUG_CONFLICT" {
			return api.Error(c, 409, "SLUG_CONFLICT", "Menu with this slug and language already exists")
		}
		return api.Error(c, 500, "CREATE_FAILED", err.Error())
	}
	return api.Created(c, menu)
}

func (h *MenuHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid menu ID")
	}
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, 400, "INVALID_BODY", "Invalid request body")
	}
	// Don't allow version/items updates through PATCH
	delete(body, "version")
	delete(body, "items")
	menu, err := h.svc.Update(id, body)
	if err != nil {
		return api.Error(c, 500, "UPDATE_FAILED", err.Error())
	}
	return api.Success(c, menu)
}

func (h *MenuHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid menu ID")
	}
	if err := h.svc.Delete(id); err != nil {
		return api.Error(c, 404, "NOT_FOUND", err.Error())
	}
	return api.Success(c, fiber.Map{"deleted": true})
}

func (h *MenuHandler) ReplaceItems(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, 400, "INVALID_ID", "Invalid menu ID")
	}
	var req replaceItemsRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, 400, "INVALID_BODY", "Invalid request body")
	}
	if err := h.svc.ReplaceItems(id, req.Version, req.Items); err != nil {
		if err.Error() == "VERSION_CONFLICT" {
			return api.Error(c, 409, "VERSION_CONFLICT", "Menu was modified by another user. Refresh and try again.")
		}
		if err.Error() == "DEPTH_EXCEEDED: max 3 levels (0-2)" {
			return api.Error(c, 400, "DEPTH_EXCEEDED", "Menu items cannot nest more than 3 levels deep")
		}
		return api.Error(c, 500, "REPLACE_FAILED", err.Error())
	}
	// Return updated menu with new tree
	menu, _ := h.svc.GetByID(id)
	return api.Success(c, menu)
}
```

- [ ] **Step 4: Wire into main.go**

```go
menuSvc := cms.NewMenuService(database)
menuHandler := cms.NewMenuHandler(menuSvc)
menuHandler.RegisterRoutes(adminAPI)
```

- [ ] **Step 5: Verify compilation**

Run: `cd /root/projects/vibecms && go build ./cmd/vibecms/`
Expected: Compiles without errors

- [ ] **Step 6: Commit**

```bash
git add internal/models/menu.go internal/cms/menu_svc.go internal/cms/menu_handler.go cmd/vibecms/main.go
git commit -m "feat: add menu model, service, handler with tree operations and optimistic locking"
```

---

## Task 5: Render Context + Layout Rendering Pipeline

**Files:**
- Create: `internal/cms/render_context.go`
- Modify: `internal/rendering/template_renderer.go`
- Modify: `internal/cms/public_handler.go`
- Modify: `internal/models/content_node.go`

- [ ] **Step 1: Add LayoutID to ContentNode model**

In `internal/models/content_node.go`, add field to the struct:

```go
LayoutID *int `gorm:"column:layout_id" json:"layout_id"`
```

- [ ] **Step 2: Create render context builder**

Create `internal/cms/render_context.go`. This builds the `.app` and `.node` template data:

```go
package cms

import (
	"html/template"
	"log"

	"vibecms/internal/models"
)

// RenderContext builds the template data passed to layouts.
type RenderContext struct {
	layoutSvc      *LayoutService
	layoutBlockSvc *LayoutBlockService
	menuSvc        *MenuService
	db             interface{ /* for site_settings */ }
	themeAssets    *ThemeAssetRegistry
}

func NewRenderContext(layoutSvc *LayoutService, lbSvc *LayoutBlockService, menuSvc *MenuService, assets *ThemeAssetRegistry) *RenderContext {
	return &RenderContext{
		layoutSvc:      layoutSvc,
		layoutBlockSvc: lbSvc,
		menuSvc:        menuSvc,
		themeAssets:    assets,
	}
}

// AppData holds the .app namespace.
type AppData struct {
	Menus        map[string]interface{} `json:"menus"`
	Settings     map[string]string      `json:"settings"`
	Languages    []models.Language      `json:"languages"`
	CurrentLang  *models.Language        `json:"currentLang"`
	HeadStyles   []string               `json:"headStyles"`
	HeadScripts  []string               `json:"headScripts"`
	FootScripts  []string               `json:"footScripts"`
	BlockStyles  template.HTML          `json:"blockStyles"`
	BlockScripts template.HTML          `json:"blockScripts"`
}

// NodeData holds the .node namespace.
type NodeData struct {
	Title        string                 `json:"title"`
	Slug         string                 `json:"slug"`
	FullURL      string                 `json:"full_url"`
	BlocksHTML   template.HTML          `json:"blocks_html"`
	Fields       map[string]interface{} `json:"fields"`
	SEO          map[string]interface{} `json:"seo"`
	NodeType     string                 `json:"node_type"`
	LanguageCode string                 `json:"language_code"`
}

// TemplateData is the top-level data passed to layout templates.
type TemplateData struct {
	App  AppData  `json:"app"`
	Node NodeData `json:"node"`
}

// BuildAppData creates the .app namespace with menus, settings, assets.
func (rc *RenderContext) BuildAppData(lang, defaultLang string, siteSettings map[string]string, languages []models.Language, currentLang *models.Language, blockSlugs []string) AppData {
	app := AppData{
		Menus:       make(map[string]interface{}),
		Settings:    siteSettings,
		Languages:   languages,
		CurrentLang: currentLang,
	}

	// Load theme assets
	if rc.themeAssets != nil {
		app.HeadStyles = rc.themeAssets.GetHeadStyles()
		app.HeadScripts = rc.themeAssets.GetHeadScripts()
		app.FootScripts = rc.themeAssets.GetFootScripts()
	}

	// Build block-scoped inline CSS/JS
	var blockCSS, blockJS string
	for _, slug := range blockSlugs {
		if asset, ok := rc.themeAssets.GetBlockAssets(slug); ok {
			if asset.CSS != "" {
				blockCSS += "<style data-block=\"" + slug + "\">" + asset.CSS + "</style>\n"
			}
			if asset.JS != "" {
				blockJS += "<script data-block=\"" + slug + "\">" + asset.JS + "</script>\n"
			}
		}
	}
	app.BlockStyles = template.HTML(blockCSS)
	app.BlockScripts = template.HTML(blockJS)

	return app
}

// BuildNodeData creates the .node namespace from a content node.
func (rc *RenderContext) BuildNodeData(node *models.ContentNode, blocksHTML string) NodeData {
	fields := make(map[string]interface{})
	if node.FieldsData != nil {
		fields = node.FieldsData
	}

	seo := make(map[string]interface{})
	if node.SEOSettings != nil {
		seo = node.SEOSettings
	}

	return NodeData{
		Title:        node.Title,
		Slug:         node.Slug,
		FullURL:      node.FullURL,
		BlocksHTML:   template.HTML(blocksHTML),
		Fields:       fields,
		SEO:          seo,
		NodeType:     node.NodeType,
		LanguageCode: node.LanguageCode,
	}
}

// LoadMenus resolves all menus for the current language into .app.menus.
func (rc *RenderContext) LoadMenus(app *AppData, lang, defaultLang string) {
	// Load all menus for this language (or default)
	menus, err := rc.menuSvc.List("")
	if err != nil {
		log.Printf("WARN: failed to load menus: %v", err)
		return
	}
	// Get unique slugs
	slugs := make(map[string]bool)
	for _, m := range menus {
		slugs[m.Slug] = true
	}
	// Resolve each slug with language fallback
	for slug := range slugs {
		menu, err := rc.menuSvc.Resolve(slug, lang, defaultLang)
		if err != nil {
			continue
		}
		app.Menus[slug] = menu
	}
}
```

- [ ] **Step 3: Update template renderer with renderLayoutBlock**

In `internal/rendering/template_renderer.go`, add a new method and update the FuncMap to support layout rendering from DB template_code with `renderLayoutBlock`:

```go
// Add to TemplateRenderer:

// RenderLayout executes a layout's template_code string with the given data.
// It registers renderLayoutBlock as a template function with recursion guard.
func (r *TemplateRenderer) RenderLayout(w io.Writer, templateCode string, data interface{}, blockResolver func(slug string) (string, error)) error {
	depth := 0
	maxDepth := 5

	var renderBlock func(slug string) template.HTML
	renderBlock = func(slug string) template.HTML {
		depth++
		if depth > maxDepth {
			log.Printf("WARN: renderLayoutBlock max recursion depth (%d) exceeded for '%s'", maxDepth, slug)
			depth--
			return ""
		}
		code, err := blockResolver(slug)
		if err != nil {
			log.Printf("WARN: layout block '%s' not found: %v", slug, err)
			depth--
			return ""
		}
		// Parse and execute the partial
		funcMap := r.funcMap
		funcMap["renderLayoutBlock"] = renderBlock
		tmpl, err := template.New("partial-" + slug).Funcs(funcMap).Parse(code)
		if err != nil {
			log.Printf("WARN: template parse error in layout block '%s': %v", slug, err)
			depth--
			return ""
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			log.Printf("WARN: template execute error in layout block '%s': %v", slug, err)
			depth--
			return ""
		}
		depth--
		return template.HTML(buf.String())
	}

	funcMap := template.FuncMap{}
	for k, v := range r.funcMap {
		funcMap[k] = v
	}
	funcMap["renderLayoutBlock"] = renderBlock

	tmpl, err := template.New("layout").Funcs(funcMap).Parse(templateCode)
	if err != nil {
		return fmt.Errorf("layout template parse error: %w", err)
	}
	return tmpl.Execute(w, data)
}
```

Add required imports: `bytes`, `log`, `fmt`.

- [ ] **Step 4: Update public handler to use layout resolution**

Modify `internal/cms/public_handler.go` to:
1. Accept `LayoutService`, `LayoutBlockService`, `MenuService`, `RenderContext` in constructor
2. After rendering blocks HTML, resolve the layout via cascade
3. Build `TemplateData` with `.app` and `.node`
4. Execute layout template_code via `renderer.RenderLayout()`
5. The `blockResolver` function calls `layoutBlockSvc.Resolve(slug, lang, defaultLang)`

Key changes to the serve/render flow:

```go
// In the page handler, after renderBlocks():

// 1. Resolve layout
defaultLang := getDefaultLanguage(h.db) // query languages where is_default=true
layout, err := h.layoutSvc.ResolveForNode(node, defaultLang)
if err != nil {
	// Hardcoded fallback
	log.Printf("WARN: no layout found for node %d: %v", node.ID, err)
	// render with minimal HTML wrapping
}

// 2. Build template data
appData := h.renderCtx.BuildAppData(node.LanguageCode, defaultLang, siteSettings, languages, currentLang, blockSlugs)
h.renderCtx.LoadMenus(&appData, node.LanguageCode, defaultLang)
nodeData := h.renderCtx.BuildNodeData(node, renderedBlocksHTML)
templateData := TemplateData{App: appData, Node: nodeData}

// 3. Execute layout
blockResolver := func(slug string) (string, error) {
	block, err := h.layoutBlockSvc.Resolve(slug, node.LanguageCode, defaultLang)
	if err != nil {
		return "", err
	}
	return block.TemplateCode, nil
}

var buf bytes.Buffer
if err := h.renderer.RenderLayout(&buf, layout.TemplateCode, templateData, blockResolver); err != nil {
	log.Printf("ERROR: layout render failed: %v", err)
	// fallback rendering
}

c.Set("Content-Type", "text/html; charset=utf-8")
return c.Send(buf.Bytes())
```

- [ ] **Step 5: Update node_handler.go to accept layout_id**

In `internal/cms/node_handler.go`, add `LayoutID *int` to create/update request structs and pass through to the service.

- [ ] **Step 6: Verify compilation and test rendering**

Run: `cd /root/projects/vibecms && go build ./cmd/vibecms/`
Expected: Compiles. Visit a published page — should render with the seeded default layout.

- [ ] **Step 7: Commit**

```bash
git add internal/cms/render_context.go internal/rendering/template_renderer.go internal/cms/public_handler.go internal/models/content_node.go internal/cms/node_handler.go
git commit -m "feat: integrate layout rendering pipeline with cascade resolution and renderLayoutBlock"
```

---

## Task 6: Theme Loader + Asset Registry

**Files:**
- Create: `internal/cms/theme_loader.go`
- Modify: `cmd/vibecms/main.go`

- [ ] **Step 1: Create theme loader and asset registry**

Create `internal/cms/theme_loader.go`:

```go
package cms

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"vibecms/internal/models"

	"gorm.io/gorm"
)

// ThemeManifest represents theme.json
type ThemeManifest struct {
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Description string              `json:"description"`
	Author      string              `json:"author"`
	Styles      []ThemeAssetDef     `json:"styles"`
	Scripts     []ThemeAssetDef     `json:"scripts"`
	Layouts     []ThemeLayoutDef    `json:"layouts"`
	Partials    []ThemePartialDef   `json:"partials"`
	Blocks      []ThemeBlockDef     `json:"blocks"`
}

type ThemeAssetDef struct {
	Handle   string   `json:"handle"`
	Src      string   `json:"src"`
	Position string   `json:"position"` // "head" or "footer"
	Defer    bool     `json:"defer"`
	Deps     []string `json:"deps"`
}

type ThemeLayoutDef struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	File      string `json:"file"`
	IsDefault bool   `json:"is_default"`
}

type ThemePartialDef struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	File string `json:"file"`
}

type ThemeBlockDef struct {
	Slug string `json:"slug"`
	Dir  string `json:"dir"`
}

// ThemeAssetRegistry holds resolved theme assets in memory.
type ThemeAssetRegistry struct {
	mu          sync.RWMutex
	headStyles  []string // resolved URLs
	headScripts []string
	footScripts []string
	blockAssets map[string]*BlockAsset // slug -> CSS/JS
	themeDir    string
}

type BlockAsset struct {
	CSS string
	JS  string
}

func NewThemeAssetRegistry() *ThemeAssetRegistry {
	return &ThemeAssetRegistry{
		blockAssets: make(map[string]*BlockAsset),
	}
}

func (r *ThemeAssetRegistry) GetHeadStyles() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.headStyles
}

func (r *ThemeAssetRegistry) GetHeadScripts() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.headScripts
}

func (r *ThemeAssetRegistry) GetFootScripts() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.footScripts
}

func (r *ThemeAssetRegistry) GetBlockAssets(slug string) (*BlockAsset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.blockAssets[slug]
	return a, ok
}

// ThemeLoader loads a theme from disk and registers into DB + asset registry.
type ThemeLoader struct {
	db       *gorm.DB
	registry *ThemeAssetRegistry
}

func NewThemeLoader(db *gorm.DB, registry *ThemeAssetRegistry) *ThemeLoader {
	return &ThemeLoader{db: db, registry: registry}
}

// LoadTheme reads theme.json from themeDir and registers everything.
func (tl *ThemeLoader) LoadTheme(themeDir string) error {
	manifestPath := filepath.Join(themeDir, "theme.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		log.Printf("WARN: theme.json not found at %s: %v", manifestPath, err)
		return nil // soft-fail per CLAUDE.md
	}

	var manifest ThemeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		log.Printf("ERROR: failed to parse theme.json at %s: %v", manifestPath, err)
		return nil // soft-fail
	}

	themeName := manifest.Name
	tl.registry.themeDir = themeDir

	// Get default language
	var defaultLang models.Language
	if err := tl.db.Where("is_default = true").First(&defaultLang).Error; err != nil {
		log.Printf("WARN: no default language found, skipping theme registration")
		return nil
	}

	// Register assets
	tl.registerAssets(manifest, themeDir)

	// Register layouts
	for _, l := range manifest.Layouts {
		code, err := os.ReadFile(filepath.Join(themeDir, l.File))
		if err != nil {
			log.Printf("WARN: layout file not found: %s", l.File)
			continue
		}
		tl.upsertLayout(themeName, l, string(code), defaultLang.Code)
	}

	// Register partials
	for _, p := range manifest.Partials {
		code, err := os.ReadFile(filepath.Join(themeDir, p.File))
		if err != nil {
			log.Printf("WARN: partial file not found: %s", p.File)
			continue
		}
		tl.upsertPartial(themeName, p, string(code), defaultLang.Code)
	}

	// Register blocks
	for _, b := range manifest.Blocks {
		tl.registerBlock(themeName, b, themeDir)
	}

	log.Printf("Theme '%s' v%s loaded from %s", manifest.Name, manifest.Version, themeDir)
	return nil
}

func (tl *ThemeLoader) registerAssets(manifest ThemeManifest, themeDir string) {
	tl.registry.mu.Lock()
	defer tl.registry.mu.Unlock()

	tl.registry.headStyles = nil
	tl.registry.headScripts = nil
	tl.registry.footScripts = nil

	// Resolve dependency order for scripts
	scripts := resolveDeps(manifest.Scripts)

	for _, s := range manifest.Styles {
		url := "/theme/assets/" + s.Src
		tl.registry.headStyles = append(tl.registry.headStyles, url)
	}

	for _, s := range scripts {
		url := "/theme/assets/" + s.Src
		pos := s.Position
		if pos == "" {
			pos = "footer"
		}
		if pos == "head" {
			tl.registry.headScripts = append(tl.registry.headScripts, url)
		} else {
			tl.registry.footScripts = append(tl.registry.footScripts, url)
		}
	}
}

// resolveDeps performs topological sort on script dependencies.
func resolveDeps(scripts []ThemeAssetDef) []ThemeAssetDef {
	byHandle := make(map[string]ThemeAssetDef)
	for _, s := range scripts {
		byHandle[s.Handle] = s
	}

	visited := make(map[string]bool)
	var result []ThemeAssetDef

	var visit func(handle string)
	visit = func(handle string) {
		if visited[handle] {
			return
		}
		visited[handle] = true
		if s, ok := byHandle[handle]; ok {
			for _, dep := range s.Deps {
				visit(dep)
			}
			result = append(result, s)
		}
	}

	// Visit in original order for stability
	for _, s := range scripts {
		visit(s.Handle)
	}
	return result
}

func (tl *ThemeLoader) upsertLayout(themeName string, def ThemeLayoutDef, code, langCode string) {
	var existing models.Layout
	err := tl.db.Where("slug = ? AND language_code = ? AND source = 'theme'", def.Slug, langCode).First(&existing).Error
	if err == nil {
		// Update
		tl.db.Model(&existing).Updates(map[string]interface{}{
			"name":          def.Name,
			"template_code": code,
			"theme_name":    themeName,
			"is_default":    def.IsDefault,
		})
	} else {
		// Create
		layout := models.Layout{
			Slug:         def.Slug,
			Name:         def.Name,
			LanguageCode: langCode,
			TemplateCode: code,
			Source:       "theme",
			ThemeName:    &themeName,
			IsDefault:    def.IsDefault,
		}
		if err := tl.db.Create(&layout).Error; err != nil {
			log.Printf("WARN: failed to create layout '%s': %v", def.Slug, err)
		}
	}
}

func (tl *ThemeLoader) upsertPartial(themeName string, def ThemePartialDef, code, langCode string) {
	var existing models.LayoutBlock
	err := tl.db.Where("slug = ? AND language_code = ? AND source = 'theme'", def.Slug, langCode).First(&existing).Error
	if err == nil {
		tl.db.Model(&existing).Updates(map[string]interface{}{
			"name":          def.Name,
			"template_code": code,
			"theme_name":    themeName,
		})
	} else {
		block := models.LayoutBlock{
			Slug:         def.Slug,
			Name:         def.Name,
			LanguageCode: langCode,
			TemplateCode: code,
			Source:       "theme",
			ThemeName:    &themeName,
		}
		if err := tl.db.Create(&block).Error; err != nil {
			log.Printf("WARN: failed to create partial '%s': %v", def.Slug, err)
		}
	}
}

func (tl *ThemeLoader) registerBlock(themeName string, def ThemeBlockDef, themeDir string) {
	blockDir := filepath.Join(themeDir, def.Dir)

	// Read block.json
	schemaPath := filepath.Join(blockDir, "block.json")
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Printf("WARN: block.json not found for '%s'", def.Slug)
		return
	}

	// Read view.html
	viewPath := filepath.Join(blockDir, "view.html")
	viewData, err := os.ReadFile(viewPath)
	if err != nil {
		log.Printf("WARN: view.html not found for '%s'", def.Slug)
		return
	}

	// Read optional style.css and script.js
	var blockCSS, blockJS string
	if css, err := os.ReadFile(filepath.Join(blockDir, "style.css")); err == nil {
		blockCSS = string(css)
	}
	if js, err := os.ReadFile(filepath.Join(blockDir, "script.js")); err == nil {
		blockJS = string(js)
	}

	// Register block-scoped assets
	if blockCSS != "" || blockJS != "" {
		tl.registry.mu.Lock()
		tl.registry.blockAssets[def.Slug] = &BlockAsset{CSS: blockCSS, JS: blockJS}
		tl.registry.mu.Unlock()
	}

	// Upsert block_type in DB
	var existing models.BlockType
	err = tl.db.Where("slug = ? AND source = 'theme'", def.Slug).First(&existing).Error
	if err == nil {
		tl.db.Model(&existing).Updates(map[string]interface{}{
			"field_schema":  string(schemaData),
			"html_template": string(viewData),
			"theme_name":    themeName,
			"block_css":     blockCSS,
			"block_js":      blockJS,
		})
	} else {
		bt := models.BlockType{
			Slug:         def.Slug,
			Label:        def.Slug, // Use slug as label if not specified
			FieldSchema:  models.JSONB(schemaData),
			HTMLTemplate: string(viewData),
			Source:       "theme",
			ThemeName:    &themeName,
			BlockCSS:     blockCSS,
			BlockJS:      blockJS,
		}
		if err := tl.db.Create(&bt).Error; err != nil {
			log.Printf("WARN: failed to create block type '%s': %v", def.Slug, err)
		}
	}
}
```

Note: Adjust field names on `models.BlockType` to match existing struct fields. The new `BlockCSS`, `BlockJS`, `ThemeName` fields need to be added to the BlockType model.

- [ ] **Step 2: Add new fields to BlockType model**

In `internal/models/block_type.go`, add:

```go
ThemeName *string `gorm:"column:theme_name;type:varchar(100)" json:"theme_name"`
ViewFile  string  `gorm:"column:view_file;type:varchar(255)" json:"view_file"`
BlockCSS  string  `gorm:"column:block_css;type:text" json:"block_css"`
BlockJS   string  `gorm:"column:block_js;type:text" json:"block_js"`
```

- [ ] **Step 3: Wire theme loader into main.go**

In `cmd/vibecms/main.go`:

```go
// After DB init, before route registration:
themeAssets := cms.NewThemeAssetRegistry()
themeLoader := cms.NewThemeLoader(database, themeAssets)

// Load theme (configurable path, default to "themes/default")
themePath := os.Getenv("THEME_PATH")
if themePath == "" {
	themePath = "themes/default"
}
themeLoader.LoadTheme(themePath)

// Serve theme static assets
app.Static("/theme/assets", filepath.Join(themePath, "assets"))

// Pass themeAssets to RenderContext
renderCtx := cms.NewRenderContext(layoutSvc, layoutBlockSvc, menuSvc, themeAssets)
```

- [ ] **Step 4: Verify compilation**

Run: `cd /root/projects/vibecms && go build ./cmd/vibecms/`
Expected: Compiles without errors

- [ ] **Step 5: Commit**

```bash
git add internal/cms/theme_loader.go internal/models/block_type.go cmd/vibecms/main.go
git commit -m "feat: add theme loader with manifest parsing, asset registry, and block registration"
```

---

## Task 7: Admin UI — API Client + Routes

**Files:**
- Modify: `admin-ui/src/api/client.ts`
- Modify: `admin-ui/src/App.tsx` (or router config)

- [ ] **Step 1: Add TypeScript interfaces and API functions**

Add to `admin-ui/src/api/client.ts`:

```typescript
// Layout types
export interface Layout {
  id: number;
  slug: string;
  name: string;
  description: string;
  language_code: string;
  template_code: string;
  source: string;
  theme_name: string | null;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export async function getLayouts(params?: { language_code?: string; source?: string }): Promise<Layout[]> {
  const q = new URLSearchParams(params as Record<string, string>).toString();
  return api<Layout[]>(`/admin/api/layouts${q ? '?' + q : ''}`);
}
export async function getLayout(id: number | string): Promise<Layout> {
  return api<Layout>(`/admin/api/layouts/${id}`);
}
export async function createLayout(data: Partial<Layout>): Promise<Layout> {
  return api<Layout>('/admin/api/layouts', { method: 'POST', body: JSON.stringify(data) });
}
export async function updateLayout(id: number | string, data: Partial<Layout>): Promise<Layout> {
  return api<Layout>(`/admin/api/layouts/${id}`, { method: 'PATCH', body: JSON.stringify(data) });
}
export async function deleteLayout(id: number | string): Promise<void> {
  return api<void>(`/admin/api/layouts/${id}`, { method: 'DELETE' });
}
export async function detachLayout(id: number | string): Promise<Layout> {
  return api<Layout>(`/admin/api/layouts/${id}/detach`, { method: 'POST' });
}

// LayoutBlock types
export interface LayoutBlock {
  id: number;
  slug: string;
  name: string;
  description: string;
  language_code: string;
  template_code: string;
  source: string;
  theme_name: string | null;
  created_at: string;
  updated_at: string;
}

export async function getLayoutBlocks(params?: { language_code?: string; source?: string }): Promise<LayoutBlock[]> {
  const q = new URLSearchParams(params as Record<string, string>).toString();
  return api<LayoutBlock[]>(`/admin/api/layout-blocks${q ? '?' + q : ''}`);
}
export async function getLayoutBlock(id: number | string): Promise<LayoutBlock> {
  return api<LayoutBlock>(`/admin/api/layout-blocks/${id}`);
}
export async function createLayoutBlock(data: Partial<LayoutBlock>): Promise<LayoutBlock> {
  return api<LayoutBlock>('/admin/api/layout-blocks', { method: 'POST', body: JSON.stringify(data) });
}
export async function updateLayoutBlock(id: number | string, data: Partial<LayoutBlock>): Promise<LayoutBlock> {
  return api<LayoutBlock>(`/admin/api/layout-blocks/${id}`, { method: 'PATCH', body: JSON.stringify(data) });
}
export async function deleteLayoutBlock(id: number | string): Promise<void> {
  return api<void>(`/admin/api/layout-blocks/${id}`, { method: 'DELETE' });
}
export async function detachLayoutBlock(id: number | string): Promise<LayoutBlock> {
  return api<LayoutBlock>(`/admin/api/layout-blocks/${id}/detach`, { method: 'POST' });
}

// Menu types
export interface MenuItem {
  id?: number;
  title: string;
  item_type: 'node' | 'url' | 'anchor';
  node_id?: number | null;
  url?: string;
  target: string;
  css_class?: string;
  children?: MenuItem[];
}

export interface Menu {
  id: number;
  slug: string;
  name: string;
  language_code: string;
  version: number;
  items: MenuItem[];
  created_at: string;
  updated_at: string;
}

export async function getMenus(params?: { language_code?: string }): Promise<Menu[]> {
  const q = new URLSearchParams(params as Record<string, string>).toString();
  return api<Menu[]>(`/admin/api/menus${q ? '?' + q : ''}`);
}
export async function getMenu(id: number | string): Promise<Menu> {
  return api<Menu>(`/admin/api/menus/${id}`);
}
export async function createMenu(data: Partial<Menu>): Promise<Menu> {
  return api<Menu>('/admin/api/menus', { method: 'POST', body: JSON.stringify(data) });
}
export async function updateMenu(id: number | string, data: Partial<Menu>): Promise<Menu> {
  return api<Menu>(`/admin/api/menus/${id}`, { method: 'PATCH', body: JSON.stringify(data) });
}
export async function deleteMenu(id: number | string): Promise<void> {
  return api<void>(`/admin/api/menus/${id}`, { method: 'DELETE' });
}
export async function replaceMenuItems(id: number | string, version: number, items: MenuItem[]): Promise<Menu> {
  return api<Menu>(`/admin/api/menus/${id}/items`, {
    method: 'PUT',
    body: JSON.stringify({ version, items }),
  });
}
```

- [ ] **Step 2: Add routes to React router**

In the router config file (check `admin-ui/src/App.tsx` or equivalent), add:

```tsx
import LayoutsList from './pages/layouts-list';
import LayoutEditor from './pages/layout-editor';
import LayoutBlocksList from './pages/layout-blocks-list';
import LayoutBlockEditor from './pages/layout-block-editor';
import MenusList from './pages/menus-list';
import MenuEditor from './pages/menu-editor';

// Inside routes:
<Route path="/layouts" element={<LayoutsList />} />
<Route path="/layouts/new" element={<LayoutEditor />} />
<Route path="/layouts/:id" element={<LayoutEditor />} />
<Route path="/layout-blocks" element={<LayoutBlocksList />} />
<Route path="/layout-blocks/new" element={<LayoutBlockEditor />} />
<Route path="/layout-blocks/:id" element={<LayoutBlockEditor />} />
<Route path="/menus" element={<MenusList />} />
<Route path="/menus/new" element={<MenuEditor />} />
<Route path="/menus/:id" element={<MenuEditor />} />
```

Also add navigation sidebar links for Layouts, Layout Blocks, and Menus.

- [ ] **Step 3: Commit**

```bash
git add admin-ui/src/api/client.ts admin-ui/src/App.tsx
git commit -m "feat: add layout, layout block, menu API client and routes"
```

---

## Task 8: Admin UI — Layouts List + Editor Pages

**Files:**
- Create: `admin-ui/src/pages/layouts-list.tsx`
- Create: `admin-ui/src/pages/layout-editor.tsx`

- [ ] **Step 1: Create layouts list page**

Create `admin-ui/src/pages/layouts-list.tsx`. Follow the pattern from `block-types-list.tsx`:
- Table with columns: Name, Slug, Language, Source (badge), Default (badge), Actions
- Filter by language_code dropdown
- Delete button (disabled for source='theme')
- "New Layout" button
- Empty state

- [ ] **Step 2: Create layout editor page**

Create `admin-ui/src/pages/layout-editor.tsx`:
- Form fields: name, slug (auto-generated from name), description, language_code (select), is_default (checkbox)
- Monaco or CodeMirror editor for `template_code` with Go template syntax highlighting
- Reference panel (collapsible sidebar) showing available variables: `.app.menus`, `.app.settings`, `.node.title`, etc. and `{{renderLayoutBlock "slug"}}`
- Save / Delete buttons
- Read-only mode when source='theme', with "Detach" button to convert to custom
- Check if existing CodeEditor component exists in the codebase; reuse if so

- [ ] **Step 3: Verify pages render**

Run: `cd /root/projects/vibecms/admin-ui && npm run dev`
Navigate to `/admin/layouts` and `/admin/layouts/new`
Expected: Pages render, CRUD operations work

- [ ] **Step 4: Commit**

```bash
git add admin-ui/src/pages/layouts-list.tsx admin-ui/src/pages/layout-editor.tsx
git commit -m "feat: add layout list and editor admin pages with code editor"
```

---

## Task 9: Admin UI — Layout Blocks List + Editor Pages

**Files:**
- Create: `admin-ui/src/pages/layout-blocks-list.tsx`
- Create: `admin-ui/src/pages/layout-block-editor.tsx`

- [ ] **Step 1: Create layout blocks list page**

Same pattern as layouts list. Table: Name, Slug, Language, Source, Actions.

- [ ] **Step 2: Create layout block editor page**

Same pattern as layout editor: name, slug, description, language_code, template_code (code editor), source badge, detach button.

- [ ] **Step 3: Verify pages**

Navigate to `/admin/layout-blocks` and `/admin/layout-blocks/new`
Expected: CRUD works

- [ ] **Step 4: Commit**

```bash
git add admin-ui/src/pages/layout-blocks-list.tsx admin-ui/src/pages/layout-block-editor.tsx
git commit -m "feat: add layout block list and editor admin pages"
```

---

## Task 10: Admin UI — Menu Editor

**Files:**
- Create: `admin-ui/src/pages/menus-list.tsx`
- Create: `admin-ui/src/pages/menu-editor.tsx`
- Create: `admin-ui/src/components/menu-tree.tsx`

- [ ] **Step 1: Create menus list page**

Table: Name, Slug, Language, Item count, Actions. "New Menu" button.

- [ ] **Step 2: Create menu tree component**

Create `admin-ui/src/components/menu-tree.tsx`:
- Nested sortable list (use `@dnd-kit/sortable` or a similar React DnD library — check package.json for what's already installed)
- Each item shows: drag handle, title, item type badge, expand/collapse arrow
- Expanded item shows editable fields: title, item_type (select), URL (for 'url' type), node search (for 'node' type), target (select), css_class
- Indent for nesting (max 3 levels)
- "Add Item" dropdown: "Page/Post" (opens node search), "Custom URL", "Anchor"
- Drag to reorder and re-nest

- [ ] **Step 3: Create menu editor page**

Create `admin-ui/src/pages/menu-editor.tsx`:
- Header: menu name, slug, language_code
- Left panel: "Add Items" — node search with autocomplete (reuse existing node search if available), custom URL form, anchor form
- Right panel: MenuTree component
- Save button: calls `replaceMenuItems(id, version, items)` — handles 409 conflict with refresh prompt
- Version display for debugging

- [ ] **Step 4: Verify menu editor**

Navigate to `/admin/menus/new`, create a menu, add items, nest them, save.
Expected: Tree persists correctly with parent_id relationships.

- [ ] **Step 5: Commit**

```bash
git add admin-ui/src/pages/menus-list.tsx admin-ui/src/pages/menu-editor.tsx admin-ui/src/components/menu-tree.tsx
git commit -m "feat: add menu editor with drag-and-drop tree and submenu support"
```

---

## Task 11: Node Editor — Layout Picker

**Files:**
- Modify: `admin-ui/src/pages/node-editor.tsx` (or equivalent)

- [ ] **Step 1: Add layout picker to node editor**

In the existing node editor page, add a layout dropdown:
- Label: "Layout"
- Options: "Auto (cascade)" (value: null) + list from `getLayouts({ language_code: node.language_code })`
- Show layout name + source badge
- Save sends `layout_id` in the PATCH payload

- [ ] **Step 2: Verify**

Edit a node, select a layout, save. Verify the public page renders with the selected layout.

- [ ] **Step 3: Commit**

```bash
git add admin-ui/src/pages/node-editor.tsx
git commit -m "feat: add layout picker dropdown to node editor"
```

---

## Task 12: Integration Testing

- [ ] **Step 1: End-to-end flow test**

Manual test sequence:
1. Start server with `docker compose up` (or however it runs)
2. Create a layout via admin API
3. Create a layout block (header partial) via admin API
4. Create a menu with nested items via admin API
5. Create/publish a content node with blocks
6. Visit the public URL — verify layout wraps the page, partial renders, menu renders with submenus
7. Switch to a different language — verify fallback behavior
8. Assign a specific layout to a node — verify it overrides the cascade

- [ ] **Step 2: Test theme loading**

1. Create a `themes/default/theme.json` with a layout + partial + block + CSS/JS
2. Create the corresponding template files
3. Restart server
4. Verify layouts/partials appear in admin with source='theme'
5. Verify theme CSS loads on public pages
6. Verify block-scoped CSS only loads when that block is on the page

- [ ] **Step 3: Test error scenarios**

1. Delete a layout that's assigned to a node → node should fall back to cascade
2. Create a circular partial reference → should hit recursion guard, not crash
3. Edit menu items in two browser tabs → second save should get 409 conflict
4. Break theme.json syntax → server should start in custom-only mode

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete layout, template, menu system with theme support"
```
