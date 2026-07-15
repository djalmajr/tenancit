-- name: CreateTenantResource :one
INSERT INTO tenant_resources (tenant_id, resource_definition_id, alias, display_name, source_resource_id)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateTenantResourceIdentity :one
UPDATE tenant_resources
SET alias = sqlc.arg(alias), display_name = sqlc.arg(display_name), updated_at = now()
WHERE id = sqlc.arg(id)
  AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;

-- name: SetTenantResourceStatus :one
UPDATE tenant_resources SET status = sqlc.arg(status), updated_at = now()
WHERE id = sqlc.arg(id)
  AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;

-- name: DeleteTenantResource :execrows
DELETE FROM tenant_resources
WHERE id = sqlc.arg(id)
  AND tenant_id = sqlc.arg(tenant_id);

-- name: ListTenantResources :many
SELECT * FROM tenant_resources WHERE tenant_id = $1;

-- name: ListActiveResourcesByTenant :many
SELECT * FROM tenant_resources WHERE tenant_id = $1 AND status = 'active';

-- name: ListResourceHeadersByTenant :many
SELECT tr.id,
       tr.tenant_id,
       tr.resource_definition_id,
       tr.alias,
       tr.display_name,
       tr.source_resource_id,
       tr.status,
       tr.created_at,
       tr.updated_at,
       rd.key AS definition_key,
       rd.name AS definition_name,
       rd.updated_at AS definition_updated_at,
       source.alias AS source_alias,
       source.updated_at AS source_updated_at
FROM tenant_resources tr
JOIN resource_definitions rd ON rd.id = tr.resource_definition_id
LEFT JOIN tenant_resources source ON source.id = tr.source_resource_id
WHERE tr.tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.arg(include_inactive)::boolean OR tr.status = 'active')
ORDER BY tr.created_at, tr.id;

-- name: GetResourceHeader :one
SELECT tr.id,
       tr.tenant_id,
       tr.resource_definition_id,
       tr.alias,
       tr.display_name,
       tr.source_resource_id,
       tr.status,
       tr.created_at,
       tr.updated_at,
       rd.key AS definition_key,
       rd.name AS definition_name,
       rd.updated_at AS definition_updated_at,
       source.alias AS source_alias,
       source.updated_at AS source_updated_at
FROM tenant_resources tr
JOIN resource_definitions rd ON rd.id = tr.resource_definition_id
LEFT JOIN tenant_resources source ON source.id = tr.source_resource_id
WHERE tr.id = $1;

-- name: GetTenantResource :one
SELECT * FROM tenant_resources
WHERE id = sqlc.arg(id) AND tenant_id = sqlc.arg(tenant_id)
FOR UPDATE;

-- name: GetActiveResourceByTenantAndAlias :one
SELECT tr.*
FROM tenant_resources tr
WHERE tr.tenant_id = $1
  AND lower(btrim(tr.alias)) = lower(btrim($2))
  AND tr.status = 'active';

-- name: GetResourceForLink :one
SELECT * FROM tenant_resources
WHERE id = sqlc.arg(id) AND tenant_id = sqlc.arg(tenant_id);

-- name: DeleteResourceValue :execrows
DELETE FROM tenant_resource_values
WHERE tenant_resource_id = sqlc.arg(tenant_resource_id)
  AND resource_field_id = sqlc.arg(resource_field_id);

-- name: UpsertResourceValue :one
WITH upserted AS (
    INSERT INTO tenant_resource_values
      (tenant_resource_id, resource_field_id, value_plain, value_cipher, nonce, key_version)
    VALUES ($1, $2, $3, $4, $5, $6)
    ON CONFLICT (tenant_resource_id, resource_field_id)
    DO UPDATE SET value_plain = EXCLUDED.value_plain,
                  value_cipher = EXCLUDED.value_cipher,
                  nonce = EXCLUDED.nonce,
                  key_version = EXCLUDED.key_version
    RETURNING *
), touched AS (
    UPDATE tenant_resources tr
    SET updated_at = clock_timestamp()
    FROM upserted u
    WHERE tr.id = u.tenant_resource_id
    RETURNING tr.id
)
SELECT u.*
FROM upserted u
JOIN touched t ON t.id = u.tenant_resource_id;

