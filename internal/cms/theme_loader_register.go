package cms

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"squilla/internal/cms/field_types"
	"squilla/internal/models"

	"gorm.io/gorm"
)

// This file owns the standalone Register* helpers used by both
// the theme loader and the extension loader to upsert templates,
// layouts, partials, and block types into the database. Plus the
// per-theme private wrappers that delegate to those helpers.

// RegisterTemplateFromFile reads a template JSON file and upserts the template record.
// Shared helper used by both ThemeLoader and ExtensionLoader.
func RegisterTemplateFromFile(db *gorm.DB, filePath string, slug string, source string, sourceName string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("template file not found %s: %w", filePath, err)
	}

	var tmplFile themeTemplateFile
	if err := json.Unmarshal(data, &tmplFile); err != nil {
		return fmt.Errorf("failed to parse template %s: %w", slug, err)
	}

	// Convert {type, fields} to {block_type_slug, default_values}
	blockConfig := make([]map[string]interface{}, 0, len(tmplFile.Blocks))
	for _, b := range tmplFile.Blocks {
		blockConfig = append(blockConfig, map[string]interface{}{
			"block_type_slug": b.Type,
			"default_values":  b.Fields,
		})
	}

	configJSON, err := json.Marshal(blockConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal template block_config for %s: %w", slug, err)
	}

	label := tmplFile.Name
	if label == "" {
		label = slug
	}

	var sourceNamePtr *string
	if sourceName != "" {
		sourceNamePtr = &sourceName
	}

	h := sha256.New()
	h.Write(data)
	contentHash := hex.EncodeToString(h.Sum(nil))

	var existing models.Template
	result := db.Where("slug = ?", slug).First(&existing)

	if result.Error == nil {
		if existing.Source == "custom" {
			return nil
		}
		if existing.ContentHash == contentHash && existing.Source == source {
			return nil
		}
		existing.Label = label
		existing.Description = tmplFile.Description
		existing.BlockConfig = models.JSONB(configJSON)
		existing.Source = source
		existing.ThemeName = sourceNamePtr
		existing.ContentHash = contentHash
		if err := db.Save(&existing).Error; err != nil {
			return fmt.Errorf("failed to update template %s: %w", slug, err)
		}
	} else {
		tmpl := models.Template{
			Slug:        slug,
			Label:       label,
			Description: tmplFile.Description,
			BlockConfig: models.JSONB(configJSON),
			Source:      source,
			ThemeName:   sourceNamePtr,
			ContentHash: contentHash,
		}
		if err := db.Create(&tmpl).Error; err != nil {
			return fmt.Errorf("failed to create template %s: %w", slug, err)
		}
	}
	return nil
}

// registerTemplate reads a theme template file and upserts the template record.
func (tl *ThemeLoader) registerTemplate(themeName string, def ThemeTemplateDef, themeDir string) {
	filePath := filepath.Join(themeDir, "templates", def.File)
	if err := RegisterTemplateFromFile(tl.db, filePath, def.Slug, "theme", themeName); err != nil {
		log.Printf("WARN: %v", err)
	}
}

