package rendering

import "testing"

// imageURL is a unit-level helper — these tests exercise the function
// directly rather than going through a TemplateRenderer because the
// transform has no dependency on parsing or caching.

func TestImageURL_TransformsMediaPath(t *testing.T) {
	got := imageURL("/media/cache/", "/media/2026/03/photo.jpg", "thumb")
	want := "/media/cache/thumb/2026/03/photo.jpg"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestImageURL_PassthroughForNonMediaURL(t *testing.T) {
	// External CDN URLs and absolute http(s) URLs are returned
	// unchanged — themes can mix asset sources without each call site
	// having to special-case.
	cases := []string{
		"https://cdn.example.com/img.jpg",
		"/static/logo.png",
		"data:image/png;base64,abc",
	}
	for _, in := range cases {
		got := imageURL("/media/cache/", in, "thumb")
		if got != in {
			t.Errorf("non-/media URL %q rewritten to %q", in, got)
		}
	}
}

func TestImageURL_EmptyPrefixIsPassthrough(t *testing.T) {
	// Documented "no media-manager configured" fallback — keeps a
	// theme that calls image_url usable even before the extension
	// has populated the prefix setting.
	got := imageURL("", "/media/foo.jpg", "thumb")
	if got != "/media/foo.jpg" {
		t.Errorf("empty prefix should passthrough; got %q", got)
	}
}

func TestImageURL_TrailingSlashOnPrefix_NotDoubled(t *testing.T) {
	// Operators may set the prefix with or without a trailing slash.
	// The output must be a single, well-formed URL either way.
	with := imageURL("/media/cache/", "/media/foo.jpg", "thumb")
	without := imageURL("/media/cache", "/media/foo.jpg", "thumb")
	if with != without {
		t.Errorf("trailing-slash inconsistency: %q vs %q", with, without)
	}
}

func TestImageSrcset_BuildsCSV(t *testing.T) {
	got := imageSrcset("/media/cache", "/media/foo.jpg", []string{"sm", "md", "lg"})
	want := "/media/cache/sm/foo.jpg, /media/cache/md/foo.jpg, /media/cache/lg/foo.jpg"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestImageSrcset_EmptyForNonMedia(t *testing.T) {
	if got := imageSrcset("/media/cache/", "https://cdn/x.jpg", []string{"sm"}); got != "" {
		t.Errorf("non-media srcset should be empty; got %q", got)
	}
	if got := imageSrcset("", "/media/foo.jpg", []string{"sm"}); got != "" {
		t.Errorf("empty-prefix srcset should be empty; got %q", got)
	}
}

func TestSetImageURLPrefix_OverridesAtRuntime(t *testing.T) {
	r := NewTemplateRenderer(t.TempDir(), false)
	// Default → uses defaultImageURLPrefix.
	defaultGot := renderFunc(t, r, `{{image_url "/media/x.jpg" "thumb"}}`, nil)
	if defaultGot != "/media/cache/thumb/x.jpg" {
		t.Fatalf("default prefix path: got %q", defaultGot)
	}
	// Override → helper picks up the new prefix on the next call,
	// without having to rebuild the funcMap or invalidate the cache.
	r.SetImageURLPrefix("/cdn/img/")
	overrideGot := renderFunc(t, r, `{{image_url "/media/x.jpg" "thumb"}}`, nil)
	if overrideGot != "/cdn/img/thumb/x.jpg" {
		t.Fatalf("after override: got %q", overrideGot)
	}
	// Empty → passthrough.
	r.SetImageURLPrefix("")
	passGot := renderFunc(t, r, `{{image_url "/media/x.jpg" "thumb"}}`, nil)
	if passGot != "/media/x.jpg" {
		t.Fatalf("empty-prefix passthrough: got %q", passGot)
	}
}
