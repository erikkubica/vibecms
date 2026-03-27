package models

import "time"

// Extension represents an installed CMS extension (plugin).
type Extension struct {
	ID          int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Slug        string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"slug"`
	Name        string    `gorm:"type:varchar(150);not null" json:"name"`
	Version     string    `gorm:"type:varchar(50);not null;default:'1.0.0'" json:"version"`
	Description string    `gorm:"type:text;not null;default:''" json:"description"`
	Author      string    `gorm:"type:varchar(150);not null;default:''" json:"author"`
	Path        string    `gorm:"type:text;not null" json:"path"`
	IsActive    bool      `gorm:"not null;default:false" json:"is_active"`
	Priority    int       `gorm:"not null;default:50" json:"priority"`
	Settings    JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	Manifest    JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"manifest"`
	InstalledAt time.Time `gorm:"autoCreateTime" json:"installed_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (Extension) TableName() string { return "extensions" }
