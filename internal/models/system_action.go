package models

import "time"

// SystemAction represents a named action that can trigger email rules.
type SystemAction struct {
	ID            int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Slug          string    `gorm:"column:slug;type:varchar(100);uniqueIndex;not null" json:"slug"`
	Label         string    `gorm:"column:label;type:varchar(150);not null" json:"label"`
	Category      string    `gorm:"column:category;type:varchar(50);not null" json:"category"`
	Description   string    `gorm:"column:description;type:text" json:"description"`
	PayloadSchema JSONB     `gorm:"column:payload_schema;type:jsonb;not null;default:'{}'" json:"payload_schema"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (SystemAction) TableName() string {
	return "system_actions"
}
