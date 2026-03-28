package coreapi

import (
	"context"
	"os"
	"path/filepath"
)

func (c *coreImpl) StoreFile(_ context.Context, path string, data []byte) (string, error) {
	fullPath := filepath.Join("storage", path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", NewInternal("store file mkdir: " + err.Error())
	}
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", NewInternal("store file write: " + err.Error())
	}
	return "/" + path, nil
}

func (c *coreImpl) DeleteFile(_ context.Context, path string) error {
	fullPath := filepath.Join("storage", path)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return NewNotFound("file", path)
		}
		return NewInternal("delete file: " + err.Error())
	}
	return nil
}
