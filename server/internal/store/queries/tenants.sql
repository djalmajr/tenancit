-- name: CreateTenant :one
INSERT INTO tenants (slug, name) VALUES ($1, $2) RETURNING *;

-- name: GetTenant :one
SELECT * FROM tenants WHERE id = $1;

-- name: ListTenants :many
SELECT * FROM tenants ORDER BY name;

-- name: UpdateTenant :one
UPDATE tenants SET name = $2, slug = $3, status = $4, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteTenant :execrows
-- Cascades to tenant_domains, tenant_resources and tenant_resource_values (FK ON DELETE CASCADE).
DELETE FROM tenants WHERE id = $1;

-- name: AddTenantDomain :one
INSERT INTO tenant_domains (tenant_id, hostname) VALUES ($1, $2) RETURNING *;

-- name: RemoveTenantDomain :execrows
DELETE FROM tenant_domains
WHERE id = sqlc.arg(id)
  AND tenant_id = sqlc.arg(tenant_id);

-- name: UpdateTenantDomain :one
UPDATE tenant_domains
SET hostname = sqlc.arg(hostname)
WHERE id = sqlc.arg(id)
  AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;

-- name: ListTenantDomains :many
SELECT * FROM tenant_domains WHERE tenant_id = $1 ORDER BY hostname;

-- name: GetTenantDomain :one
SELECT d.* FROM tenant_domains d
WHERE d.id = sqlc.arg(domain_id) AND d.tenant_id = sqlc.arg(target_tenant_id)
FOR UPDATE;

-- name: CountTenantChildren :one
SELECT
  (SELECT count(*) FROM tenant_domains d WHERE d.tenant_id = sqlc.arg(target_tenant_id))::int AS domains,
  (SELECT count(*) FROM tenant_resources r WHERE r.tenant_id = sqlc.arg(target_tenant_id))::int AS resources;

-- name: GetTenantByHostname :one
SELECT t.* FROM tenants t
JOIN tenant_domains d ON d.tenant_id = t.id
WHERE d.hostname = $1;

-- name: GetTenantBySlug :one
SELECT * FROM tenants WHERE slug = $1;
