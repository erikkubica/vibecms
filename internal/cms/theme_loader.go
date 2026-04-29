package cms

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"sync"

	"squilla/internal/events"
	"squilla/internal/models"

	"gorm.io/gorm"
)

// ThemeManifest represents the parsed theme.json file.
type ThemeManifest struct {
	Name        string               `json:"name"`
	Version     string               `json:"version"`
	Description string               `json:"description"`
	Author      string               `json:"author"`
	Styles      []ThemeAssetDef      `json:"styles"`
	Scripts     []ThemeAssetDef      `json:"scripts"`
	Layouts     []ThemeLayoutDef     `json:"layouts"`
	Partials    []ThemePartialDef    `json:"partials"`
	Blocks      []ThemeBlockDef      `json:"blocks"`
	Templates   []ThemeTemplateDef   `json:"templates"`
	Assets      []ThemeMediaAssetDef `json:"assets"`
	ImageSizes  []ThemeImageSizeDef  `json:"image_sizes"`
	// SettingsPages declares per-page setting schemas. Each entry references
	// a JSON file (relative to the theme directory) that follows the standard
	// field-schema format. Optional — themes without settings UI omit this.
	SettingsPages []ThemeSettingsPageDef `json:"settings_pages,omitempty"`
}

// ThemeSettingsPageDef declares a single settings page in theme.json.
// The actual field schema lives in the referenced JSON file so theme.json
// stays compact when many pages are declared.
type ThemeSettingsPageDef struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	File string `json:"file"`
	Icon string `json:"icon,omitempty"`
}

// ThemeImageSizeDef declares a named image size the theme depends on
// (e.g. `{name:"showcase-thumb", width:450, height:350, mode:"crop"}`).
// Carried in the theme.activated payload so the media-manager extension
// can upsert the row into media_image_sizes; URLs of the form
// /media/cache/<name>/<storage-path> then resolve.
type ThemeImageSizeDef struct {
	Name    string `json:"name"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Mode    string `json:"mode,omitempty"`    // "fit" (default) | "crop"
	Quality int    `json:"quality,omitempty"` // 0 = use site default
}

// ThemeMediaAssetDef defines an image or media asset shipped with the theme.
// When the theme is activated, the theme loader emits theme.activated with
// these assets in the payload (resolved abs_paths included). Extensions such
// as media-manager subscribe to that event and import the files into their
// own storage (media_files table) tagged source='theme'.
type ThemeMediaAssetDef struct {
	Key    string `json:"key"`
	Src    string `json:"src"`
	Alt    string `json:"alt,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// ThemeAssetDef defines a CSS or JS asset declared in theme.json.
type ThemeAssetDef struct {
	Handle   string   `json:"handle"`
	Src      string   `json:"src"`
	Position string   `json:"position"` // "head" or "footer", default "footer"
	Defer    bool     `json:"defer"`
	Deps     []string `json:"deps"`
}

// ThemeLayoutDef defines a layout declared in theme.json.
type ThemeLayoutDef struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	File      string `json:"file"`
	IsDefault bool   `json:"is_default"`
	// SupportsBlocks defaults to true when omitted. When explicitly false,
	// the admin node editor hides the blocks section for pages on this layout.
	SupportsBlocks *bool `json:"supports_blocks,omitempty"`
}

// ThemePartialDef defines a partial declared in theme.json.
type ThemePartialDef struct {
	Slug        string          `json:"slug"`
	Name        string          `json:"name"`
	File        string          `json:"file"`
	FieldSchema json.RawMessage `json:"field_schema"`
}

// ThemeBlockDef defines a block type declared in theme.json.
type ThemeBlockDef struct {
	Slug string `json:"slug"`
	Dir  string `json:"dir"`
}

// ThemeTemplateDef defines a page template declared in theme.json.
type ThemeTemplateDef struct {
	Slug string `json:"slug"`
	File string `json:"file"`
}

// BlockAsset holds scoped CSS and JS for a block type.
type BlockAsset struct {
	CSS string
	JS  string
}

// ThemeAssetRegistry is an in-memory store for resolved theme assets.
type ThemeAssetRegistry struct {
	mu          sync.RWMutex
	headStyles  []string
	headScripts []string
	footScripts []string
	blockAssets map[string]*BlockAsset
	themeDir    string
}

