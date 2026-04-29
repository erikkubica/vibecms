package cms

import (
	"fmt"
	"html"
	"html/template"
	"strings"

	"squilla/internal/models"
)

// BuildHeadMeta composes the <meta> tag block emitted in the layout's <head>
// for SEO and social previews. Per-node SEO wins; site-wide defaults from
// site_settings fill in the gaps. Returns template.HTML so callers can drop
// it directly into a Go template (auto-escape would otherwise break the
// generated tags).
//
// Outputs:
//   - canonical URL
//   - meta robots (when site-wide indexing is disabled)
//   - og:title / og:description / og:image / og:url / og:type / og:site_name
//   - twitter:card, twitter:site, twitter:title, twitter:description,
//     twitter:image
//   - hreflang alternates per translation (when translations exist)
//
// If site_url isn't configured the per-page URL falls back to a relative
// path; absolute URLs are preferred but the kernel can't fabricate one.
func BuildHeadMeta(node *models.ContentNode, seo map[string]interface{}, settings map[string]string, translations []map[string]interface{}, languages []models.Language) template.HTML {
	siteURL := strings.TrimRight(settings["site_url"], "/")
	siteName := stringOr(settings["seo_og_site_name"], settings["site_name"])

	title := pickString(seo, "meta_title")
	if title == "" {
		title = stringOr(node.Title, settings["seo_default_meta_title"])
	}
	if title == "" {
		title = settings["site_name"]
	}

	desc := pickString(seo, "meta_description")
	if desc == "" {
		desc = stringOr(node.Excerpt, settings["seo_default_meta_description"])
	}
	if desc == "" {
		desc = settings["site_description"]
	}

	ogImage := pickString(seo, "og_image")
	if ogImage == "" {
		ogImage = featuredImageURL(node.FeaturedImage)
	}
	if ogImage == "" {
		ogImage = settings["seo_default_og_image"]
	}
	ogImage = absoluteURL(siteURL, ogImage)

	canonical := absoluteURL(siteURL, node.FullURL)

	twitterHandle := strings.TrimSpace(settings["seo_twitter_handle"])
	if twitterHandle != "" && !strings.HasPrefix(twitterHandle, "@") {
		twitterHandle = "@" + twitterHandle
	}

	var b strings.Builder
	tag := func(format string, args ...any) {
		b.WriteString(fmt.Sprintf(format, args...))
		b.WriteByte('\n')
	}

	if canonical != "" {
		tag(`<link rel="canonical" href="%s">`, html.EscapeString(canonical))
	}

	// Meta robots — always emitted so the indexing intent is explicit
	// in the rendered HTML, not just in the X-Robots-Tag response
	// header. Allowed mode pairs `index, follow` with the modern
	// rich-snippet opt-ins (max-image-preview, max-snippet,
	// max-video-preview) so Google can surface large previews; without
	// these flags Google falls back to small thumbnails.
	tag(`<meta name="robots" content="%s">`, html.EscapeString(robotsDirective(settings)))

	// Open Graph
	if title != "" {
		tag(`<meta property="og:title" content="%s">`, html.EscapeString(title))
	}
	if desc != "" {
		tag(`<meta property="og:description" content="%s">`, html.EscapeString(desc))
	}
	if canonical != "" {
		tag(`<meta property="og:url" content="%s">`, html.EscapeString(canonical))
	}
	tag(`<meta property="og:type" content="article">`)
	if siteName != "" {
		tag(`<meta property="og:site_name" content="%s">`, html.EscapeString(siteName))
	}
	if ogImage != "" {
		tag(`<meta property="og:image" content="%s">`, html.EscapeString(ogImage))
	}
	if node.LanguageCode != "" {
		tag(`<meta property="og:locale" content="%s">`, html.EscapeString(node.LanguageCode))
	}

	// Twitter
	cardType := "summary"
	if ogImage != "" {
		cardType = "summary_large_image"
	}
	tag(`<meta name="twitter:card" content="%s">`, cardType)
	if twitterHandle != "" {
		tag(`<meta name="twitter:site" content="%s">`, html.EscapeString(twitterHandle))
	}
	if title != "" {
		tag(`<meta name="twitter:title" content="%s">`, html.EscapeString(title))
	}
	if desc != "" {
		tag(`<meta name="twitter:description" content="%s">`, html.EscapeString(desc))
	}
	if ogImage != "" {
		tag(`<meta name="twitter:image" content="%s">`, html.EscapeString(ogImage))
	}

	// hreflang alternates — emit one for each translation (including this
	// node) so search engines and social previews surface the right
	// locale.
	if len(translations) > 0 && siteURL != "" {
		// The current node is implicit in translations only when the helper
		// includes it; emit explicitly so search engines see the canonical
		// pair.
		if node.LanguageCode != "" && node.FullURL != "" {
			tag(`<link rel="alternate" hreflang="%s" href="%s">`,
				html.EscapeString(node.LanguageCode), html.EscapeString(canonical))
		}
		for _, tr := range translations {
			lc, _ := tr["language_code"].(string)
			full, _ := tr["full_url"].(string)
			if lc == "" || full == "" || lc == node.LanguageCode {
				continue
			}
			tag(`<link rel="alternate" hreflang="%s" href="%s">`,
				html.EscapeString(lc), html.EscapeString(absoluteURL(siteURL, full)))
		}
	}

	return template.HTML(b.String())
}

// robotsDirective returns the canonical robots directive driven by the
// seo_robots_index site setting. Used by both the X-Robots-Tag header
// middleware and the <meta name=robots> emitted in head_meta so the two
// signals can never disagree on a given page.
//
// "false" → noindex,nofollow (operator opted out — staging or hidden site).
// anything else (including empty / unset) → index,follow + modern preview
// opt-ins so search engines render rich snippets / large images / long
// snippets when our content is good enough to deserve them.
func robotsDirective(settings map[string]string) string {
	if settings["seo_robots_index"] == "false" {
		return "noindex, nofollow"
	}
	return "index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1"
}

// mapClone returns a shallow copy of a string→string map. Used when a
// caller needs to override a single setting (e.g. force noindex on 404s)
// without mutating the cached site-settings map.
func mapClone(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// stringOr returns primary unless empty, otherwise fallback.
func stringOr(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

// pickString reads a string value from a map, returning "" when missing or
// non-string (the seo map comes from JSONB and may carry mixed types).
func pickString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// featuredImageURL extracts a `url` string from the featured_image JSONB
// payload. Tolerates the legacy bare-string shape and the canonical
// {url, alt, ...} object shape.
func featuredImageURL(fi models.JSONB) string {
	if len(fi) == 0 {
		return ""
	}
	// JSONB is bytes; the raw text is good enough for a cheap probe — full
	// unmarshal happens in the per-node hydration upstream. Looking for
	// "url":"..." avoids a second decode here.
	s := string(fi)
	idx := strings.Index(s, `"url"`)
	if idx < 0 {
		return ""
	}
	tail := s[idx+5:]
	colon := strings.IndexByte(tail, ':')
	if colon < 0 {
		return ""
	}
	tail = strings.TrimSpace(tail[colon+1:])
	if !strings.HasPrefix(tail, `"`) {
		return ""
	}
	tail = tail[1:]
	end := strings.IndexByte(tail, '"')
	if end < 0 {
		return ""
	}
	return tail[:end]
}

// absoluteURL returns siteURL+path when path is relative, path itself when
// it's already absolute, and "" when the input is empty. Empty siteURL keeps
// relative paths as-is.
func absoluteURL(siteURL, path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "//") {
		return path
	}
	if siteURL == "" {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return siteURL + path
}
