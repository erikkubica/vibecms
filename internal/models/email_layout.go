package models

import "time"

// EmailLayout represents a base HTML wrapper applied to all outgoing emails.
// LanguageID is nullable — NULL means universal/fallback layout.
type EmailLayout struct {
	ID           int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"column:name;type:varchar(150);not null" json:"name"`
	LanguageID   *int      `gorm:"column:language_id" json:"language_id"`
	BodyTemplate string    `gorm:"column:body_template;type:text;not null" json:"body_template"`
	IsDefault    bool      `gorm:"column:is_default;not null;default:false" json:"is_default"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (EmailLayout) TableName() string {
	return "email_layouts"
}
