\set ON_ERROR_STOP on

BEGIN;

TRUNCATE TABLE
  tenant_resource_values,
  tenant_resources,
  resource_fields,
  tenant_domains,
  api_clients,
  resource_definitions,
  tenants
RESTART IDENTITY CASCADE;

INSERT INTO tenants (id, slug, name, status, created_at, updated_at)
SELECT
  (substr(md5('tenant-' || n::text), 1, 8) || '-' ||
   substr(md5('tenant-' || n::text), 9, 4) || '-' ||
   substr(md5('tenant-' || n::text), 13, 4) || '-' ||
   substr(md5('tenant-' || n::text), 17, 4) || '-' ||
   substr(md5('tenant-' || n::text), 21, 12))::uuid,
  'bench-tenant-' || lpad(n::text, 5, '0'),
  'Benchmark Tenant ' || lpad(n::text, 5, '0'),
  CASE WHEN n % 10 = 0 THEN 'inactive' ELSE 'active' END,
  timestamptz '2026-01-01 00:00:00+00' + n * interval '1 second',
  timestamptz '2026-01-01 00:00:00+00' + n * interval '1 second'
FROM generate_series(1, :size) AS n;

INSERT INTO tenant_domains (id, tenant_id, hostname, created_at)
SELECT
  (substr(md5('domain-' || n::text), 1, 8) || '-' ||
   substr(md5('domain-' || n::text), 9, 4) || '-' ||
   substr(md5('domain-' || n::text), 13, 4) || '-' ||
   substr(md5('domain-' || n::text), 17, 4) || '-' ||
   substr(md5('domain-' || n::text), 21, 12))::uuid,
  (substr(md5('tenant-' || n::text), 1, 8) || '-' ||
   substr(md5('tenant-' || n::text), 9, 4) || '-' ||
   substr(md5('tenant-' || n::text), 13, 4) || '-' ||
   substr(md5('tenant-' || n::text), 17, 4) || '-' ||
   substr(md5('tenant-' || n::text), 21, 12))::uuid,
  'bench-tenant-' || lpad(n::text, 5, '0') || '.example.invalid',
  timestamptz '2026-01-01 00:00:00+00' + n * interval '1 second'
FROM generate_series(1, :size) AS n;

INSERT INTO resource_definitions (id, key, name, description, status, created_at, updated_at)
SELECT
  (substr(md5('definition-' || n::text), 1, 8) || '-' ||
   substr(md5('definition-' || n::text), 9, 4) || '-' ||
   substr(md5('definition-' || n::text), 13, 4) || '-' ||
   substr(md5('definition-' || n::text), 17, 4) || '-' ||
   substr(md5('definition-' || n::text), 21, 12))::uuid,
  'bench-definition-' || lpad(n::text, 5, '0'),
  'Benchmark Definition ' || lpad(n::text, 5, '0'),
  'Synthetic scale benchmark record',
  CASE WHEN n % 10 = 0 THEN 'inactive' ELSE 'active' END,
  timestamptz '2026-01-01 00:00:00+00' + n * interval '1 second',
  timestamptz '2026-01-01 00:00:00+00' + n * interval '1 second'
FROM generate_series(1, :size) AS n;

INSERT INTO resource_fields
  (id, resource_definition_id, key, label, data_type, required, is_secret, sort_order)
SELECT
  (substr(md5('field-' || n::text || '-' || field_number::text), 1, 8) || '-' ||
   substr(md5('field-' || n::text || '-' || field_number::text), 9, 4) || '-' ||
   substr(md5('field-' || n::text || '-' || field_number::text), 13, 4) || '-' ||
   substr(md5('field-' || n::text || '-' || field_number::text), 17, 4) || '-' ||
   substr(md5('field-' || n::text || '-' || field_number::text), 21, 12))::uuid,
  (substr(md5('definition-' || n::text), 1, 8) || '-' ||
   substr(md5('definition-' || n::text), 9, 4) || '-' ||
   substr(md5('definition-' || n::text), 13, 4) || '-' ||
   substr(md5('definition-' || n::text), 17, 4) || '-' ||
   substr(md5('definition-' || n::text), 21, 12))::uuid,
  'field_' || field_number,
  'Field ' || field_number,
  'string',
  true,
  field_number = 4,
  field_number
