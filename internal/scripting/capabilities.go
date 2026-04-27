package scripting

// defaultThemeCapabilities returns the capability set granted to theme
// Tengo scripts. Themes are first-party, operator-installed code, but the
// capability guard still applies defense-in-depth: themes get content
// read/write (so theme.tengo can register node types and seed menus on
// activation), event/filter/route registration, log/http/file/setting
// access — but NOT raw data store access (data:read/write/delete/exec)
// or media management (extension responsibility).
func defaultThemeCapabilities() map[string]bool {
	return map[string]bool{
		"nodes:read":       true,
		"nodes:write":      true,
		"nodes:delete":     true,
		"nodetypes:read":   true,
		"nodetypes:write":  true,
		"menus:read":       true,
		"menus:write":      true,
		"menus:delete":     true,
		"settings:read":    true,
		"settings:write":   true,
		"users:read":       true,
		"events:emit":      true,
		"events:subscribe": true,
		"filters:register": true,
		"filters:apply":    true,
		"routes:register":  true,
		"log:write":        true,
		"http:fetch":       true,
		"files:write":      true,
		"files:delete":     true,
		"email:send":       true,
	}
}
