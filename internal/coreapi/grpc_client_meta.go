package coreapi

import (
	"context"
	"encoding/json"

	pb "squilla/pkg/plugin/coreapipb"
)

// This file holds the taxonomy, settings, event, email, menu,
// route/filter/media-stub, and user methods of GRPCHostClient.

func (c *GRPCHostClient) RegisterTaxonomy(ctx context.Context, input TaxonomyInput) (*Taxonomy, error) {
	resp, err := c.client.RegisterTaxonomy(ctx, taxonomyInputToProto(input))
	if err != nil {
		return nil, fromGRPCError(err)
	}
	t := taxonomyFromProto(resp.Taxonomy)
	return &t, nil
}

func (c *GRPCHostClient) GetTaxonomy(ctx context.Context, slug string) (*Taxonomy, error) {
	resp, err := c.client.GetTaxonomy(ctx, &pb.GetTaxonomyRequest{Slug: slug})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	t := taxonomyFromProto(resp.Taxonomy)
	return &t, nil
}

func (c *GRPCHostClient) ListTaxonomies(ctx context.Context) ([]*Taxonomy, error) {
	resp, err := c.client.ListTaxonomies(ctx, &pb.Empty{})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	list := make([]*Taxonomy, len(resp.Taxonomies))
	for i, t := range resp.Taxonomies {
		tax := taxonomyFromProto(t)
		list[i] = &tax
	}
	return list, nil
}

func (c *GRPCHostClient) UpdateTaxonomy(ctx context.Context, slug string, input TaxonomyInput) (*Taxonomy, error) {
	resp, err := c.client.UpdateTaxonomy(ctx, &pb.UpdateTaxonomyRequest{
		Slug:  slug,
		Input: taxonomyInputToProto(input),
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	t := taxonomyFromProto(resp.Taxonomy)
	return &t, nil
}

func (c *GRPCHostClient) DeleteTaxonomy(ctx context.Context, slug string) error {
	_, err := c.client.DeleteTaxonomy(ctx, &pb.DeleteTaxonomyRequest{Slug: slug})
	if err != nil {
		return fromGRPCError(err)
	}
	return nil
}

// --- Settings ---

func (c *GRPCHostClient) GetSetting(ctx context.Context, key string) (string, error) {
	resp, err := c.client.GetSetting(ctx, &pb.GetSettingRequest{Key: key})
	if err != nil {
		return "", fromGRPCError(err)
	}
	return resp.Value, nil
}

func (c *GRPCHostClient) SetSetting(ctx context.Context, key, value string) error {
	_, err := c.client.SetSetting(ctx, &pb.SetSettingRequest{Key: key, Value: value})
	if err != nil {
		return fromGRPCError(err)
	}
	return nil
}

func (c *GRPCHostClient) GetSettings(ctx context.Context, prefix string) (map[string]string, error) {
	resp, err := c.client.GetSettings(ctx, &pb.GetSettingsRequest{Prefix: prefix})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	return resp.Settings, nil
}

// Locale-aware variants. Until the plugin proto carries a language_code field
// these delegate to the non-Loc versions so existing extensions keep working
// against the new core. Locale-sensitive plugins must be updated alongside a
// proto change in a follow-up.
func (c *GRPCHostClient) GetSettingLoc(ctx context.Context, key, _ string) (string, error) {
	return c.GetSetting(ctx, key)
}

func (c *GRPCHostClient) SetSettingLoc(ctx context.Context, key, _, value string) error {
	return c.SetSetting(ctx, key, value)
}

func (c *GRPCHostClient) GetSettingsLoc(ctx context.Context, prefix, _ string) (map[string]string, error) {
	return c.GetSettings(ctx, prefix)
}

// --- Events ---

func (c *GRPCHostClient) Emit(ctx context.Context, action string, payload map[string]any) error {
	var payloadJSON string
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return NewInternal("failed to marshal payload: " + err.Error())
		}
		payloadJSON = string(b)
	}
	_, err := c.client.EmitEvent(ctx, &pb.EmitEventRequest{Action: action, PayloadJson: payloadJSON})
	if err != nil {
		return fromGRPCError(err)
	}
	return nil
}

