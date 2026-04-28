package rendering

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
)

// defaultCacheSize bounds each LRU cache. 1024 templates is generous —
// most sites don't have more than a few hundred distinct ones. Operators
// with massive content can override via VIBECMS_TEMPLATE_CACHE_SIZE.
const defaultCacheSize = 1024

func cacheSize() int {
	if v := os.Getenv("VIBECMS_TEMPLATE_CACHE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultCacheSize
}

// TemplateRenderer handles parsing and rendering of Go html/template files.
// It supports caching in production and always re-parses in dev mode.
//
// Caches are bounded LRU — historically these were unbounded maps that
// grew indefinitely on sites with many distinct templates (block content
// hashes especially). LRU evicts the least recently used so memory stays
// bounded under hostile or just busy conditions.
type TemplateRenderer struct {
	templateDir  string
	cache        *lru.Cache[string, *template.Template]
	layoutCache  *lru.Cache[string, *template.Template]
	blockCache   *lru.Cache[string, *template.Template]
	mu           sync.RWMutex // synchronizes parse-on-cache-miss with concurrent reads
	isDev        bool
	funcMap      template.FuncMap
	eventRunner  func(string, interface{}, []interface{}) template.HTML
	filterRunner func(string, interface{}, interface{}) interface{}
	// imageURLPrefix configures the media-manager URL convention
	// consumed by the image_url / image_srcset funcMap entries.
	// Empty disables size transforms (helpers passthrough). See
	// media_funcs.go for the rationale and migration plan.
	imageURLPrefix string
}

// SetImageURLPrefix overrides the prefix used by the image_url /
// image_srcset template helpers. main.go reads the
// `image_cache_url_prefix` site setting at boot (and on theme reload)
// to wire this — letting the media-manager extension own the URL
// convention without core hardcoding it.
func (r *TemplateRenderer) SetImageURLPrefix(prefix string) {
	r.mu.Lock()
	r.imageURLPrefix = prefix
	r.mu.Unlock()
}

// imagePrefixSnapshot returns the current prefix under read-lock. The
// helper closures use it on every call so a runtime SetImageURLPrefix
// is picked up without rebuilding the funcMap (which would require
// re-parsing every cached template).
func (r *TemplateRenderer) imagePrefixSnapshot() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.imageURLPrefix
}

// ClearCache completely resets the template caches.
func (r *TemplateRenderer) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache.Purge()
	r.layoutCache.Purge()
	r.blockCache.Purge()
}

// SetEventRunner sets the function called by {{event "name" .}} in templates.
// This connects the template engine to the scripting event system.
// The second argument receives the current template context (node, app, user data).
func (r *TemplateRenderer) SetEventRunner(fn func(string, interface{}, []interface{}) template.HTML) {
	r.eventRunner = fn
}

// SetFilterRunner sets the function called by {{filter "name" value}} in templates.
// This connects the template engine to the scripting filter system.
func (r *TemplateRenderer) SetFilterRunner(fn func(string, interface{}, interface{}) interface{}) {
	r.filterRunner = fn
}

