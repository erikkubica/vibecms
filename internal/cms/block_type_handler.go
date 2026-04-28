package cms

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"vibecms/internal/api"
	"vibecms/internal/auth"
	"vibecms/internal/models"
)

// activeThemeChrome extracts the <head> inner HTML and the <body class="...">
// attribute from the active theme's default.html layout, so the preview iframe
// can render blocks in a frame that looks like the live site. Template
// directives referencing runtime-only data (.app.*, .node.*) are stripped —
// then the runtime head_styles / block_styles from the theme asset registry
// are substituted in, matching what the live renderer emits.
func (h *BlockTypeHandler) activeThemeChrome() (head, bodyClass string) {
	var theme models.Theme
	if err := h.db.Where("is_active = ?", true).First(&theme).Error; err != nil || theme.Path == "" {
		return "", ""
	}
	b, err := os.ReadFile(filepath.Join(theme.Path, "layouts", "default.html"))
	if err != nil {
		return "", ""
	}
	src := string(b)
	if m := regexp.MustCompile(`(?is)<head[^>]*>(.*?)</head>`).FindStringSubmatch(src); len(m) == 2 {
		head = m[1]
	}
	if m := regexp.MustCompile(`(?is)<body([^>]*)>`).FindStringSubmatch(src); len(m) == 2 {
		if c := regexp.MustCompile(`class\s*=\s*"([^"]*)"`).FindStringSubmatch(m[1]); len(c) == 2 {
			bodyClass = c[1]
		}
	}
	head = regexp.MustCompile(`(?s)\{\{[-]?.*?[-]?\}\}`).ReplaceAllString(head, "")
	if h.themeAssets != nil {
		var sb strings.Builder
		for _, href := range h.themeAssets.GetHeadStyles() {
			sb.WriteString(`<link rel="stylesheet" href="`)
			sb.WriteString(template.HTMLEscapeString(href))
			sb.WriteString(`">`)
		}
		sb.WriteString(string(h.themeAssets.BuildBlockStyleTags()))
		head += sb.String()
	}
	return head, bodyClass
}

// BlockTypeHandler provides HTTP handlers for block type CRUD operations.
type BlockTypeHandler struct {
	svc         *BlockTypeService
	db          *gorm.DB
	themeAssets *ThemeAssetRegistry
}

// NewBlockTypeHandler creates a new BlockTypeHandler with the given BlockTypeService.
func NewBlockTypeHandler(svc *BlockTypeService, db *gorm.DB) *BlockTypeHandler {
	return &BlockTypeHandler{svc: svc, db: db}
}

// SetThemeAssets wires the theme asset registry so the preview endpoint can
// emit the same head_styles + block_styles the live renderer uses.
func (h *BlockTypeHandler) SetThemeAssets(r *ThemeAssetRegistry) {
	h.themeAssets = r
}

// resolveTestDataSlice replaces any "theme-asset:<key>" or
// "extension-asset:<slug>:<key>" references inside test_data with full media
// objects. No-op when the media-manager extension hasn't run its ownership
// migrations (columns missing → empty lookup). Mutates by index so
// value-typed slices take effect.
func (h *BlockTypeHandler) resolveTestDataSlice(bts []models.BlockType) {
	if h.db == nil || len(bts) == 0 {
		return
	}
	lookup := LoadAssetLookup(h.db, ActiveThemeName(h.db))
	if lookup.Empty() {
		return
	}
	for i := range bts {
		if len(bts[i].TestData) == 0 {
			continue
		}
		resolved := ResolveThemeAssetRefsInJSON([]byte(bts[i].TestData), lookup)
		bts[i].TestData = models.JSONB(resolved)
	}
}

// resolveTestDataOne resolves a single block type in place.
func (h *BlockTypeHandler) resolveTestDataOne(bt *models.BlockType) {
	if h.db == nil || bt == nil || len(bt.TestData) == 0 {
		return
	}
	lookup := LoadAssetLookup(h.db, ActiveThemeName(h.db))
	if lookup.Empty() {
		return
	}
	bt.TestData = models.JSONB(ResolveThemeAssetRefsInJSON([]byte(bt.TestData), lookup))
}

// RegisterRoutes registers all block type routes on the provided router group.
// Reads + preview are open to authenticated users; mutations require manage_layouts.
func (h *BlockTypeHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/block-types", h.List)
	router.Post("/block-types/preview", h.PreviewBlockTemplate)
	router.Get("/block-types/:id", h.Get)
	manage := auth.CapabilityRequired("manage_layouts")
	router.Post("/block-types", manage, h.Create)
	router.Patch("/block-types/:id", manage, h.Update)
	router.Delete("/block-types/:id", manage, h.Delete)
	router.Post("/block-types/:id/detach", manage, h.Detach)
	router.Post("/block-types/:id/reattach", manage, h.Reattach)
}

// List handles GET /block-types to retrieve paginated block types.
func (h *BlockTypeHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 1000 {
		perPage = 50
	}

	blockTypes, total, err := h.svc.List(page, perPage)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", "Failed to list block types")
	}

	h.resolveTestDataSlice(blockTypes)
	return api.Paginated(c, blockTypes, total, page, perPage)
}

