-- Add manage_content capability to admin role
UPDATE roles SET capabilities = capabilities || '{"manage_content": true}'::jsonb WHERE slug = 'admin' AND NOT (capabilities ? 'manage_content');