// NewTemplateRenderer creates a new TemplateRenderer.
// templateDir is the root directory containing template files.
// isDev controls whether templates are cached (false) or re-parsed on every request (true).
func NewTemplateRenderer(templateDir string, isDev bool) *TemplateRenderer {
	size := cacheSize()
	mainCache, _ := lru.New[string, *template.Template](size)
	layoutCache, _ := lru.New[string, *template.Template](size)
	blockCache, _ := lru.New[string, *template.Template](size)
	r := &TemplateRenderer{
		templateDir:    templateDir,
		cache:          mainCache,
		layoutCache:    layoutCache,
		blockCache:     blockCache,
		isDev:          isDev,
		imageURLPrefix: defaultImageURLPrefix,
	}
	// Default no-op runners (replaced when scripting engine is loaded)
	r.eventRunner = func(name string, ctx interface{}, args []interface{}) template.HTML { return "" }
	r.filterRunner = func(name string, value interface{}, ctx interface{}) interface{} { return value }

	r.funcMap = template.FuncMap{
		"filter": func(name string, value interface{}) interface{} {
			return r.filterRunner(name, value, nil)
		},
		"event": func(name string, args ...interface{}) template.HTML {
			var ctx interface{}
			var extra []interface{}
			if len(args) > 0 {
				ctx = args[0]
			}
			if len(args) > 1 {
				extra = args[1:]
			}
			return r.eventRunner(name, ctx, extra)
		},
		"deref": func(v interface{}) interface{} {
			if v == nil {
				return ""
			}
			switch p := v.(type) {
			case *string:
				if p == nil {
					return ""
				}
				return *p
			case *int:
				if p == nil {
					return 0
				}
				return *p
			default:
				return v
			}
		},
		"safeHTML": func(s interface{}) template.HTML {
			switch v := s.(type) {
			case string:
				return template.HTML(v)
			case template.HTML:
				return v
			default:
				return template.HTML(fmt.Sprintf("%v", v))
			}
		},
		"raw": func(s interface{}) template.HTML {
			switch v := s.(type) {
			case string:
				return template.HTML(v)
			case template.HTML:
				return v
			default:
				return template.HTML(fmt.Sprintf("%v", v))
			}
		},
		"safeURL": func(s interface{}) template.URL {
			switch v := s.(type) {
			case string:
				return template.URL(v)
			case template.URL:
				return v
			default:
				return template.URL(fmt.Sprintf("%v", v))
			}
		},
		"json": func(v interface{}) string {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Sprintf("{\"error\": %q}", err.Error())
			}
			return string(b)
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call: must have even number of arguments")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		"lastWord": func(s string) string {
			s = strings.TrimSpace(s)
			if i := strings.LastIndexAny(s, " \t\n"); i >= 0 {
				return s[i+1:]
			}
			return s
		},
		"beforeLastWord": func(s string) string {
			s = strings.TrimSpace(s)
			if i := strings.LastIndexAny(s, " \t\n"); i >= 0 {
				return s[:i]
			}
			return ""
		},
		"split": func(sep, s string) []string {
			return strings.Split(s, sep)
		},
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"list": func(items ...interface{}) []interface{} {
			return items
		},
		"seq": func(n int) []int {
			if n <= 0 {
				return nil
			}
			out := make([]int, n)
			for i := range out {
				out[i] = i
			}
			return out
		},
		"image_url": func(originalURL string, sizeName string) string {
			return imageURL(r.imagePrefixSnapshot(), originalURL, sizeName)
		},
		"image_srcset": func(originalURL string, sizeNames ...string) string {
			return imageSrcset(r.imagePrefixSnapshot(), originalURL, sizeNames)
		},
	}
	return r
}

// Render parses and executes a layout + page template combination.
// layoutName is relative to templateDir, e.g. "layouts/base.html".
// pageName is relative to templateDir, e.g. "public/home.html".
// The page template must define blocks expected by the layout (e.g. "title", "content").
func (r *TemplateRenderer) Render(w io.Writer, layoutName, pageName string, data interface{}) error {
	cacheKey := layoutName + ":" + pageName

	if !r.isDev {
		if tmpl, ok := r.cache.Get(cacheKey); ok {
			return tmpl.Execute(w, data)
		}
	}

	layoutPath := filepath.Join(r.templateDir, layoutName)
	pagePath := filepath.Join(r.templateDir, pageName)

	tmpl, err := template.New(filepath.Base(layoutName)).Funcs(r.funcMap).ParseFiles(layoutPath, pagePath)
	if err != nil {
		return fmt.Errorf("template parse error [%s + %s]: %w", layoutName, pageName, err)
	}

	if !r.isDev {
		r.cache.Add(cacheKey, tmpl)
	}

	return tmpl.Execute(w, data)
}

// RenderPage is a convenience method that renders a page template with the
// default base layout ("layouts/base.html").
func (r *TemplateRenderer) RenderPage(w io.Writer, pageName string, data interface{}) error {
	return r.Render(w, "layouts/base.html", pageName, data)
}

