package models

import "time"

// EmailLog records each email sent or attempted by the system.
type EmailLog struct {
	ID             int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RuleID         *int      `gorm:"column:rule_id" json:"rule_id,omitempty"`
	TemplateSlug   string    `gorm:"column:template_slug;type:varchar(100);not null" json:"template_slug"`
	Action         string    `gorm:"column:action;type:varchar(100);not null" json:"action"`
	RecipientEmail string    `gorm:"column:recipient_email;type:varchar(255);not null" json:"recipient_email"`
	Subject        string    `gorm:"column:subject;type:text;not null" json:"subject"`
	RenderedBody   string    `gorm:"column:rendered_body;type:text;not null" json:"rendered_body"`
	Status         string    `gorm:"column:status;type:varchar(20);not null;default:'pending'" json:"status"`
	ErrorMessage   *string   `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	Provider       *string   `gorm:"column:provider;type:varchar(50)" json:"provider,omitempty"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (EmailLog) TableName() string {
	return "email_logs"
}
