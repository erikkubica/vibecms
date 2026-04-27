package cms

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// This file owns small filesystem helpers used by the theme
// management service: zip extraction, manifest discovery, and
// recursive copy. Pure utilities, no DB access.

func extractZipFile(f *zip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	// Limit copy size to prevent zip bombs (256 MB per file).
	_, err = io.Copy(out, io.LimitReader(rc, 256<<20))
	return err
}

// findAndParseManifest looks for theme.json in dir or one level deep.
func findAndParseManifest(dir string) (*themeMgmtManifest, string, error) {
	// Check root.
	rootManifest := filepath.Join(dir, "theme.json")
	if data, err := os.ReadFile(rootManifest); err == nil {
		var m themeMgmtManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, "", fmt.Errorf("failed to parse theme.json: %w", err)
		}
		return &m, dir, nil
	}

	// Check one level deep.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read temp dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subManifest := filepath.Join(dir, entry.Name(), "theme.json")
		if data, err := os.ReadFile(subManifest); err == nil {
			var m themeMgmtManifest
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, "", fmt.Errorf("failed to parse theme.json: %w", err)
			}
			return &m, filepath.Join(dir, entry.Name()), nil
		}
	}

	return nil, "", fmt.Errorf("theme.json not found in archive (checked root and one level deep)")
}

// copyDir recursively copies a directory tree from src to dst.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
