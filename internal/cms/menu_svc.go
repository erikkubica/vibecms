package cms

import (
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"

	"vibecms/internal/events"
	"vibecms/internal/models"
)

// MenuService provides business logic for managing menus and menu items.
type MenuService struct {
	db       *gorm.DB
	cache    sync.Map
	eventBus *events.EventBus
}

// NewMenuService creates a new MenuService with the given database connection.
func NewMenuService(db *gorm.DB, eventBus *events.EventBus) *MenuService {
	return &MenuService{db: db, eventBus: eventBus}
}

// List retrieves menus with an optional language_id filter.
func (s *MenuService) List(languageID *int) ([]models.Menu, error) {
	cacheKey := "list:all"
	if languageID != nil {
		cacheKey = fmt.Sprintf("list:%d", *languageID)
	}
	
	if cached, ok := s.cache.Load(cacheKey); ok {
		if cached != nil {
			return cached.([]models.Menu), nil
		}
	}

	var menus []models.Menu
	q := s.db.Order("name ASC")
	if languageID != nil {
		q = q.Where("language_id = ?", *languageID)
	}
	if err := q.Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("failed to list menus: %w", err)
	}
	
	s.cache.Store(cacheKey, menus)
	return menus, nil
}

// ListWithItems retrieves all matches for a language (or all) and populates their nested items tree.
// Employs only 2 database queries regardless of the number of menus.
func (s *MenuService) ListWithItems(languageID *int) ([]models.Menu, error) {
	cacheKey := "list-items:all"
	if languageID != nil {
		cacheKey = fmt.Sprintf("list-items:%d", *languageID)
	}

	if cached, ok := s.cache.Load(cacheKey); ok {
		if cached != nil {
			return cached.([]models.Menu), nil
		}
	}

	var menus []models.Menu
	q := s.db.Order("name ASC")
	if languageID != nil {
		q = q.Where("language_id = ?", *languageID).Or("language_id IS NULL")
	}
	if err := q.Find(&menus).Error; err != nil {
		return nil, err
	}

	if len(menus) == 0 {
		return nil, nil
	}

	menuIDs := make([]int, len(menus))
	for i, m := range menus {
		menuIDs[i] = m.ID
	}

	var allItems []models.MenuItem
	if err := s.db.Where("menu_id IN ?", menuIDs).Order("sort_order ASC").Find(&allItems).Error; err != nil {
		return nil, err
	}

	// Group items by menu_id
	itemsByMenu := make(map[int][]models.MenuItem)
	for _, item := range allItems {
		itemsByMenu[item.MenuID] = append(itemsByMenu[item.MenuID], item)
	}

	// Build trees for each menu
	for i := range menus {
		menus[i].Items = buildTree(itemsByMenu[menus[i].ID])
	}

	s.cache.Store(cacheKey, menus)
	return menus, nil
}

// GetByID retrieves a single menu by its ID, including nested items tree.
func (s *MenuService) GetByID(id int) (*models.Menu, error) {
	var menu models.Menu
	if err := s.db.First(&menu, id).Error; err != nil {
		return nil, err
	}

	var items []models.MenuItem
	if err := s.db.Where("menu_id = ?", id).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to load menu items: %w", err)
	}
	menu.Items = buildTree(items)

	return &menu, nil
}

// Create inserts a new menu after validating slug+language uniqueness.
func (s *MenuService) Create(menu *models.Menu) error {
	var count int64
	if menu.LanguageID != nil {
		s.db.Model(&models.Menu{}).Where("slug = ? AND language_id = ?", menu.Slug, *menu.LanguageID).Count(&count)
	} else {
		s.db.Model(&models.Menu{}).Where("slug = ? AND language_id IS NULL", menu.Slug).Count(&count)
	}
	if count > 0 {
		return fmt.Errorf("SLUG_CONFLICT")
	}

	if err := s.db.Create(menu).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return fmt.Errorf("SLUG_CONFLICT")
		}
		return fmt.Errorf("failed to create menu: %w", err)
	}

	s.InvalidateCache()

	if s.eventBus != nil {
		go s.eventBus.Publish("menu.created", events.Payload{
			"menu_id":   menu.ID,
			"menu_slug": menu.Slug,
			"menu_name": menu.Name,
		})
	}

	return nil
}

