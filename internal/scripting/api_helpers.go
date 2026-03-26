package scripting

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/d5/tengo/v2"
	"golang.org/x/text/unicode/norm"
)

// helpersModule returns the cms/helpers built-in module.
// Provides utility functions commonly needed in theme scripts.
//
// Usage:
//
//	helpers := import("cms/helpers")
//	slug := helpers.slugify("Hello World")
//	short := helpers.truncate("Long text...", 50)
func helpersModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"slugify":      &tengo.UserFunction{Name: "slugify", Value: helpersSlugify},
		"truncate":     &tengo.UserFunction{Name: "truncate", Value: helpersTruncate},
		"strip_html":   &tengo.UserFunction{Name: "strip_html", Value: helpersStripHTML},
		"escape_html":  &tengo.UserFunction{Name: "escape_html", Value: helpersEscapeHTML},
		"lower":        &tengo.UserFunction{Name: "lower", Value: helpersLower},
		"upper":        &tengo.UserFunction{Name: "upper", Value: helpersUpper},
		"title_case":   &tengo.UserFunction{Name: "title_case", Value: helpersTitleCase},
		"contains":     &tengo.UserFunction{Name: "contains", Value: helpersContains},
		"starts_with":  &tengo.UserFunction{Name: "starts_with", Value: helpersStartsWith},
		"ends_with":    &tengo.UserFunction{Name: "ends_with", Value: helpersEndsWith},
		"replace":      &tengo.UserFunction{Name: "replace", Value: helpersReplace},
		"split":        &tengo.UserFunction{Name: "split", Value: helpersSplit},
		"join":         &tengo.UserFunction{Name: "join", Value: helpersJoin},
		"trim":         &tengo.UserFunction{Name: "trim", Value: helpersTrim},
		"md5":          &tengo.UserFunction{Name: "md5", Value: helpersMD5},
		"repeat":       &tengo.UserFunction{Name: "repeat", Value: helpersRepeat},
		"word_count":   &tengo.UserFunction{Name: "word_count", Value: helpersWordCount},
		"excerpt":      &tengo.UserFunction{Name: "excerpt", Value: helpersExcerpt},
		"pluralize":    &tengo.UserFunction{Name: "pluralize", Value: helpersPluralize},
		"default":      &tengo.UserFunction{Name: "default", Value: helpersDefault},
	}
}

var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)
var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// helpersSlugify converts a string to a URL-safe slug.
// slugify("Hello World!") -> "hello-world"
func helpersSlugify(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return &tengo.String{Value: ""}, nil
	}
	s := getString(args[0])

	// Normalize unicode characters
	s = norm.NFKD.String(s)

	// Remove non-ASCII characters, lowercase
	var result strings.Builder
	for _, r := range s {
		if r <= unicode.MaxASCII {
			result.WriteRune(unicode.ToLower(r))
		}
	}
	slug := result.String()

	// Replace non-alphanumeric sequences with hyphens
	slug = nonAlphanumRe.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	return &tengo.String{Value: slug}, nil
}

// helpersTruncate truncates a string to max_len, appending "..." if truncated.
// truncate("Hello World", 5) -> "Hello..."
// Optional 3rd arg: custom suffix (default "...")
func helpersTruncate(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.UndefinedValue, fmt.Errorf("helpers.truncate: requires string and max_len arguments")
	}
	s := getString(args[0])
	maxLen := getInt(args[1])
	suffix := "..."
	if len(args) > 2 {
		suffix = getString(args[2])
	}

	if maxLen <= 0 || len(s) <= maxLen {
		return &tengo.String{Value: s}, nil
	}
	return &tengo.String{Value: s[:maxLen] + suffix}, nil
}

// helpersStripHTML removes all HTML tags from a string.
func helpersStripHTML(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return &tengo.String{Value: ""}, nil
	}
	s := getString(args[0])
	return &tengo.String{Value: htmlTagRe.ReplaceAllString(s, "")}, nil
}

// helpersEscapeHTML escapes HTML special characters.
func helpersEscapeHTML(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return &tengo.String{Value: ""}, nil
	}
	s := getString(args[0])
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return &tengo.String{Value: s}, nil
}

func helpersLower(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return &tengo.String{Value: ""}, nil
	}
	return &tengo.String{Value: strings.ToLower(getString(args[0]))}, nil
}

func helpersUpper(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return &tengo.String{Value: ""}, nil
	}
	return &tengo.String{Value: strings.ToUpper(getString(args[0]))}, nil
}

func helpersTitleCase(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return &tengo.String{Value: ""}, nil
	}
	return &tengo.String{Value: strings.Title(getString(args[0]))}, nil
}

func helpersContains(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.FalseValue, nil
	}
	if strings.Contains(getString(args[0]), getString(args[1])) {
		return tengo.TrueValue, nil
	}
	return tengo.FalseValue, nil
}

