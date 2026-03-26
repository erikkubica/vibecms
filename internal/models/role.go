package models

import "time"

// Role represents a user role with capabilities.
type Role struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Slug         string    `gorm:"column:slug;type:varchar(50);uniqueIndex;not null" json:"slug"`
	Name         string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description  string    `gorm:"column:description;type:text" json:"description"`
	IsSystem     bool      `gorm:"column:is_system;not null;default:false" json:"is_system"`
	Capabilities JSONB     `gorm:"column:capabilities;type:jsonb;not null;default:'{}'" json:"capabilities"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (Role) TableName() string {
	return "roles"
}
