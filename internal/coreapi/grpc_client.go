package coreapi

import (
	"context"
	"encoding/json"
	"time"

	pb "vibecms/pkg/plugin/coreapipb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Compile-time check that GRPCHostClient implements CoreAPI.
var _ CoreAPI = (*GRPCHostClient)(nil)

// GRPCHostClient implements CoreAPI by calling VibeCMSHost over gRPC.
type GRPCHostClient struct {
	client pb.VibeCMSHostClient
}

// NewGRPCHostClient creates a new CoreAPI client that delegates to a gRPC connection.
func NewGRPCHostClient(client pb.VibeCMSHostClient) *GRPCHostClient {
	return &GRPCHostClient{client: client}
}

// --- Nodes ---

func (c *GRPCHostClient) GetNode(ctx context.Context, id uint) (*Node, error) {
	resp, err := c.client.GetNode(ctx, &pb.GetNodeRequest{Id: uint32(id)})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	return nodeFromProto(resp.Node), nil
}

func (c *GRPCHostClient) QueryNodes(ctx context.Context, query NodeQuery) (*NodeList, error) {
	req := &pb.QueryNodesRequest{
		NodeType:     query.NodeType,
		Status:       query.Status,
		LanguageCode: query.LanguageCode,
		Slug:         query.Slug,
		Search:       query.Search,
		Limit:        int32(query.Limit),
		Offset:       int32(query.Offset),
		OrderBy:      query.OrderBy,
	}
	if query.ParentID != nil {
		req.HasParentId = true
		req.ParentId = uint32(*query.ParentID)
	}
	resp, err := c.client.QueryNodes(ctx, req)
	if err != nil {
		return nil, fromGRPCError(err)
	}
	nodes := make([]*Node, len(resp.Nodes))
	for i, n := range resp.Nodes {
		nodes[i] = nodeFromProto(n)
	}
	return &NodeList{Nodes: nodes, Total: resp.Total}, nil
}

func (c *GRPCHostClient) CreateNode(ctx context.Context, input NodeInput) (*Node, error) {
	pbInput, err := nodeInputToProto(input)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.CreateNode(ctx, &pb.CreateNodeRequest{Input: pbInput})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	return nodeFromProto(resp.Node), nil
}

func (c *GRPCHostClient) UpdateNode(ctx context.Context, id uint, input NodeInput) (*Node, error) {
	pbInput, err := nodeInputToProto(input)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.UpdateNode(ctx, &pb.UpdateNodeRequest{Id: uint32(id), Input: pbInput})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	return nodeFromProto(resp.Node), nil
}

func (c *GRPCHostClient) DeleteNode(ctx context.Context, id uint) error {
	_, err := c.client.DeleteNode(ctx, &pb.DeleteNodeRequest{Id: uint32(id)})
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

func (c *GRPCHostClient) Fetch(ctx context.Context, req FetchRequest) (*FetchResponse, error) {
	resp, err := c.client.Fetch(ctx, &pb.FetchRequest{
		Method:  req.Method,
		Url:     req.URL,
		Headers: req.Headers,
		Body:    req.Body,
		Timeout: int32(req.Timeout),
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	return &FetchResponse{
		StatusCode: int(resp.StatusCode),
		Headers:    resp.Headers,
		Body:       resp.Body,
	}, nil
}

// --- Log ---

func (c *GRPCHostClient) Log(ctx context.Context, level, message string, fields map[string]any) error {
	var fieldsJSON string
	if fields != nil {
		b, err := json.Marshal(fields)
		if err != nil {
			return NewInternal("failed to marshal fields: " + err.Error())
		}
		fieldsJSON = string(b)
	}
	_, err := c.client.Log(ctx, &pb.LogRequest{
		Level:      level,
		Message:    message,
		FieldsJson: fieldsJSON,
	})
	if err != nil {
		return fromGRPCError(err)
	}
	return nil
}

// --- Data Store ---

func (c *GRPCHostClient) DataGet(ctx context.Context, table string, id uint) (map[string]any, error) {
	resp, err := c.client.DataGet(ctx, &pb.DataGetRequest{Table: table, Id: uint32(id)})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	var row map[string]any
	if err := json.Unmarshal(resp.RowJson, &row); err != nil {
		return nil, NewInternal("failed to unmarshal row: " + err.Error())
	}
	return row, nil
}

func (c *GRPCHostClient) DataQuery(ctx context.Context, table string, query DataStoreQuery) (*DataStoreResult, error) {
	req := &pb.DataQueryRequest{
		Table:   table,
		Search:  query.Search,
		OrderBy: query.OrderBy,
		Limit:   int32(query.Limit),
		Offset:  int32(query.Offset),
		Raw:     query.Raw,
	}
	if query.Where != nil {
		b, err := json.Marshal(query.Where)
		if err != nil {
			return nil, NewInternal("failed to marshal where: " + err.Error())
		}
		req.WhereJson = string(b)
	}
	if query.Args != nil {
		b, err := json.Marshal(query.Args)
		if err != nil {
			return nil, NewInternal("failed to marshal args: " + err.Error())
		}
		req.ArgsJson = string(b)
	}
	resp, err := c.client.DataQuery(ctx, req)
	if err != nil {
		return nil, fromGRPCError(err)
	}
	rows := make([]map[string]any, len(resp.RowsJson))
	for i, b := range resp.RowsJson {
		var row map[string]any
		if err := json.Unmarshal(b, &row); err != nil {
			return nil, NewInternal("failed to unmarshal row: " + err.Error())
		}
		rows[i] = row
	}
	return &DataStoreResult{Rows: rows, Total: resp.Total}, nil
}

func (c *GRPCHostClient) DataCreate(ctx context.Context, table string, data map[string]any) (map[string]any, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, NewInternal("failed to marshal data: " + err.Error())
	}
	resp, err := c.client.DataCreate(ctx, &pb.DataCreateRequest{Table: table, DataJson: b})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	var row map[string]any
	if err := json.Unmarshal(resp.RowJson, &row); err != nil {
		return nil, NewInternal("failed to unmarshal row: " + err.Error())
	}
	return row, nil
}

func (c *GRPCHostClient) DataUpdate(ctx context.Context, table string, id uint, data map[string]any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return NewInternal("failed to marshal data: " + err.Error())
	}
	_, err = c.client.DataUpdate(ctx, &pb.DataUpdateRequest{Table: table, Id: uint32(id), DataJson: b})
	if err != nil {
		return fromGRPCError(err)
	}
	return nil
}

func (c *GRPCHostClient) DataDelete(ctx context.Context, table string, id uint) error {
	_, err := c.client.DataDelete(ctx, &pb.DataDeleteRequest{Table: table, Id: uint32(id)})
	if err != nil {
		return fromGRPCError(err)
	}
	return nil
}

func (c *GRPCHostClient) DataExec(ctx context.Context, sqlStr string, args ...any) (int64, error) {
	var argsJSON string
	if len(args) > 0 {
		b, err := json.Marshal(args)
		if err != nil {
			return 0, NewInternal("failed to marshal args: " + err.Error())
		}
		argsJSON = string(b)
	}
	resp, err := c.client.DataExec(ctx, &pb.DataExecRequest{Sql: sqlStr, ArgsJson: argsJSON})
	if err != nil {
		return 0, fromGRPCError(err)
	}
	return resp.RowsAffected, nil
}

// --- File Storage ---

func (c *GRPCHostClient) StoreFile(ctx context.Context, path string, data []byte) (string, error) {
	resp, err := c.client.StoreFile(ctx, &pb.StoreFileRequest{Path: path, Data: data})
	if err != nil {
		return "", fromGRPCError(err)
	}
	return resp.Url, nil
}

func (c *GRPCHostClient) DeleteFile(ctx context.Context, path string) error {
	_, err := c.client.DeleteFile(ctx, &pb.DeleteFileRequest{Path: path})
	if err != nil {
		return fromGRPCError(err)
	}
	return nil
}

// --- Helper functions ---

func nodeFromProto(msg *pb.NodeMessage) *Node {
	if msg == nil {
		return nil
	}
	n := &Node{
		ID:           uint(msg.Id),
		UUID:         msg.Uuid,
		NodeType:     msg.NodeType,
		Status:       msg.Status,
		LanguageCode: msg.LanguageCode,
		Slug:         msg.Slug,
		FullURL:      msg.FullUrl,
		Title:        msg.Title,
		SeoSettings:  msg.SeoSettings,
	}
	if msg.HasParentId {
		pid := uint(msg.ParentId)
		n.ParentID = &pid
	}
	if msg.PublishedAt != "" {
		if t, err := time.Parse(time.RFC3339, msg.PublishedAt); err == nil {
			n.PublishedAt = &t
		}
	}
	if msg.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, msg.CreatedAt); err == nil {
			n.CreatedAt = t
		}
	}
	if msg.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, msg.UpdatedAt); err == nil {
			n.UpdatedAt = t
		}
	}
	if msg.BlocksDataJson != "" {
		var blocks any
		if err := json.Unmarshal([]byte(msg.BlocksDataJson), &blocks); err == nil {
			n.BlocksData = blocks
		}
	}
	if msg.FieldsDataJson != "" {
		var fields map[string]any
		if err := json.Unmarshal([]byte(msg.FieldsDataJson), &fields); err == nil {
			n.FieldsData = fields
		}
	}
	return n
}

