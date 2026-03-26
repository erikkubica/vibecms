package models

import "time"

// User represents a CMS user account.
type User struct {
	ID           int        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Email        string     `gorm:"column:email;type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	RoleID       int        `gorm:"column:role_id;not null" json:"role_id"`
	Role         Role       `gorm:"foreignKey:RoleID;references:ID" json:"role,omitempty"`
	LanguageID   *int       `gorm:"column:language_id" json:"language_id,omitempty"`
	FullName     *string    `gorm:"column:full_name;type:varchar(100)" json:"full_name,omitempty"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at" json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (User) TableName() string {
	return "users"
}
