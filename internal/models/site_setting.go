package models

import "time"

// SiteSetting represents a key-value configuration entry. The composite
// primary key (key, language_code) lets a single setting key carry per-locale
// values; empty language_code ('') means "fallback / applies to all languages".
type SiteSetting struct {
	Key          string    `gorm:"column:key;type:varchar(100);primaryKey" json:"key"`
	LanguageCode string    `gorm:"column:language_code;type:varchar(8);primaryKey;default:''" json:"language_code"`
	Value        *string   `gorm:"column:value;type:text" json:"value,omitempty"`
	IsEncrypted  bool      `gorm:"column:is_encrypted;default:false" json:"is_encrypted"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (SiteSetting) TableName() string {
	return "site_settings"
}
