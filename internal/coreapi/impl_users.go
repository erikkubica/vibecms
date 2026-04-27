package coreapi

import (
	"context"

	"vibecms/internal/models"
)

func userFromModel(u *models.User) *User {
	user := &User{
		ID:         uint(u.ID),
		Email:      u.Email,
		RoleID:     ptrUint(uint(u.RoleID)),
		LanguageID: u.LanguageID,
	}
	if u.FullName != nil {
		user.Name = *u.FullName
	}
	if u.Role.Slug != "" {
		user.RoleSlug = u.Role.Slug
	}
	return user
}

func ptrUint(v uint) *uint {
	return &v
}

func (c *coreImpl) GetUser(ctx context.Context, id uint) (*User, error) {
	var u models.User
	if err := c.db.WithContext(ctx).Preload("Role").First(&u, id).Error; err != nil {
		return nil, NewNotFound("user", id)
	}
	return userFromModel(&u), nil
}

func (c *coreImpl) QueryUsers(ctx context.Context, query UserQuery) ([]*User, error) {
	tx := c.db.WithContext(ctx).Model(&models.User{}).Preload("Role")

	if query.RoleSlug != "" {
		tx = tx.Joins("JOIN roles ON roles.id = users.role_id").
			Where("roles.slug = ?", query.RoleSlug)
	}

	if query.Search != "" {
		pattern := "%" + query.Search + "%"
		tx = tx.Where("users.email ILIKE ? OR users.full_name ILIKE ?", pattern, pattern)
	}

	if query.Limit > 0 {
		tx = tx.Limit(query.Limit)
	} else {
		tx = tx.Limit(50)
	}

	if query.Offset > 0 {
		tx = tx.Offset(query.Offset)
	}

	var rows []models.User
	if err := tx.Find(&rows).Error; err != nil {
		return nil, NewInternal("query users failed: " + err.Error())
	}

	users := make([]*User, len(rows))
	for i := range rows {
		users[i] = userFromModel(&rows[i])
	}
	return users, nil
}
