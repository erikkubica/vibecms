package main

import (
	"context"
	"encoding/json"
	"strconv"

	"vibecms/internal/coreapi"
	pb "vibecms/pkg/plugin/proto"
)

func (p *FormsPlugin) handleRender(ctx context.Context, identifier string) (*pb.PluginHTTPResponse, error) {
	if id, err := strconv.ParseUint(identifier, 10, 64); err == nil {
		res, err := p.host.DataGet(ctx, formsTable, uint(id))
		if err != nil {
			return jsonError(404, "FORM_NOT_FOUND", "Form not found"), nil
		}
		res = normalizeForm(res)
		html, err := p.renderFormHTML(res)
		if err != nil {
			return jsonError(500, "RENDER_FAILED", err.Error()), nil
		}
		return jsonResponse(200, map[string]any{"html": html, "form": res}), nil
	}

	query := coreapi.DataStoreQuery{
		Where: map[string]any{"slug": identifier},
		Limit: 1,
	}
	res, err := p.host.DataQuery(ctx, formsTable, query)
	if err != nil || res.Total == 0 {
		return jsonError(404, "FORM_NOT_FOUND", "Form not found"), nil
	}
	form := normalizeForm(res.Rows[0])

	html, err := p.renderFormHTML(form)
	if err != nil {
		return jsonError(500, "RENDER_FAILED", err.Error()), nil
	}

	return jsonResponse(200, map[string]any{"html": html, "form": form}), nil
}

func (p *FormsPlugin) handlePreview(ctx context.Context, req *pb.PluginHTTPRequest) (*pb.PluginHTTPResponse, error) {
	var input map[string]any
	if err := json.Unmarshal(req.GetBody(), &input); err != nil {
		return jsonError(400, "INVALID_BODY", err.Error()), nil
	}

	html, err := p.renderFormHTML(input)
	if err != nil {
		return jsonError(500, "RENDER_FAILED", err.Error()), nil
	}

	return jsonResponse(200, map[string]any{"html": html}), nil
}