-- name: ListResourceValues :many
SELECT * FROM tenant_resource_values WHERE tenant_resource_id = $1;

-- name: ListResourceFieldValuesByResourceIDs :many
SELECT tr.id AS tenant_resource_id,
       rf.id AS resource_field_id,
       rf.key AS field_key,
       rf.label AS field_label,
       rf.data_type,
       rf.required,
       rf.is_secret,
       (local_value.id IS NOT NULL OR source_value.id IS NOT NULL)::boolean AS has_value,
       (local_value.id IS NOT NULL)::boolean AS is_override,
       coalesce(local_value.value_plain, source_value.value_plain) AS value_plain,
       coalesce(local_value.value_cipher, source_value.value_cipher) AS value_cipher,
       coalesce(local_value.nonce, source_value.nonce) AS nonce,
       coalesce(local_value.key_version, source_value.key_version) AS key_version
FROM tenant_resources tr
JOIN resource_fields rf ON rf.resource_definition_id = tr.resource_definition_id
LEFT JOIN tenant_resource_values local_value
  ON local_value.tenant_resource_id = tr.id
 AND local_value.resource_field_id = rf.id
LEFT JOIN tenant_resource_values source_value
  ON source_value.tenant_resource_id = tr.source_resource_id
 AND source_value.resource_field_id = rf.id
WHERE tr.id = ANY(sqlc.arg(resource_ids)::uuid[])
ORDER BY tr.id, rf.sort_order, rf.key;

-- name: CreateAPIClient :one
INSERT INTO api_clients (name, key_hash, token_preview, rpm_limit, expires_at)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetAPIClientByHash :one
SELECT * FROM api_clients WHERE key_hash = $1;

-- name: GetAPIClientAuthByHash :one
SELECT c.*, coalesce(array_agg(s.scope ORDER BY s.scope) FILTER (WHERE s.scope IS NOT NULL), '{}')::text[] AS scopes
FROM api_clients c
LEFT JOIN api_client_scopes s ON s.api_client_id = c.id
WHERE c.key_hash = sqlc.arg(key_hash)
   OR EXISTS (
     SELECT 1 FROM api_client_previous_tokens previous
     WHERE previous.api_client_id = c.id
       AND previous.key_hash = sqlc.arg(key_hash)
       AND previous.valid_until > clock_timestamp()
   )
GROUP BY c.id;

-- name: GetAPIClient :one
SELECT * FROM api_clients WHERE id = $1 FOR UPDATE;

-- name: ReplaceAPIClientScopes :exec
WITH deleted AS (
  DELETE FROM api_client_scopes WHERE api_client_id = sqlc.arg(api_client_id)
  RETURNING 1
)
INSERT INTO api_client_scopes (api_client_id, scope)
SELECT sqlc.arg(api_client_id), unnest(sqlc.arg(scopes)::text[])
FROM (SELECT count(*) FROM deleted) AS deletion_barrier;

-- name: ListAPIClientScopes :many
SELECT scope FROM api_client_scopes WHERE api_client_id = $1 ORDER BY scope;

-- name: TouchAPIClientLastUsed :exec
UPDATE api_clients
SET last_used_at = GREATEST(coalesce(last_used_at, sqlc.arg(used_at)::timestamptz), sqlc.arg(used_at)::timestamptz)
WHERE id = sqlc.arg(api_client_id)
  AND (last_used_at IS NULL OR last_used_at < sqlc.arg(used_at)::timestamptz - interval '1 minute');