// NewThemeAssetRegistry creates a new ThemeAssetRegistry.
func NewThemeAssetRegistry() *ThemeAssetRegistry {
	return &ThemeAssetRegistry{
		blockAssets: make(map[string]*BlockAsset),
	}
}

// LoadBlockAssetsFromDB seeds the in-memory block asset registry from the
// block_types table. Disk-based sync (ThemeLoader, ExtensionLoader) skips the
// registry write when the content hash is unchanged, which leaves the registry
// empty on every restart after first install. Calling this at boot guarantees
// the registry reflects whatever block_css/block_js is currently persisted.
func (r *ThemeAssetRegistry) LoadBlockAssetsFromDB(db *gorm.DB) error {
	var bts []models.BlockType
	if err := db.Select("slug, block_css, block_js").Find(&bts).Error; err != nil {
		return fmt.Errorf("load block assets from db: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, bt := range bts {
		if bt.BlockCSS == "" && bt.BlockJS == "" {
			continue
		}
		r.blockAssets[bt.Slug] = &BlockAsset{
			CSS: bt.BlockCSS,
			JS:  bt.BlockJS,
		}
	}
	return nil
}

// GetHeadStyles returns the resolved head style URLs.
func (r *ThemeAssetRegistry) GetHeadStyles() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.headStyles))
	copy(out, r.headStyles)
	return out
}

// GetHeadScripts returns the resolved head script URLs.
func (r *ThemeAssetRegistry) GetHeadScripts() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.headScripts))
	copy(out, r.headScripts)
	return out
}

// GetFootScripts returns the resolved footer script URLs.
func (r *ThemeAssetRegistry) GetFootScripts() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.footScripts))
	copy(out, r.footScripts)
	return out
}

// GetBlockAssets returns the scoped CSS/JS for a block slug.
func (r *ThemeAssetRegistry) GetBlockAssets(slug string) (*BlockAsset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ba, ok := r.blockAssets[slug]
	return ba, ok
}

// BuildBlockStyleTags returns inline <style> tags for block assets.
// If usedSlugs is provided, only those blocks are included; otherwise all blocks.
func (r *ThemeAssetRegistry) BuildBlockStyleTags(usedSlugs ...[]string) template.HTML {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var sb strings.Builder

	if len(usedSlugs) > 0 && len(usedSlugs[0]) > 0 {
		// Only include CSS for blocks used on this page.
		seen := make(map[string]bool, len(usedSlugs[0]))
		for _, slug := range usedSlugs[0] {
			if seen[slug] {
				continue
			}
			seen[slug] = true
			if ba, ok := r.blockAssets[slug]; ok && ba.CSS != "" {
				sb.WriteString(fmt.Sprintf("<style data-block=\"%s\">\n%s\n</style>\n", slug, ba.CSS))
			}
		}
	} else {
		for slug, ba := range r.blockAssets {
			if ba.CSS != "" {
				sb.WriteString(fmt.Sprintf("<style data-block=\"%s\">\n%s\n</style>\n", slug, ba.CSS))
			}
		}
	}
	return template.HTML(sb.String())
}

// BuildBlockScriptTags returns inline <script> tags for block assets.
// If usedSlugs is provided, only those blocks are included; otherwise all blocks.
func (r *ThemeAssetRegistry) BuildBlockScriptTags(usedSlugs ...[]string) template.HTML {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var sb strings.Builder

	if len(usedSlugs) > 0 && len(usedSlugs[0]) > 0 {
		seen := make(map[string]bool, len(usedSlugs[0]))
		for _, slug := range usedSlugs[0] {
			if seen[slug] {
				continue
			}
			seen[slug] = true
			if ba, ok := r.blockAssets[slug]; ok && ba.JS != "" {
				sb.WriteString(fmt.Sprintf("<script data-block=\"%s\">\n%s\n</script>\n", slug, ba.JS))
			}
		}
	} else {
		for slug, ba := range r.blockAssets {
			if ba.JS != "" {
				sb.WriteString(fmt.Sprintf("<script data-block=\"%s\">\n%s\n</script>\n", slug, ba.JS))
			}
		}
	}
	return template.HTML(sb.String())
}

// ThemeLoader reads theme.json and registers layouts, partials, blocks, and assets.
type ThemeLoader struct {
	db               *gorm.DB
	registry         *ThemeAssetRegistry
	eventBus         *events.EventBus
	SettingsRegistry *ThemeSettingsRegistry
}
