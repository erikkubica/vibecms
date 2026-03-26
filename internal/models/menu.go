package models

import "time"

// Menu represents a navigation menu with versioning for optimistic locking.
type Menu struct {
	ID           int        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Slug         string     `gorm:"column:slug;type:varchar(255);not null" json:"slug"`
	Name         string     `gorm:"column:name;type:varchar(255);not null" json:"name"`
	LanguageID   *int       `gorm:"column:language_id" json:"language_id"`
	Version      int        `gorm:"column:version;type:int;not null;default:1" json:"version"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Items        []MenuItem `gorm:"-" json:"items,omitempty"`
}

func (Menu) TableName() string { return "menus" }

// MenuItem represents a single item within a menu, supporting nested trees via ParentID.
type MenuItem struct {
	ID        int        `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MenuID    int        `gorm:"column:menu_id;not null" json:"menu_id"`
	ParentID  *int       `gorm:"column:parent_id" json:"parent_id"`
	Title     string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	ItemType  string     `gorm:"column:item_type;type:varchar(20);not null;default:'custom'" json:"item_type"`
	NodeID    *int       `gorm:"column:node_id" json:"node_id"`
	URL       string     `gorm:"column:url;type:varchar(2048)" json:"url"`
	Target    string     `gorm:"column:target;type:varchar(20);not null;default:'_self'" json:"target"`
	CSSClass  string     `gorm:"column:css_class;type:varchar(255)" json:"css_class"`
	SortOrder int        `gorm:"column:sort_order;type:int;not null;default:0" json:"sort_order"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Children  []MenuItem `gorm:"-" json:"children,omitempty"`
}

func (MenuItem) TableName() string { return "menu_items" }

// MenuItemTree is an input type for replacing menu items as a nested tree.
type MenuItemTree struct {
	Title    string         `json:"title"`
	ItemType string         `json:"item_type"`
	NodeID   *int           `json:"node_id"`
	URL      string         `json:"url"`
	Target   string         `json:"target"`
	CSSClass string         `json:"css_class"`
	Children []MenuItemTree `json:"children,omitempty"`
}
