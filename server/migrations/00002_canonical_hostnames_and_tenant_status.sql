-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM tenant_domains
        WHERE octet_length(hostname) <> char_length(hostname)
    ) THEN
        RAISE EXCEPTION 'tenant_domains contains non-ASCII hostnames; migrate to explicit punycode before upgrade';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM tenant_domains
        GROUP BY lower(rtrim(btrim(hostname), '.'))
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'tenant_domains contains hostnames that collide after canonicalization';
    END IF;

    IF EXISTS (
        SELECT 1 FROM tenants WHERE status NOT IN ('active', 'inactive')
    ) THEN
        RAISE EXCEPTION 'tenants contains status values outside active|inactive';
    END IF;
END $$;
-- +goose StatementEnd

UPDATE tenant_domains
SET hostname = lower(rtrim(btrim(hostname), '.'));

ALTER TABLE tenant_domains
    ADD CONSTRAINT tenant_domains_hostname_canonical
    CHECK (
        hostname <> ''
        AND hostname = lower(rtrim(btrim(hostname), '.'))
        AND octet_length(hostname) = char_length(hostname)
    );

ALTER TABLE tenants
    ADD CONSTRAINT tenants_status_check
    CHECK (status IN ('active', 'inactive'));

-- +goose Down
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_status_check;
ALTER TABLE tenant_domains DROP CONSTRAINT IF EXISTS tenant_domains_hostname_canonical;
