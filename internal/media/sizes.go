package media

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// ImageSize represents a registered image size for on-the-fly resizing.
type ImageSize struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"uniqueIndex;size:50;not null"`
	Width     int       `json:"width" gorm:"not null"`
	Height    int       `json:"height" gorm:"not null"`
	Mode      string    `json:"mode" gorm:"size:20;not null;default:fit"` // "crop" | "fit" | "width"
	Source    string    `json:"source" gorm:"size:100;not null;default:default"`
	Quality   int       `json:"quality" gorm:"not null;default:0"` // 0 = use global default
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (ImageSize) TableName() string {
	return "media_image_sizes"
}

// SizeRegistry manages image sizes in memory with DB backing.
type SizeRegistry struct {
	mu    sync.RWMutex
	sizes map[string]ImageSize
	db    *gorm.DB
}

// DefaultSizes are seeded on first run when the table is empty.
var DefaultSizes = []ImageSize{
	{Name: "thumbnail", Width: 150, Height: 150, Mode: "crop", Source: "default"},
	{Name: "medium", Width: 250, Height: 250, Mode: "fit", Source: "default"},
	{Name: "large", Width: 500, Height: 500, Mode: "fit", Source: "default"},
}

// NewSizeRegistry creates a new SizeRegistry and auto-migrates the table.
func NewSizeRegistry(db *gorm.DB) *SizeRegistry {
	r := &SizeRegistry{
		sizes: make(map[string]ImageSize),
		db:    db,
	}
	// Auto-migrate the media_image_sizes table.
	_ = db.AutoMigrate(&ImageSize{})

	// Seed defaults if table is empty.
	var count int64
	db.Model(&ImageSize{}).Count(&count)
	if count == 0 {
		for _, s := range DefaultSizes {
			db.Create(&s)
		}
	}
	return r
}

// Load reads all sizes from the database into memory.
func (r *SizeRegistry) Load() error {
	var rows []ImageSize
	if err := r.db.Find(&rows).Error; err != nil {
		return fmt.Errorf("load image sizes: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.sizes = make(map[string]ImageSize, len(rows))
	for _, s := range rows {
		r.sizes[s.Name] = s
	}
	return nil
}

// Get returns a size by name.
func (r *SizeRegistry) Get(name string) (ImageSize, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sizes[name]
	return s, ok
}

// GetAll returns all registered sizes.
func (r *SizeRegistry) GetAll() []ImageSize {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]ImageSize, 0, len(r.sizes))
	for _, s := range r.sizes {
		out = append(out, s)
	}
	return out
}

// Register upserts a size to the database and in-memory cache.
func (r *SizeRegistry) Register(size ImageSize) error {
	// Validate mode.
	switch size.Mode {
	case "crop", "fit", "width":
	default:
		return fmt.Errorf("invalid mode: %s (must be crop, fit, or width)", size.Mode)
	}

	if size.Name == "" {
		return fmt.Errorf("size name is required")
	}
	if size.Width <= 0 && size.Mode != "fit" {
		return fmt.Errorf("width must be positive")
	}

	// Upsert: update on conflict by name.
	var existing ImageSize
	result := r.db.Where("name = ?", size.Name).First(&existing)
	if result.Error == nil {
		// Update existing.
		existing.Width = size.Width
		existing.Height = size.Height
		existing.Mode = size.Mode
		existing.Source = size.Source
		existing.Quality = size.Quality
		if err := r.db.Save(&existing).Error; err != nil {
			return fmt.Errorf("update image size: %w", err)
		}
		size = existing
	} else {
		// Create new.
		if err := r.db.Create(&size).Error; err != nil {
			return fmt.Errorf("create image size: %w", err)
		}
	}

	r.mu.Lock()
	r.sizes[size.Name] = size
	r.mu.Unlock()

	return nil
}

// Delete removes a size from the database and in-memory cache.
func (r *SizeRegistry) Delete(name string) error {
	if err := r.db.Where("name = ?", name).Delete(&ImageSize{}).Error; err != nil {
		return fmt.Errorf("delete image size: %w", err)
	}

	r.mu.Lock()
	delete(r.sizes, name)
	r.mu.Unlock()

	return nil
}
