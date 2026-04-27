package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// JSONB is a custom type wrapping json.RawMessage that implements the
// sql.Scanner and driver.Valuer interfaces for GORM compatibility.
type JSONB json.RawMessage

// Scan implements the sql.Scanner interface.
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = JSONB("null")
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("JSONB.Scan: expected []byte")
	}
	*j = JSONB(append([]byte{}, bytes...))
	return nil
}

// Value implements the driver.Valuer interface.
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

// MarshalJSON returns the raw JSON bytes.
func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

// UnmarshalJSON sets the raw JSON bytes.
func (j *JSONB) UnmarshalJSON(data []byte) error {
	*j = JSONB(append([]byte{}, data...))
	return nil
}

// ContentNode represents a content entity (page, post, etc.) in the CMS.
type ContentNode struct {
	ID                 int            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UUID               string         `gorm:"column:uuid;type:uuid;default:gen_random_uuid();uniqueIndex;not null" json:"uuid"`
	ParentID           *int           `gorm:"column:parent_id" json:"parent_id,omitempty"`
	Parent             *ContentNode   `gorm:"foreignKey:ParentID;references:ID" json:"parent,omitempty"`
	NodeType           string         `gorm:"column:node_type;type:varchar(50);not null;default:'page'" json:"node_type"`
	Status             string         `gorm:"column:status;type:varchar(20);not null;default:'draft'" json:"status"`
	LanguageCode       string         `gorm:"column:language_code;type:varchar(10);not null;default:'en'" json:"language_code"`
	LanguageID         *int           `gorm:"column:language_id" json:"language_id,omitempty"`
	Slug               string         `gorm:"column:slug;type:varchar(255);not null" json:"slug"`
	FullURL            string         `gorm:"column:full_url;type:text;uniqueIndex;not null" json:"full_url"`
	Title              string         `gorm:"column:title;type:varchar(255);not null" json:"title"`
	FeaturedImage      JSONB          `gorm:"column:featured_image;type:jsonb;not null;default:'{}'" json:"featured_image"`
	Excerpt            string         `gorm:"column:excerpt;type:text;not null;default:''" json:"excerpt"`
	Taxonomies         JSONB          `gorm:"column:taxonomies;type:jsonb;not null;default:'{}'" json:"taxonomies"`
	BlocksData         JSONB          `gorm:"column:blocks_data;type:jsonb;not null;default:'[]'" json:"blocks_data"`
	SeoSettings        JSONB          `gorm:"column:seo_settings;type:jsonb;not null;default:'{}'" json:"seo_settings"`
	FieldsData         JSONB          `gorm:"column:fields_data;type:jsonb;not null;default:'{}'" json:"fields_data"`
	LayoutData         JSONB          `gorm:"column:layout_data;type:jsonb;not null;default:'{}'" json:"layout_data"`
	AuthorID           *int           `gorm:"column:author_id" json:"author_id,omitempty"`
	Author             *User          `gorm:"foreignKey:AuthorID;references:ID" json:"author,omitempty"`
	LayoutID           *int           `gorm:"column:layout_id" json:"layout_id,omitempty"`
	LayoutSlug         *string        `gorm:"column:layout_slug" json:"layout_slug,omitempty"`
	TranslationGroupID *string        `gorm:"column:translation_group_id;type:uuid" json:"translation_group_id,omitempty"`
	Version            int            `gorm:"column:version;not null;default:1" json:"version"`
	PublishedAt        *time.Time     `gorm:"column:published_at" json:"published_at,omitempty"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

// BeforeSave keeps LayoutSlug in sync with LayoutID so a node's layout
// reference survives theme deactivate/reactivate cycles (layouts may be
// deleted/recreated; slugs persist across such cycles).
//
// LayoutSlug set directly (without LayoutID) is preserved — that path is used
// by theme seeds and the slug-first cascade in LayoutService.ResolveForNode.
func (n *ContentNode) BeforeSave(tx *gorm.DB) error {
	if n.LayoutID != nil {
		var slug string
		if err := tx.Model(&Layout{}).Select("slug").Where("id = ?", *n.LayoutID).Scan(&slug).Error; err == nil && slug != "" {
			n.LayoutSlug = &slug
		}
	}
	return nil
}

// TableName overrides the default GORM table name.
func (ContentNode) TableName() string {
	return "content_nodes"
}
