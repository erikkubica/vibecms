package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strconv"
	"strings"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"vibecms/internal/coreapi"
	vibeplugin "vibecms/pkg/plugin"
	coreapipb "vibecms/pkg/plugin/coreapipb"
	pb "vibecms/pkg/plugin/proto"
)

const formsTable = "forms"
const submissionsTable = "form_submissions"

type FormsPlugin struct {
	host           coreapi.CoreAPI
	rateLimiter    *RateLimiter
	shutdownCancel context.CancelFunc
}

func (p *FormsPlugin) GetSubscriptions() ([]*pb.Subscription, error) {
	return []*pb.Subscription{
		{EventName: "forms:render", Priority: 0},
		{EventName: "forms:upsert", Priority: 0},
	}, nil
}

func (p *FormsPlugin) HandleEvent(action string, payload []byte) (*pb.EventResponse, error) {
	ctx := context.Background()
	switch action {
	case "forms:render":
		return p.handleRenderEvent(ctx, payload)
	case "forms:upsert":
		return p.handleUpsertEvent(ctx, payload)
	}
	return &pb.EventResponse{Handled: false}, nil
}

// handleRenderEvent renders a form to raw HTML for template inclusion.
// Payload: { form_id: "<slug-or-id>", hidden: { key: value, ... } }
// `hidden` keys are injected as <input type="hidden"> before </form>, letting
// templates pass per-page context (trip_slug, price, etc) into submissions.
func (p *FormsPlugin) handleRenderEvent(ctx context.Context, payload []byte) (*pb.EventResponse, error) {
	var data struct {
		FormID string         `json:"form_id"`
		Hidden map[string]any `json:"hidden"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return &pb.EventResponse{Handled: false, Error: err.Error()}, nil
	}
	if data.FormID == "" {
		return &pb.EventResponse{Handled: false, Error: "form_id required"}, nil
	}

	form, err := p.lookupForm(ctx, data.FormID)
	if err != nil {
		return &pb.EventResponse{Handled: false, Error: err.Error()}, nil
	}

	html, err := p.renderFormHTML(form)
	if err != nil {
		return &pb.EventResponse{Handled: false, Error: err.Error()}, nil
	}

	if len(data.Hidden) > 0 {
		html = injectHiddenInputs(html, data.Hidden)
	}

	return &pb.EventResponse{Handled: true, Result: []byte(html)}, nil
}

// handleUpsertEvent creates a form by slug if no row with that slug exists.
// Payload: { slug, name, fields, layout?, notifications?, settings?, force? }
// If a form with the slug already exists and `force` is not true, the call is
// a no-op so editor changes survive theme re-imports.
func (p *FormsPlugin) handleUpsertEvent(ctx context.Context, payload []byte) (*pb.EventResponse, error) {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return &pb.EventResponse{Handled: false, Error: err.Error()}, nil
	}
	slug, _ := data["slug"].(string)
	name, _ := data["name"].(string)
	if slug == "" || name == "" {
		return &pb.EventResponse{Handled: false, Error: "slug and name required"}, nil
	}

	existing, _ := p.host.DataQuery(ctx, formsTable, coreapi.DataStoreQuery{
		Where: map[string]any{"slug": slug},
		Limit: 1,
	})
	force, _ := data["force"].(bool)
	if existing != nil && existing.Total > 0 && !force {
		return &pb.EventResponse{Handled: true, Result: []byte(`{"status":"skipped"}`)}, nil
	}

	// Default to the "simple" layout style when none specified, so seeded forms
	// render out of the box. Callers can pass `layout: "<go-template>"` for a
	// raw template, or `style: "grid"|"card"|"inline"|"simple"` for a preset.
	layout, _ := data["layout"].(string)
	if layout == "" {
		style, _ := data["style"].(string)
		layout = defaultLayoutForStyle(style)
	}
	row := map[string]any{
		"slug":   slug,
		"name":   name,
		"fields": data["fields"],
		"layout": layout,
	}
	if v, ok := data["notifications"]; ok {
		row["notifications"] = v
	}
	if v, ok := data["settings"]; ok {
		row["settings"] = v
	}

	if existing != nil && existing.Total > 0 {
		id, _ := toUint(existing.Rows[0]["id"])
		if err := p.host.DataUpdate(ctx, formsTable, id, row); err != nil {
			return &pb.EventResponse{Handled: false, Error: err.Error()}, nil
		}
		return &pb.EventResponse{Handled: true, Result: []byte(`{"status":"updated"}`)}, nil
	}
	if _, err := p.host.DataCreate(ctx, formsTable, row); err != nil {
		return &pb.EventResponse{Handled: false, Error: err.Error()}, nil
	}
	return &pb.EventResponse{Handled: true, Result: []byte(`{"status":"created"}`)}, nil
}

// lookupForm fetches a form by numeric id or slug and returns the normalized row.
func (p *FormsPlugin) lookupForm(ctx context.Context, identifier string) (map[string]any, error) {
	if id, err := strconv.ParseUint(identifier, 10, 64); err == nil {
		row, err := p.host.DataGet(ctx, formsTable, uint(id))
		if err != nil {
			return nil, fmt.Errorf("form not found: %s", identifier)
		}
		return normalizeForm(row), nil
	}
	res, err := p.host.DataQuery(ctx, formsTable, coreapi.DataStoreQuery{
		Where: map[string]any{"slug": identifier},
		Limit: 1,
	})
	if err != nil || res.Total == 0 {
		return nil, fmt.Errorf("form not found: %s", identifier)
	}
	return normalizeForm(res.Rows[0]), nil
}

// injectHiddenInputs inserts <input type="hidden"> tags for each key/value pair
// just before the closing </form> tag. Values are HTML-escaped.
func injectHiddenInputs(html string, hidden map[string]any) string {
	if len(hidden) == 0 {
		return html
	}
	var b strings.Builder
	for k, v := range hidden {
		val := fmt.Sprintf("%v", v)
		b.WriteString(fmt.Sprintf(`<input type="hidden" name=%q value=%q />`,
			template.HTMLEscapeString(k), template.HTMLEscapeString(val)))
	}
	idx := strings.LastIndex(strings.ToLower(html), "</form>")
	if idx == -1 {
		return html + b.String()
	}
	return html[:idx] + b.String() + html[idx:]
}

func toUint(v any) (uint, bool) {
	switch n := v.(type) {
	case uint:
		return n, true
	case int:
		return uint(n), true
	case int64:
		return uint(n), true
	case float64:
		return uint(n), true
	case string:
		i, err := strconv.ParseUint(n, 10, 64)
		if err != nil {
			return 0, false
		}
		return uint(i), true
	}
	return 0, false
}

func (p *FormsPlugin) Initialize(hostConn *grpc.ClientConn) error {
	p.host = coreapi.NewGRPCHostClient(coreapipb.NewVibeCMSHostClient(hostConn))
	p.rateLimiter = NewRateLimiter(10000)
	ctx, cancel := context.WithCancel(context.Background())
	p.shutdownCancel = cancel
	p.startRetentionWorker(ctx)
	return nil
}

func (p *FormsPlugin) Shutdown() error {
	if p.shutdownCancel != nil {
		p.shutdownCancel()
	}
	return nil
}

func (p *FormsPlugin) HandleHTTPRequest(req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	return p.routeRequest(context.Background(), req)
}











func main() {
	p := &FormsPlugin{}
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: vibeplugin.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"extension": &vibeplugin.ExtensionGRPCPlugin{Impl: p},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}
