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
		Category:     req.Category,
	}
	if req.TaxQueryJson != "" {
		var tq map[string][]string
		if err := json.Unmarshal([]byte(req.TaxQueryJson), &tq); err == nil {
			q.TaxQuery = tq
		}
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

func (s *GRPCHostServer) ListTaxonomyTerms(ctx context.Context, req *pb.ListTaxonomyTermsRequest) (*pb.ListTaxonomyTermsResponse, error) {
	terms, err := s.api.ListTaxonomyTerms(s.ctx(ctx), req.NodeType, req.Taxonomy)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.ListTaxonomyTermsResponse{Terms: terms}, nil
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

// --- Taxonomies ---

func (s *GRPCHostServer) ListTerms(ctx context.Context, req *pb.ListTermsRequest) (*pb.ListTermsResponse, error) {
	list, err := s.api.ListTerms(s.ctx(ctx), req.NodeType, req.Taxonomy)
	if err != nil {
		return nil, grpcError(err)
	}
	terms := make([]*pb.TermMessage, len(list))
	for i, t := range list {
		terms[i] = termToProto(t)
	}
	return &pb.ListTermsResponse{Terms: terms}, nil
}

func (s *GRPCHostServer) GetTerm(ctx context.Context, req *pb.GetTermRequest) (*pb.TermResponse, error) {
	t, err := s.api.GetTerm(s.ctx(ctx), uint(req.Id))
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.TermResponse{Term: termToProto(t)}, nil
}

func (s *GRPCHostServer) CreateTerm(ctx context.Context, req *pb.CreateTermRequest) (*pb.TermResponse, error) {
	t := termFromProto(req.Term)
	created, err := s.api.CreateTerm(s.ctx(ctx), &t)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.TermResponse{Term: termToProto(created)}, nil
}

func (s *GRPCHostServer) UpdateTerm(ctx context.Context, req *pb.UpdateTermRequest) (*pb.TermResponse, error) {
	var updates map[string]interface{}
	if err := json.Unmarshal([]byte(req.UpdatesJson), &updates); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid updates JSON: %v", err)
	}
	updated, err := s.api.UpdateTerm(s.ctx(ctx), uint(req.Id), updates)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.TermResponse{Term: termToProto(updated)}, nil
}

func (s *GRPCHostServer) DeleteTerm(ctx context.Context, req *pb.DeleteTermRequest) (*pb.Empty, error) {
	if err := s.api.DeleteTerm(s.ctx(ctx), uint(req.Id)); err != nil {
		return nil, grpcError(err)
	}
	return &pb.Empty{}, nil
}

// --- Taxonomy Definitions ---

func (s *GRPCHostServer) RegisterTaxonomy(ctx context.Context, req *pb.TaxonomyInputMessage) (*pb.TaxonomyResponse, error) {
	input := taxonomyInputFromProto(req)
	t, err := s.api.RegisterTaxonomy(s.ctx(ctx), input)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.TaxonomyResponse{Taxonomy: taxonomyToProto(t)}, nil
}

func (s *GRPCHostServer) GetTaxonomy(ctx context.Context, req *pb.GetTaxonomyRequest) (*pb.TaxonomyResponse, error) {
	t, err := s.api.GetTaxonomy(s.ctx(ctx), req.Slug)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.TaxonomyResponse{Taxonomy: taxonomyToProto(t)}, nil
}

func (s *GRPCHostServer) ListTaxonomies(ctx context.Context, _ *pb.Empty) (*pb.TaxonomyListResponse, error) {
	list, err := s.api.ListTaxonomies(s.ctx(ctx))
	if err != nil {
		return nil, grpcError(err)
	}
	pbList := make([]*pb.TaxonomyMessage, len(list))
	for i, t := range list {
		pbList[i] = taxonomyToProto(t)
	}
	return &pb.TaxonomyListResponse{Taxonomies: pbList}, nil
}

func (s *GRPCHostServer) UpdateTaxonomy(ctx context.Context, req *pb.UpdateTaxonomyRequest) (*pb.TaxonomyResponse, error) {
	input := taxonomyInputFromProto(req.Input)
	t, err := s.api.UpdateTaxonomy(s.ctx(ctx), req.Slug, input)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.TaxonomyResponse{Taxonomy: taxonomyToProto(t)}, nil
}

func (s *GRPCHostServer) DeleteTaxonomy(ctx context.Context, req *pb.DeleteTaxonomyRequest) (*pb.Empty, error) {
	if err := s.api.DeleteTaxonomy(s.ctx(ctx), req.Slug); err != nil {
		return nil, grpcError(err)
	}
	return &pb.Empty{}, nil
}

// --- Settings ---

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

// --- Node Type RPCs ---

func (s *GRPCHostServer) RegisterNodeType(ctx context.Context, req *pb.NodeTypeInputMessage) (*pb.NodeTypeResponse, error) {
	input := nodeTypeInputFromProto(req)
	nt, err := s.api.RegisterNodeType(s.ctx(ctx), input)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.NodeTypeResponse{NodeType: nodeTypeToProto(nt)}, nil
}

func (s *GRPCHostServer) GetNodeType(ctx context.Context, req *pb.GetNodeTypeRequest) (*pb.NodeTypeResponse, error) {
	nt, err := s.api.GetNodeType(s.ctx(ctx), req.Slug)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.NodeTypeResponse{NodeType: nodeTypeToProto(nt)}, nil
}

func (s *GRPCHostServer) ListNodeTypes(ctx context.Context, _ *pb.Empty) (*pb.NodeTypeListResponse, error) {
	list, err := s.api.ListNodeTypes(s.ctx(ctx))
	if err != nil {
		return nil, grpcError(err)
	}
	pbTypes := make([]*pb.NodeTypeMessage, len(list))
	for i, nt := range list {
		pbTypes[i] = nodeTypeToProto(nt)
	}
	return &pb.NodeTypeListResponse{NodeTypes: pbTypes}, nil
}

func (s *GRPCHostServer) UpdateNodeType(ctx context.Context, req *pb.UpdateNodeTypeRequest) (*pb.NodeTypeResponse, error) {
	input := nodeTypeInputFromProto(req.Input)
	nt, err := s.api.UpdateNodeType(s.ctx(ctx), req.Slug, input)
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.NodeTypeResponse{NodeType: nodeTypeToProto(nt)}, nil
}

func (s *GRPCHostServer) DeleteNodeType(ctx context.Context, req *pb.DeleteNodeTypeRequest) (*pb.Empty, error) {
	if err := s.api.DeleteNodeType(s.ctx(ctx), req.Slug); err != nil {
		return nil, grpcError(err)
	}
	return &pb.Empty{}, nil
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
		Excerpt:      n.Excerpt,
		SeoSettings:  n.SeoSettings,
		CreatedAt:    n.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    n.UpdatedAt.Format(time.RFC3339),
	}
	if n.Taxonomies != nil {
		if b, err := json.Marshal(n.Taxonomies); err == nil {
			msg.TaxonomiesJson = string(b)
		}
	}
	if n.FeaturedImage != nil {
		if b, err := json.Marshal(n.FeaturedImage); err == nil {
			msg.FeaturedImageJson = string(b)
		}
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
		Excerpt:      inp.Excerpt,
		SeoSettings:  inp.SeoSettings,
	}
	if inp.TaxonomiesJson != "" {
		var tx map[string][]string
		if err := json.Unmarshal([]byte(inp.TaxonomiesJson), &tx); err == nil {
			ni.Taxonomies = tx
		}
	}
	if inp.FeaturedImageJson != "" {
		var img any
		if err := json.Unmarshal([]byte(inp.FeaturedImageJson), &img); err != nil {
			return NodeInput{}, fmt.Errorf("invalid featured_image JSON: %w", err)
		}
		ni.FeaturedImage = img
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

func nodeTypeToProto(nt *NodeType) *pb.NodeTypeMessage {
	if nt == nil {
		return nil
	}
	fields := make([]*pb.NodeTypeFieldMessage, len(nt.FieldSchema))
	for i, f := range nt.FieldSchema {
		fields[i] = &pb.NodeTypeFieldMessage{
			Name:     f.Name,
			Label:    f.Label,
			Type:     f.Type,
			Required: f.Required,
			Options:  f.OptionsToStrings(),
		}
	}
	msg := &pb.NodeTypeMessage{
		Id:          int32(nt.ID),
		Slug:        nt.Slug,
		Label:       nt.Label,
		Icon:        nt.Icon,
		Description: nt.Description,
		FieldSchema: fields,
		UrlPrefixes: nt.URLPrefixes,
		CreatedAt:   nt.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   nt.UpdatedAt.Format(time.RFC3339),
	}
	if nt.Taxonomies != nil {
		if b, err := json.Marshal(nt.Taxonomies); err == nil {
			msg.TaxonomiesJson = string(b)
		}
	}
	return msg
}

func nodeTypeInputFromProto(inp *pb.NodeTypeInputMessage) NodeTypeInput {
	if inp == nil {
		return NodeTypeInput{}
	}
	var fields []NodeTypeField
	for _, f := range inp.FieldSchema {
		fields = append(fields, NodeTypeField{
			Name:     f.Name,
			Label:    f.Label,
			Type:     f.Type,
			Required: f.Required,
			Options:  OptionsFromStrings(f.Options),
		})
	}
	ni := NodeTypeInput{
		Slug:        inp.Slug,
		Label:       inp.Label,
		Icon:        inp.Icon,
		Description: inp.Description,
		FieldSchema: fields,
		URLPrefixes: inp.UrlPrefixes,
	}
	if inp.TaxonomiesJson != "" {
		var taxes []TaxonomyDefinition
		if err := json.Unmarshal([]byte(inp.TaxonomiesJson), &taxes); err == nil {
			ni.Taxonomies = taxes
		}
	}
	return ni
}

func termToProto(t *TaxonomyTerm) *pb.TermMessage {
	if t == nil {
		return nil
	}
	msg := &pb.TermMessage{
		Id:          uint32(t.ID),
		NodeType:    t.NodeType,
		Taxonomy:    t.Taxonomy,
		Slug:        t.Slug,
		Name:        t.Name,
		Description: t.Description,
		Count:       int32(t.Count),
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
	}
	if t.ParentID != nil {
		msg.HasParentId = true
		msg.ParentId = uint32(*t.ParentID)
	}
	if t.FieldsData != nil {
		if b, err := json.Marshal(t.FieldsData); err == nil {
			msg.FieldsDataJson = string(b)
		}
	}
	return msg
}

func termFromProto(msg *pb.TermMessage) TaxonomyTerm {
	if msg == nil {
		return TaxonomyTerm{}
	}
	t := TaxonomyTerm{
		ID:          uint(msg.Id),
		NodeType:    msg.NodeType,
		Taxonomy:    msg.Taxonomy,
		Slug:        msg.Slug,
		Name:        msg.Name,
		Description: msg.Description,
		Count:       int(msg.Count),
	}
	if msg.HasParentId {
		pid := uint(msg.ParentId)
		t.ParentID = &pid
	}
	if msg.FieldsDataJson != "" {
		var fields map[string]any
		if err := json.Unmarshal([]byte(msg.FieldsDataJson), &fields); err == nil {
			t.FieldsData = fields
		}
	}
	return t
}

func taxonomyToProto(t *Taxonomy) *pb.TaxonomyMessage {
	if t == nil {
		return nil
	}
	fields := make([]*pb.NodeTypeFieldMessage, len(t.FieldSchema))
	for i, f := range t.FieldSchema {
		fields[i] = &pb.NodeTypeFieldMessage{
			Name:     f.Name,
			Label:    f.Label,
			Type:     f.Type,
			Required: f.Required,
			Options:  f.OptionsToStrings(),
		}
	}
	return &pb.TaxonomyMessage{
		Id:          uint32(t.ID),
		Slug:        t.Slug,
		Label:       t.Label,
		Description: t.Description,
		NodeTypes:   t.NodeTypes,
		FieldSchema: fields,
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
	}
}

func taxonomyInputFromProto(inp *pb.TaxonomyInputMessage) TaxonomyInput {
	if inp == nil {
		return TaxonomyInput{}
	}
	var fields []NodeTypeField
	for _, f := range inp.FieldSchema {
		fields = append(fields, NodeTypeField{
			Name:     f.Name,
			Label:    f.Label,
			Type:     f.Type,
			Required: f.Required,
			Options:  OptionsFromStrings(f.Options),
		})
	}
	return TaxonomyInput{
		Slug:        inp.Slug,
		Label:       inp.Label,
		Description: inp.Description,
		NodeTypes:   inp.NodeTypes,
		FieldSchema: fields,
	}
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
