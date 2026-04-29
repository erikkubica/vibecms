package models

import "time"

// ContentNodeRevision stores a historical snapshot of a content node's data.
// Every Update creates one row capturing the pre-update state so the editor
// can browse and restore. Migration 0041 expanded the snapshot to cover
// every editable field; pre-0041 rows still resolve via the column defaults.
type ContentNodeRevision struct {
	ID                 int64       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	NodeID             int         `gorm:"column:node_id;not null" json:"node_id"`
	Node               ContentNode `gorm:"foreignKey:NodeID;references:ID" json:"node,omitempty"`
	Title              string      `gorm:"column:title;not null;default:''" json:"title"`
	Slug               string      `gorm:"column:slug;not null;default:''" json:"slug"`
	Status             string      `gorm:"column:status;not null;default:'draft'" json:"status"`
	LanguageCode       string      `gorm:"column:language_code;not null;default:'en'" json:"language_code"`
	LayoutSlug         *string     `gorm:"column:layout_slug" json:"layout_slug,omitempty"`
	Excerpt            string      `gorm:"column:excerpt;not null;default:''" json:"excerpt"`
	FeaturedImage      JSONB       `gorm:"column:featured_image;type:jsonb;not null;default:'{}'" json:"featured_image"`
	BlocksSnapshot     JSONB       `gorm:"column:blocks_snapshot;type:jsonb;not null" json:"blocks_snapshot"`
	FieldsSnapshot     JSONB       `gorm:"column:fields_snapshot;type:jsonb;not null;default:'{}'" json:"fields_snapshot"`
	SeoSnapshot        JSONB       `gorm:"column:seo_snapshot;type:jsonb;not null" json:"seo_snapshot"`
	TaxonomiesSnapshot JSONB       `gorm:"column:taxonomies_snapshot;type:jsonb;not null;default:'{}'" json:"taxonomies_snapshot"`
	VersionNumber      int         `gorm:"column:version_number;not null;default:0" json:"version_number"`
	CreatedBy          *int        `gorm:"column:created_by" json:"created_by,omitempty"`
	Creator            *User       `gorm:"foreignKey:CreatedBy;references:ID" json:"creator,omitempty"`
	CreatedAt          time.Time   `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (ContentNodeRevision) TableName() string {
	return "content_node_revisions"
}
