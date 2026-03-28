package coreapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	pb "vibecms/pkg/plugin/coreapipb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHostServer implements VibeCMSHostServer by wrapping a CoreAPI instance.
type GRPCHostServer struct {
	pb.UnimplementedVibeCMSHostServer
	api    CoreAPI
	caller CallerInfo
}

// NewGRPCHostServer creates a new gRPC host server that delegates to the given CoreAPI.
func NewGRPCHostServer(api CoreAPI, caller CallerInfo) *GRPCHostServer {
	return &GRPCHostServer{api: api, caller: caller}
}

func (s *GRPCHostServer) ctx(ctx context.Context) context.Context {
	return WithCaller(ctx, s.caller)
}

// --- Node RPCs ---

func (s *GRPCHostServer) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.NodeResponse, error) {
	node, err := s.api.GetNode(s.ctx(ctx), uint(req.Id))
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.NodeResponse{Node: nodeToProto(node)}, nil
}

func (s *GRPCHostServer) QueryNodes(ctx context.Context, req *pb.QueryNodesRequest) (*pb.QueryNodesResponse, error) {
	q := NodeQuery{
		NodeType:     req.NodeType,
		Status:       req.Status,
		LanguageCode: req.LanguageCode,
		Slug:         req.Slug,
		Search:       req.Search,
		Limit:        int(req.Limit),
		Offset:       int(req.Offset),
		OrderBy:      req.OrderBy,
	}
	if req.HasParentId {
		pid := uint(req.ParentId)
		q.ParentID = &pid
	}
	list, err := s.api.QueryNodes(s.ctx(ctx), q)
	if err != nil {
		return nil, grpcError(err)
	}
	nodes := make([]*pb.NodeMessage, len(list.Nodes))
	for i, n := range list.Nodes {
		nodes[i] = nodeToProto(n)
	}
	return &pb.QueryNodesResponse{Nodes: nodes, Total: list.Total}, nil
}

func (s *GRPCHostServer) CreateNode(ctx context.Context, req *pb.CreateNodeRequest) (*pb.NodeResponse, error) {
	input, err := nodeInputFromProto(req.Input)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid input: %v", err)
	}
	node, err := s.api.CreateNode(s.ctx(ctx), input)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.NodeResponse{Node: nodeToProto(node)}, nil
}

func (s *GRPCHostServer) UpdateNode(ctx context.Context, req *pb.UpdateNodeRequest) (*pb.NodeResponse, error) {
	input, err := nodeInputFromProto(req.Input)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid input: %v", err)
	}
	node, err := s.api.UpdateNode(s.ctx(ctx), uint(req.Id), input)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.NodeResponse{Node: nodeToProto(node)}, nil
}

func (s *GRPCHostServer) DeleteNode(ctx context.Context, req *pb.DeleteNodeRequest) (*pb.Empty, error) {
	if err := s.api.DeleteNode(s.ctx(ctx), uint(req.Id)); err != nil {
		return nil, grpcError(err)
	}
	return &pb.Empty{}, nil
}

// --- Settings RPCs ---

func (s *GRPCHostServer) GetSetting(ctx context.Context, req *pb.GetSettingRequest) (*pb.SettingResponse, error) {
	val, err := s.api.GetSetting(s.ctx(ctx), req.Key)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.SettingResponse{Value: val}, nil
}

func (s *GRPCHostServer) SetSetting(ctx context.Context, req *pb.SetSettingRequest) (*pb.Empty, error) {
	if err := s.api.SetSetting(s.ctx(ctx), req.Key, req.Value); err != nil {
		return nil, grpcError(err)
	}
	return &pb.Empty{}, nil
}

func (s *GRPCHostServer) GetSettings(ctx context.Context, req *pb.GetSettingsRequest) (*pb.SettingsResponse, error) {
	settings, err := s.api.GetSettings(s.ctx(ctx), req.Prefix)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.SettingsResponse{Settings: settings}, nil
}

// --- Event RPCs ---

