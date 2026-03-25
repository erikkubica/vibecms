package cms

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	slugPattern    = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	nonAlphaNum    = regexp.MustCompile(`[^a-z0-9-]+`)
	multiHyphen    = regexp.MustCompile(`-{2,}`)
)

// Slugify converts a title string into a URL-safe slug.
// It lowercases, normalizes unicode, replaces spaces and special characters
// with hyphens, removes consecutive hyphens, and trims leading/trailing hyphens.
func Slugify(title string) string {
	// Normalize unicode characters
	s := norm.NFKD.String(title)

	// Remove non-ASCII characters (accents, etc.)
	var b strings.Builder
	for _, r := range s {
		if r <= unicode.MaxASCII {
			b.WriteRune(r)
		}
	}
	s = b.String()

	// Lowercase
	s = strings.ToLower(s)

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Remove any character that isn't alphanumeric or hyphen
	s = nonAlphaNum.ReplaceAllString(s, "")

	// Collapse consecutive hyphens
	s = multiHyphen.ReplaceAllString(s, "-")

	// Trim leading and trailing hyphens
	s = strings.Trim(s, "-")

	return s
}

// ValidateSlug ensures the slug matches the allowed pattern: lowercase
// alphanumeric segments separated by single hyphens.
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("slug must match pattern: lowercase letters, numbers, and hyphens (e.g. 'my-page-1')")
	}
	return nil
}
