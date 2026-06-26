-- name: CreateTenant :one
INSERT INTO tenants (slug, name) VALUES ($1, $2) RETURNING *;

-- name: GetTenant :one
SELECT * FROM tenants WHERE id = $1;

-- name: ListTenants :many
SELECT * FROM tenants ORDER BY name;

-- name: UpdateTenant :one
UPDATE tenants SET name = $2, slug = $3, status = $4, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: AddTenantDomain :one
INSERT INTO tenant_domains (tenant_id, hostname) VALUES ($1, $2) RETURNING *;

-- name: RemoveTenantDomain :exec
DELETE FROM tenant_domains WHERE id = $1;

-- name: ListTenantDomains :many
SELECT * FROM tenant_domains WHERE tenant_id = $1 ORDER BY hostname;

-- name: GetTenantByHostname :one
SELECT t.* FROM tenants t
JOIN tenant_domains d ON d.tenant_id = t.id
WHERE d.hostname = $1;

-- name: GetTenantBySlug :one
SELECT * FROM tenants WHERE slug = $1;
