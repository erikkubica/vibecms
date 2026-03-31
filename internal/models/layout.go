package models

import "time"

// Layout represents a page layout template in the CMS.
type Layout struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Slug         string    `gorm:"column:slug;type:varchar(255);not null" json:"slug"`
	Name         string    `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description  string    `gorm:"column:description;type:text" json:"description"`
	LanguageID   *int      `gorm:"column:language_id" json:"language_id"`
	TemplateCode string    `gorm:"column:template_code;type:text;not null" json:"template_code"`
	Source       string    `gorm:"column:source;type:varchar(20);not null;default:'custom'" json:"source"`
	ThemeName    *string   `gorm:"column:theme_name;type:varchar(100)" json:"theme_name"`
	IsDefault    bool      `gorm:"column:is_default;type:boolean;not null;default:false" json:"is_default"`
	ContentHash  string    `gorm:"column:content_hash;type:varchar(64);not null;default:''" json:"content_hash"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (Layout) TableName() string { return "layouts" }
