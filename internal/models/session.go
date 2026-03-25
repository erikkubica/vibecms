package models

import "time"

// Session represents an authenticated user session.
type Session struct {
	ID        string    `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    int       `gorm:"column:user_id;not null" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	TokenHash string    `gorm:"column:token_hash;type:varchar(255);uniqueIndex;not null" json:"-"`
	IPAddress *string   `gorm:"column:ip_address;type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent *string   `gorm:"column:user_agent;type:text" json:"user_agent,omitempty"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null" json:"expires_at"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (Session) TableName() string {
	return "sessions"
}
