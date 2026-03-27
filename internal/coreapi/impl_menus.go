package coreapi

import (
	"context"
	"strings"

	"vibecms/internal/models"
)

func (c *coreImpl) GetMenu(_ context.Context, slug string) (*Menu, error) {
	var m models.Menu
	if err := c.db.Where("slug = ?", slug).First(&m).Error; err != nil {
		return nil, NewNotFound("menu", slug)
	}

	resolved, err := c.menuSvc.GetByID(m.ID)
	if err != nil {
		return nil, NewInternal(err.Error())
	}

	result := menuFromModel(resolved)
	return &result, nil
}

func (c *coreImpl) GetMenus(_ context.Context) ([]*Menu, error) {
	list, err := c.menuSvc.List(nil)
	if err != nil {
		return nil, NewInternal(err.Error())
	}

	out := make([]*Menu, len(list))
	for i := range list {
		m := menuFromModel(&list[i])
		out[i] = &m
	}
	return out, nil
}

func (c *coreImpl) CreateMenu(_ context.Context, input MenuInput) (*Menu, error) {
	slug := input.Slug
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(input.Name, " ", "-"))
	}

	m := models.Menu{
		Name: input.Name,
		Slug: slug,
	}
	if err := c.menuSvc.Create(&m); err != nil {
		if strings.Contains(err.Error(), "SLUG_CONFLICT") {
			return nil, NewValidation("menu slug already exists")
		}
		return nil, NewInternal(err.Error())
	}

	result := menuFromModel(&m)
	return &result, nil
}

func (c *coreImpl) UpdateMenu(_ context.Context, slug string, input MenuInput) (*Menu, error) {
	var existing models.Menu
	if err := c.db.Where("slug = ?", slug).First(&existing).Error; err != nil {
		return nil, NewNotFound("menu", slug)
	}

	updates := map[string]interface{}{
		"name": input.Name,
	}
	if input.Slug != "" {
		updates["slug"] = input.Slug
	}

	updated, err := c.menuSvc.Update(existing.ID, updates)
	if err != nil {
		if strings.Contains(err.Error(), "SLUG_CONFLICT") {
			return nil, NewValidation("menu slug already exists")
		}
		return nil, NewInternal(err.Error())
	}

	result := menuFromModel(updated)
	return &result, nil
}

func (c *coreImpl) DeleteMenu(_ context.Context, slug string) error {
	var existing models.Menu
	if err := c.db.Where("slug = ?", slug).First(&existing).Error; err != nil {
		return NewNotFound("menu", slug)
	}

	if err := c.menuSvc.Delete(existing.ID); err != nil {
		return NewInternal(err.Error())
	}
	return nil
}

func menuFromModel(m *models.Menu) Menu {
	return Menu{
		ID:        uint(m.ID),
		Name:      m.Name,
		Slug:      m.Slug,
		Items:     menuItemsFromModels(m.Items),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func menuItemsFromModels(items []models.MenuItem) []MenuItem {
	out := make([]MenuItem, len(items))
	for i, item := range items {
		var parentID *uint
		if item.ParentID != nil {
			p := uint(*item.ParentID)
			parentID = &p
		}
		out[i] = MenuItem{
			ID:       uint(item.ID),
			Label:    item.Title,
			URL:      item.URL,
			Target:   item.Target,
			ParentID: parentID,
			Position: item.SortOrder,
			Children: menuItemsFromModels(item.Children),
		}
	}
	return out
}
