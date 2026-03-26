package scripting

import (
	"fmt"

	"vibecms/internal/models"

	"github.com/d5/tengo/v2"
)

// menusModule returns the cms/menus built-in module.
func (e *ScriptEngine) menusModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"get":  &tengo.UserFunction{Name: "get", Value: e.menusGet},
		"list": &tengo.UserFunction{Name: "list", Value: e.menusList},
	}
}

// menusGet handles menus.get(slug) -> menu map with items
// Optional second argument: language_id (int)
func (e *ScriptEngine) menusGet(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return tengo.UndefinedValue, fmt.Errorf("menus.get: requires slug argument")
	}
	slug := getString(args[0])
	if slug == "" {
		return tengo.UndefinedValue, nil
	}

	var langID *int
	if len(args) > 1 {
		lid := getInt(args[1])
		if lid > 0 {
			langID = &lid
		}
	}

	menu, err := e.menuSvc.Resolve(slug, langID)
	if err != nil {
		return tengo.UndefinedValue, nil
	}

	return menuToTengo(menu), nil
}

// menusList handles menus.list() -> [menu maps]
func (e *ScriptEngine) menusList(args ...tengo.Object) (tengo.Object, error) {
	menus, err := e.menuSvc.List(nil)
	if err != nil {
		return &tengo.ImmutableArray{Value: []tengo.Object{}}, nil
	}

	items := make([]tengo.Object, len(menus))
	for i := range menus {
		items[i] = menuToTengo(&menus[i])
	}

	return &tengo.ImmutableArray{Value: items}, nil
}

// menuToTengo converts a Menu to a Tengo ImmutableMap.
func menuToTengo(menu *models.Menu) tengo.Object {
	m := map[string]tengo.Object{
		"id":   &tengo.Int{Value: int64(menu.ID)},
		"slug": &tengo.String{Value: menu.Slug},
		"name": &tengo.String{Value: menu.Name},
	}

	if menu.LanguageID != nil {
		m["language_id"] = &tengo.Int{Value: int64(*menu.LanguageID)}
	} else {
		m["language_id"] = tengo.UndefinedValue
	}

	m["items"] = menuItemsToTengo(menu.Items)
	return &tengo.ImmutableMap{Value: m}
}

// menuItemsToTengo converts menu items to Tengo arrays recursively.
func menuItemsToTengo(items []models.MenuItem) tengo.Object {
	arr := make([]tengo.Object, len(items))
	for i, item := range items {
		m := map[string]tengo.Object{
			"id":        &tengo.Int{Value: int64(item.ID)},
			"title":     &tengo.String{Value: item.Title},
			"item_type": &tengo.String{Value: item.ItemType},
			"url":       &tengo.String{Value: item.URL},
			"target":    &tengo.String{Value: item.Target},
			"css_class": &tengo.String{Value: item.CSSClass},
			"children":  menuItemsToTengo(item.Children),
		}
		if item.NodeID != nil {
			m["node_id"] = &tengo.Int{Value: int64(*item.NodeID)}
		} else {
			m["node_id"] = tengo.UndefinedValue
		}
		arr[i] = &tengo.ImmutableMap{Value: m}
	}
	return &tengo.ImmutableArray{Value: arr}
}
