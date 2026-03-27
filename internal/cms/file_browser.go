package cms

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxFileSize = 1 << 20 // 1MB

// FileEntry represents a single file or directory in a listing.
type FileEntry struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // "file" or "directory"
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// DirectoryResponse is returned when the requested path is a directory.
type DirectoryResponse struct {
	Type    string      `json:"type"` // "directory"
	Path    string      `json:"path"`
	Entries []FileEntry `json:"entries"`
}

// FileResponse is returned when the requested path is a file.
type FileResponse struct {
	Type     string `json:"type"` // "file"
	Path     string `json:"path"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	Language string `json:"language"`
	Binary   bool   `json:"binary,omitempty"`
	TooLarge bool   `json:"too_large,omitempty"`
}

// binaryExtensions lists file extensions that should not have their content read.
var binaryExtensions = map[string]bool{
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".ico":   true,
	".woff":  true,
	".woff2": true,
	".ttf":   true,
	".eot":   true,
}

// languageMap maps file extensions to language identifiers for syntax highlighting.
var languageMap = map[string]string{
	".go":    "go",
	".html":  "html",
	".json":  "json",
	".css":   "css",
	".js":    "javascript",
	".tengo": "tengo",
	".tgo":   "tengo",
	".md":    "markdown",
	".sql":   "sql",
	".txt":   "text",
	".yaml":  "yaml",
	".yml":   "yaml",
	".ts":    "typescript",
	".tsx":   "typescript",
	".jsx":   "javascript",
	".xml":   "xml",
	".svg":   "xml",
	".sh":    "shell",
	".toml":  "toml",
}

// detectLanguage returns a language identifier based on file extension.
func detectLanguage(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if lang, ok := languageMap[ext]; ok {
		return lang
	}
	return "text"
}

// isBinary returns true if the file extension indicates a binary file.
func isBinary(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return binaryExtensions[ext]
}

// BrowseFilesInDir handles browsing files within a base directory.
// requestedPath is relative to baseDir. It returns a DirectoryResponse or FileResponse.
func BrowseFilesInDir(baseDir, requestedPath string) (interface{}, error) {
	// Clean and resolve the full path.
	cleanBase := filepath.Clean(baseDir)
	fullPath := filepath.Clean(filepath.Join(cleanBase, requestedPath))

	// Security: ensure resolved path is within the base directory.
	if !strings.HasPrefix(fullPath, cleanBase+string(os.PathSeparator)) && fullPath != cleanBase {
		return nil, fmt.Errorf("INVALID_PATH")
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("NOT_FOUND")
		}
		return nil, fmt.Errorf("STAT_FAILED")
	}

	if info.IsDir() {
		return browseDirectory(fullPath, requestedPath)
	}
	return browseFile(fullPath, requestedPath, info)
}

func browseDirectory(fullPath, requestedPath string) (*DirectoryResponse, error) {
	dirEntries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("READ_DIR_FAILED")
	}

	entries := make([]FileEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		// Skip hidden files/directories (starting with .)
		if strings.HasPrefix(de.Name(), ".") {
			continue
		}

		fi, err := de.Info()
		if err != nil {
			continue
		}

		entryType := "file"
		var size int64
		if de.IsDir() {
			entryType = "directory"
		} else {
			size = fi.Size()
		}

		entries = append(entries, FileEntry{
			Name:     de.Name(),
			Type:     entryType,
			Size:     size,
			Modified: fi.ModTime().UTC().Format(time.RFC3339),
		})
	}

	// Sort: directories first, then files, alphabetically within each group.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type != entries[j].Type {
			return entries[i].Type == "directory"
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return &DirectoryResponse{
		Type:    "directory",
		Path:    requestedPath,
		Entries: entries,
	}, nil
}

func browseFile(fullPath, requestedPath string, info os.FileInfo) (*FileResponse, error) {
	name := filepath.Base(fullPath)
	resp := &FileResponse{
		Type:     "file",
		Path:     requestedPath,
		Name:     name,
		Size:     info.Size(),
		Language: detectLanguage(name),
	}

	if isBinary(name) {
		resp.Binary = true
		return resp, nil
	}

	if info.Size() > maxFileSize {
		resp.TooLarge = true
		return resp, nil
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("READ_FAILED")
	}
	resp.Content = string(data)

	return resp, nil
}
