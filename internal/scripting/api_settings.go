package scripting

import (
	"fmt"

	"vibecms/internal/models"

	"github.com/d5/tengo/v2"
)

// settingsModule returns the cms/settings built-in module.
func (e *ScriptEngine) settingsModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"get": &tengo.UserFunction{Name: "get", Value: e.settingsGet},
		"set": &tengo.UserFunction{Name: "set", Value: e.settingsSet},
		"all": &tengo.UserFunction{Name: "all", Value: e.settingsAll},
	}
}

// settingsGet handles settings.get(key) -> string or undefined
func (e *ScriptEngine) settingsGet(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return tengo.UndefinedValue, fmt.Errorf("settings.get: requires key argument")
	}
	key := getString(args[0])
	if key == "" {
		return tengo.UndefinedValue, nil
	}

	var setting models.SiteSetting
	if err := e.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return tengo.UndefinedValue, nil
	}
	if setting.Value == nil {
		return tengo.UndefinedValue, nil
	}
	return &tengo.String{Value: *setting.Value}, nil
}

// settingsSet handles settings.set(key, value) -> bool
func (e *ScriptEngine) settingsSet(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 2 {
		return tengo.FalseValue, fmt.Errorf("settings.set: requires key and value arguments")
	}
	key := getString(args[0])
	if key == "" {
		return tengo.FalseValue, fmt.Errorf("settings.set: empty key")
	}

	// Prevent scripts from modifying sensitive settings
	sensitive := map[string]bool{
		"email_smtp_password": true,
		"license_key":         true,
		"monitor_bearer_token": true,
	}
	if sensitive[key] {
		return tengo.FalseValue, fmt.Errorf("settings.set: cannot modify sensitive setting %q", key)
	}

	value := getString(args[1])

	result := e.db.Where("key = ?", key).First(&models.SiteSetting{})
	if result.Error != nil {
		// Create new setting
		setting := models.SiteSetting{Key: key, Value: &value}
		if err := e.db.Create(&setting).Error; err != nil {
			return tengo.FalseValue, nil
		}
	} else {
		// Update existing
		if err := e.db.Model(&models.SiteSetting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			return tengo.FalseValue, nil
		}
	}

	return tengo.TrueValue, nil
}

// settingsAll handles settings.all() -> {key: value, ...}
func (e *ScriptEngine) settingsAll(args ...tengo.Object) (tengo.Object, error) {
	var settings []models.SiteSetting
	if err := e.db.Where("is_encrypted = ?", false).Find(&settings).Error; err != nil {
		return &tengo.ImmutableMap{Value: map[string]tengo.Object{}}, nil
	}

	m := make(map[string]tengo.Object, len(settings))
	for _, s := range settings {
		if s.Value != nil {
			m[s.Key] = &tengo.String{Value: *s.Value}
		} else {
			m[s.Key] = tengo.UndefinedValue
		}
	}

	return &tengo.ImmutableMap{Value: m}, nil
}
