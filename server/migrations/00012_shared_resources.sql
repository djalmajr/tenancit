-- +goose Up
ALTER TABLE tenant_resources
    ADD COLUMN alias text,
    ADD COLUMN source_resource_id uuid REFERENCES tenant_resources(id) ON DELETE RESTRICT;

UPDATE tenant_resources tr
SET alias = rd.key
FROM resource_definitions rd
WHERE rd.id = tr.resource_definition_id;

ALTER TABLE tenant_resources
    ALTER COLUMN alias SET NOT NULL,
    ALTER COLUMN alias SET DEFAULT gen_random_uuid()::text,
    ADD CONSTRAINT tenant_resource_alias_not_blank CHECK (btrim(alias) <> ''),
    ADD CONSTRAINT tenant_resource_alias_format CHECK (alias ~ '^[a-z0-9][a-z0-9._-]{0,62}$'),
    ADD CONSTRAINT tenant_resource_source_not_self CHECK (source_resource_id IS NULL OR source_resource_id <> id);

-- +goose StatementBegin
CREATE FUNCTION validate_tenant_resource_source() RETURNS trigger AS $$
DECLARE
    source tenant_resources%ROWTYPE;
BEGIN
    IF NEW.source_resource_id IS NULL THEN
        RETURN NEW;
    END IF;
    SELECT * INTO source FROM tenant_resources WHERE id = NEW.source_resource_id;
    IF NOT FOUND
       OR source.tenant_id <> NEW.tenant_id
       OR source.resource_definition_id <> NEW.resource_definition_id
       OR source.source_resource_id IS NOT NULL THEN
        RAISE EXCEPTION 'invalid tenant resource source' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER tenant_resource_source_contract
BEFORE INSERT OR UPDATE OF source_resource_id, tenant_id, resource_definition_id
ON tenant_resources
FOR EACH ROW EXECUTE FUNCTION validate_tenant_resource_source();

DROP INDEX uq_tenant_resource_active;
CREATE UNIQUE INDEX uq_tenant_resource_alias
    ON tenant_resources (tenant_id, lower(btrim(alias)));
CREATE INDEX idx_tenant_resources_source ON tenant_resources(source_resource_id)
    WHERE source_resource_id IS NOT NULL;

-- +goose Down
DROP TRIGGER IF EXISTS tenant_resource_source_contract ON tenant_resources;
DROP FUNCTION IF EXISTS validate_tenant_resource_source();
DROP INDEX IF EXISTS idx_tenant_resources_source;
DROP INDEX IF EXISTS uq_tenant_resource_alias;
ALTER TABLE tenant_resources
    DROP CONSTRAINT IF EXISTS tenant_resource_source_not_self,
    DROP CONSTRAINT IF EXISTS tenant_resource_alias_format,
    DROP CONSTRAINT IF EXISTS tenant_resource_alias_not_blank,
    DROP COLUMN IF EXISTS source_resource_id,
    DROP COLUMN IF EXISTS alias;
CREATE UNIQUE INDEX uq_tenant_resource_active
    ON tenant_resources(tenant_id, resource_definition_id)
    WHERE status = 'active';
