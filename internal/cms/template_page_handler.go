package cms

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	"vibecms/internal/api"
)

// pageTemplateFile represents the JSON structure of a page template file.
type pageTemplateFile struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Thumbnail   string            `json:"thumbnail"`
	Blocks      []json.RawMessage `json:"blocks"`
}

// pageTemplateListItem is the response for the list endpoint.
type pageTemplateListItem struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
}

// pageTemplateDetail is the response for the get endpoint.
type pageTemplateDetail struct {
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Thumbnail   string            `json:"thumbnail"`
	Blocks      []json.RawMessage `json:"blocks"`
}

// PageTemplateHandler provides HTTP handlers for page templates defined in themes.
type PageTemplateHandler struct {
	themeAssets *ThemeAssetRegistry
}

// NewPageTemplateHandler creates a new PageTemplateHandler.
func NewPageTemplateHandler(themeAssets *ThemeAssetRegistry) *PageTemplateHandler {
	return &PageTemplateHandler{themeAssets: themeAssets}
}

// RegisterRoutes registers page template API routes on the provided router group.
func (h *PageTemplateHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/page-templates", h.List)
	router.Get("/page-templates/:slug", h.Get)
}

// loadManifest reads and parses the theme.json from the active theme directory.
func (h *PageTemplateHandler) loadManifest() (*ThemeManifest, string, error) {
	h.themeAssets.mu.RLock()
	themeDir := h.themeAssets.themeDir
	h.themeAssets.mu.RUnlock()

	if themeDir == "" {
		return nil, "", fiber.NewError(fiber.StatusInternalServerError, "no theme loaded")
	}

	data, err := os.ReadFile(filepath.Join(themeDir, "theme.json"))
	if err != nil {
		return nil, "", err
	}

	var manifest ThemeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, "", err
	}

	return &manifest, themeDir, nil
}

// readTemplateFile reads and parses a page template JSON file from the theme directory.
func (h *PageTemplateHandler) readTemplateFile(themeDir, file string) (*pageTemplateFile, error) {
	data, err := os.ReadFile(filepath.Join(themeDir, "templates", file))
	if err != nil {
		return nil, err
	}

	var tmpl pageTemplateFile
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}

	return &tmpl, nil
}

// List handles GET /admin/api/page-templates — returns all available page templates.
func (h *PageTemplateHandler) List(c *fiber.Ctx) error {
	manifest, themeDir, err := h.loadManifest()
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "MANIFEST_READ_FAILED", "Failed to read theme manifest")
	}

	items := make([]pageTemplateListItem, 0, len(manifest.Templates))
	for _, def := range manifest.Templates {
		tmpl, err := h.readTemplateFile(themeDir, def.File)
		if err != nil {
			continue // skip templates with missing/invalid files
		}
		items = append(items, pageTemplateListItem{
			Slug:        def.Slug,
			Name:        tmpl.Name,
			Description: tmpl.Description,
			Thumbnail:   tmpl.Thumbnail,
		})
	}

	return api.Success(c, items)
}

// Get handles GET /admin/api/page-templates/:slug — returns a specific template with blocks.
func (h *PageTemplateHandler) Get(c *fiber.Ctx) error {
	slug := c.Params("slug")

	manifest, themeDir, err := h.loadManifest()
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "MANIFEST_READ_FAILED", "Failed to read theme manifest")
	}

	// Find the template definition by slug.
	var found *ThemeTemplateDef
	for i := range manifest.Templates {
		if manifest.Templates[i].Slug == slug {
			found = &manifest.Templates[i]
			break
		}
	}

	if found == nil {
		return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Page template not found")
	}

	tmpl, err := h.readTemplateFile(themeDir, found.File)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "TEMPLATE_READ_FAILED", "Failed to read template file")
	}

	return api.Success(c, pageTemplateDetail{
		Slug:        found.Slug,
		Name:        tmpl.Name,
		Description: tmpl.Description,
		Thumbnail:   tmpl.Thumbnail,
		Blocks:      tmpl.Blocks,
	})
}
