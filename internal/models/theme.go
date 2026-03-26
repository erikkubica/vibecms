package models

import "time"

// Theme represents an installed CMS theme.
type Theme struct {
	ID          int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Slug        string    `gorm:"column:slug;type:varchar(100);uniqueIndex;not null" json:"slug"`
	Name        string    `gorm:"column:name;type:varchar(200);not null" json:"name"`
	Description string    `gorm:"column:description;type:text;not null;default:''" json:"description"`
	Version     string    `gorm:"column:version;type:varchar(50);not null;default:''" json:"version"`
	Author      string    `gorm:"column:author;type:varchar(200);not null;default:''" json:"author"`
	Source      string    `gorm:"column:source;type:varchar(20);not null;default:'upload'" json:"source"`
	GitURL      *string   `gorm:"column:git_url;type:text" json:"git_url"`
	GitBranch   string    `gorm:"column:git_branch;type:varchar(100);not null;default:'main'" json:"git_branch"`
	GitToken    *string   `gorm:"column:git_token;type:text" json:"-"`
	IsActive    bool      `gorm:"column:is_active;type:boolean;not null;default:false" json:"is_active"`
	Path        string    `gorm:"column:path;type:varchar(500);not null" json:"path"`
	Thumbnail   *string   `gorm:"column:thumbnail;type:varchar(500)" json:"thumbnail"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (Theme) TableName() string { return "themes" }
