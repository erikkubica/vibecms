// Package field_types is the single source of truth for Squilla's built-in
// content field types. Both the admin UI (via /admin/api/field-types) and the
// MCP core.field_types.list tool read from this registry.
package field_types

// FieldTypeDef describes a field type available in the CMS, including
// authoring guidance so both humans and AI can pick the right one.
type FieldTypeDef struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Description string `json:"description"`
	HowTo       string `json:"how_to"`
	Group       string `json:"group"`
	Icon        string `json:"icon"`
}

// IsBuiltin reports whether typ matches a known kernel field type. Used by
// validators that need to flag unknown types declared in theme.json /
// block.json / nodetype field schemas — common typos like "boolean" (vs
// "toggle"), "wysiwyg" (vs "richtext"), "dropdown" (vs "select"). Extension-
// contributed types live outside this check; callers that have access to the
// extension registry should layer their own check on top.
func IsBuiltin(typ string) bool {
	for _, def := range Builtin() {
		if def.Type == typ {
			return true
		}
	}
	return false
}

// Builtin returns the 20 field types that ship with the Squilla kernel.
// Extensions can contribute additional types through their admin_ui.field_types
// manifest entry.
func Builtin() []FieldTypeDef {
	return []FieldTypeDef{
		// --- Basic ---
		{
			Type: "text", Label: "Text", Group: "Basic", Icon: "Type",
			Description: "Single-line text input",
			HowTo: "Use for short plain-text values that fit on one line — titles, headings, button labels, names, short captions. Not for addresses, long descriptions, or rich content. If the value needs formatting (bold, links), use Rich Text instead; if it spans multiple sentences, use Textarea.",
		},
		{
			Type: "textarea", Label: "Textarea", Group: "Basic", Icon: "AlignLeft",
			Description: "Multi-line text input",
			HowTo: "Use for plain-text content that spans multiple lines but doesn't need formatting — short bios, excerpts, alt text paragraphs, plain-text notes. If the author needs bold/italics/links/headings, use Rich Text instead.",
		},
		{
			Type: "richtext", Label: "Rich Text", Group: "Basic", Icon: "FileText",
			Description: "WYSIWYG rich text editor",
			HowTo: "Use for body copy where the author needs formatting — article bodies, long descriptions with links, marketing copy. Outputs HTML and should be rendered with the safeHTML template helper. Overkill for plain single-line strings; use Text instead.",
		},
		{
			Type: "number", Label: "Number", Group: "Basic", Icon: "Hash",
			Description: "Numeric input with constraints",
			HowTo: "Use for integers or decimals — counts, prices, weights, scores. Supports min/max/step validation in the schema. Prefer Range Slider when the value is on a known scale the author tunes visually (opacity, spacing).",
		},
		{
			Type: "range", Label: "Range Slider", Group: "Basic", Icon: "SlidersHorizontal",
			Description: "Slider with min/max values",
			HowTo: "Use when the author picks a value on a bounded scale and a slider is clearer than a number — opacity (0–1), column count (1–12), blur radius (0–40px). Always set min/max/step in the schema. If there is no natural maximum, use Number.",
		},
		{
			Type: "email", Label: "Email", Group: "Basic", Icon: "Mail",
			Description: "Email address input",
			HowTo: "Use when the field must hold a valid email address — contact forms, mailing-list fields, author emails. The input enforces email format on the client. For generic text that merely looks like an email, use Text.",
		},
		{
			Type: "url", Label: "URL", Group: "Basic", Icon: "Globe",
			Description: "Web address input",
			HowTo: "Use for standalone absolute URLs where no additional metadata is needed (canonical URL, RSS feed source, external docs link). When the URL appears as a clickable button or link in the UI, prefer Link — it carries label, target, and rel.",
		},
		{
			Type: "date", Label: "Date", Group: "Basic", Icon: "Calendar",
			Description: "Date picker",
			HowTo: "Use for dates the author should pick from a calendar — event dates, publish overrides, deadlines. Stores ISO-8601 date strings. Not for timestamps or timezone-sensitive datetimes; those need a dedicated datetime type (not yet in core).",
		},
		{
			Type: "color", Label: "Color Picker", Group: "Basic", Icon: "Palette",
			Description: "Color selection with hex value",
			HowTo: "Use when the author selects a color — background colors, accents, badges. Outputs a hex string like '#aabbcc'. If you only want a fixed palette, use Select with the palette values instead so authors can't drift from the design system.",
		},

		// --- Choice ---
		{
			Type: "toggle", Label: "Toggle", Group: "Choice", Icon: "ToggleLeft",
			Description: "On/off boolean switch",
			HowTo: "Use for binary choices — 'show/hide', 'featured', 'enabled'. Stores a true/false boolean. If there are three or more mutually exclusive states, use Radio Buttons or Select.",
		},
		{
			Type: "select", Label: "Select", Group: "Choice", Icon: "ListOrdered",
			Description: "Dropdown with predefined options",
			HowTo: "Use when there are 4+ mutually exclusive options and space is tight — variant pickers, sort orders, theme variants. For 2–3 options where visibility matters, Radio Buttons read faster. Define options in the schema under 'options'.",
		},
		{
			Type: "radio", Label: "Radio Buttons", Group: "Choice", Icon: "CircleDot",
			Description: "Single choice from options",
			HowTo: "Use when there are 2–4 mutually exclusive options and authors benefit from seeing them all at once — layout direction, alignment, size preset. For more than 5 options, switch to Select to save space.",
		},
		{
			Type: "checkbox", Label: "Checkbox Group", Group: "Choice", Icon: "CheckSquare",
			Description: "Multiple choice from options",
			HowTo: "Use for multi-select from a fixed list — feature flags, category filters, shown-on pages. Stores an array of selected values. For single-choice, use Radio Buttons or Select.",
		},

		// --- Media ---
		{
			Type: "image", Label: "Image", Group: "Media", Icon: "Image",
			Description: "Single image upload",
			HowTo: "Use when exactly one image is expected — hero image, thumbnail, avatar. Stored as an object with url, alt, width, height; templates access fields like {{ .image.url }}, {{ .image.alt }}. Never store the URL as a plain string field.",
		},
		{
			Type: "gallery", Label: "Gallery", Group: "Media", Icon: "Images",
			Description: "Multiple image uploads",
			HowTo: "Use for ordered image collections — product galleries, photo grids, carousels. Stores an array of image objects. If the author picks exactly one image, use Image instead; if each item needs metadata beyond image fields, use a Repeater with an Image sub-field.",
		},
		{
			Type: "file", Label: "File", Group: "Media", Icon: "File",
			Description: "File upload with type filtering",
			HowTo: "Use for non-image downloads — PDF brochures, ZIP assets, CSVs. Schema's 'accept' option filters allowed types. For images, always use the Image type — it carries alt text and dimensions.",
		},

		// --- Relational ---
		{
			Type: "link", Label: "Link", Group: "Relational", Icon: "Link2",
			Description: "URL with text, alt, and target",
			HowTo: "Use for clickable actions — buttons, CTAs, menu items, card links. Stores {url, label, target, rel}. Always prefer Link over URL when the value is rendered as an <a> or button; the extra fields are how the designer controls accessibility and UX.",
		},
		{
			Type: "node", Label: "Node Selector", Group: "Relational", Icon: "FileSearch",
			Description: "Reference to content nodes",
			HowTo: "Use to reference other CMS content — 'related posts', 'featured page', 'parent product'. Stores node IDs and resolves to full node data at render time. Filter by node type with the 'node_types' schema option. If you just want an external URL, use Link.",
		},
		{
			Type: "term", Label: "Term Selector", Group: "Relational", Icon: "Tags",
			Description: "Reference to taxonomy terms",
			HowTo: "Use to assign taxonomy terms (categories, tags) to a node or block field — 'trip tag', 'product category', 'blog topic'. Stores {id, slug, name, taxonomy, node_type} and resolves live so term renames propagate. Declare taxonomy+node_type in the schema. For free-form strings, use Text; for a fixed list of non-taxonomy options, use Select.",
		},

		// --- Layout ---
		{
			Type: "group", Label: "Group", Group: "Layout", Icon: "Layers",
			Description: "Container for nested fields",
			HowTo: "Use to nest related fields under one logical object — 'seo' with title/description/image, 'cta' with heading/button. Authors see a fieldset; templates access children with dot notation ({{ .seo.title }}). Sub-fields go in 'sub_fields' (not 'fields').",
		},
		{
			Type: "repeater", Label: "Repeater", Group: "Layout", Icon: "Repeat",
			Description: "Repeatable set of fields",
			HowTo: "Use for a variable-length list of compound items — FAQ entries, feature cards, team members. Declare child fields in 'sub_fields' (not 'fields'). Supports min/max row counts. Wrong choice when items are simple strings — use Checkbox Group or Select for fixed options, or a plain array inside a richtext if it's free-form.",
		},
	}
}
