package sdui

import (
	"encoding/json"
	"fmt"
	"strconv"

	"gorm.io/gorm"

	"squilla/internal/models"
)

// buildLanguageList shapes active languages into the wire format the
// SearchToolbar expects. Shared between node list and term list so the
// language filter dropdown looks identical in both places.
func buildLanguageList(db *gorm.DB) []map[string]interface{} {
	var languages []models.Language
	db.Where("is_active = ?", true).Order("sort_order ASC, name ASC").Find(&languages)
	out := make([]map[string]interface{}, 0, len(languages))
	for _, lang := range languages {
		out = append(out, map[string]interface{}{
			"code": lang.Code,
			"name": lang.Name,
			"flag": lang.Flag,
		})
	}
	return out
}

// nodeListLayout renders /admin/{pages|posts|content/<type>}.
// Counts per status come from a separate raw query (rather than
// re-reading the filtered slice) so tabs reflect the unfiltered total
// while the table reflects the active status filter.
func (e *Engine) nodeListLayout(params map[string]string) *LayoutNode {
	nodeTypeSlug := params["nodeType"]
	if nodeTypeSlug == "" {
		return e.defaultLayout("node-list")
	}

	// 1. Get NodeType by slug
	var nt models.NodeType
	if err := e.db.Where("slug = ?", nodeTypeSlug).First(&nt).Error; err != nil {
		return e.defaultLayout("node-list")
	}

	// Resolve display labels
	labelPlural := nt.LabelPlural
	if labelPlural == "" {
		labelPlural = nt.Label
	}

	// 2. Get taxonomy definitions for this node type
	var taxSlugs []string
	if err := json.Unmarshal(nt.Taxonomies, &taxSlugs); err != nil {
		taxSlugs = []string{}
	}

	var taxonomyDefs []models.Taxonomy
	if len(taxSlugs) > 0 {
		e.db.Where("slug IN ?", taxSlugs).Order("label ASC").Find(&taxonomyDefs)
	}

	taxonomyDefsList := make([]map[string]interface{}, 0, len(taxonomyDefs))
	for _, t := range taxonomyDefs {
		taxonomyDefsList = append(taxonomyDefsList, map[string]interface{}{
			"slug":  t.Slug,
			"label": t.Label,
		})
	}

	// 3. Build base query for ContentNode
	page, _ := strconv.Atoi(params["page"])
	if page < 1 {
		page = 1
	}
	perPage := getPerPage(params)
	offset := (page - 1) * perPage

	sortBy := params["sort"]
	sortOrder := params["order"]
	switch sortBy {
	case "title", "updated_at", "created_at":
	default:
		sortBy = "updated_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	query := e.db.Model(&models.ContentNode{}).Where("node_type = ? AND deleted_at IS NULL", nodeTypeSlug)

	// Optional search filter (applied before status counting)
	if search := params["search"]; search != "" {
		query = query.Where("title ILIKE ?", "%"+search+"%")
	}

	// Optional language filter (applied before status counting)
	if lang := params["language"]; lang != "" && lang != "all" {
		query = query.Where("language_code = ?", lang)
	}

	// Apply taxonomy term filters from URL params
	activeTaxFilters := make([]map[string]interface{}, 0)
	for _, taxDef := range taxonomyDefs {
		if termName := params[taxDef.Slug]; termName != "" {
			query = query.Where("taxonomies->? @> ?::jsonb", taxDef.Slug, fmt.Sprintf(`["%s"]`, termName))
			activeTaxFilters = append(activeTaxFilters, map[string]interface{}{
				"taxonomy": taxDef.Slug,
				"term":     termName,
				"label":    taxDef.Label,
			})
		}
	}

	// 4. Query status counts (unfiltered by status, but filtered by node_type, search, language, taxonomy)
	baseCountQuery := "SELECT status, COUNT(*) as count FROM content_nodes WHERE node_type = ? AND deleted_at IS NULL"
	countArgs := []interface{}{nodeTypeSlug}
	if search := params["search"]; search != "" {
		baseCountQuery += " AND title ILIKE ?"
		countArgs = append(countArgs, "%"+search+"%")
	}
	if lang := params["language"]; lang != "" && lang != "all" {
		baseCountQuery += " AND language_code = ?"
		countArgs = append(countArgs, lang)
	}
	for _, taxDef := range taxonomyDefs {
		if termName := params[taxDef.Slug]; termName != "" {
			baseCountQuery += " AND taxonomies->? @> ?::jsonb"
			countArgs = append(countArgs, taxDef.Slug, fmt.Sprintf(`["%s"]`, termName))
		}
	}
	baseCountQuery += " GROUP BY status"

	type statusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []statusCount
	e.db.Raw(baseCountQuery, countArgs...).Scan(&statusCounts)

	statusMap := make(map[string]int64)
	var totalAll int64
	for _, sc := range statusCounts {
		statusMap[sc.Status] = sc.Count
		totalAll += sc.Count
	}

	tabs := []map[string]interface{}{
		{"value": "all", "label": "All", "count": totalAll},
		{"value": "published", "label": "Published", "count": statusMap["published"]},
		{"value": "draft", "label": "Drafts", "count": statusMap["draft"]},
		{"value": "archived", "label": "Archived", "count": statusMap["archived"]},
	}

	// 5. Query languages
	var languages []models.Language
	e.db.Where("is_active = ?", true).Order("sort_order ASC, name ASC").Find(&languages)

	langList := make([]map[string]interface{}, 0, len(languages))
	for _, lang := range languages {
		langList = append(langList, map[string]interface{}{
			"code": lang.Code,
			"name": lang.Name,
			"flag": lang.Flag,
		})
	}

	// 6. Apply status filter for pagination
	if status := params["status"]; status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	// 7. Count total matching nodes (with status filter applied)
	var totalCount int64
	query.Count(&totalCount)

	// 8. Fetch paginated nodes
	var nodes []models.ContentNode
	query.Order(sortBy + " " + sortOrder).Offset(offset).Limit(perPage).Find(&nodes)

	// Calculate base path
	basePath := basePathForNodeType(nodeTypeSlug)

	// 9. Build rows
	rows := make([]map[string]interface{}, 0, len(nodes))
	for _, n := range nodes {
		// Parse taxonomies JSONB — map of taxonomy_slug → []term_names
		var nodeTax map[string][]string
		if err := json.Unmarshal(n.Taxonomies, &nodeTax); err != nil {
			nodeTax = map[string][]string{}
		}

		rows = append(rows, map[string]interface{}{
			"id":            n.ID,
			"title":         n.Title,
			"slug":          n.Slug,
			"status":        n.Status,
			"language_code": n.LanguageCode,
			"taxonomies":    nodeTax,
			"updated_at":    n.UpdatedAt.Format("2006-01-02"),
			"editPath":      fmt.Sprintf("%s/%d/edit", basePath, n.ID),
		})
	}

	totalPages := int(totalCount) / perPage
	if int(totalCount)%perPage > 0 {
		totalPages++
	}

	// Build children dynamically to support conditional TaxonomyFilterChips
	children := []LayoutNode{
		{
			Type: "PageHeader",
			Props: map[string]interface{}{
				"title": labelPlural,
				"tabs":  tabs,
				"activeTab": func() string {
					if s := params["status"]; s != "" && s != "all" {
						return s
					}
					return "all"
				}(),
				"newLabel":         "New " + nt.Label,
				"newPath":          basePath + "/new",
				"taxonomyDefs":     taxonomyDefsList,
				"activeTaxFilters": activeTaxFilters,
			},
			Actions: map[string]ActionDef{
				"onNew": {Type: "NAVIGATE", To: basePath + "/new"},
			},
		},
	}

	// Add taxonomy filter chips if any active
	if len(activeTaxFilters) > 0 {
		children = append(children, LayoutNode{
			Type: "TaxonomyFilterChips",
			Props: map[string]interface{}{
				"filters": activeTaxFilters,
			},
		})
	}

	children = append(children, []LayoutNode{
		{
			Type: "SearchToolbar",
			Props: map[string]interface{}{
				"searchPlaceholder": "Search by title or slug…",
				"languages":         langList,
				"activeLanguage":    params["language"],
			},
		},
		{
			Type: "ContentNodeTable",
			Props: map[string]interface{}{
				"nodeType":            nodeTypeSlug,
				"columns":             []string{"title", "status", "taxonomies", "language", "updated_at", "actions"},
				"rows":                rows,
				"pagination":          map[string]interface{}{"page": page, "perPage": perPage, "total": int(totalCount), "totalPages": totalPages},
				"taxonomyDefs":        taxonomyDefsList,
				"basePath":            basePath,
				"nodeTypeLabel":       nt.Label,
				"nodeTypeLabelPlural": labelPlural,
				"hasActiveFilters":    len(activeTaxFilters) > 0 || params["search"] != "" || (params["status"] != "" && params["status"] != "all"),
				"sortBy":              sortBy,
				"sortOrder":           sortOrder,
			},
			Actions: map[string]ActionDef{
				"onRowDelete": {
					Type: "SEQUENCE",
					Steps: []ActionDef{
						{Type: "CONFIRM", Message: "Delete this item? This cannot be undone."},
						{Type: "CORE_API", Method: "nodes:delete", Params: map[string]interface{}{"id": "$event.id"}},
						{Type: "TOAST", Message: "Deleted", Variant: "success"},
						{Type: "INVALIDATE", Keys: []string{"layout"}},
					},
				},
			},
		},
	}...)

	return &LayoutNode{
		Type:     "VerticalStack",
		Props:    map[string]interface{}{"gap": 0},
		Children: children,
	}
}

// taxonomyTermsLayout renders the term-management page nested under a
// node-type's content area (e.g. /admin/posts/taxonomies/category).
func (e *Engine) taxonomyTermsLayout(params map[string]string) *LayoutNode {
	taxonomySlug := params["taxonomy"]
	nodeTypeSlug := params["nodeType"]
	if taxonomySlug == "" || nodeTypeSlug == "" {
		return e.defaultLayout("taxonomy-terms")
	}

	// 1. Get Taxonomy by slug
	var tax models.Taxonomy
	if err := e.db.Where("slug = ?", taxonomySlug).First(&tax).Error; err != nil {
		return e.defaultLayout("taxonomy-terms")
	}

	// Resolve display labels
	labelPlural := tax.LabelPlural
	if labelPlural == "" {
		labelPlural = tax.Label
	}
	_ = labelPlural // currently unused in layout but kept for symmetry with sibling layouts

	// 2. Get NodeType for context (best-effort, doesn't fail layout)
	var nt models.NodeType
	e.db.Where("slug = ?", nodeTypeSlug).First(&nt)

	basePath := basePathForNodeType(nodeTypeSlug)

	// 3. Sort + search params
	termPage, _ := strconv.Atoi(params["page"])
	if termPage < 1 {
		termPage = 1
	}
	termPerPage := getPerPage(params)
	termSearch := params["search"]

	termSortBy := params["sort"]
	termSortOrder := params["order"]
	switch termSortBy {
	case "name", "count":
	default:
		termSortBy = "name"
	}
	if termSortOrder != "asc" && termSortOrder != "desc" {
		if termSortBy == "count" {
			termSortOrder = "desc"
		} else {
			termSortOrder = "asc"
		}
	}

	// 4. Query taxonomy terms with search + sort + pagination
	termQuery := e.db.Model(&models.TaxonomyTerm{}).
		Where("node_type = ? AND taxonomy = ?", nodeTypeSlug, taxonomySlug)
	if termSearch != "" {
		termQuery = termQuery.Where("name ILIKE ? OR slug ILIKE ?", "%"+termSearch+"%", "%"+termSearch+"%")
	}
	// Optional language filter — same convention as the node list. "all"
	// disables the filter so operators can audit every translation.
	if lang := params["language"]; lang != "" && lang != "all" {
		termQuery = termQuery.Where("language_code = ?", lang)
	}

	var termTotal int64
	termQuery.Count(&termTotal)

	termOffset := (termPage - 1) * termPerPage
	var terms []models.TaxonomyTerm
	termQuery.Order(termSortBy + " " + termSortOrder).Offset(termOffset).Limit(termPerPage).Find(&terms)

	// 5. Build rows
	rows := make([]map[string]interface{}, 0, len(terms))
	for _, t := range terms {
		editPath := fmt.Sprintf("/admin/content/%s/taxonomies/%s/%d/edit", nodeTypeSlug, taxonomySlug, t.ID)
		rows = append(rows, map[string]interface{}{
			"id":            t.ID,
			"name":          t.Name,
			"slug":          t.Slug,
			"description":   t.Description,
			"count":         t.Count,
			"language_code": t.LanguageCode,
			"editPath":      editPath,
		})
	}

	termTotalPages := int(termTotal) / termPerPage
	if int(termTotal)%termPerPage > 0 {
		termTotalPages++
	}

	hasFilters := termSearch != ""

	return &LayoutNode{
		Type:  "VerticalStack",
		Props: map[string]interface{}{"gap": 0},
		Children: []LayoutNode{
			{
				Type: "PageHeader",
				Props: map[string]interface{}{
					"tabs":      []map[string]interface{}{{"value": "all", "label": "All", "count": int(termTotal)}},
					"activeTab": "all",
					"newLabel":  "New " + tax.Label,
				},
				Actions: map[string]ActionDef{
					"onBack": {Type: "NAVIGATE", To: basePath},
					"onNew":  {Type: "NAVIGATE", To: fmt.Sprintf("/admin/content/%s/taxonomies/%s/new", nodeTypeSlug, taxonomySlug)},
				},
			},
			{
				Type: "SearchToolbar",
				Props: map[string]interface{}{
					"searchPlaceholder": "Search terms…",
					"languages":         buildLanguageList(e.db),
					"activeLanguage":    params["language"],
				},
			},
			{
				Type: "TaxonomyTermsTable",
				Props: map[string]interface{}{
					"taxonomy":         taxonomySlug,
					"nodeType":         nodeTypeSlug,
					"rows":             rows,
					"sortBy":           termSortBy,
					"sortOrder":        termSortOrder,
					"hasActiveFilters": hasFilters,
					"pagination": map[string]interface{}{
						"page": termPage, "perPage": termPerPage,
						"total": int(termTotal), "totalPages": termTotalPages,
					},
				},
				Actions: map[string]ActionDef{
					"onRowDelete": {
						Type: "SEQUENCE",
						Steps: []ActionDef{
							{Type: "CONFIRM", Message: "Delete this term? This cannot be undone."},
							{Type: "CORE_API", Method: "terms:delete", Params: map[string]interface{}{"id": "$event.id"}},
							{Type: "TOAST", Message: "Term deleted", Variant: "success"},
							{Type: "INVALIDATE", Keys: []string{"layout"}},
						},
					},
				},
			},
		},
	}
}
