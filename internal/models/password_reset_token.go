package models

import "time"

// PasswordResetToken records an in-flight password reset request. Only
// the SHA-256 hash of the raw token is stored — the raw value is sent
// once to the user's email and never persisted. UsedAt is set when the
// token is consumed by a successful reset; setting it (rather than
// deleting the row) lets us detect replay attempts and audit them.
type PasswordResetToken struct {
	ID        int        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    int        `gorm:"column:user_id;not null" json:"user_id"`
	User      User       `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	TokenHash string     `gorm:"column:token_hash;type:varchar(64);uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"column:expires_at;not null" json:"expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at" json:"used_at,omitempty"`
	IPAddress *string    `gorm:"column:ip_address;type:varchar(64)" json:"ip_address,omitempty"`
	UserAgent *string    `gorm:"column:user_agent;type:text" json:"user_agent,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}
