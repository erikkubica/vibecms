package cms

import (
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// RegisterAdminPreviewRoutes mounts the admin-side preview endpoints used
// by the node editor's Preview button. Two shapes:
//
//   GET  /admin/api/nodes/:id/preview
//        Render the persisted row as-is. Useful for previewing a node
//        the way it would render right now without any editor context.
//
//   POST /admin/api/nodes/:id/preview
//        Render the node with the request body's draft overrides applied
//        in-memory only. Nothing is written to the DB. Use this when the
//        editor has unsaved changes and the operator wants to see the
//        page exactly as it will look — without committing.
//
// Both endpoints return text/html (the rendered page) on success and an
// inline HTML error page on failure. JSON envelopes don't render well in
// fresh browser tabs.
func (h *PublicHandler) RegisterAdminPreviewRoutes(router fiber.Router) {
	router.Get("/nodes/:id/preview", h.AdminNodePreview)
	router.Post("/nodes/:id/preview", h.AdminNodePreviewWithDraft)
}

// AdminNodePreview renders the persisted row. No DB writes, no events.
func (h *PublicHandler) AdminNodePreview(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return previewHTMLError(c, fiber.StatusBadRequest, "Node id must be a positive integer")
	}
	return renderPreview(c, h, uint(id), nil)
}

// adminPreviewBody is the POST payload shape. Mirrors the node editor's
// in-flight form state. Every field is optional — omitted fields keep
// the persisted value. JSONB shapes (blocks_data, fields_data, etc.) are
// accepted as decoded JSON; we re-encode before handing them to the
// renderer.
type adminPreviewBody struct {
	Title         *string         `json:"title,omitempty"`
	Slug          *string         `json:"slug,omitempty"`
	Status        *string         `json:"status,omitempty"`
	LanguageCode  *string         `json:"language_code,omitempty"`
	Excerpt       *string         `json:"excerpt,omitempty"`
	LayoutSlug    *string         `json:"layout_slug,omitempty"`
	LayoutID      *int            `json:"layout_id,omitempty"`
	BlocksData    json.RawMessage `json:"blocks_data,omitempty"`
	FieldsData    json.RawMessage `json:"fields_data,omitempty"`
	SeoSettings   json.RawMessage `json:"seo_settings,omitempty"`
	FeaturedImage json.RawMessage `json:"featured_image,omitempty"`
	Taxonomies    json.RawMessage `json:"taxonomies,omitempty"`
}

// AdminNodePreviewWithDraft renders a node with in-flight form state
// applied in-memory. Body must be JSON-shaped (adminPreviewBody) and
// every field is optional. Nothing is written to the database.
func (h *PublicHandler) AdminNodePreviewWithDraft(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return previewHTMLError(c, fiber.StatusBadRequest, "Node id must be a positive integer")
	}

	var body adminPreviewBody
	if err := c.BodyParser(&body); err != nil {
		return previewHTMLError(c, fiber.StatusBadRequest, "Could not parse preview payload: "+err.Error())
	}

	draft := &NodeDraftOverrides{
		Title:         body.Title,
		Slug:          body.Slug,
		Status:        body.Status,
		LanguageCode:  body.LanguageCode,
		Excerpt:       body.Excerpt,
		LayoutSlug:    body.LayoutSlug,
		LayoutID:      body.LayoutID,
		BlocksData:    rawOrNil(body.BlocksData),
		FieldsData:    rawOrNil(body.FieldsData),
		SeoSettings:   rawOrNil(body.SeoSettings),
		FeaturedImage: rawOrNil(body.FeaturedImage),
		Taxonomies:    rawOrNil(body.Taxonomies),
	}
	return renderPreview(c, h, uint(id), draft)
}

// rawOrNil returns the underlying bytes when the RawMessage is non-empty,
// otherwise nil so the renderer keeps the persisted JSONB column as-is.
func rawOrNil(m json.RawMessage) []byte {
	if len(m) == 0 || string(m) == "null" {
		return nil
	}
	return m
}

func renderPreview(c *fiber.Ctx, h *PublicHandler, id uint, draft *NodeDraftOverrides) error {
	rendered, err := h.RenderNodePreview(id, draft)
	if err != nil {
		return previewHTMLError(c, fiber.StatusNotFound, err.Error())
	}
	if rendered == "" {
		return previewHTMLError(c, fiber.StatusInternalServerError,
			"Renderer returned an empty document. Verify the node has a layout assigned and that the active theme provides templates for it.")
	}
	// Inject <base href> so absolute paths in the rendered HTML resolve
	// against the live site origin even when the editor opens this
	// response inside a blob: URL. Without this, /theme/assets/...,
	// /media/..., and /forms/submit/... 404 because the browser tries
	// to resolve them against the blob URL's path component.
	rendered = injectBaseHref(rendered, baseHrefForRequest(c))
	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("Cache-Control", "no-store, max-age=0")
	return c.Status(fiber.StatusOK).SendString(rendered)
}

// baseHrefForRequest returns the absolute origin (scheme + host) the
// editor opened the admin from, so the preview response carries a real
// base URL even when it's loaded inside a blob: document.
func baseHrefForRequest(c *fiber.Ctx) string {
	scheme := c.Protocol()
	host := c.Hostname()
	if host == "" {
		host = string(c.Request().Host())
	}
	if host == "" {
		// No way to recover — return relative root and hope the consumer
		// is still loading the response on the same origin.
		return "/"
	}
	return scheme + "://" + host + "/"
}

// injectBaseHref inserts a <base href="..."> tag immediately after <head>
// in the rendered HTML. Idempotent: if the document already declares a
// base tag we leave it alone (themes can opt out by shipping their own).
func injectBaseHref(htmlDoc, base string) string {
	// Quick guard — if the theme already wrote its own base tag, don't
	// double up.
	if strings.Contains(htmlDoc, "<base ") {
		return htmlDoc
	}
	idx := strings.Index(htmlDoc, "<head>")
	if idx < 0 {
		idx = strings.Index(htmlDoc, "<head ")
	}
	if idx < 0 {
		return htmlDoc
	}
	closeTag := strings.Index(htmlDoc[idx:], ">")
	if closeTag < 0 {
		return htmlDoc
	}
	insertPos := idx + closeTag + 1
	return htmlDoc[:insertPos] + `<base href="` + html.EscapeString(base) + `">` + htmlDoc[insertPos:]
}

// previewHTMLError renders a minimal-but-readable error page. Used in
// AdminNodePreview where the response always lands in a fresh browser tab.
func previewHTMLError(c *fiber.Ctx, status int, message string) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("Cache-Control", "no-store, max-age=0")
	body := fmt.Sprintf(`<!doctype html>
<html><head><meta charset="utf-8"><title>Preview unavailable</title>
<style>body{font:14px/1.5 system-ui, sans-serif; color:#0f172a; max-width:560px; margin:48px auto; padding:0 24px}
h1{font-size:18px; margin:0 0 12px} .code{font-family:ui-monospace, monospace; background:#f1f5f9; padding:2px 6px; border-radius:4px}
.box{padding:16px 20px; border:1px solid #e2e8f0; border-radius:8px; background:#fafafa}</style>
</head><body>
<h1>Preview unavailable</h1>
<div class="box">
  <p><strong>%d</strong> — could not render this node.</p>
  <p class="code">%s</p>
  <p style="color:#64748b">If the node loads on the public site, this is most likely a missing layout assignment or a draft node referencing a layout that no longer exists.</p>
</div>
</body></html>`, status, html.EscapeString(message))
	return c.Status(status).SendString(body)
}
