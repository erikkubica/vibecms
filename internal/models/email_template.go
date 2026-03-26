package models

import "time"

// EmailTemplate represents a reusable email template with subject and body.
// LanguageID is nullable — NULL means universal/fallback template.
type EmailTemplate struct {
	ID              int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Slug            string    `gorm:"column:slug;type:varchar(100);not null" json:"slug"`
	Name            string    `gorm:"column:name;type:varchar(150);not null" json:"name"`
	LanguageID      *int      `gorm:"column:language_id" json:"language_id"`
	SubjectTemplate string    `gorm:"column:subject_template;type:text;not null" json:"subject_template"`
	BodyTemplate    string    `gorm:"column:body_template;type:text;not null" json:"body_template"`
	TestData        JSONB     `gorm:"column:test_data;type:jsonb;not null;default:'{}'" json:"test_data"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (EmailTemplate) TableName() string {
	return "email_templates"
}