FROM generate_series(1, :size) AS n
CROSS JOIN generate_series(1, 4) AS field_number;

INSERT INTO tenant_resources
  (id, tenant_id, resource_definition_id, status, created_at, updated_at)
SELECT
  (substr(md5('resource-' || n::text), 1, 8) || '-' ||
   substr(md5('resource-' || n::text), 9, 4) || '-' ||
   substr(md5('resource-' || n::text), 13, 4) || '-' ||
   substr(md5('resource-' || n::text), 17, 4) || '-' ||
   substr(md5('resource-' || n::text), 21, 12))::uuid,
  (substr(md5('tenant-' || n::text), 1, 8) || '-' ||
   substr(md5('tenant-' || n::text), 9, 4) || '-' ||
   substr(md5('tenant-' || n::text), 13, 4) || '-' ||
   substr(md5('tenant-' || n::text), 17, 4) || '-' ||
   substr(md5('tenant-' || n::text), 21, 12))::uuid,
  (substr(md5('definition-' || n::text), 1, 8) || '-' ||
   substr(md5('definition-' || n::text), 9, 4) || '-' ||
   substr(md5('definition-' || n::text), 13, 4) || '-' ||
   substr(md5('definition-' || n::text), 17, 4) || '-' ||
   substr(md5('definition-' || n::text), 21, 12))::uuid,
  CASE WHEN n % 10 = 0 THEN 'inactive' ELSE 'active' END,
  timestamptz '2026-01-01 00:00:00+00' + n * interval '1 second',
  timestamptz '2026-01-01 00:00:00+00' + n * interval '1 second'
FROM generate_series(1, :size) AS n;

INSERT INTO api_clients
  (id, name, key_hash, token_preview, rpm_limit, expires_at, status, revoked_at, created_at, updated_at)
SELECT
  (substr(md5('client-' || n::text), 1, 8) || '-' ||
   substr(md5('client-' || n::text), 9, 4) || '-' ||
   substr(md5('client-' || n::text), 13, 4) || '-' ||
   substr(md5('client-' || n::text), 17, 4) || '-' ||
   substr(md5('client-' || n::text), 21, 12))::uuid,
  'Benchmark Client ' || lpad(n::text, 5, '0'),
  encode(digest('benchmark-client-' || n::text, 'sha256'), 'hex'),
  'tnc_' || substr(md5('preview-' || n::text), 1, 8),
  300,
  timestamptz '2027-01-01 00:00:00+00' + n * interval '1 second',
  CASE WHEN n % 10 = 0 THEN 'revoked' ELSE 'active' END,
  CASE WHEN n % 10 = 0 THEN timestamptz '2026-06-01 00:00:00+00' + n * interval '1 second' END,
  timestamptz '2026-01-01 00:00:00+00' + n * interval '1 second',
  timestamptz '2026-01-01 00:00:00+00' + n * interval '1 second'
FROM generate_series(1, :size) AS n;

INSERT INTO api_client_scopes (api_client_id, scope)
SELECT
  (substr(md5('client-' || n::text), 1, 8) || '-' ||
   substr(md5('client-' || n::text), 9, 4) || '-' ||
   substr(md5('client-' || n::text), 13, 4) || '-' ||
   substr(md5('client-' || n::text), 17, 4) || '-' ||
   substr(md5('client-' || n::text), 21, 12))::uuid,
  scope
FROM generate_series(1, :size) AS n
CROSS JOIN (VALUES ('tenant:identify'), ('resource:resolve')) AS scopes(scope);

SELECT 1 / ((count(*) = :size)::integer) FROM tenants;
SELECT 1 / ((count(*) = :size)::integer) FROM tenant_domains;
SELECT 1 / ((count(*) = :size)::integer) FROM resource_definitions;
SELECT 1 / ((count(*) = :size * 4)::integer) FROM resource_fields;
SELECT 1 / ((count(*) = :size)::integer) FROM tenant_resources;
SELECT 1 / ((count(*) = :size)::integer) FROM api_clients;
SELECT 1 / ((count(*) = :size * 2)::integer) FROM api_client_scopes;

ANALYZE tenants;
ANALYZE tenant_domains;
ANALYZE resource_definitions;
ANALYZE api_clients;

COMMIT;