func helpersStartsWith(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.FalseValue, nil
	}
	if strings.HasPrefix(getString(args[0]), getString(args[1])) {
		return tengo.TrueValue, nil
	}
	return tengo.FalseValue, nil
}

func helpersEndsWith(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.FalseValue, nil
	}
	if strings.HasSuffix(getString(args[0]), getString(args[1])) {
		return tengo.TrueValue, nil
	}
	return tengo.FalseValue, nil
}

// helpersReplace replaces all occurrences of old with new.
// replace("hello world", "world", "tengo")
func helpersReplace(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 3 {
		return tengo.UndefinedValue, fmt.Errorf("helpers.replace: requires string, old, new arguments")
	}
	return &tengo.String{Value: strings.ReplaceAll(getString(args[0]), getString(args[1]), getString(args[2]))}, nil
}

// helpersSplit splits a string by separator.
func helpersSplit(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.UndefinedValue, fmt.Errorf("helpers.split: requires string and separator arguments")
	}
	parts := strings.Split(getString(args[0]), getString(args[1]))
	arr := make([]tengo.Object, len(parts))
	for i, p := range parts {
		arr[i] = &tengo.String{Value: p}
	}
	return &tengo.ImmutableArray{Value: arr}, nil
}

// helpersJoin joins an array of strings with separator.
func helpersJoin(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.UndefinedValue, fmt.Errorf("helpers.join: requires array and separator arguments")
	}
	var parts []string
	switch v := args[0].(type) {
	case *tengo.Array:
		for _, obj := range v.Value {
			parts = append(parts, getString(obj))
		}
	case *tengo.ImmutableArray:
		for _, obj := range v.Value {
			parts = append(parts, getString(obj))
		}
	}
	sep := getString(args[1])
	return &tengo.String{Value: strings.Join(parts, sep)}, nil
}

func helpersTrim(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return &tengo.String{Value: ""}, nil
	}
	return &tengo.String{Value: strings.TrimSpace(getString(args[0]))}, nil
}

// helpersMD5 returns the MD5 hex digest of a string (useful for Gravatar URLs, etc.)
func helpersMD5(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return &tengo.String{Value: ""}, nil
	}
	hash := md5.Sum([]byte(getString(args[0])))
	return &tengo.String{Value: hex.EncodeToString(hash[:])}, nil
}

// helpersRepeat repeats a string n times.
func helpersRepeat(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.UndefinedValue, fmt.Errorf("helpers.repeat: requires string and count arguments")
	}
	n := getInt(args[1])
	if n < 0 || n > 1000 {
		n = 0
	}
	return &tengo.String{Value: strings.Repeat(getString(args[0]), n)}, nil
}

// helpersWordCount counts words in a string.
func helpersWordCount(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return &tengo.Int{Value: 0}, nil
	}
	words := strings.Fields(getString(args[0]))
	return &tengo.Int{Value: int64(len(words))}, nil
}

// helpersExcerpt extracts a word-aware excerpt from text.
// excerpt("Hello beautiful world of Go", 3) -> "Hello beautiful world..."
func helpersExcerpt(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.UndefinedValue, fmt.Errorf("helpers.excerpt: requires text and word_count arguments")
	}
	text := getString(args[0])
	// Strip HTML first
	text = htmlTagRe.ReplaceAllString(text, "")
	maxWords := getInt(args[1])
	if maxWords <= 0 {
		return &tengo.String{Value: ""}, nil
	}

	words := strings.Fields(text)
	if len(words) <= maxWords {
		return &tengo.String{Value: strings.Join(words, " ")}, nil
	}
	return &tengo.String{Value: strings.Join(words[:maxWords], " ") + "..."}, nil
}

// helpersPluralize returns singular or plural form based on count.
// pluralize(1, "item", "items") -> "item"
// pluralize(5, "item", "items") -> "items"
func helpersPluralize(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 3 {
		return tengo.UndefinedValue, fmt.Errorf("helpers.pluralize: requires count, singular, plural arguments")
	}
	count := getInt(args[0])
	if count == 1 {
		return args[1], nil
	}
	return args[2], nil
}

// helpersDefault returns the first non-empty/non-undefined argument.
// default(val, "fallback") -> val if truthy, else "fallback"
func helpersDefault(args ...tengo.Object) (tengo.Object, error) {
	for _, arg := range args {
		if arg == nil || arg == tengo.UndefinedValue {
			continue
		}
		if s, ok := arg.(*tengo.String); ok && s.Value == "" {
			continue
		}
		if arg.IsFalsy() {
			continue
		}
		return arg, nil
	}
	if len(args) > 0 {
		return args[len(args)-1], nil
	}
	return tengo.UndefinedValue, nil
}