// registerAssets resolves dependency order and populates the registry.
// Resets any previously-registered theme assets so switching themes doesn't
// leak the old theme's stylesheets/scripts into the new active theme.
func (tl *ThemeLoader) registerAssets(manifest ThemeManifest) {
	tl.registry.mu.Lock()
	defer tl.registry.mu.Unlock()

	tl.registry.headStyles = tl.registry.headStyles[:0]
	tl.registry.headScripts = tl.registry.headScripts[:0]
	tl.registry.footScripts = tl.registry.footScripts[:0]

	// Resolve styles (styles typically don't have deps but support it).
	for _, s := range manifest.Styles {
		url := "/theme/assets/" + s.Src
		tl.registry.headStyles = append(tl.registry.headStyles, url)
	}

	// Resolve scripts with dependency ordering.
	sorted := tl.resolveDeps(manifest.Scripts)
	for _, s := range sorted {
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

// upsertLayout creates or updates a layout from a theme definition.
func (tl *ThemeLoader) upsertLayout(themeName string, def ThemeLayoutDef, code string) {
	if err := RegisterLayoutFromFile(tl.db, def, code, "theme", themeName); err != nil {
		log.Printf("WARN: %v", err)
	}
}

// upsertPartial creates or updates a layout block from a theme partial definition.
func (tl *ThemeLoader) upsertPartial(themeName string, def ThemePartialDef, code string) {
	if err := RegisterPartialFromFile(tl.db, def, code, "theme", themeName); err != nil {
		log.Printf("WARN: %v", err)
	}
}

// RegisterLayoutFromFile upserts a layout with hash-based change detection.
func RegisterLayoutFromFile(db *gorm.DB, def ThemeLayoutDef, code string, source string, sourceName string) error {
	h := sha256.New()
	h.Write([]byte(code))
	contentHash := hex.EncodeToString(h.Sum(nil))

	var sourceNamePtr *string
	if sourceName != "" {
		sourceNamePtr = &sourceName
	}

	var existing models.Layout
	result := db.Where("slug = ? AND language_id IS NULL", def.Slug).First(&existing)

	if result.Error == nil {
		// User-customized layouts are sacred — never overwrite.
		// Seed-created layouts (source "seed") are intentionally
		// overridable so themes can replace the placeholder template.
		if existing.Source == "custom" {
			return nil
		}
		if existing.ContentHash == contentHash && existing.Source == source {
			return nil
		}
		existing.Name = def.Name
		existing.TemplateCode = code
		existing.Source = source
		existing.ThemeName = sourceNamePtr
		existing.IsDefault = def.IsDefault
		if def.SupportsBlocks != nil {
			existing.SupportsBlocks = *def.SupportsBlocks
		}
		existing.ContentHash = contentHash
		if err := db.Save(&existing).Error; err != nil {
			return fmt.Errorf("failed to update layout %s: %w", def.Slug, err)
		}
	} else {
		supportsBlocks := true
		if def.SupportsBlocks != nil {
			supportsBlocks = *def.SupportsBlocks
		}
		layout := models.Layout{
			Slug:           def.Slug,
			Name:           def.Name,
			LanguageID:     nil,
			TemplateCode:   code,
			Source:         source,
			ThemeName:      sourceNamePtr,
			IsDefault:      def.IsDefault,
			SupportsBlocks: supportsBlocks,
			ContentHash:    contentHash,
		}
		if err := db.Create(&layout).Error; err != nil {
			return fmt.Errorf("failed to create layout %s: %w", def.Slug, err)
		}
	}
	return nil
}

// RegisterPartialFromFile upserts a layout block (partial) with hash-based change detection.
func RegisterPartialFromFile(db *gorm.DB, def ThemePartialDef, code string, source string, sourceName string) error {
	h := sha256.New()
	h.Write([]byte(code))
	h.Write(def.FieldSchema) // include field_schema in hash
	contentHash := hex.EncodeToString(h.Sum(nil))

	var sourceNamePtr *string
	if sourceName != "" {
		sourceNamePtr = &sourceName
	}

	// Prepare field_schema JSONB
	fieldSchema := models.JSONB("[]")
	if len(def.FieldSchema) > 0 {
		fieldSchema = models.JSONB(def.FieldSchema)
	}

	var existing models.LayoutBlock
	result := db.Where("slug = ? AND language_id IS NULL", def.Slug).First(&existing)

	if result.Error == nil {
		if existing.Source == "custom" {
			return nil
		}
		if existing.ContentHash == contentHash && existing.Source == source {
			return nil
		}
		existing.Name = def.Name
		existing.TemplateCode = code
		existing.FieldSchema = fieldSchema
		existing.Source = source
		existing.ThemeName = sourceNamePtr
		existing.ContentHash = contentHash
		if err := db.Save(&existing).Error; err != nil {
			return fmt.Errorf("failed to update partial %s: %w", def.Slug, err)
		}
	} else {
		lb := models.LayoutBlock{
			Slug:         def.Slug,
			Name:         def.Name,
			LanguageID:   nil,
			TemplateCode: code,
			FieldSchema:  fieldSchema,
			Source:       source,
			ThemeName:    sourceNamePtr,
			ContentHash:  contentHash,
		}
		if err := db.Create(&lb).Error; err != nil {
			return fmt.Errorf("failed to create partial %s: %w", def.Slug, err)
		}
	}
	return nil
}

// validateBlockFieldSchema enforces block.json invariants that, if violated,
// silently break the admin or render path. Today it catches:
//   - select-typed fields whose options contain {value,label} objects (admin
//     crashes with React error #31 — must be plain string options).
//   - term-typed fields without term_node_type (hydration won't match).
//   - mixing `name:` instead of `key:` at the field-schema level (block.json
//     readers expect `key`, admin will render empty inputs).
//
// Recurses into sub_fields for repeater/group fields.
func validateBlockFieldSchema(blockSlug string, raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var fields []map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		// Schema isn't an array — let the rest of the load surface that.
		return nil
	}
	return validateBlockFieldsRecursive(blockSlug, "", fields)
}

func validateBlockFieldsRecursive(blockSlug, parentPath string, fields []map[string]any) error {
	for _, f := range fields {
		key, _ := f["key"].(string)
		if key == "" {
			if name, ok := f["name"].(string); ok && name != "" {
				// Accept `name:` as a deprecated alias for `key:` — some
				// theme/block authors come from the Tengo nodetype convention
				// where `name` is the canonical field. Mirror it into `key`
				// so the rest of the load + admin renderer sees a consistent
				// shape, and warn so the author migrates.
				log.Printf("WARN: block %q field schema uses deprecated `name:` (=%q) — block.json should use `key:`. Mirroring into `key` for back-compat.",
					blockSlug, name)
				f["key"] = name
				key = name
			}
		}
		path := key
		if parentPath != "" {
			path = parentPath + "." + key
		}
		typ, _ := f["type"].(string)
		// Validate type against the kernel field-type registry. Extension-
		// contributed types (image/file/gallery from media-manager, etc.)
		// are not in the builtin list — log a warning rather than fail so a
		// theme that uses an extension-provided type still registers when
		// that extension is active. Common typos surface here:
		//   "boolean" → use "toggle", "dropdown" → use "select",
		//   "wysiwyg" → use "richtext".
		if typ != "" && !field_types.IsBuiltin(typ) {
			log.Printf("WARN: block %q field %q type=%q is not a built-in field type. If this is provided by an extension (e.g. image/file/gallery from media-manager), ensure that extension is active. Common typos: boolean→toggle, dropdown→select, wysiwyg→richtext.",
				blockSlug, path, typ)
		}
		switch typ {
		case "select", "radio":
			if opts, ok := f["options"].([]any); ok {
				for _, o := range opts {
					if _, isObj := o.(map[string]any); isObj {
						return fmt.Errorf("block %q field %q: type=%q options must be plain strings, not {value,label} objects (admin will crash with React error #31)",
							blockSlug, path, typ)
					}
				}
			}
		case "term":
			tnt, _ := f["term_node_type"].(string)
			if tnt == "" {
				log.Printf("WARN: block %q field %q is type=term but term_node_type is empty — hydration will not match any term row", blockSlug, path)
			}
		}
		if sub, ok := f["sub_fields"].([]any); ok {
			subFields := make([]map[string]any, 0, len(sub))
			for _, s := range sub {
				if sm, ok := s.(map[string]any); ok {
					subFields = append(subFields, sm)
				}
			}
			if err := validateBlockFieldsRecursive(blockSlug, path, subFields); err != nil {
				return err
			}
		}
	}
	return nil
}

// blockManifest is the structure of a block's block.json file.
type blockManifest struct {
	Slug        string          `json:"slug"`
	Label       string          `json:"label"`
	Icon        string          `json:"icon"`
	Description string          `json:"description"`
	FieldSchema json.RawMessage `json:"field_schema"`
	TestData    json.RawMessage `json:"test_data"`
}

// RegisterBlockFromDir reads block files from blockDir and upserts the block type.
// This is a shared helper used by both ThemeLoader and ExtensionLoader.
func RegisterBlockFromDir(db *gorm.DB, registry *ThemeAssetRegistry, blockDir string, slug string, source string, sourceName string) error {
	// Read block.json.
	bjData, err := os.ReadFile(filepath.Join(blockDir, "block.json"))
	if err != nil {
		return fmt.Errorf("block.json not found for block %s: %w", slug, err)
	}

	var bm blockManifest
	if err := json.Unmarshal(bjData, &bm); err != nil {
		return fmt.Errorf("failed to parse block.json for %s: %w", slug, err)
	}

	if err := validateBlockFieldSchema(slug, bm.FieldSchema); err != nil {
		return err
	}

	// Read view.html (the HTML template for the block).
	viewData, err := os.ReadFile(filepath.Join(blockDir, "view.html"))
	if err != nil {
		return fmt.Errorf("view.html not found for block %s: %w", slug, err)
	}

	// Read optional style.css and script.js.
	var blockCSS, blockJS string
	if cssData, err := os.ReadFile(filepath.Join(blockDir, "style.css")); err == nil {
		blockCSS = string(cssData)
	}
	if jsData, err := os.ReadFile(filepath.Join(blockDir, "script.js")); err == nil {
		blockJS = string(jsData)
	}

	// Prepare field_schema and test_data as JSONB.
	fieldSchema := models.JSONB("[]")
	if len(bm.FieldSchema) > 0 {
		fieldSchema = models.JSONB(bm.FieldSchema)
	}
	testData := models.JSONB("{}")
	if len(bm.TestData) > 0 {
		testData = models.JSONB(bm.TestData)
	}

	// Set defaults.
	label := bm.Label
	if label == "" {
		label = slug
	}
	icon := bm.Icon
	if icon == "" {
		icon = "square"
	}

	// Source name pointer (nil for "custom").
	var sourceNamePtr *string
	if sourceName != "" {
		sourceNamePtr = &sourceName
	}

	// Compute content hash from all block files.
	h := sha256.New()
	h.Write(bjData)
	h.Write(viewData)
	h.Write([]byte(blockCSS))
	h.Write([]byte(blockJS))
	contentHash := hex.EncodeToString(h.Sum(nil))

	// Upsert block type.
	var existing models.BlockType
	result := db.Where("slug = ?", slug).First(&existing)

	viewFile := filepath.Join("blocks", filepath.Base(blockDir), "view.html")

	if result.Error == nil {
		// Skip update if block is detached (custom) — user owns it now.
		if existing.Source == "custom" {
			return nil
		}

		// Skip update if content hasn't changed.
		if existing.ContentHash == contentHash && existing.Source == source {
			return nil
		}

		existing.Label = label
		existing.Icon = icon
		existing.Description = bm.Description
		existing.FieldSchema = fieldSchema
		existing.HTMLTemplate = string(viewData)
		existing.TestData = testData
		existing.Source = source
		existing.ThemeName = sourceNamePtr
		existing.ViewFile = viewFile
		existing.BlockCSS = blockCSS
		existing.BlockJS = blockJS
		existing.ContentHash = contentHash
		if err := db.Save(&existing).Error; err != nil {
			return fmt.Errorf("failed to update block_type %s: %w", slug, err)
		}
	} else {
		bt := models.BlockType{
			Slug:         slug,
			Label:        label,
			Icon:         icon,
			Description:  bm.Description,
			FieldSchema:  fieldSchema,
			HTMLTemplate: string(viewData),
			TestData:     testData,
			Source:       source,
			ThemeName:    sourceNamePtr,
			ViewFile:     viewFile,
			BlockCSS:     blockCSS,
			BlockJS:      blockJS,
			ContentHash:  contentHash,
		}
		if err := db.Create(&bt).Error; err != nil {
			return fmt.Errorf("failed to create block_type %s: %w", slug, err)
		}
	}

	// Store block assets in registry.
	if blockCSS != "" || blockJS != "" {
		registry.mu.Lock()
		registry.blockAssets[slug] = &BlockAsset{
			CSS: blockCSS,
			JS:  blockJS,
		}
		registry.mu.Unlock()
	}

	return nil
}

