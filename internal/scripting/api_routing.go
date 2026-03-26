package scripting

import (
	"fmt"
	"strconv"

	"vibecms/internal/models"

	"github.com/d5/tengo/v2"
)

// routingModule returns the cms/routing built-in module, scoped to the current
// page render context. Each hook script execution gets a fresh module with
// closures that capture the current node/app/user data.
//
// Available outside of render context too (functions return undefined/false).
//
// Usage:
//
//	routing := import("cms/routing")
//	if routing.is_homepage() { ... }
//	if routing.is_404() { ... }
//	node := routing.get_node()
func (e *ScriptEngine) routingModule(renderCtx interface{}) map[string]tengo.Object {
	var nodeCtx map[string]interface{}
	var appCtx map[string]interface{}
	var userCtx map[string]interface{}

	if ctx, ok := renderCtx.(map[string]interface{}); ok {
		if n, ok := ctx["node"].(map[string]interface{}); ok {
			nodeCtx = n
		}
		if a, ok := ctx["app"].(map[string]interface{}); ok {
			appCtx = a
		}
		if u, ok := ctx["user"].(map[string]interface{}); ok {
			userCtx = u
		}
	}

	return map[string]tengo.Object{
		// Node info
		"get_node":         &tengo.UserFunction{Name: "get_node", Value: routingGetNode(nodeCtx)},
		"current_url":      &tengo.UserFunction{Name: "current_url", Value: routingSimpleString(nodeCtx, "full_url")},
		"current_language": &tengo.UserFunction{Name: "current_language", Value: routingSimpleString(nodeCtx, "language_code")},
		"current_node_type": &tengo.UserFunction{Name: "current_node_type", Value: routingSimpleString(nodeCtx, "node_type")},

		// Checks
		"is_homepage":  &tengo.UserFunction{Name: "is_homepage", Value: e.routingIsHomepage(nodeCtx)},
		"is_404":       &tengo.UserFunction{Name: "is_404", Value: routingIs404(nodeCtx)},
		"is_node_type": &tengo.UserFunction{Name: "is_node_type", Value: routingIsNodeType(nodeCtx)},
		"is_language":  &tengo.UserFunction{Name: "is_language", Value: routingIsLanguage(nodeCtx)},
		"is_slug":      &tengo.UserFunction{Name: "is_slug", Value: routingIsSlug(nodeCtx)},
		"is_logged_in": &tengo.UserFunction{Name: "is_logged_in", Value: routingIsLoggedIn(userCtx)},

		// User & app
		"get_user":     &tengo.UserFunction{Name: "get_user", Value: routingGetUser(userCtx)},
		"site_setting": &tengo.UserFunction{Name: "site_setting", Value: e.routingSiteSetting(appCtx)},
	}
}

// --- Node accessors ---

// routingGetNode returns the current node as a map, or undefined if not in a render.
func routingGetNode(nodeCtx map[string]interface{}) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if nodeCtx == nil {
			return tengo.UndefinedValue, nil
		}
		return goToTengo(nodeCtx), nil
	}
}

// routingSimpleString returns a closure that reads a single string field from nodeCtx.
func routingSimpleString(nodeCtx map[string]interface{}, field string) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if nodeCtx == nil {
			return tengo.UndefinedValue, nil
		}
		if val, ok := nodeCtx[field].(string); ok {
			return &tengo.String{Value: val}, nil
		}
		return tengo.UndefinedValue, nil
	}
}

// --- Page type checks ---

// routingIsHomepage checks whether the current page is the homepage.
// Language-aware: resolves through translation groups so all translations
// of the homepage are also considered the homepage.
func (e *ScriptEngine) routingIsHomepage(nodeCtx map[string]interface{}) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if nodeCtx == nil {
			return tengo.FalseValue, nil
		}

		currentID := extractNodeID(nodeCtx)
		if currentID <= 0 {
			return tengo.FalseValue, nil
		}

		// Get homepage_node_id from settings
		var setting models.SiteSetting
		if err := e.db.Where("key = ?", "homepage_node_id").First(&setting).Error; err != nil || setting.Value == nil {
			return tengo.FalseValue, nil
		}
		homepageID, err := strconv.Atoi(*setting.Value)
		if err != nil || homepageID <= 0 {
			return tengo.FalseValue, nil
		}

		// Direct match
		if currentID == homepageID {
			return tengo.TrueValue, nil
		}

		// Translation group match
		var homepage models.ContentNode
		if err := e.db.Select("id, translation_group_id").First(&homepage, homepageID).Error; err != nil {
			return tengo.FalseValue, nil
		}
		if homepage.TranslationGroupID == nil || *homepage.TranslationGroupID == "" {
			return tengo.FalseValue, nil
		}

		var currentNode models.ContentNode
		if err := e.db.Select("id, translation_group_id").First(&currentNode, currentID).Error; err != nil {
			return tengo.FalseValue, nil
		}
		if currentNode.TranslationGroupID == nil || *currentNode.TranslationGroupID == "" {
			return tengo.FalseValue, nil
		}

		if *currentNode.TranslationGroupID == *homepage.TranslationGroupID {
			return tengo.TrueValue, nil
		}

		return tengo.FalseValue, nil
	}
}

