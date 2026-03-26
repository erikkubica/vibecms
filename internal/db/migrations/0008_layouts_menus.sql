-- Layouts
CREATE TABLE IF NOT EXISTS layouts (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    language_code VARCHAR(10) NOT NULL REFERENCES languages(code),
    template_code TEXT NOT NULL DEFAULT '',
    source VARCHAR(20) NOT NULL DEFAULT 'custom',
    theme_name VARCHAR(100),
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(slug, language_code)
);

CREATE INDEX IF NOT EXISTS idx_layouts_source_theme ON layouts(source, theme_name);
CREATE UNIQUE INDEX IF NOT EXISTS layouts_one_default_per_lang ON layouts(language_code) WHERE is_default = true;

-- Layout Blocks (partials)
CREATE TABLE IF NOT EXISTS layout_blocks (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    language_code VARCHAR(10) NOT NULL REFERENCES languages(code),
    template_code TEXT NOT NULL DEFAULT '',
    source VARCHAR(20) NOT NULL DEFAULT 'custom',
    theme_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(slug, language_code)
);

CREATE INDEX IF NOT EXISTS idx_layout_blocks_source_theme ON layout_blocks(source, theme_name);

-- Menus
CREATE TABLE IF NOT EXISTS menus (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    language_code VARCHAR(10) NOT NULL REFERENCES languages(code),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(slug, language_code)
);

-- Menu Items
CREATE TABLE IF NOT EXISTS menu_items (
    id SERIAL PRIMARY KEY,
    menu_id INT NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    parent_id INT REFERENCES menu_items(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    item_type VARCHAR(20) NOT NULL DEFAULT 'url',
    node_id INT REFERENCES content_nodes(id),
    url VARCHAR(2048),
    target VARCHAR(20) NOT NULL DEFAULT '_self',
    css_class VARCHAR(255),
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_menu_items_menu_sort ON menu_items(menu_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_menu_items_menu_parent ON menu_items(menu_id, parent_id);
CREATE INDEX IF NOT EXISTS idx_menu_items_node ON menu_items(node_id);

-- Add layout_id to content_nodes
ALTER TABLE content_nodes ADD COLUMN IF NOT EXISTS layout_id INT REFERENCES layouts(id) ON DELETE SET NULL;

-- Add theme fields to block_types
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS theme_name VARCHAR(100);
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS view_file VARCHAR(255);
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS block_css TEXT;
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS block_js TEXT;

-- Seed default layout for default language
INSERT INTO layouts (slug, name, description, language_code, template_code, source, is_default)
SELECT 'default', 'Default Layout', 'Default page layout', code,
'<!DOCTYPE html>
<html lang="{{.app.currentLang.Code}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.node.seo.title}}</title>
    {{range .app.headStyles}}<link rel="stylesheet" href="{{.}}">{{end}}
    {{range .app.headScripts}}<script src="{{.}}"></script>{{end}}
    {{.app.blockStyles}}
</head>
<body>
    <main>{{.node.blocks_html}}</main>
    {{range .app.footScripts}}<script src="{{.}}" defer></script>{{end}}
    {{.app.blockScripts}}
</body>
</html>',
'custom', true
FROM languages WHERE is_default = true
ON CONFLICT (slug, language_code) DO NOTHING;
