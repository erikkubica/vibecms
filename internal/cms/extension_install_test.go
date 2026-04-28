package cms

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildZip composes an in-memory zip from the given path→bytes map. Names
// containing trailing slashes become directory entries.
func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		if strings.HasSuffix(name, "/") {
			if _, err := zw.Create(name); err != nil {
				t.Fatalf("create dir entry %s: %v", name, err)
			}
			continue
		}
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create entry %s: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("write entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

// readZip is a small helper that converts the buffer back to a *zip.Reader so
// the package-internal helpers can be exercised directly.
func readZip(t *testing.T, data []byte) *zip.Reader {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	return r
}

func TestFindExtensionManifest_RootAndOneLevelDeep(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		zipData := buildZip(t, map[string]string{
			"extension.json": `{"slug":"hello","name":"Hello"}`,
			"scripts/x.tgo":  "x",
		})
		m, prefix, err := findExtensionManifestInZip(readZip(t, zipData))
		if err != nil {
			t.Fatalf("expected manifest, got: %v", err)
		}
		if m.Slug != "hello" {
			t.Fatalf("slug=%q want hello", m.Slug)
		}
		if prefix != "" {
			t.Fatalf("prefix=%q want empty", prefix)
		}
	})

	t.Run("one level deep", func(t *testing.T) {
		zipData := buildZip(t, map[string]string{
			"hello/extension.json": `{"slug":"hello","name":"Hello"}`,
			"hello/scripts/x.tgo":  "x",
		})
		m, prefix, err := findExtensionManifestInZip(readZip(t, zipData))
		if err != nil {
			t.Fatalf("expected manifest, got: %v", err)
		}
		if m.Slug != "hello" {
			t.Fatalf("slug=%q want hello", m.Slug)
		}
		if prefix != "hello/" {
			t.Fatalf("prefix=%q want 'hello/'", prefix)
		}
	})

	t.Run("missing", func(t *testing.T) {
		zipData := buildZip(t, map[string]string{
			"a/b/extension.json": `{"slug":"hello"}`, // too deep, must reject
		})
		if _, _, err := findExtensionManifestInZip(readZip(t, zipData)); err == nil {
			t.Fatal("expected error for too-deep manifest, got nil")
		}
	})
}

func TestExtractZipToDir_BlocksZipSlip(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "ext")
	zipData := buildZip(t, map[string]string{
		"../escape.txt": "pwn",
	})
	err := extractZipToDir(readZip(t, zipData), dest, "")
	if err == nil {
		t.Fatal("expected zip-slip rejection, got nil")
	}
	if !strings.Contains(err.Error(), "zip slip") {
		t.Fatalf("expected zip-slip error, got: %v", err)
	}
}

func TestExtractZipToDir_HappyPath(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "ext")
	zipData := buildZip(t, map[string]string{
		"hello/extension.json":   `{"slug":"hello"}`,
		"hello/scripts/main.tgo": "// ok",
		"hello/dir/":             "",
	}, // prefix-stripped on extract
	)
	if err := extractZipToDir(readZip(t, zipData), dest, "hello/"); err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "extension.json")); err != nil {
		t.Fatalf("extension.json not extracted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "scripts", "main.tgo")); err != nil {
		t.Fatalf("main.tgo not extracted: %v", err)
	}
}

func TestIsValidSettingsKey_BlocksTraversal(t *testing.T) {
	cases := map[string]bool{
		"hello":                  true,
		"hello-world":            true,
		"hello_world":            true,
		"":                       false,
		"../escape":              false,
		"foo/bar":                false,
		"foo bar":                false,
		"foo.bar":                false,
		strings.Repeat("a", 129): false,
	}
	for k, want := range cases {
		if got := isValidSettingsKey(k); got != want {
			t.Errorf("isValidSettingsKey(%q)=%v want %v", k, got, want)
		}
	}
}
