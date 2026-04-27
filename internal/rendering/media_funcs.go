package rendering

import "strings"

// TRANSITIONAL — the image_url and image_srcset template helpers below
// know about the media-manager extension's URL routing scheme
// (/media/cache/{size}/...). Per CLAUDE.md, feature-specific code
// belongs in the extension, not core. This file is a deliberate seam:
// the helpers stay reachable from any theme's template, but the URL
// prefix that encodes the extension's convention is configurable
// (TemplateRenderer.SetImageURLPrefix). When the renderer eventually
// gains an extension-registered funcMap surface, this whole file moves
// into extensions/media-manager.

// defaultImageURLPrefix is the path prefix prepended to the size
// segment when transforming /media/foo.jpg → <prefix>/<size>/foo.jpg.
// Empty string means "no media-manager wired up — passthrough" so a
// deployment without the extension renders unbroken (if size-unaware)
// URLs instead of producing 404s pointing at /media/cache/...
const defaultImageURLPrefix = "/media/cache/"

// imageURL produces a sized-variant URL for a media path. The
// transformation only applies to URLs already inside the /media/
// namespace; absolute URLs and other paths are passed through
// unchanged so a theme can mix CDN-hosted and locally-hosted images.
//
// When prefix is empty, the original URL is returned — this is the
// "no media-manager configured" fallback.
func imageURL(prefix, originalURL, sizeName string) string {
	if prefix == "" || !strings.HasPrefix(originalURL, "/media/") {
		return originalURL
	}
	path := strings.TrimPrefix(originalURL, "/media/")
	return strings.TrimSuffix(prefix, "/") + "/" + sizeName + "/" + path
}

// imageSrcset returns a comma-separated list of sized variants suitable
// for an HTML <img srcset> attribute. Returns "" when the input URL
// isn't under /media/ or no prefix is configured — so themes can call
// it unconditionally without producing junk attributes.
func imageSrcset(prefix, originalURL string, sizeNames []string) string {
	if prefix == "" || !strings.HasPrefix(originalURL, "/media/") {
		return ""
	}
	path := strings.TrimPrefix(originalURL, "/media/")
	prefix = strings.TrimSuffix(prefix, "/")
	parts := make([]string, 0, len(sizeNames))
	for _, name := range sizeNames {
		parts = append(parts, prefix+"/"+name+"/"+path)
	}
	return strings.Join(parts, ", ")
}
