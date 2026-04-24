-- Layouts
CREATE TABLE IF NOT EXISTS layouts (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    language_id INT REFERENCES languages(id) ON DELETE SET NULL,
    template_code TEXT NOT NULL DEFAULT '',
    source VARCHAR(20) NOT NULL DEFAULT 'custom',
    theme_name VARCHAR(100),
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Partial unique indexes to handle NULL language_id (universal/all languages)
CREATE UNIQUE INDEX IF NOT EXISTS layouts_slug_lang ON layouts(slug, language_id) WHERE language_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS layouts_slug_universal ON layouts(slug) WHERE language_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_layouts_source_theme ON layouts(source, theme_name);
CREATE UNIQUE INDEX IF NOT EXISTS layouts_one_default_per_lang ON layouts(language_id) WHERE is_default = true AND language_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS layouts_one_default_universal ON layouts(is_default) WHERE is_default = true AND language_id IS NULL;

-- Layout Blocks (partials)
CREATE TABLE IF NOT EXISTS layout_blocks (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    language_id INT REFERENCES languages(id) ON DELETE SET NULL,
    template_code TEXT NOT NULL DEFAULT '',
    source VARCHAR(20) NOT NULL DEFAULT 'custom',
    theme_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS layout_blocks_slug_lang ON layout_blocks(slug, language_id) WHERE language_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS layout_blocks_slug_universal ON layout_blocks(slug) WHERE language_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_layout_blocks_source_theme ON layout_blocks(source, theme_name);

-- Menus
CREATE TABLE IF NOT EXISTS menus (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    language_id INT REFERENCES languages(id) ON DELETE SET NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS menus_slug_lang ON menus(slug, language_id) WHERE language_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS menus_slug_universal ON menus(slug) WHERE language_id IS NULL;

-- Menu Items
CREATE TABLE IF NOT EXISTS menu_items (
    id SERIAL PRIMARY KEY,
    menu_id INT NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    parent_id INT REFERENCES menu_items(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    item_type VARCHAR(20) NOT NULL DEFAULT 'custom',
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

-- Add layout_id and language_id to content_nodes
ALTER TABLE content_nodes ADD COLUMN IF NOT EXISTS layout_id INT REFERENCES layouts(id) ON DELETE SET NULL;
ALTER TABLE content_nodes ADD COLUMN IF NOT EXISTS language_id INT REFERENCES languages(id) ON DELETE SET NULL;

-- Add theme fields to block_types
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS theme_name VARCHAR(100);
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS view_file VARCHAR(255);
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS block_css TEXT;
ALTER TABLE block_types ADD COLUMN IF NOT EXISTS block_js TEXT;

-- Seed default layout (universal / all languages — language_id = NULL)
INSERT INTO layouts (slug, name, description, language_id, template_code, source, is_default)
SELECT 'default', 'Default Layout', 'Default page layout', NULL,
'<!DOCTYPE html>
<html lang="{{.app.current_lang.code}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{or (index .node.seo "title") .node.title "VibeCMS"}}</title>
    {{range .app.head_styles}}<link rel="stylesheet" href="{{.}}">{{end}}
    {{range .app.head_scripts}}<script src="{{.}}"></script>{{end}}
    {{.app.block_styles}}
</head>
<body>
    <main>{{.node.blocks_html}}</main>
    {{range .app.foot_scripts}}<script src="{{.}}" defer></script>{{end}}
    {{.app.block_scripts}}
</body>
</html>',
'seed', true
WHERE NOT EXISTS (SELECT 1 FROM layouts WHERE slug = 'default' AND language_id IS NULL);
