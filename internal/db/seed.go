package db

import (
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"vibecms/internal/models"
)

// Seed populates the database with initial data including a default admin
// user and a sample content node.
func Seed(db *gorm.DB) error {
	// Create default admin user
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	fullName := "Admin"
	admin := models.User{
		Email:        "admin@vibecms.local",
		PasswordHash: string(hash),
		Role:         "admin",
		FullName:     &fullName,
	}

	result := db.Where("email = ?", admin.Email).FirstOrCreate(&admin)
	if result.Error != nil {
		return fmt.Errorf("failed to seed admin user: %w", result.Error)
	}

	// Create sample content node
	blocksData := json.RawMessage(`[{"type":"heading","data":{"text":"Welcome to VibeCMS","level":1}},{"type":"paragraph","data":{"text":"This is your first page. Edit it from the admin panel."}}]`)
	seoSettings := json.RawMessage(`{"meta_title":"Welcome to VibeCMS","meta_description":"A high-performance, AI-native CMS."}`)
	now := time.Now()

	node := models.ContentNode{
		NodeType:     "page",
		Status:       "published",
		LanguageCode: "en",
		Slug:         "home",
		FullURL:      "/",
		Title:        "Welcome to VibeCMS",
		BlocksData:   models.JSONB(blocksData),
		SeoSettings:  models.JSONB(seoSettings),
		Version:      1,
		PublishedAt:  &now,
	}

	result = db.Where("full_url = ?", node.FullURL).FirstOrCreate(&node)
	if result.Error != nil {
		return fmt.Errorf("failed to seed sample content node: %w", result.Error)
	}

	return nil
}
