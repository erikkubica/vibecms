package main

import (
	"strings"
	"testing"
)

// ---- validateFile ----

func TestValidateFile_MaxSize(t *testing.T) {
	field := map[string]any{"max_size": float64(1)} // 1 MB

	t.Run("within limit", func(t *testing.T) {
		f := uploadedFile{FileName: "a.pdf", Size: 500 * 1024}
		if msg := validateFile(field, f); msg != "" {
			t.Errorf("expected no error, got %q", msg)
		}
	})

	t.Run("exceeds limit", func(t *testing.T) {
		f := uploadedFile{FileName: "a.pdf", Size: 2 * 1024 * 1024}
		if msg := validateFile(field, f); msg == "" {
			t.Error("expected size error, got none")
		}
	})
}

func TestValidateFile_DefaultMaxSize(t *testing.T) {
	field := map[string]any{} // no max_size → default 5MB

	small := uploadedFile{FileName: "small.jpg", Size: 4 * 1024 * 1024}
	if msg := validateFile(field, small); msg != "" {
		t.Errorf("4MB file with default 5MB limit: unexpected error %q", msg)
	}

	large := uploadedFile{FileName: "large.jpg", Size: 6 * 1024 * 1024}
	if msg := validateFile(field, large); msg == "" {
		t.Error("6MB file with default 5MB limit should error")
	}
}

func TestValidateFile_AllowedTypes(t *testing.T) {
	field := map[string]any{"allowed_types": "pdf,jpg,png"}

	cases := []struct {
		filename string
		wantErr  bool
	}{
		{"resume.pdf", false},
		{"photo.jpg", false},
		{"logo.PNG", false}, // case-insensitive
		{"script.exe", true},
		{"archive.ZIP", true},
	}
	for _, tc := range cases {
		f := uploadedFile{FileName: tc.filename, Size: 100}
		msg := validateFile(field, f)
		if tc.wantErr && msg == "" {
			t.Errorf("%s: expected type error, got none", tc.filename)
		}
		if !tc.wantErr && msg != "" {
			t.Errorf("%s: expected no error, got %q", tc.filename, msg)
		}
	}
}

func TestValidateFile_NoAllowedTypes(t *testing.T) {
	field := map[string]any{} // no restriction
	f := uploadedFile{FileName: "anything.xyz", Size: 100}
	if msg := validateFile(field, f); msg != "" {
		t.Errorf("no allowed_types constraint: unexpected error %q", msg)
	}
}

// ---- storeFile ----

func TestStoreFile(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	f := uploadedFile{
		FileName: "test.pdf",
		MimeType: "application/pdf",
		Size:     1234,
		Body:     []byte("pdf-content"),
	}

	meta, err := p.storeFile(ctx(), 42, f)
	if err != nil {
		t.Fatalf("storeFile failed: %v", err)
	}

	if meta["name"] != "test.pdf" {
		t.Errorf("name: got %v, want test.pdf", meta["name"])
	}
	if meta["mime_type"] != "application/pdf" {
		t.Errorf("mime_type: got %v", meta["mime_type"])
	}
	if meta["size"] != int64(1234) {
		t.Errorf("size: got %v, want 1234", meta["size"])
	}
	url, _ := meta["url"].(string)
	if !strings.HasPrefix(url, "/forms/submissions/42/") {
		t.Errorf("url path: got %q, want prefix /forms/submissions/42/", url)
	}
}

// ---- sanitizeFilename ----

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		input string
		check func(string) bool
		desc  string
	}{
		{"/etc/passwd", func(s string) bool { return s == "passwd" }, "strips path"},
		{"../../etc/secret", func(s string) bool { return !strings.Contains(s, "/") && !strings.Contains(s, "..") }, "strips traversal"},
		{strings.Repeat("x", 300), func(s string) bool { return len(s) <= 200 }, "truncates to 200"},
		{"normal.txt", func(s string) bool { return s == "normal.txt" }, "normal name unchanged"},
	}
	for _, tc := range cases {
		got := sanitizeFilename(tc.input)
		if !tc.check(got) {
			t.Errorf("%s: sanitizeFilename(%q) = %q", tc.desc, tc.input, got)
		}
	}
}

// ---- deleteFileValueIfPresent ----

func TestDeleteFileValueIfPresent_SingleFile(t *testing.T) {
	h := NewFakeHost()
	// Pre-populate a stored file
	h.StoredFiles["forms/submissions/1/file.pdf"] = []byte("data")

	val := map[string]any{"url": "/forms/submissions/1/file.pdf", "name": "file.pdf"}
	deleteFileValueIfPresent(ctx(), h, val)

	if _, ok := h.StoredFiles["forms/submissions/1/file.pdf"]; ok {
		t.Error("file should have been deleted")
	}
}

func TestDeleteFileValueIfPresent_ArrayOfFiles(t *testing.T) {
	h := NewFakeHost()
	h.StoredFiles["forms/submissions/1/a.pdf"] = []byte("a")
	h.StoredFiles["forms/submissions/1/b.pdf"] = []byte("b")

	val := []any{
		map[string]any{"url": "/forms/submissions/1/a.pdf"},
		map[string]any{"url": "/forms/submissions/1/b.pdf"},
	}
	deleteFileValueIfPresent(ctx(), h, val)

	if len(h.StoredFiles) != 0 {
		t.Errorf("expected 0 files, got %d", len(h.StoredFiles))
	}
}

func TestDeleteFileValueIfPresent_NonFileValue(t *testing.T) {
	h := NewFakeHost()
	// Non-file values (strings, ints) should not panic
	deleteFileValueIfPresent(ctx(), h, "plain string")
	deleteFileValueIfPresent(ctx(), h, 42)
	deleteFileValueIfPresent(ctx(), h, nil)
}

func TestDeleteFileValueIfPresent_NonFormsURL(t *testing.T) {
	h := NewFakeHost()
	// Only /forms/submissions/ URLs are deleted
	h.StoredFiles["media/images/photo.jpg"] = []byte("photo")
	val := map[string]any{"url": "/media/images/photo.jpg"}
	deleteFileValueIfPresent(ctx(), h, val)

	if _, ok := h.StoredFiles["media/images/photo.jpg"]; !ok {
		t.Error("non-forms file should NOT be deleted")
	}
}
