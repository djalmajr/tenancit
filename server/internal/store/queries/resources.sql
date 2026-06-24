-- name: CreateTenantResource :one
INSERT INTO tenant_resources (tenant_id, resource_definition_id)
VALUES ($1, $2) RETURNING *;

-- name: SetTenantResourceStatus :one
UPDATE tenant_resources SET status = $2, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteTenantResource :exec
DELETE FROM tenant_resources WHERE id = $1;

-- name: ListTenantResources :many
SELECT * FROM tenant_resources WHERE tenant_id = $1;

-- name: ListActiveResourcesByTenant :many
SELECT * FROM tenant_resources WHERE tenant_id = $1 AND status = 'active';

-- name: GetActiveResourceByTenantAndDefinitionKey :one
SELECT tr.*
FROM tenant_resources tr
JOIN resource_definitions rd ON rd.id = tr.resource_definition_id
WHERE tr.tenant_id = $1
  AND rd.key = $2
  AND tr.status = 'active';

-- name: UpsertResourceValue :one
INSERT INTO tenant_resource_values
  (tenant_resource_id, resource_field_id, value_plain, value_cipher, nonce, key_version)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_resource_id, resource_field_id)
DO UPDATE SET value_plain = EXCLUDED.value_plain,
              value_cipher = EXCLUDED.value_cipher,
              nonce = EXCLUDED.nonce,
              key_version = EXCLUDED.key_version
RETURNING *;

-- name: ListResourceValues :many
SELECT * FROM tenant_resource_values WHERE tenant_resource_id = $1;

-- name: CreateAPIClient :one
INSERT INTO api_clients (name, key_hash) VALUES ($1, $2) RETURNING *;

-- name: GetAPIClientByHash :one
SELECT * FROM api_clients WHERE key_hash = $1;

-- name: ListAPIClients :many
SELECT * FROM api_clients ORDER BY created_at DESC;

-- name: SetAPIClientStatus :one
UPDATE api_clients SET status = $2 WHERE id = $1 RETURNING *;

-- name: ListOverviewTenantCards :many
SELECT t.id,
       t.name,
       t.slug,
       t.status,
       coalesce((array_agg(d.hostname ORDER BY d.created_at) FILTER (WHERE d.id IS NOT NULL))[1], '')::text AS primary_host,
       count(DISTINCT d.id)::int AS domain_count,
       count(DISTINCT tr.id) FILTER (WHERE tr.status = 'active')::int AS resource_count
FROM tenants t
LEFT JOIN tenant_domains d ON d.tenant_id = t.id
LEFT JOIN tenant_resources tr ON tr.tenant_id = t.id
GROUP BY t.id
ORDER BY t.name;

-- name: CountDefinitionsSummary :one
SELECT count(*)::int AS definitions,
       (count(*) FILTER (WHERE status = 'active'))::int AS active_definitions
FROM resource_definitions;

-- name: CountAPIClients :one
SELECT count(*)::int AS api_clients FROM api_clients;
