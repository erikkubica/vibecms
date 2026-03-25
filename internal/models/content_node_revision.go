package models

import "time"

// ContentNodeRevision stores a historical snapshot of a content node's data.
type ContentNodeRevision struct {
	ID             int64       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	NodeID         int         `gorm:"column:node_id;not null" json:"node_id"`
	Node           ContentNode `gorm:"foreignKey:NodeID;references:ID" json:"node,omitempty"`
	BlocksSnapshot JSONB       `gorm:"column:blocks_snapshot;type:jsonb;not null" json:"blocks_snapshot"`
	SeoSnapshot    JSONB       `gorm:"column:seo_snapshot;type:jsonb;not null" json:"seo_snapshot"`
	CreatedBy      *int        `gorm:"column:created_by" json:"created_by,omitempty"`
	Creator        *User       `gorm:"foreignKey:CreatedBy;references:ID" json:"creator,omitempty"`
	CreatedAt      time.Time   `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (ContentNodeRevision) TableName() string {
	return "content_node_revisions"
}
