package cms

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"

	"squilla/internal/models"
)

// This file owns the block-rendering side of the public handler:
// resolving block-type templates, the per-block render path, the
// batch-hydration variant, and the block-output cache key. Keeping it
// separate from the page-flow code in public_handler.go means the
// render code can be reasoned about (and tuned) without scrolling
// past 1500 lines of unrelated handler logic.

// getBlockType reads a block-type record from the in-memory cache,
// loading from the database on first use. The cache is invalidated
// via PublicHandler.ClearCache when block_type events fire.
func (h *PublicHandler) getBlockType(slug string) (models.BlockType, bool) {
	h.cacheMu.RLock()
	blocks := h.blockTypes
	h.cacheMu.RUnlock()

	if blocks == nil {
		h.cacheMu.Lock()
		if h.blockTypes == nil {
			var dt []models.BlockType
			h.db.Find(&dt)
			blocks = make(map[string]models.BlockType)
			for _, b := range dt {
				blocks[b.Slug] = b
			}
			h.blockTypes = blocks
		} else {
			blocks = h.blockTypes
		}
		h.cacheMu.Unlock()
	}

	bt, ok := blocks[slug]
	return bt, ok
}

// renderBlocks renders each block's HTML template with its field values.
func (h *PublicHandler) renderBlocks(blocks []map[string]interface{}) []string {
	// Load theme settings once per call so the GetSettings round-trip
	// happens exactly once, not once per block.
	ts := h.loadThemeSettingsForRender(context.Background())

	var rendered []string
	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		fields, _ := block["fields"].(map[string]interface{})
		if fields == nil {
			fields = block // fallback for old format
		}

		bt, ok := h.getBlockType(blockType)
		if !ok || bt.HTMLTemplate == "" {
			// Fallback: render raw JSON debug block
			jsonBytes, _ := json.MarshalIndent(fields, "", "  ")
			rendered = append(rendered, fmt.Sprintf(
				`<div class="mb-8 bg-slate-50 rounded-xl p-6 border border-slate-200">
					<div class="flex items-center gap-2 mb-3">
						<span class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-semibold bg-slate-200 text-slate-700">%s</span>
						<span class="text-xs text-slate-400">No template defined</span>
					</div>
					<pre class="text-sm text-slate-600 overflow-x-auto bg-white rounded-lg p-4 border border-slate-200"><code>%s</code></pre>
				</div>`, blockType, string(jsonBytes)))
			continue
		}

		// Check for theme file override with caching
		tmplContent := bt.HTMLTemplate
		themeFile := fmt.Sprintf("themes/default/blocks/%s.html", blockType)

		h.cacheMu.RLock()
		cachedContent, hasCache := h.themeBlockCache[themeFile]
		h.cacheMu.RUnlock()

		if hasCache {
			if cachedContent != "" {
				tmplContent = cachedContent
			}
		} else {
			if fileContent, err := os.ReadFile(themeFile); err == nil {
				tmplContent = string(fileContent)
				h.cacheMu.Lock()
				h.themeBlockCache[themeFile] = tmplContent
				h.cacheMu.Unlock()
			} else {
				h.cacheMu.Lock()
				h.themeBlockCache[themeFile] = "" // Cache the miss
				h.cacheMu.Unlock()
			}
		}

		// Hydrate node references — resolve node selector fields to full node data
		markRichTextFields(fields, bt.FieldSchema)
		h.hydrateTermFields(fields, bt.FieldSchema)

		// Use the new RenderParsed method for cached template execution
		cacheKey := "block:" + blockType + ":" + tmplContent
		var buf bytes.Buffer
		err := h.renderer.RenderParsed(&buf, cacheKey, tmplContent, fields, mergeFuncMaps(template.FuncMap{
			"safeHTML": func(s interface{}) template.HTML {
				return template.HTML(fmt.Sprintf("%v", s))
			},
			"safeURL": func(s interface{}) template.URL {
				return template.URL(fmt.Sprintf("%v", s))
			},
		}, themeSettingsFuncs(ts)))
		if err != nil {
			// Escape blockType and err — both can carry user input
			// (slug from a malicious manifest, parser error echoing
			// raw template source) into the public-facing response.
			rendered = append(rendered, fmt.Sprintf(`<div class="mb-4 text-red-500 text-sm">Template error in %s: %s</div>`,
				template.HTMLEscapeString(blockType),
				template.HTMLEscapeString(err.Error()),
			))
			continue
		}

		rendered = append(rendered, buf.String())
	}
	return rendered
}