// Update performs a partial update on menu metadata by ID.
func (s *MenuService) Update(id int, updates map[string]interface{}) (*models.Menu, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate slug+language uniqueness if slug is being changed.
	if newSlug, ok := updates["slug"].(string); ok && newSlug != "" && newSlug != existing.Slug {
		langID := existing.LanguageID
		if lid, ok := updates["language_id"]; ok {
			if lid == nil {
				langID = nil
			} else if lidFloat, ok := lid.(float64); ok {
				lidInt := int(lidFloat)
				langID = &lidInt
			}
		}
		var count int64
		if langID != nil {
			s.db.Model(&models.Menu{}).Where("slug = ? AND language_id = ? AND id != ?", newSlug, *langID, id).Count(&count)
		} else {
			s.db.Model(&models.Menu{}).Where("slug = ? AND language_id IS NULL AND id != ?", newSlug, id).Count(&count)
		}
		if count > 0 {
			return nil, fmt.Errorf("SLUG_CONFLICT")
		}
	}

	if err := s.db.Model(&models.Menu{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return nil, fmt.Errorf("SLUG_CONFLICT")
		}
		return nil, fmt.Errorf("failed to update menu: %w", err)
	}

	s.InvalidateCache()

	updated, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if s.eventBus != nil {
		go s.eventBus.Publish("menu.updated", events.Payload{
			"menu_id":   updated.ID,
			"menu_slug": updated.Slug,
			"menu_name": updated.Name,
		})
	}

	return updated, nil
}

// Delete removes a menu and all its items by ID.
func (s *MenuService) Delete(id int) error {
	existing, err := s.GetByID(id)
	if err != nil {
		return err
	}
	_ = existing // used in event payload below

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("menu_id = ?", id).Delete(&models.MenuItem{}).Error; err != nil {
			return fmt.Errorf("failed to delete menu items: %w", err)
		}
		result := tx.Delete(&models.Menu{}, id)
		if result.Error != nil {
			return fmt.Errorf("failed to delete menu: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		s.InvalidateCache()

		if s.eventBus != nil {
			go s.eventBus.Publish("menu.deleted", events.Payload{
				"menu_id":   existing.ID,
				"menu_slug": existing.Slug,
				"menu_name": existing.Name,
			})
		}

		return nil
	})
}

// ReplaceItems atomically replaces all items in a menu with a new tree,
// using optimistic locking via the version field.
func (s *MenuService) ReplaceItems(menuID, clientVersion int, tree []models.MenuItemTree) error {
	// Fetch menu and check version.
	var menu models.Menu
	if err := s.db.First(&menu, menuID).Error; err != nil {
		return err
	}
	if menu.Version != clientVersion {
		return fmt.Errorf("VERSION_CONFLICT")
	}

	// Validate tree depth (max 3 levels: 0, 1, 2).
	if err := validateTreeDepth(tree, 0); err != nil {
		return err
	}

	// Transaction: delete old items, insert new tree, bump version.
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("menu_id = ?", menuID).Delete(&models.MenuItem{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing items: %w", err)
		}

		if err := insertItems(tx, menuID, nil, tree, 0); err != nil {
			return fmt.Errorf("failed to insert items: %w", err)
		}

		if err := tx.Model(&models.Menu{}).Where("id = ?", menuID).Updates(map[string]interface{}{
			"version": menu.Version + 1,
		}).Error; err != nil {
			return fmt.Errorf("failed to bump version: %w", err)
		}
		
		s.InvalidateCache()

		return nil
	})
}