func (s *GRPCHostServer) EmitEvent(ctx context.Context, req *pb.EmitEventRequest) (*pb.Empty, error) {
	var payload map[string]any
	if req.PayloadJson != "" {
		if err := json.Unmarshal([]byte(req.PayloadJson), &payload); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid payload JSON: %v", err)
		}
	}
	if err := s.api.Emit(s.ctx(ctx), req.Action, payload); err != nil {
		return nil, grpcError(err)
	}
	return &pb.Empty{}, nil
}

// --- Email RPCs ---

func (s *GRPCHostServer) SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.Empty, error) {
	emailReq := EmailRequest{
		To:      req.To,
		Subject: req.Subject,
		HTML:    req.Html,
	}
	if err := s.api.SendEmail(s.ctx(ctx), emailReq); err != nil {
		return nil, grpcError(err)
	}
	return &pb.Empty{}, nil
}

// --- Menu RPCs ---

func (s *GRPCHostServer) GetMenu(ctx context.Context, req *pb.GetMenuRequest) (*pb.MenuResponse, error) {
	menu, err := s.api.GetMenu(s.ctx(ctx), req.Slug)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.MenuResponse{Menu: menuToProto(menu)}, nil
}

func (s *GRPCHostServer) GetMenus(ctx context.Context, _ *pb.Empty) (*pb.MenuListResponse, error) {
	menus, err := s.api.GetMenus(s.ctx(ctx))
	if err != nil {
		return nil, grpcError(err)
	}
	pbMenus := make([]*pb.MenuMessage, len(menus))
	for i, m := range menus {
		pbMenus[i] = menuToProto(m)
	}
	return &pb.MenuListResponse{Menus: pbMenus}, nil
}

// --- User RPCs ---

func (s *GRPCHostServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	user, err := s.api.GetUser(s.ctx(ctx), uint(req.Id))
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.UserResponse{User: userToProto(user)}, nil
}

func (s *GRPCHostServer) QueryUsers(ctx context.Context, req *pb.QueryUsersRequest) (*pb.UserListResponse, error) {
	q := UserQuery{
		RoleSlug: req.RoleSlug,
		Search:   req.Search,
		Limit:    int(req.Limit),
		Offset:   int(req.Offset),
	}
	users, err := s.api.QueryUsers(s.ctx(ctx), q)
	if err != nil {
		return nil, grpcError(err)
	}
	pbUsers := make([]*pb.UserMessage, len(users))
	for i, u := range users {
		pbUsers[i] = userToProto(u)
	}
	return &pb.UserListResponse{Users: pbUsers}, nil
}

// --- Fetch RPCs ---

func (s *GRPCHostServer) Fetch(ctx context.Context, req *pb.FetchRequest) (*pb.FetchResponse, error) {
	fetchReq := FetchRequest{
		Method:  req.Method,
		URL:     req.Url,
		Headers: req.Headers,
		Body:    req.Body,
		Timeout: int(req.Timeout),
	}
	resp, err := s.api.Fetch(s.ctx(ctx), fetchReq)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.FetchResponse{
		StatusCode: int32(resp.StatusCode),
		Headers:    resp.Headers,
		Body:       resp.Body,
	}, nil
}

// --- Log RPCs ---

func (s *GRPCHostServer) Log(ctx context.Context, req *pb.LogRequest) (*pb.Empty, error) {
	var fields map[string]any
	if req.FieldsJson != "" {
		if err := json.Unmarshal([]byte(req.FieldsJson), &fields); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid fields JSON: %v", err)
		}
	}
	if err := s.api.Log(s.ctx(ctx), req.Level, req.Message, fields); err != nil {
		return nil, grpcError(err)
	}
	return &pb.Empty{}, nil
}

// --- Data Store RPCs ---

func (s *GRPCHostServer) DataGet(ctx context.Context, req *pb.DataGetRequest) (*pb.DataRowResponse, error) {
	row, err := s.api.DataGet(s.ctx(ctx), req.Table, uint(req.Id))
	if err != nil {
		return nil, grpcError(err)
	}
	b, err := json.Marshal(row)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal row: %v", err)
	}
	return &pb.DataRowResponse{RowJson: b}, nil
}

