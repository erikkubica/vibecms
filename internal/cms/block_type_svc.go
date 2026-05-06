package cms

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"squilla/internal/events"
	"squilla/internal/models"
)

// BlockTypeService provides business logic for managing block types.
type BlockTypeService struct {
	db          *gorm.DB
	eventBus    *events.EventBus
	themeAssets *ThemeAssetRegistry
}

// NewBlockTypeService creates a new BlockTypeService with the given database connection.
func NewBlockTypeService(db *gorm.DB, eventBus *events.EventBus, themeAssets *ThemeAssetRegistry) *BlockTypeService {
	return &BlockTypeService{db: db, eventBus: eventBus, themeAssets: themeAssets}
}

// List retrieves a paginated list of block types ordered by label,
// excluding heavy fields (html_template, block_css, block_js).
func (s *BlockTypeService) List(page, perPage int) ([]models.BlockType, int64, error) {
	var total int64
	if err := s.db.Model(&models.BlockType{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting block types: %w", err)
	}

	var blockTypes []models.BlockType
	err := s.db.
		Select("id, slug, label, icon, description, field_schema, source, theme_name, view_file, test_data, created_at, updated_at").
		Order("label ASC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&blockTypes).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing block types: %w", err)
	}
	return blockTypes, total, nil
}

// ListAll retrieves all block types ordered by label (for internal use).
func (s *BlockTypeService) ListAll() ([]models.BlockType, error) {
	var blockTypes []models.BlockType
	if err := s.db.Order("label ASC").Find(&blockTypes).Error; err != nil {
		return nil, fmt.Errorf("listing block types: %w", err)
	}
	return blockTypes, nil
}

// GetByID retrieves a single block type by its ID.
func (s *BlockTypeService) GetByID(id int) (*models.BlockType, error) {
	var bt models.BlockType
	if err := s.db.First(&bt, id).Error; err != nil {
		return nil, err
	}
	return &bt, nil
}

// GetBySlug retrieves a single block type by its slug.
func (s *BlockTypeService) GetBySlug(slug string) (*models.BlockType, error) {
	var bt models.BlockType
	if err := s.db.Where("slug = ?", slug).First(&bt).Error; err != nil {
		return nil, err
	}
	return &bt, nil
}

// Create inserts a new block type after validating slug uniqueness.
func (s *BlockTypeService) Create(bt *models.BlockType) error {
	if bt.Slug == "" {
		return fmt.Errorf("validation error: slug is required")
	}
	if bt.Label == "" {
		return fmt.Errorf("validation error: label is required")
	}

	// Check slug uniqueness
	var count int64
	s.db.Model(&models.BlockType{}).Where("slug = ?", bt.Slug).Count(&count)
	if count > 0 {
		return fmt.Errorf("slug conflict: block type with slug %q already exists", bt.Slug)
	}

	if err := s.db.Create(bt).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return fmt.Errorf("slug conflict: block type with slug %q already exists", bt.Slug)
		}
		return fmt.Errorf("creating block type: %w", err)
	}

	if s.eventBus != nil {
		go s.eventBus.Publish("block_type.created", events.Payload{
			"block_type_id":    bt.ID,
			"block_type_slug":  bt.Slug,
			"block_type_label": bt.Label,
		})
	}

	return nil
}

// Update performs a partial update on a block type by ID.
func (s *BlockTypeService) Update(id int, updates map[string]interface{}) (*models.BlockType, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate slug uniqueness if slug is being changed
	if newSlug, ok := updates["slug"].(string); ok && newSlug != "" && newSlug != existing.Slug {
		var count int64
		s.db.Model(&models.BlockType{}).Where("slug = ? AND id != ?", newSlug, id).Count(&count)
		if count > 0 {
			return nil, fmt.Errorf("slug conflict: block type with slug %q already exists", newSlug)
		}
	}

	// Wire uses "fields", DB column is "field_schema" — GORM's Updates(map)
	// would otherwise try to write to a non-existent "fields" column.
	if val, ok := updates["fields"]; ok {
		delete(updates, "fields")
		updates["field_schema"] = val
	}

	// Convert JSONB fields from parsed JSON (map/slice) to models.JSONB
	for _, key := range []string{"field_schema", "test_data"} {
		if val, ok := updates[key]; ok && val != nil {
			b, err := json.Marshal(val)
			if err == nil {
				updates[key] = models.JSONB(b)
			}
		}
	}

	if err := s.db.Model(existing).Updates(updates).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			slug := updates["slug"]
			return nil, fmt.Errorf("slug conflict: block type with slug %q already exists", slug)
		}
		return nil, fmt.Errorf("updating block type: %w", err)
	}

	// Re-fetch updated block type
	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if s.eventBus != nil {
		go s.eventBus.Publish("block_type.updated", events.Payload{
			"block_type_id":    updated.ID,
			"block_type_slug":  updated.Slug,
			"block_type_label": updated.Label,
		})
	}

	return updated, nil
}

