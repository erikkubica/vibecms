package coreapi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// safeStoragePath validates that the resolved path stays within the storage directory.
func safeStoragePath(path string) (string, error) {
	baseDir, _ := filepath.Abs("storage")
	fullPath := filepath.Join(baseDir, path)
	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, baseDir+string(os.PathSeparator)) && cleanPath != baseDir {
		return "", NewValidation("invalid file path: path traversal not allowed")
	}
	return cleanPath, nil
}

func (c *coreImpl) StoreFile(ctx context.Context, path string, data []byte) (string, error) {
	fullPath, err := safeStoragePath(path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", NewInternal("store file mkdir: " + err.Error())
	}
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", NewInternal("store file write: " + err.Error())
	}
	return "/" + path, nil
}

func (c *coreImpl) DeleteFile(ctx context.Context, path string) error {
	fullPath, err := safeStoragePath(path)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return NewNotFound("file", path)
		}
		return NewInternal("delete file: " + err.Error())
	}
	return nil
}
