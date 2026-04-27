package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type uploadedFile struct {
	FieldName string
	FileName  string
	MimeType  string
	Size      int64
	Body      []byte
}

// validateFile checks file against the field's allowed_types and max_size.
// Returns an error message string suitable for the validation fields map.
func validateFile(field map[string]any, f uploadedFile) string {
	// max_size in MB
	maxMB := 5.0
	if v, ok := field["max_size"]; ok {
		switch n := v.(type) {
		case float64:
			maxMB = n
		case int:
			maxMB = float64(n)
		}
	}
	if float64(f.Size) > maxMB*1024*1024 {
		return fmt.Sprintf("File exceeds %.1fMB limit", maxMB)
	}

	allowed, _ := field["allowed_types"].(string)
	if allowed != "" {
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(f.FileName)), ".")
		ok := false
		for _, a := range strings.Split(allowed, ",") {
			if strings.TrimSpace(strings.ToLower(a)) == ext {
				ok = true
				break
			}
		}
		if !ok {
			return "File type not allowed (expected: " + allowed + ")"
		}
	}
	return ""
}

// storeFile writes via host.StoreFile and returns the file metadata to embed
// in the submission JSON.
func (p *FormsPlugin) storeFile(ctx context.Context, formID uint, f uploadedFile) (map[string]any, error) {
	safeName := sanitizeFilename(f.FileName)
	storagePath := fmt.Sprintf("forms/submissions/%d/%d_%s", formID, time.Now().UnixNano(), safeName)
	url, err := p.host.StoreFile(ctx, storagePath, f.Body)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"name":      f.FileName,
		"url":       url,
		"size":      f.Size,
		"mime_type": f.MimeType,
	}, nil
}

// sanitizeFilename strips path separators and unsafe chars.
func sanitizeFilename(s string) string {
	s = filepath.Base(s)
	s = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return '_'
		}
		return r
	}, s)
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