// Resolve finds a menu by slug, trying the specific language_id first then NULL (all languages).
// Results are cached.
func (s *MenuService) Resolve(slug string, languageID *int) (*models.Menu, error) {
	type langQuery struct {
		id       *int
		cacheKey string
	}

	queries := []langQuery{}
	if languageID != nil {
		queries = append(queries, langQuery{id: languageID, cacheKey: fmt.Sprintf("resolve:%s:%d", slug, *languageID)})
	}
	queries = append(queries, langQuery{id: nil, cacheKey: fmt.Sprintf("resolve:%s:null", slug)})

	for _, q := range queries {
		if cached, ok := s.cache.Load(q.cacheKey); ok {
			if cached == nil {
				continue
			}
			return cached.(*models.Menu), nil
		}

		var menu models.Menu
		var err error
		if q.id != nil {
			err = s.db.Where("slug = ? AND language_id = ?", slug, *q.id).First(&menu).Error
		} else {
			err = s.db.Where("slug = ? AND language_id IS NULL", slug).First(&menu).Error
		}
		if err != nil {
			s.cache.Store(q.cacheKey, nil)
			continue
		}

		// Load items and build tree.
		var items []models.MenuItem
		if err := s.db.Where("menu_id = ?", menu.ID).Order("sort_order ASC").Find(&items).Error; err != nil {
			return nil, fmt.Errorf("failed to load menu items: %w", err)
		}
		menu.Items = buildTree(items)

		s.cache.Store(q.cacheKey, &menu)
		return &menu, nil
	}

	return nil, fmt.Errorf("menu not found for slug=%s", slug)
}

// InvalidateCache resets the entire menu cache.
func (s *MenuService) InvalidateCache() {
	s.cache.Range(func(key, value interface{}) bool {
		s.cache.Delete(key)
		return true
	})
}

// buildTree converts a flat list of MenuItem records into a nested tree structure.
// Root items have nil ParentID.
func buildTree(flat []models.MenuItem) []models.MenuItem {
	byID := make(map[int]*models.MenuItem, len(flat))
	var roots []models.MenuItem

	// Index all items by ID.
	for i := range flat {
		flat[i].Children = nil
		byID[flat[i].ID] = &flat[i]
	}

	// Assign children to their parents.
	for i := range flat {
		if flat[i].ParentID == nil {
			roots = append(roots, flat[i])
		} else {
			if parent, ok := byID[*flat[i].ParentID]; ok {
				parent.Children = append(parent.Children, flat[i])
			}
		}
	}

	// Update roots with populated children from the map.
	for i := range roots {
		if mapped, ok := byID[roots[i].ID]; ok {
			roots[i].Children = mapped.Children
		}
	}

	return roots
}

// validateTreeDepth ensures no branch exceeds maxDepth (3 levels: 0, 1, 2).
func validateTreeDepth(items []models.MenuItemTree, depth int) error {
	if depth > 2 {
		return fmt.Errorf("DEPTH_EXCEEDED: max 3 levels (0-2)")
	}
	for _, item := range items {
		if len(item.Children) > 0 {
			if err := validateTreeDepth(item.Children, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// insertItems recursively inserts MenuItemTree nodes into the database.
func insertItems(tx *gorm.DB, menuID int, parentID *int, items []models.MenuItemTree, sortStart int) error {
	for i, item := range items {
		target := item.Target
		if target == "" {
			target = "_self"
		}
		itemType := item.ItemType
		if itemType == "" {
			itemType = "custom"
		}

		mi := models.MenuItem{
			MenuID:    menuID,
			ParentID:  parentID,
			Title:     item.Title,
			ItemType:  itemType,
			NodeID:    item.NodeID,
			URL:       item.URL,
			Target:    target,
			CSSClass:  item.CSSClass,
			SortOrder: sortStart + i,
		}

		if err := tx.Create(&mi).Error; err != nil {
			return err
		}

		if len(item.Children) > 0 {
			if err := insertItems(tx, menuID, &mi.ID, item.Children, 0); err != nil {
				return err
			}
		}
	}
	return nil
}
