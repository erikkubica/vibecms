-- Activate Forms extension
UPDATE extensions SET is_active = true WHERE slug = 'forms' AND is_active = false;
