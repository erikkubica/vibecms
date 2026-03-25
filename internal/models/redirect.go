package models

import "time"

// Redirect represents a URL redirect rule.
type Redirect struct {
	ID        int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OldURL    string    `gorm:"column:old_url;type:text;uniqueIndex;not null" json:"old_url"`
	NewURL    string    `gorm:"column:new_url;type:text;not null" json:"new_url"`
	HTTPCode  int       `gorm:"column:http_code;not null;default:301" json:"http_code"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (Redirect) TableName() string {
	return "redirects"
}
