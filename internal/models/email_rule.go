package models

import "time"

// EmailRule maps a system action to an email template and recipient.
type EmailRule struct {
	ID             int           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Action         string        `gorm:"column:action;type:varchar(100);not null" json:"action"`
	NodeType       *string       `gorm:"column:node_type;type:varchar(50)" json:"node_type,omitempty"`
	TemplateID     int           `gorm:"column:template_id;not null" json:"template_id"`
	Template       EmailTemplate `gorm:"foreignKey:TemplateID;references:ID" json:"template,omitempty"`
	RecipientType  string        `gorm:"column:recipient_type;type:varchar(20);not null" json:"recipient_type"`
	RecipientValue string        `gorm:"column:recipient_value;type:varchar(500);not null" json:"recipient_value"`
	Enabled        bool          `gorm:"column:enabled;not null;default:true" json:"enabled"`
	CreatedAt      time.Time     `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time     `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (EmailRule) TableName() string {
	return "email_rules"
}
