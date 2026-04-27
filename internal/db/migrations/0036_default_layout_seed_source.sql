-- 0036_default_layout_seed_source.sql
--
-- Earlier migration 0008 seeded the "default" layout with source='custom',
-- which RegisterLayoutFromFile treats as user-owned and never overwrites.
-- That prevented any activated theme from installing its own default layout
-- (so renderLayoutBlock "site-header"/"site-footer" calls never fired on
-- existing DBs).
--
-- Demote the seeded placeholder to source='seed' so theme layouts can
-- replace it. We only touch rows that look like the original seed (no
-- renderLayoutBlock calls) — genuinely user-customised layouts keep their
-- 'custom' source and remain untouched.
UPDATE layouts
SET source = 'seed'
WHERE slug = 'default'
  AND source = 'custom'
  AND template_code NOT LIKE '%renderLayoutBlock%';
