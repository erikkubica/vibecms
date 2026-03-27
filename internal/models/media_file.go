package models

import "time"

type MediaFile struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Filename     string    `gorm:"not null" json:"filename"`      // stored filename (uuid-based)
	OriginalName string    `gorm:"not null" json:"original_name"` // original upload name
	MimeType     string    `gorm:"not null" json:"mime_type"`
	Size         int64     `gorm:"not null" json:"size"`  // bytes
	Path         string    `gorm:"not null" json:"path"`  // relative storage path
	URL          string    `gorm:"not null" json:"url"`   // public URL
	Width        *int      `json:"width,omitempty"`        // image width (nil for non-images)
	Height       *int      `json:"height,omitempty"`       // image height (nil for non-images)
	Alt          string    `json:"alt"`                    // alt text
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