func (c *GRPCHostClient) Subscribe(_ context.Context, _ string, _ EventHandler) (UnsubscribeFunc, error) {
	return nil, NewInternal("not supported via gRPC")
}

// --- Email ---

func (c *GRPCHostClient) SendEmail(ctx context.Context, req EmailRequest) error {
	_, err := c.client.SendEmail(ctx, &pb.SendEmailRequest{
		To:      req.To,
		Subject: req.Subject,
		Html:    req.HTML,
	})
	if err != nil {
		return fromGRPCError(err)
	}
	return nil
}

// --- Menus ---

func (c *GRPCHostClient) GetMenu(ctx context.Context, slug string) (*Menu, error) {
	resp, err := c.client.GetMenu(ctx, &pb.GetMenuRequest{Slug: slug})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	return menuFromProto(resp.Menu), nil
}

func (c *GRPCHostClient) GetMenus(ctx context.Context) ([]*Menu, error) {
	resp, err := c.client.GetMenus(ctx, &pb.Empty{})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	menus := make([]*Menu, len(resp.Menus))
	for i, m := range resp.Menus {
		menus[i] = menuFromProto(m)
	}
	return menus, nil
}

func (c *GRPCHostClient) CreateMenu(_ context.Context, _ MenuInput) (*Menu, error) {
	return nil, NewInternal("not supported via gRPC")
}

func (c *GRPCHostClient) UpdateMenu(_ context.Context, _ string, _ MenuInput) (*Menu, error) {
	return nil, NewInternal("not supported via gRPC")
}

func (c *GRPCHostClient) UpsertMenu(_ context.Context, _ MenuInput) (*Menu, error) {
	return nil, NewInternal("not supported via gRPC")
}

func (c *GRPCHostClient) DeleteMenu(_ context.Context, _ string) error {
	return NewInternal("not supported via gRPC")
}

// --- Routes ---

func (c *GRPCHostClient) RegisterRoute(_ context.Context, _, _ string, _ RouteMeta) error {
	return NewInternal("not supported via gRPC")
}

func (c *GRPCHostClient) RemoveRoute(_ context.Context, _, _ string) error {
	return NewInternal("not supported via gRPC")
}

// --- Filters ---

func (c *GRPCHostClient) RegisterFilter(_ context.Context, _ string, _ int, _ FilterHandler) (UnsubscribeFunc, error) {
	return nil, NewInternal("not supported via gRPC")
}

func (c *GRPCHostClient) ApplyFilters(_ context.Context, _ string, _ any) (any, error) {
	return nil, NewInternal("not supported via gRPC")
}

// --- Media ---

func (c *GRPCHostClient) UploadMedia(_ context.Context, _ MediaUploadRequest) (*MediaFile, error) {
	return nil, NewInternal("not supported via gRPC")
}

func (c *GRPCHostClient) GetMedia(_ context.Context, _ uint) (*MediaFile, error) {
	return nil, NewInternal("not supported via gRPC")
}

func (c *GRPCHostClient) QueryMedia(_ context.Context, _ MediaQuery) ([]*MediaFile, error) {
	return nil, NewInternal("not supported via gRPC")
}

func (c *GRPCHostClient) DeleteMedia(_ context.Context, _ uint) error {
	return NewInternal("not supported via gRPC")
}

// --- Users ---

func (c *GRPCHostClient) GetUser(ctx context.Context, id uint) (*User, error) {
	resp, err := c.client.GetUser(ctx, &pb.GetUserRequest{Id: uint32(id)})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	return userFromProto(resp.User), nil
}

func (c *GRPCHostClient) QueryUsers(ctx context.Context, query UserQuery) ([]*User, error) {
	resp, err := c.client.QueryUsers(ctx, &pb.QueryUsersRequest{
		RoleSlug: query.RoleSlug,
		Search:   query.Search,
		Limit:    int32(query.Limit),
		Offset:   int32(query.Offset),
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	users := make([]*User, len(resp.Users))
	for i, u := range resp.Users {
		users[i] = userFromProto(u)
	}
	return users, nil
}

// --- Fetch ---

