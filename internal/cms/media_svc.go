package cms

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"squilla/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MediaService handles file uploads, listing, and deletion for the media manager.
type MediaService struct {
	db         *gorm.DB
	storageDir string // e.g. "storage/media"
}

// NewMediaService creates a new MediaService.
func NewMediaService(db *gorm.DB, storageDir string) *MediaService {
	return &MediaService{db: db, storageDir: storageDir}
}

// GetStorageDir returns the base storage directory.
func (s *MediaService) GetStorageDir() string {
	return s.storageDir
}

// Upload saves a file to disk and creates a database record.
func (s *MediaService) Upload(file io.Reader, originalName string, mimeType string, size int64) (*models.MediaFile, error) {
	now := time.Now()
	dateDir := fmt.Sprintf("%04d/%02d", now.Year(), now.Month())

	ext := filepath.Ext(originalName)
	storedName := uuid.New().String() + ext

	// Store path WITH the "media/" prefix so it matches the media-manager
	// extension's convention. media-manager resolves files via
	// filepath.Join("storage", row.path) for optimize / re-optimize /
	// restore — without this prefix, a kernel-uploaded row points at
	// storage/<date>/<file> while the actual file is at storage/media/
	// <date>/<file>, producing a confusing "source image not found" while
	// the public URL still serves fine. storageDir is the filesystem root
	// (e.g. "storage"); the "media/" segment lives inside the stored path.
	relPath := filepath.Join("media", dateDir, storedName)
	fullDir := filepath.Join(s.storageDir, "media", dateDir)

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	fullPath := filepath.Join(fullDir, storedName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	if size == 0 {
		size = written
	}

	publicURL := "/media/" + dateDir + "/" + storedName

	mf := &models.MediaFile{
		Filename:     storedName,
		OriginalName: originalName,
		MimeType:     mimeType,
		Size:         size,
		Path:         relPath,
		URL:          publicURL,
	}

	if err := s.db.Create(mf).Error; err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("failed to save media record: %w", err)
	}

	return mf, nil
}

// GetByID retrieves a single media file by ID.
func (s *MediaService) GetByID(id uint) (*models.MediaFile, error) {
	var mf models.MediaFile
	if err := s.db.First(&mf, id).Error; err != nil {
		return nil, err
	}
	return &mf, nil
}

// List retrieves media files with optional filtering and pagination. Returns items and total count.
func (s *MediaService) List(mimeType string, search string, limit, offset int) ([]models.MediaFile, int64, error) {
	q := s.db.Model(&models.MediaFile{})

	if mimeType != "" {
		if strings.Contains(mimeType, "/") {
			q = q.Where("mime_type = ?", mimeType)
		} else {
			q = q.Where("mime_type LIKE ?", mimeType+"/%")
		}
	}

	if search != "" {
		q = q.Where("original_name ILIKE ?", "%"+search+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}

	var files []models.MediaFile
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&files).Error; err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

// Delete removes a media file from disk and database.
func (s *MediaService) Delete(id uint) error {
	var mf models.MediaFile
	if err := s.db.First(&mf, id).Error; err != nil {
		return err
	}

	fullPath := filepath.Join(s.storageDir, mf.Path)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return s.db.Delete(&mf).Error
}