// Delete removes a block type by ID.
func (s *BlockTypeService) Delete(id int) error {
	existing, err := s.GetByID(id)
	if err != nil {
		return err
	}

	result := s.db.Delete(&models.BlockType{}, id)
	if result.Error != nil {
		return fmt.Errorf("deleting block type: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	if s.eventBus != nil {
		go s.eventBus.Publish("block_type.deleted", events.Payload{
			"block_type_id":    existing.ID,
			"block_type_slug":  existing.Slug,
			"block_type_label": existing.Label,
		})
	}

	return nil
}

// Detach converts a theme-sourced block type to custom.
func (s *BlockTypeService) Detach(id int) (*models.BlockType, error) {
	var existing models.BlockType
	if err := s.db.First(&existing, id).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&existing).Updates(map[string]interface{}{
		"source":     "custom",
		"theme_name": nil,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to detach block type: %w", err)
	}
	s.db.First(&existing, id)
	return &existing, nil
}

// Reattach restores a detached block type to its theme version.
func (s *BlockTypeService) Reattach(id int) (*models.BlockType, error) {
	var existing models.BlockType
	if err := s.db.First(&existing, id).Error; err != nil {
		return nil, err
	}
	if existing.Source == "theme" {
		return &existing, nil // already attached
	}

	if s.themeAssets == nil {
		return nil, fmt.Errorf("no theme loaded")
	}

	s.themeAssets.mu.RLock()
	themeDir := s.themeAssets.themeDir
	s.themeAssets.mu.RUnlock()

	if themeDir == "" {
		return nil, fmt.Errorf("no theme directory configured")
	}

	// Read theme.json to find the matching block definition.
	manifestData, err := os.ReadFile(filepath.Join(themeDir, "theme.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read theme.json: %w", err)
	}
	var manifest ThemeManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse theme.json: %w", err)
	}

	for _, def := range manifest.Blocks {
		if def.Slug == existing.Slug {
			blockDir := filepath.Join(themeDir, "blocks", def.Dir)

			// Read block.json.
			bjData, err := os.ReadFile(filepath.Join(blockDir, "block.json"))
			if err != nil {
				return nil, fmt.Errorf("failed to read block.json: %w", err)
			}
			var bm blockManifest
			if err := json.Unmarshal(bjData, &bm); err != nil {
				return nil, fmt.Errorf("failed to parse block.json: %w", err)
			}

			// Read view.html.
			viewData, err := os.ReadFile(filepath.Join(blockDir, "view.html"))
			if err != nil {
				return nil, fmt.Errorf("failed to read view.html: %w", err)
			}

			// Read optional style.css and script.js.
			var blockCSS, blockJS string
			if cssData, err := os.ReadFile(filepath.Join(blockDir, "style.css")); err == nil {
				blockCSS = string(cssData)
			}
			if jsData, err := os.ReadFile(filepath.Join(blockDir, "script.js")); err == nil {
				blockJS = string(jsData)
			}

			// Prepare field_schema and test_data.
			fieldSchema := models.JSONB("[]")
			if len(bm.Fields) > 0 {
				fieldSchema = models.JSONB(bm.Fields)
			}
			testData := models.JSONB("{}")
			if len(bm.TestData) > 0 {
				testData = models.JSONB(bm.TestData)
			}

			label := bm.Label
			if label == "" {
				label = def.Slug
			}
			icon := bm.Icon
			if icon == "" {
				icon = "square"
			}

			themeName := manifest.Name
			viewFile := filepath.Join("blocks", def.Dir, "view.html")

			if err := s.db.Model(&existing).Updates(map[string]interface{}{
				"label":         label,
				"icon":          icon,
				"description":   bm.Description,
				"field_schema":  fieldSchema,
				"html_template": string(viewData),
				"test_data":     testData,
				"source":        "theme",
				"theme_name":    &themeName,
				"view_file":     viewFile,
				"block_css":     blockCSS,
				"block_js":      blockJS,
			}).Error; err != nil {
				return nil, fmt.Errorf("failed to reattach block type: %w", err)
			}

			return s.GetByID(id)
		}
	}

	return nil, fmt.Errorf("block %q not found in theme", existing.Slug)
}