-- name: UpsertAPIClientUsageDaily :exec
INSERT INTO api_client_usage_daily (
  day, api_client_id, operation, status_class, request_count, rate_limited_count
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (day, api_client_id, operation, status_class)
DO UPDATE SET
  request_count = api_client_usage_daily.request_count + EXCLUDED.request_count,
  rate_limited_count = api_client_usage_daily.rate_limited_count + EXCLUDED.rate_limited_count;

-- name: ListAPIClientUsage :many
SELECT * FROM api_client_usage_daily
WHERE api_client_id = sqlc.arg(api_client_id)
  AND day >= sqlc.arg(from_day)
  AND day <= sqlc.arg(to_day)
ORDER BY day DESC, operation, status_class
LIMIT sqlc.arg(page_limit);

-- name: ListAPIClientUsageOverview :many
SELECT u.*, coalesce(c.name, u.api_client_id::text)::text AS client_name
FROM api_client_usage_daily u
LEFT JOIN api_clients c ON c.id = u.api_client_id
WHERE u.day >= sqlc.arg(from_day) AND u.day <= sqlc.arg(to_day)
ORDER BY u.day DESC, client_name, u.operation, u.status_class
LIMIT sqlc.arg(page_limit);

-- name: DeleteExpiredAPIClientUsage :execrows
DELETE FROM api_client_usage_daily WHERE day < sqlc.arg(cutoff_day);

-- name: ListAPIClients :many
SELECT * FROM api_clients ORDER BY created_at DESC;

-- name: SetAPIClientStatus :one
UPDATE api_clients
SET status = $2,
    revoked_at = CASE WHEN $2 = 'revoked' THEN clock_timestamp() ELSE NULL END,
    updated_at = clock_timestamp()
WHERE id = $1 RETURNING *;

-- name: UpdateAPIClientPolicy :one
UPDATE api_clients
SET name = sqlc.arg(name), rpm_limit = sqlc.arg(rpm_limit),
    expires_at = sqlc.arg(expires_at), updated_at = clock_timestamp()
WHERE id = sqlc.arg(id) AND status = 'active'
RETURNING *;

-- name: SavePreviousAPIClientToken :exec
INSERT INTO api_client_previous_tokens (api_client_id, key_hash, valid_until)
VALUES ($1, $2, $3);

-- name: RotateAPIClientToken :one
UPDATE api_clients
SET key_hash = sqlc.arg(key_hash), token_preview = sqlc.arg(token_preview),
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(id) AND status = 'active'
RETURNING *;

-- name: DeleteAPIClient :execrows
DELETE FROM api_clients WHERE id = $1 AND status = 'revoked';

-- name: InsertAdminAuditEvent :one
INSERT INTO admin_audit_events (
  request_id, actor_kind, actor_issuer, actor_subject, actor_label,
  action, target_type, target_id, result, http_method, route_template,
  http_status, error_code, metadata
) VALUES (
  sqlc.arg(request_id), sqlc.arg(actor_kind), sqlc.narg(actor_issuer),
  sqlc.arg(actor_subject), sqlc.narg(actor_label), sqlc.arg(action),
  sqlc.arg(target_type), sqlc.arg(target_id), sqlc.arg(result),
  sqlc.arg(http_method), sqlc.arg(route_template), sqlc.arg(http_status),
  sqlc.narg(error_code), sqlc.arg(metadata)
)
RETURNING *;

-- name: ListAdminAuditEvents :many
SELECT * FROM admin_audit_events
WHERE occurred_at >= sqlc.arg(from_time)
  AND occurred_at < sqlc.arg(to_time)
  AND (sqlc.arg(actor_kind)::text = '' OR actor_kind = sqlc.arg(actor_kind))
  AND (sqlc.arg(actor_subject)::text = '' OR actor_subject = sqlc.arg(actor_subject))
  AND (sqlc.arg(action)::text = '' OR action = sqlc.arg(action))
  AND (sqlc.arg(target_type)::text = '' OR target_type = sqlc.arg(target_type))
  AND (sqlc.arg(target_id)::text = '' OR target_id = sqlc.arg(target_id))
  AND (sqlc.arg(request_id)::text = '' OR request_id = sqlc.arg(request_id))
  AND (sqlc.arg(result)::text = '' OR result = sqlc.arg(result))
  AND (
    NOT sqlc.arg(has_cursor)::boolean
    OR (occurred_at, id) < (sqlc.arg(cursor_time), sqlc.arg(cursor_id)::uuid)
  )
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg(page_limit);

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