func (s *GRPCHostServer) DataQuery(ctx context.Context, req *pb.DataQueryRequest) (*pb.DataQueryResponse, error) {
	q := DataStoreQuery{
		Search:  req.Search,
		OrderBy: req.OrderBy,
		Limit:   int(req.Limit),
		Offset:  int(req.Offset),
		Raw:     req.Raw,
	}
	if req.WhereJson != "" {
		if err := json.Unmarshal([]byte(req.WhereJson), &q.Where); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid where JSON: %v", err)
		}
	}
	if req.ArgsJson != "" {
		if err := json.Unmarshal([]byte(req.ArgsJson), &q.Args); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid args JSON: %v", err)
		}
	}
	result, err := s.api.DataQuery(s.ctx(ctx), req.Table, q)
	if err != nil {
		return nil, grpcError(err)
	}
	rowsJSON := make([][]byte, len(result.Rows))
	for i, row := range result.Rows {
		b, err := json.Marshal(row)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal row: %v", err)
		}
		rowsJSON[i] = b
	}
	return &pb.DataQueryResponse{RowsJson: rowsJSON, Total: result.Total}, nil
}

func (s *GRPCHostServer) DataCreate(ctx context.Context, req *pb.DataCreateRequest) (*pb.DataRowResponse, error) {
	var data map[string]any
	if err := json.Unmarshal(req.DataJson, &data); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid data JSON: %v", err)
	}
	row, err := s.api.DataCreate(s.ctx(ctx), req.Table, data)
	if err != nil {
		return nil, grpcError(err)
	}
	b, err := json.Marshal(row)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal row: %v", err)
	}
	return &pb.DataRowResponse{RowJson: b}, nil
}

func (s *GRPCHostServer) DataUpdate(ctx context.Context, req *pb.DataUpdateRequest) (*pb.Empty, error) {
	var data map[string]any
	if err := json.Unmarshal(req.DataJson, &data); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid data JSON: %v", err)
	}
	if err := s.api.DataUpdate(s.ctx(ctx), req.Table, uint(req.Id), data); err != nil {
		return nil, grpcError(err)
	}
	return &pb.Empty{}, nil
}

func (s *GRPCHostServer) DataDelete(ctx context.Context, req *pb.DataDeleteRequest) (*pb.Empty, error) {
	if err := s.api.DataDelete(s.ctx(ctx), req.Table, uint(req.Id)); err != nil {
		return nil, grpcError(err)
	}
	return &pb.Empty{}, nil
}

func (s *GRPCHostServer) DataExec(ctx context.Context, req *pb.DataExecRequest) (*pb.DataExecResponse, error) {
	var args []any
	if req.ArgsJson != "" {
		if err := json.Unmarshal([]byte(req.ArgsJson), &args); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid args JSON: %v", err)
		}
	}
	affected, err := s.api.DataExec(s.ctx(ctx), req.Sql, args...)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.DataExecResponse{RowsAffected: affected}, nil
}

// --- File Storage RPCs ---

func (s *GRPCHostServer) StoreFile(ctx context.Context, req *pb.StoreFileRequest) (*pb.StoreFileResponse, error) {
	url, err := s.api.StoreFile(s.ctx(ctx), req.Path, req.Data)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.StoreFileResponse{Url: url}, nil
}

func (s *GRPCHostServer) DeleteFile(ctx context.Context, req *pb.DeleteFileRequest) (*pb.Empty, error) {
	if err := s.api.DeleteFile(s.ctx(ctx), req.Path); err != nil {
		return nil, grpcError(err)
	}
	return &pb.Empty{}, nil
}

// --- Helper functions ---

func nodeToProto(n *Node) *pb.NodeMessage {
	if n == nil {
		return nil
	}
	msg := &pb.NodeMessage{
		Id:           uint32(n.ID),
		Uuid:         n.UUID,
		NodeType:     n.NodeType,
		Status:       n.Status,
		LanguageCode: n.LanguageCode,
		Slug:         n.Slug,
		FullUrl:      n.FullURL,
		Title:        n.Title,
		SeoSettings:  n.SeoSettings,
		CreatedAt:    n.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    n.UpdatedAt.Format(time.RFC3339),
	}
	if n.ParentID != nil {
		msg.HasParentId = true
		msg.ParentId = uint32(*n.ParentID)
	}
	if n.PublishedAt != nil {
		msg.PublishedAt = n.PublishedAt.Format(time.RFC3339)
	}
	if n.BlocksData != nil {
		if b, err := json.Marshal(n.BlocksData); err == nil {
			msg.BlocksDataJson = string(b)
		}
	}
	if n.FieldsData != nil {
		if b, err := json.Marshal(n.FieldsData); err == nil {
			msg.FieldsDataJson = string(b)
		}
	}
	return msg
}

