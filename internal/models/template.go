package models

import "time"

// Template represents a page template definition in the CMS.
type Template struct {
	ID          int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Slug        string    `gorm:"column:slug;type:varchar(50);uniqueIndex;not null" json:"slug"`
	Label       string    `gorm:"column:label;type:varchar(100);not null" json:"label"`
	Description string    `gorm:"column:description;type:text;not null;default:''" json:"description"`
	BlockConfig JSONB     `gorm:"column:block_config;type:jsonb;not null;default:'[]'" json:"block_config"`
	Source      string    `gorm:"column:source;type:varchar(20);not null;default:'custom'" json:"source"`
	ThemeName   *string   `gorm:"column:theme_name;type:varchar(100)" json:"theme_name"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (Template) TableName() string { return "templates" }
