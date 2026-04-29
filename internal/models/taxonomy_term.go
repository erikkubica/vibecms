package models

import "time"

// TaxonomyTerm represents a single term (category, tag, etc.) in a taxonomy.
//
// Each language version of a term is its own row. Translations of the same
// logical concept share a translation_group_id (UUID); a single-language
// term has translation_group_id = nil until a sibling translation is
// created, at which point both the source and the new row share a freshly
// minted UUID. Slug uniqueness is per (node_type, taxonomy, language_code).
type TaxonomyTerm struct {
	ID                 int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	NodeType           string    `gorm:"column:node_type;type:varchar(50);not null" json:"node_type"`
	Taxonomy           string    `gorm:"column:taxonomy;type:varchar(50);not null" json:"taxonomy"`
	LanguageCode       string    `gorm:"column:language_code;type:varchar(10);not null;default:'en'" json:"language_code"`
	TranslationGroupID *string   `gorm:"column:translation_group_id;type:uuid" json:"translation_group_id,omitempty"`
	Slug               string    `gorm:"column:slug;type:varchar(255);not null" json:"slug"`
	Name               string    `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description        string    `gorm:"column:description;type:text;not null;default:''" json:"description"`
	ParentID           *int      `gorm:"column:parent_id" json:"parent_id,omitempty"`
	Count              int       `gorm:"column:count;not null;default:0" json:"count"`
	FieldsData         JSONB     `gorm:"column:fields_data;type:jsonb;not null;default:'{}'" json:"fields_data"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Optional relations
	Parent *TaxonomyTerm `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
}

// TableName overrides the default GORM table name.
func (TaxonomyTerm) TableName() string {
	return "taxonomy_terms"
}
