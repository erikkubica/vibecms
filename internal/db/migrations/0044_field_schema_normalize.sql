-- 0044_field_schema_normalize.sql
--
-- Normalize stored field schemas to the canonical vocabulary used by the
-- back-compat reader added in this release. Every JSONB column that holds
-- a field-schema array is rewritten in place:
--
--   key         -> name
--   label       -> title
--   help        -> description
--   default     -> initialValue
--   sub_fields  -> fields
--   type "text"     -> "string"
--   type "repeater" -> "array"
--   type "group"    -> "object"
--   type "node"     -> "reference"
--
-- Idempotent: running again on already-normalized data is a no-op
-- because the legacy keys won't be present.
--
-- Tables touched (column "field_schema"):
--   node_types, taxonomies, block_types, layout_blocks
--
-- Column names are unchanged — this migration only rewrites the JSONB
-- payload, not the schema. Safe under concurrent writes because the
-- runtime back-compat reader handles either shape.

CREATE OR REPLACE FUNCTION pg_temp.normalize_field_schema(input jsonb)
RETURNS jsonb
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    item   jsonb;
    out_arr jsonb := '[]'::jsonb;
    obj    jsonb;
    typ    text;
    nested jsonb;
BEGIN
    IF input IS NULL OR jsonb_typeof(input) <> 'array' THEN
        RETURN input;
    END IF;

    FOR item IN SELECT jsonb_array_elements(input)
    LOOP
        obj := item;
        IF jsonb_typeof(obj) <> 'object' THEN
            out_arr := out_arr || jsonb_build_array(obj);
            CONTINUE;
        END IF;

        -- Identifier: key -> name (only fill if name is empty)
        IF (obj ? 'key') AND NOT (obj ? 'name') THEN
            obj := obj || jsonb_build_object('name', obj->'key');
        END IF;
        obj := obj - 'key';

        -- Display label: label -> title
        IF (obj ? 'label') AND NOT (obj ? 'title') THEN
            obj := obj || jsonb_build_object('title', obj->'label');
        END IF;
        obj := obj - 'label';

        -- Helper text: help -> description
        IF (obj ? 'help') AND NOT (obj ? 'description') THEN
            obj := obj || jsonb_build_object('description', obj->'help');
        END IF;
        obj := obj - 'help';

        -- Default value: default -> initialValue
        IF (obj ? 'default') AND NOT (obj ? 'initialValue') THEN
            obj := obj || jsonb_build_object('initialValue', obj->'default');
        END IF;
        obj := obj - 'default';

        -- Type aliases.
        typ := obj->>'type';
        IF typ = 'text' THEN
            obj := jsonb_set(obj, '{type}', '"string"');
        ELSIF typ = 'repeater' THEN
            obj := jsonb_set(obj, '{type}', '"array"');
        ELSIF typ = 'group' THEN
            obj := jsonb_set(obj, '{type}', '"object"');
        ELSIF typ = 'node' THEN
            obj := jsonb_set(obj, '{type}', '"reference"');
        END IF;

        -- Nested fields: sub_fields -> fields, then recurse.
        IF (obj ? 'sub_fields') AND NOT (obj ? 'fields') THEN
            obj := obj || jsonb_build_object('fields', obj->'sub_fields');
        END IF;
        obj := obj - 'sub_fields';

        IF (obj ? 'fields') AND jsonb_typeof(obj->'fields') = 'array' THEN
            nested := pg_temp.normalize_field_schema(obj->'fields');
            obj := jsonb_set(obj, '{fields}', nested);
        END IF;

        out_arr := out_arr || jsonb_build_array(obj);
    END LOOP;

    RETURN out_arr;
END;
$$;

-- Apply normalization to every table that stores a field schema.
UPDATE node_types
   SET field_schema = pg_temp.normalize_field_schema(field_schema)
 WHERE field_schema IS NOT NULL
   AND field_schema::text <> pg_temp.normalize_field_schema(field_schema)::text;

UPDATE taxonomies
   SET field_schema = pg_temp.normalize_field_schema(field_schema)
 WHERE field_schema IS NOT NULL
   AND field_schema::text <> pg_temp.normalize_field_schema(field_schema)::text;

UPDATE block_types
   SET field_schema = pg_temp.normalize_field_schema(field_schema)
 WHERE field_schema IS NOT NULL
   AND field_schema::text <> pg_temp.normalize_field_schema(field_schema)::text;

UPDATE layout_blocks
   SET field_schema = pg_temp.normalize_field_schema(field_schema)
 WHERE field_schema IS NOT NULL
   AND field_schema::text <> pg_temp.normalize_field_schema(field_schema)::text;
