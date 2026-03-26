package models

import "time"

// BlockType represents a block type definition in the CMS.
type BlockType struct {
	ID          int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Slug        string    `gorm:"column:slug;type:varchar(50);uniqueIndex;not null" json:"slug"`
	Label       string    `gorm:"column:label;type:varchar(100);not null" json:"label"`
	Icon        string    `gorm:"column:icon;type:varchar(50);not null;default:'square'" json:"icon"`
	Description string    `gorm:"column:description;type:text;not null;default:''" json:"description"`
	FieldSchema  JSONB     `gorm:"column:field_schema;type:jsonb;not null;default:'[]'" json:"field_schema"`
	HTMLTemplate string    `gorm:"column:html_template;type:text;not null;default:''" json:"html_template"`
	TestData     JSONB     `gorm:"column:test_data;type:jsonb;not null;default:'{}'" json:"test_data"`
	Source       string    `gorm:"column:source;type:varchar(20);not null;default:'custom'" json:"source"`
	ThemeName    *string   `gorm:"column:theme_name;type:varchar(100)" json:"theme_name"`
	ViewFile     string    `gorm:"column:view_file;type:varchar(255)" json:"view_file"`
	BlockCSS     string    `gorm:"column:block_css;type:text" json:"block_css"`
	BlockJS      string    `gorm:"column:block_js;type:text" json:"block_js"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (BlockType) TableName() string { return "block_types" }