// renderBlocksBatch is the optimized version that performs batch node hydration.
func (h *PublicHandler) renderBlocksBatch(blocks []map[string]interface{}) []string {
	var rendered []string
	if len(blocks) == 0 {
		return rendered
	}

	// Load theme settings once per page render so the GetSettings round-trip
	// happens exactly once, regardless of how many blocks we render.
	ts := h.loadThemeSettingsForRender(context.Background())

	// Step 1: Collect all node IDs across all blocks
	allNodeIDs := make(map[int]bool)
	for _, block := range blocks {
		fields, _ := block["fields"].(map[string]interface{})
		if fields == nil {
			fields = block
		}
		collectNodeIDs(fields, allNodeIDs)
	}

	// Step 2: Batch fetch nodes if any IDs were found
	nodeMap := make(map[int]map[string]interface{})
	if len(allNodeIDs) > 0 {
		var ids []int
		for id := range allNodeIDs {
			ids = append(ids, id)
		}
		var nodes []models.ContentNode
		if err := h.db.Where("id IN ?", ids).Find(&nodes).Error; err == nil {
			// Get node types for schema info
			typeSlugs := make(map[string]bool)
			for _, n := range nodes {
				if n.NodeType != "" {
					typeSlugs[n.NodeType] = true
				}
			}
			var slugs []string
			for s := range typeSlugs {
				slugs = append(slugs, s)
			}
			var nodeTypes []models.NodeType
			if len(slugs) > 0 {
				h.db.Where("slug IN ?", slugs).Find(&nodeTypes)
			}
			typeSchemaMap := make(map[string]models.JSONB)
			for _, nt := range nodeTypes {
				typeSchemaMap[nt.Slug] = nt.FieldSchema
			}

			for _, n := range nodes {
				fields := make(map[string]interface{})
				json.Unmarshal(n.FieldsData, &fields)
				schema := typeSchemaMap[n.NodeType]
				markRichTextFields(fields, schema)

				var featuredImage interface{}
				if len(n.FeaturedImage) > 0 {
					json.Unmarshal(n.FeaturedImage, &featuredImage)
				}

				var taxonomies map[string][]string
				if len(n.Taxonomies) > 0 {
					json.Unmarshal(n.Taxonomies, &taxonomies)
				}

				nodeMap[n.ID] = map[string]interface{}{
					"id":             n.ID,
					"title":          n.Title,
					"slug":           n.Slug,
					"full_url":       n.FullURL,
					"featured_image": featuredImage,
					"excerpt":        n.Excerpt,
					"taxonomies":     taxonomies,
					"fields":         fields,
					"node_type":      n.NodeType,
					"language_code":  n.LanguageCode,
					"status":         n.Status,
				}
			}
		}
	}

	// Step 3: Render each block with pre-hydrated nodeMap
	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		fields, _ := block["fields"].(map[string]interface{})
		if fields == nil {
			fields = block
		}

		bt, ok := h.getBlockType(blockType)
		if !ok || bt.HTMLTemplate == "" {
			jsonBytes, _ := json.MarshalIndent(fields, "", "  ")
			// Escape both before interpolation: blockType could be a
			// user-crafted slug, and the JSON payload reflects every
			// field value the editor saved — including ones an
			// attacker controls. JSON's own quoting wouldn't survive
			// raw HTML interpolation.
			rendered = append(rendered, fmt.Sprintf(`<div class="mb-8 bg-slate-50 rounded-xl p-6 border border-slate-200">
				<div class="flex items-center gap-2 mb-3">
					<span class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-semibold bg-slate-200 text-slate-700">%s</span>
				</div>
				<pre class="text-sm text-slate-600 bg-white rounded-lg p-4 border border-slate-200"><code>%s</code></pre>
			</div>`,
				template.HTMLEscapeString(blockType),
				template.HTMLEscapeString(string(jsonBytes)),
			))
			continue
		}

		// Apply batch-hydrated nodes
		applyHydratedNodes(fields, nodeMap)

		// Check block output cache (only for blocks with cache_output enabled)
		if bt.CacheOutput {
			outputKey := blockOutputKey(blockType, fields)
			h.cacheMu.RLock()
			cached, hit := h.blockOutputCache[outputKey]
			h.cacheMu.RUnlock()
			if hit {
				rendered = append(rendered, cached)
				continue
			}
		}

		tmplContent := bt.HTMLTemplate
		themeFile := fmt.Sprintf("themes/default/blocks/%s.html", blockType)
		h.cacheMu.RLock()
		cachedContent, hasCache := h.themeBlockCache[themeFile]
		h.cacheMu.RUnlock()
		if hasCache && cachedContent != "" {
			tmplContent = cachedContent
		} else if !hasCache {
			if fileContent, err := os.ReadFile(themeFile); err == nil {
				tmplContent = string(fileContent)
				h.cacheMu.Lock()
				h.themeBlockCache[themeFile] = tmplContent
				h.cacheMu.Unlock()
			} else {
				h.cacheMu.Lock()
				h.themeBlockCache[themeFile] = ""
				h.cacheMu.Unlock()
			}
		}

		markRichTextFields(fields, bt.FieldSchema)
		h.hydrateTermFields(fields, bt.FieldSchema)

		tmplCacheKey := "block:" + blockType + ":" + tmplContent
		var buf bytes.Buffer
		err := h.renderer.RenderParsed(&buf, tmplCacheKey, tmplContent, fields, mergeFuncMaps(template.FuncMap{
			"safeHTML": func(s interface{}) template.HTML {
				return template.HTML(fmt.Sprintf("%v", s))
			},
			"safeURL": func(s interface{}) template.URL {
				return template.URL(fmt.Sprintf("%v", s))
			},
		}, themeSettingsFuncs(ts)))
		if err != nil {
			log.Printf("WARN: block template render error [%s]: %v", blockType, err)
			continue
		}

		output := buf.String()

		// Store in block output cache if enabled
		if bt.CacheOutput {
			outputKey := blockOutputKey(blockType, fields)
			h.cacheMu.Lock()
			h.blockOutputCache[outputKey] = output
			h.cacheMu.Unlock()
		}

		rendered = append(rendered, output)
	}
	return rendered
}

// blockOutputKey generates a cache key for a rendered block based on
// its type and field content. The key is short (16 bytes of SHA-256
// hex) so it's cheap to compare and unlikely to collide for the
// per-block render-cache map sizes we expect.
func blockOutputKey(blockType string, fields map[string]interface{}) string {
	b, _ := json.Marshal(fields)
	hash := sha256.Sum256(b)
	return blockType + ":" + hex.EncodeToString(hash[:16])
}