// routingIs404 checks whether the current page is a 404 error page.
// 404 pages have slug "404" and no ID.
func routingIs404(nodeCtx map[string]interface{}) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if nodeCtx == nil {
			return tengo.TrueValue, nil // no context = not rendering a page
		}
		// 404 pages have slug "404" and id 0
		if slug, ok := nodeCtx["slug"].(string); ok && slug == "404" {
			return tengo.TrueValue, nil
		}
		if extractNodeID(nodeCtx) <= 0 {
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	}
}

// routingIsNodeType checks current page against a given node type.
// Usage: routing.is_node_type("page"), routing.is_node_type("post")
func routingIsNodeType(nodeCtx map[string]interface{}) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return tengo.FalseValue, fmt.Errorf("routing.is_node_type: requires type argument")
		}
		if nodeCtx == nil {
			return tengo.FalseValue, nil
		}
		expected := getString(args[0])
		if nt, ok := nodeCtx["node_type"].(string); ok && nt == expected {
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	}
}

// routingIsLanguage checks current page against a given language code.
func routingIsLanguage(nodeCtx map[string]interface{}) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return tengo.FalseValue, fmt.Errorf("routing.is_language: requires language code argument")
		}
		if nodeCtx == nil {
			return tengo.FalseValue, nil
		}
		expected := getString(args[0])
		if lang, ok := nodeCtx["language_code"].(string); ok && lang == expected {
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	}
}

// routingIsSlug checks current page against a given slug.
// Usage: routing.is_slug("about"), routing.is_slug("contact")
func routingIsSlug(nodeCtx map[string]interface{}) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return tengo.FalseValue, fmt.Errorf("routing.is_slug: requires slug argument")
		}
		if nodeCtx == nil {
			return tengo.FalseValue, nil
		}
		expected := getString(args[0])
		if slug, ok := nodeCtx["slug"].(string); ok && slug == expected {
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	}
}

// --- User checks ---

// routingIsLoggedIn checks if the current user is logged in.
func routingIsLoggedIn(userCtx map[string]interface{}) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if userCtx == nil {
			return tengo.FalseValue, nil
		}
		if loggedIn, ok := userCtx["logged_in"].(bool); ok && loggedIn {
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	}
}

// routingGetUser returns the current user data or undefined.
func routingGetUser(userCtx map[string]interface{}) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if userCtx == nil {
			return tengo.UndefinedValue, nil
		}
		return goToTengo(userCtx), nil
	}
}

// --- App / settings ---

// routingSiteSetting returns a site setting by key.
// Tries cached app context first, falls back to DB.
func (e *ScriptEngine) routingSiteSetting(appCtx map[string]interface{}) tengo.CallableFunc {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return tengo.UndefinedValue, fmt.Errorf("routing.site_setting: requires key argument")
		}
		key := getString(args[0])

		// Try from app context first (cached for this request)
		if appCtx != nil {
			if settings, ok := appCtx["settings"].(map[string]interface{}); ok {
				if val, ok := settings[key]; ok {
					if s, ok := val.(string); ok {
						return &tengo.String{Value: s}, nil
					}
				}
			}
		}

		// Fall back to DB
		var setting models.SiteSetting
		if err := e.db.Where("key = ? AND is_encrypted = ?", key, false).First(&setting).Error; err != nil {
			return tengo.UndefinedValue, nil
		}
		if setting.Value == nil {
			return tengo.UndefinedValue, nil
		}
		return &tengo.String{Value: *setting.Value}, nil
	}
}

// extractNodeID gets the node ID from the template context node map.
func extractNodeID(nodeCtx map[string]interface{}) int {
	if nodeCtx == nil {
		return 0
	}
	if id, ok := nodeCtx["id"]; ok {
		switch v := id.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}
