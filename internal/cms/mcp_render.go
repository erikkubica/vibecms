package cms

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"squilla/internal/models"
)

// MCP-facing render helpers. These let the MCP core.render.* tools preview
// content without going through the public HTTP path — no Fiber context, no
// redirects, no side-effects. They reuse the existing render primitives on
// PublicHandler so behavior stays identical to what a published page produces.

// RenderBlockPreview renders a single content block of the given type with the
// given field values. Returns the rendered HTML.
func (h *PublicHandler) RenderBlockPreview(blockType string, fields map[string]interface{}) (string, error) {
	block := map[string]interface{}{
		"type":   blockType,
		"fields": fields,
	}
	out := h.renderBlocksBatch([]map[string]interface{}{block}, "")
	if len(out) == 0 {
		return "", fmt.Errorf("render returned no output for block %q", blockType)
	}
	return out[0], nil
}

// RenderBlocksPreview renders a series of inline block specs into an HTML
// fragment (concatenated). Each block is a map with keys "type" and "fields".
func (h *PublicHandler) RenderBlocksPreview(blocks []map[string]interface{}) (string, error) {
	out := h.renderBlocksBatch(blocks, "")
	return strings.Join(out, "\n"), nil
}

// RenderLayoutPreview renders a layout with the given inline blocks. The
// layout is looked up by slug (optional languageCode narrows to a specific
// language variant). Use this to preview how a block composition looks inside
// a given layout without creating or publishing a node.
func (h *PublicHandler) RenderLayoutPreview(layoutSlug string, blocks []map[string]interface{}, languageCode string) (string, error) {
	languages := h.loadActiveLanguages()
	var languageID *int
	var currentLang *models.Language
	for i := range languages {
		if languageCode == "" && languages[i].IsDefault {
			id := languages[i].ID
			languageID = &id
			currentLang = &languages[i]
			break
		}
		if languages[i].Code == languageCode {
			id := languages[i].ID
			languageID = &id
			currentLang = &languages[i]
			break
		}
	}

	layout := h.layoutSvc.findBySlugAndLang(layoutSlug, languageID)
	if layout == nil {
		return "", fmt.Errorf("layout %q not found", layoutSlug)
	}

	renderedBlocks := h.renderBlocksBatch(blocks, languageCode)
	blocksHTML := strings.Join(renderedBlocks, "\n")

	settings := h.loadSiteSettings()
	menus := h.renderCtx.LoadMenus(languageID)
	usedSlugs := extractBlockSlugs(blocks)
	appData := h.renderCtx.BuildAppData(settings, languages, currentLang, usedSlugs)
	appData.Menus = menus

	// Stub node data — preview has no real node.
	fakeNode := &models.ContentNode{
		Title:        "[MCP Preview]",
		LanguageCode: func() string { if currentLang != nil { return currentLang.Code } ; return "" }(),
	}
	nodeData := h.renderCtx.BuildNodeData(fakeNode, blocksHTML, languages)

	templateData := TemplateData{App: appData, Node: nodeData}
	dataMap := templateData.ToMap()

	blockResolver := func(slug string) (string, error) {
		lb, err := h.layoutBlockSvc.Resolve(slug, languageID)
		if err != nil {
			return "", err
		}
		return lb.TemplateCode, nil
	}

	var buf bytes.Buffer
	if err := h.renderer.RenderLayout(&buf, layout.TemplateCode, dataMap, blockResolver); err != nil {
		return "", fmt.Errorf("render layout: %w", err)
	}
	return buf.String(), nil
}

// RenderNodePreview renders a specific node (published or draft) via the
// layout engine, returning HTML. No view counts, no node.viewed events.
func (h *PublicHandler) RenderNodePreview(nodeID uint) (string, error) {
	var node models.ContentNode
	if err := h.db.Where("id = ?", nodeID).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("node %d not found", nodeID)
		}
		return "", err
	}

	blocks := parseBlocks(node.BlocksData)
	renderedBlocks := h.renderBlocksBatch(blocks, node.LanguageCode)

	// Reuse the main render path but pass nil for Fiber ctx and nil user —
	// renderNodeWithLayout reads from c only via currentUser which we skip.
	// To avoid touching c, we inline the essentials of renderNodeWithLayout.
	languages := h.loadActiveLanguages()
	var languageID *int
	var currentLang *models.Language
	for i := range languages {
		if languages[i].Code == node.LanguageCode {
			id := languages[i].ID
			languageID = &id
			currentLang = &languages[i]
			break
		}
	}

	layout, err := h.layoutSvc.ResolveForNode(&node, languageID)
	if err != nil || layout == nil {
		return "", fmt.Errorf("no layout resolved for node %d", nodeID)
	}

	settings := h.loadSiteSettings()
	menus := h.renderCtx.LoadMenus(languageID)
	blocksHTML := strings.Join(renderedBlocks, "\n")
	usedSlugs := extractBlockSlugs(blocks)
	appData := h.renderCtx.BuildAppData(settings, languages, currentLang, usedSlugs)
	appData.Menus = menus
	nodeData := h.renderCtx.BuildNodeData(&node, blocksHTML, languages)

	templateData := TemplateData{App: appData, Node: nodeData}
	blockResolver := func(slug string) (string, error) {
		lb, err := h.layoutBlockSvc.Resolve(slug, languageID)
		if err != nil {
			return "", err
		}
		return lb.TemplateCode, nil
	}

	dataMap := templateData.ToMap()
	partialData := h.buildPartialData(&node, layout, languageID, dataMap)

	var buf bytes.Buffer
	if err := h.renderer.RenderLayout(&buf, layout.TemplateCode, dataMap, blockResolver, partialData); err != nil {
		return "", fmt.Errorf("render layout: %w", err)
	}
	return buf.String(), nil
}