// Get handles GET /block-types/:id to retrieve a single block type.
func (h *BlockTypeHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Block type ID must be a valid integer")
	}

	bt, err := h.svc.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Block type not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch block type")
	}

	h.resolveTestDataOne(bt)
	return api.Success(c, bt)
}

// createBlockTypeRequest represents the JSON body for creating a block type.
type createBlockTypeRequest struct {
	Slug         string       `json:"slug"`
	Label        string       `json:"label"`
	Icon         string       `json:"icon"`
	Description  string       `json:"description"`
	FieldSchema  models.JSONB `json:"field_schema"`
	HTMLTemplate string       `json:"html_template"`
	Source       string       `json:"source"`
}

// Create handles POST /block-types to create a new block type.
func (h *BlockTypeHandler) Create(c *fiber.Ctx) error {
	var req createBlockTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	if req.Slug == "" {
		return api.ValidationError(c, map[string]string{
			"slug": "Slug is required",
		})
	}
	if req.Label == "" {
		return api.ValidationError(c, map[string]string{
			"label": "Label is required",
		})
	}

	bt := models.BlockType{
		Slug:         req.Slug,
		Label:        req.Label,
		Icon:         req.Icon,
		Description:  req.Description,
		FieldSchema:  req.FieldSchema,
		HTMLTemplate: req.HTMLTemplate,
		Source:       req.Source,
	}

	if bt.Icon == "" {
		bt.Icon = "square"
	}
	if len(bt.FieldSchema) == 0 {
		bt.FieldSchema = models.JSONB("[]")
	}
	if bt.Source == "" {
		bt.Source = "custom"
	}

	if err := h.svc.Create(&bt); err != nil {
		if strings.Contains(err.Error(), "slug conflict") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", err.Error())
		}
		if strings.Contains(err.Error(), "validation error") {
			return api.ValidationError(c, map[string]string{
				"slug": err.Error(),
			})
		}
		return api.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create block type")
	}

	return api.Created(c, bt)
}

// Update handles PATCH /block-types/:id to partially update a block type.
func (h *BlockTypeHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Block type ID must be a valid integer")
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}

	// Remove fields that should not be directly updated
	delete(body, "id")
	delete(body, "created_at")
	delete(body, "updated_at")

	if len(body) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "NO_UPDATES", "No valid fields to update")
	}

	updated, err := h.svc.Update(id, body)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Block type not found")
		}
		if strings.Contains(err.Error(), "slug conflict") {
			return api.Error(c, fiber.StatusConflict, "SLUG_CONFLICT", err.Error())
		}
		return api.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update block type")
	}

	return api.Success(c, updated)
}

// PreviewBlockTemplate renders a block template with test data for live preview.
func (h *BlockTypeHandler) PreviewBlockTemplate(c *fiber.Ctx) error {
	var req struct {
		HTMLTemplate string                 `json:"html_template"`
		TestData     map[string]interface{} `json:"test_data"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.HTMLTemplate == "" {
		return c.JSON(fiber.Map{"html": ""})
	}

	if req.TestData == nil {
		req.TestData = make(map[string]interface{})
	}

	// Mark any values containing HTML tags as template.HTML to prevent escaping
	for k, v := range req.TestData {
		if s, ok := v.(string); ok && strings.Contains(s, "<") {
			req.TestData[k] = template.HTML(s)
		}
	}

	tmpl, err := template.New("preview").Funcs(template.FuncMap{
		"safeHTML": func(s interface{}) template.HTML {
			return template.HTML(fmt.Sprintf("%v", s))
		},
		"raw": func(s interface{}) template.HTML {
			return template.HTML(fmt.Sprintf("%v", s))
		},
		"safeURL": func(s interface{}) template.URL {
			return template.URL(fmt.Sprintf("%v", s))
		},
	}).Parse(req.HTMLTemplate)
	if err != nil {
		return c.JSON(fiber.Map{"html": fmt.Sprintf("<div class=\"text-red-500 text-sm p-3 bg-red-50 rounded-lg border border-red-200\"><strong>Template Error:</strong> %s</div>", template.HTMLEscapeString(err.Error()))})
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, req.TestData); err != nil {
		return c.JSON(fiber.Map{"html": fmt.Sprintf("<div class=\"text-red-500 text-sm p-3 bg-red-50 rounded-lg border border-red-200\"><strong>Render Error:</strong> %s</div>", template.HTMLEscapeString(err.Error()))})
	}

	head, bodyClass := h.activeThemeChrome()
	return c.JSON(fiber.Map{"html": buf.String(), "head": head, "body_class": bodyClass})
}

// Delete handles DELETE /block-types/:id to remove a block type.
func (h *BlockTypeHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Block type ID must be a valid integer")
	}

	if err := h.svc.Delete(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Block type not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete block type")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Detach handles POST /block-types/:id/detach to convert a theme block type to custom.
func (h *BlockTypeHandler) Detach(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Block type ID must be a valid integer")
	}

	bt, err := h.svc.Detach(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Block type not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "DETACH_FAILED", "Failed to detach block type")
	}

	return api.Success(c, bt)
}

// Reattach handles POST /block-types/:id/reattach to restore a block type to its theme version.
func (h *BlockTypeHandler) Reattach(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Block type ID must be a valid integer")
	}

	bt, err := h.svc.Reattach(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Block type not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "REATTACH_FAILED", "Failed to reattach block type")
	}

	return api.Success(c, bt)
}
