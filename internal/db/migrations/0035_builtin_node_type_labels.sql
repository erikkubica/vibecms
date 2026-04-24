-- Fix labels for built-in node types: sentence case and add plural forms.
UPDATE node_types SET label_plural = 'Pages' WHERE slug = 'page' AND label_plural = '';
UPDATE node_types SET label_plural = 'Posts' WHERE slug = 'post' AND label_plural = '';