// RenderFragment renders only the "content" block from a page template,
// without wrapping in any layout. Used when the site layout engine handles the wrapper.
func (r *TemplateRenderer) RenderFragment(w io.Writer, pageName string, data interface{}) error {
	pagePath := filepath.Join(r.templateDir, pageName)

	// Parse with a minimal base that just executes the content block
	base := `{{template "content" .}}`
	tmpl, err := template.New("fragment").Funcs(r.funcMap).Parse(base)
	if err != nil {
		return fmt.Errorf("fragment base parse error: %w", err)
	}
	tmpl, err = tmpl.ParseFiles(pagePath)
	if err != nil {
		return fmt.Errorf("fragment parse error [%s]: %w", pageName, err)
	}
	return tmpl.Execute(w, data)
}

// RenderParsed renders a template from a string (code), caching the parsed version.
// The cacheKey should uniquely identify the template content (e.g. slug + code hash).
func (r *TemplateRenderer) RenderParsed(w io.Writer, cacheKey string, code string, data interface{}, funcMap template.FuncMap) error {
	var tmpl *template.Template

	if !r.isDev {
		if cachedTmpl, ok := r.blockCache.Get(cacheKey); ok {
			tmpl, _ = cachedTmpl.Clone()
		}
	}

	if tmpl == nil {
		// Define a dummy renderLayoutBlock if it's missing from the funcMap
		// to allow parsing templates that reference it.
		fullFuncMap := template.FuncMap{}
		for k, v := range r.funcMap {
			fullFuncMap[k] = v
		}
		for k, v := range funcMap {
			fullFuncMap[k] = v
		}
		if _, ok := fullFuncMap["renderLayoutBlock"]; !ok {
			fullFuncMap["renderLayoutBlock"] = func(s string) template.HTML { return "" }
		}

		var err error
		tmpl, err = template.New(cacheKey).Funcs(fullFuncMap).Parse(code)
		if err != nil {
			return fmt.Errorf("template parse error: %w", err)
		}

		if !r.isDev {
			clone, _ := tmpl.Clone()
			r.blockCache.Add(cacheKey, clone)
		}
	}

	// Apply the actual funcMap with the correct closures for this request
	fullFuncMap := template.FuncMap{}
	for k, v := range r.funcMap {
		fullFuncMap[k] = v
	}
	for k, v := range funcMap {
		fullFuncMap[k] = v
	}
	tmpl.Funcs(fullFuncMap)

	return tmpl.Execute(w, data)
}

// RenderLayout renders a layout template_code string (from the DB) with a
// blockResolver that supports the renderLayoutBlock template function.
// partialData is an optional map of partial slug -> field values that gets
// injected as .partial in each layout block's template context.
func (r *TemplateRenderer) RenderLayout(w io.Writer, templateCode string, data interface{}, blockResolver func(slug string) (string, error), partialData ...map[string]map[string]interface{}) error {
	// Build partial data lookup
	var pData map[string]map[string]interface{}
	if len(partialData) > 0 && partialData[0] != nil {
		pData = partialData[0]
	}

	depth := 0
	maxDepth := 5

	var renderBlock func(slug string) template.HTML
	renderBlock = func(slug string) template.HTML {
		depth++
		if depth > maxDepth {
			log.Printf("WARN: renderLayoutBlock recursion limit reached for '%s'", slug)
			depth--
			return ""
		}
		defer func() { depth-- }()

		code, err := blockResolver(slug)
		if err != nil {
			log.Printf("WARN: layout block '%s' not found: %v", slug, err)
			return ""
		}

		// Build per-block context: clone base data and inject .partial
		blockData := data
		if pData != nil {
			if baseMap, ok := data.(map[string]interface{}); ok {
				merged := make(map[string]interface{}, len(baseMap)+1)
				for k, v := range baseMap {
					merged[k] = v
				}
				if fields, ok := pData[slug]; ok {
					merged["partial"] = fields
				} else {
					merged["partial"] = map[string]interface{}{}
				}
				blockData = merged
			}
		}

		var buf bytes.Buffer
		cacheKey := "block:" + slug + ":" + code
		err = r.RenderParsed(&buf, cacheKey, code, blockData, template.FuncMap{
			"renderLayoutBlock": renderBlock,
		})
		if err != nil {
			log.Printf("WARN: template render error in block '%s': %v", slug, err)
			return ""
		}
		return template.HTML(buf.String())
	}

	cacheKey := "layout:" + templateCode
	return r.RenderParsed(w, cacheKey, templateCode, data, template.FuncMap{
		"renderLayoutBlock": renderBlock,
	})
}
