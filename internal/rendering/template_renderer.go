package rendering

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"path/filepath"
	"sync"
)

// TemplateRenderer handles parsing and rendering of Go html/template files.
// It supports caching in production and always re-parses in dev mode.
type TemplateRenderer struct {
	templateDir  string
	cache        map[string]*template.Template
	layoutCache  map[string]*template.Template
	blockCache   map[string]*template.Template
	mu           sync.RWMutex
	isDev        bool
	funcMap      template.FuncMap
	eventRunner  func(string, interface{}, []interface{}) template.HTML
	filterRunner func(string, interface{}, interface{}) interface{}
}

// ClearCache completely resets the template caches.
func (r *TemplateRenderer) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = make(map[string]*template.Template)
	r.layoutCache = make(map[string]*template.Template)
	r.blockCache = make(map[string]*template.Template)
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
	r := &TemplateRenderer{
		templateDir: templateDir,
		cache:       make(map[string]*template.Template),
		layoutCache: make(map[string]*template.Template),
		blockCache:  make(map[string]*template.Template),
		isDev:       isDev,
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
		"json": func(v interface{}) string {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Sprintf("{\"error\": %q}", err.Error())
			}
			return string(b)
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
		r.mu.RLock()
		tmpl, ok := r.cache[cacheKey]
		r.mu.RUnlock()
		if ok {
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
		r.mu.Lock()
		r.cache[cacheKey] = tmpl
		r.mu.Unlock()
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
		r.mu.RLock()
		cachedTmpl, ok := r.blockCache[cacheKey]
		r.mu.RUnlock()
		if ok {
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
			r.mu.Lock()
			clone, _ := tmpl.Clone()
			r.blockCache[cacheKey] = clone
			r.mu.Unlock()
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
func (r *TemplateRenderer) RenderLayout(w io.Writer, templateCode string, data interface{}, blockResolver func(slug string) (string, error)) error {
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

		var buf bytes.Buffer
		cacheKey := "block:" + slug + ":" + code
		err = r.RenderParsed(&buf, cacheKey, code, data, template.FuncMap{
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