func nodeInputToProto(input NodeInput) (*pb.NodeInput, error) {
	pi := &pb.NodeInput{
		NodeType:     input.NodeType,
		Status:       input.Status,
		LanguageCode: input.LanguageCode,
		Slug:         input.Slug,
		Title:        input.Title,
		SeoSettings:  input.SeoSettings,
	}
	if input.ParentID != nil {
		pi.HasParentId = true
		pi.ParentId = uint32(*input.ParentID)
	}
	if input.BlocksData != nil {
		b, err := json.Marshal(input.BlocksData)
		if err != nil {
			return nil, NewInternal("failed to marshal blocks_data: " + err.Error())
		}
		pi.BlocksDataJson = string(b)
	}
	if input.FieldsData != nil {
		b, err := json.Marshal(input.FieldsData)
		if err != nil {
			return nil, NewInternal("failed to marshal fields_data: " + err.Error())
		}
		pi.FieldsDataJson = string(b)
	}
	return pi, nil
}

func menuFromProto(msg *pb.MenuMessage) *Menu {
	if msg == nil {
		return nil
	}
	m := &Menu{
		ID:   uint(msg.Id),
		Name: msg.Name,
		Slug: msg.Slug,
	}
	if msg.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, msg.CreatedAt); err == nil {
			m.CreatedAt = t
		}
	}
	if msg.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, msg.UpdatedAt); err == nil {
			m.UpdatedAt = t
		}
	}
	m.Items = make([]MenuItem, len(msg.Items))
	for i, item := range msg.Items {
		m.Items[i] = menuItemFromProto(item)
	}
	return m
}

