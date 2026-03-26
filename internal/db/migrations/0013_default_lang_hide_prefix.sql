-- Default language should hide URL prefix (e.g. /about instead of /en/about)
UPDATE languages SET hide_prefix = TRUE WHERE is_default = TRUE;
