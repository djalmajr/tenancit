-- +goose Up
ALTER TABLE tenant_resources
    ADD COLUMN display_name text;

UPDATE tenant_resources
SET display_name = alias;

ALTER TABLE tenant_resources
    ALTER COLUMN display_name SET NOT NULL,
    ADD CONSTRAINT tenant_resource_display_name_not_blank CHECK (btrim(display_name) <> ''),
    ADD CONSTRAINT tenant_resource_display_name_length CHECK (char_length(display_name) <= 120);

-- +goose Down
ALTER TABLE tenant_resources
    DROP CONSTRAINT IF EXISTS tenant_resource_display_name_length,
    DROP CONSTRAINT IF EXISTS tenant_resource_display_name_not_blank,
    DROP COLUMN IF EXISTS display_name;