func menuItemFromProto(msg *pb.MenuItemMessage) MenuItem {
	if msg == nil {
		return MenuItem{}
	}
	item := MenuItem{
		ID:       uint(msg.Id),
		Label:    msg.Label,
		URL:      msg.Url,
		Target:   msg.Target,
		Position: int(msg.Position),
	}
	if msg.HasParentId {
		pid := uint(msg.ParentId)
		item.ParentID = &pid
	}
	if len(msg.Children) > 0 {
		item.Children = make([]MenuItem, len(msg.Children))
		for i, child := range msg.Children {
			item.Children[i] = menuItemFromProto(child)
		}
	}
	return item
}

func userFromProto(msg *pb.UserMessage) *User {
	if msg == nil {
		return nil
	}
	u := &User{
		ID:       uint(msg.Id),
		Email:    msg.Email,
		Name:     msg.Name,
		RoleSlug: msg.RoleSlug,
	}
	if msg.HasRoleId {
		rid := uint(msg.RoleId)
		u.RoleID = &rid
	}
	if msg.HasLanguageId {
		lid := int(msg.LanguageId)
		u.LanguageID = &lid
	}
	return u
}

func fromGRPCError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return NewInternal(err.Error())
	}
	switch st.Code() {
	case codes.NotFound:
		return &APIError{Code: ErrNotFound, Message: st.Message()}
	case codes.PermissionDenied:
		return &APIError{Code: ErrCapabilityDenied, Message: st.Message()}
	case codes.InvalidArgument:
		return &APIError{Code: ErrValidation, Message: st.Message()}
	default:
		return &APIError{Code: ErrInternal, Message: st.Message()}
	}
}