func nodeInputFromProto(inp *pb.NodeInput) (NodeInput, error) {
	if inp == nil {
		return NodeInput{}, fmt.Errorf("nil input")
	}
	ni := NodeInput{
		NodeType:     inp.NodeType,
		Status:       inp.Status,
		LanguageCode: inp.LanguageCode,
		Slug:         inp.Slug,
		Title:        inp.Title,
		SeoSettings:  inp.SeoSettings,
	}
	if inp.HasParentId {
		pid := uint(inp.ParentId)
		ni.ParentID = &pid
	}
	if inp.BlocksDataJson != "" {
		var blocks any
		if err := json.Unmarshal([]byte(inp.BlocksDataJson), &blocks); err != nil {
			return NodeInput{}, fmt.Errorf("invalid blocks_data JSON: %w", err)
		}
		ni.BlocksData = blocks
	}
	if inp.FieldsDataJson != "" {
		var fields map[string]any
		if err := json.Unmarshal([]byte(inp.FieldsDataJson), &fields); err != nil {
			return NodeInput{}, fmt.Errorf("invalid fields_data JSON: %w", err)
		}
		ni.FieldsData = fields
	}
	return ni, nil
}

func menuToProto(m *Menu) *pb.MenuMessage {
	if m == nil {
		return nil
	}
	items := make([]*pb.MenuItemMessage, len(m.Items))
	for i, item := range m.Items {
		items[i] = menuItemToProto(item)
	}
	return &pb.MenuMessage{
		Id:        uint32(m.ID),
		Name:      m.Name,
		Slug:      m.Slug,
		Items:     items,
		CreatedAt: m.CreatedAt.Format(time.RFC3339),
		UpdatedAt: m.UpdatedAt.Format(time.RFC3339),
	}
}

func menuItemToProto(item MenuItem) *pb.MenuItemMessage {
	msg := &pb.MenuItemMessage{
		Id:       uint32(item.ID),
		Label:    item.Label,
		Url:      item.URL,
		Target:   item.Target,
		Position: int32(item.Position),
	}
	if item.ParentID != nil {
		msg.HasParentId = true
		msg.ParentId = uint32(*item.ParentID)
	}
	if len(item.Children) > 0 {
		msg.Children = make([]*pb.MenuItemMessage, len(item.Children))
		for i, child := range item.Children {
			msg.Children[i] = menuItemToProto(child)
		}
	}
	return msg
}

func userToProto(u *User) *pb.UserMessage {
	if u == nil {
		return nil
	}
	msg := &pb.UserMessage{
		Id:       uint32(u.ID),
		Email:    u.Email,
		Name:     u.Name,
		RoleSlug: u.RoleSlug,
	}
	if u.RoleID != nil {
		msg.HasRoleId = true
		msg.RoleId = uint32(*u.RoleID)
	}
	if u.LanguageID != nil {
		msg.HasLanguageId = true
		msg.LanguageId = int32(*u.LanguageID)
	}
	return msg
}

func grpcError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch {
		case errors.Is(apiErr.Code, ErrNotFound):
			return status.Errorf(codes.NotFound, apiErr.Message)
		case errors.Is(apiErr.Code, ErrCapabilityDenied):
			return status.Errorf(codes.PermissionDenied, apiErr.Message)
		case errors.Is(apiErr.Code, ErrValidation):
			return status.Errorf(codes.InvalidArgument, apiErr.Message)
		case errors.Is(apiErr.Code, ErrInternal):
			return status.Errorf(codes.Internal, apiErr.Message)
		}
	}
	return status.Errorf(codes.Internal, err.Error())
}
