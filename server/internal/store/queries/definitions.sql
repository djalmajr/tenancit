-- name: CreateDefinition :one
INSERT INTO resource_definitions (key, name, description, icon)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetDefinition :one
SELECT * FROM resource_definitions WHERE id = $1;

-- name: GetDefinitionByKey :one
SELECT * FROM resource_definitions WHERE key = $1;

-- name: ListDefinitions :many
SELECT * FROM resource_definitions ORDER BY name;

-- name: SetDefinitionStatus :one
UPDATE resource_definitions SET status = $2, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: AddField :one
INSERT INTO resource_fields
  (resource_definition_id, key, label, hint, data_type, required, is_secret, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: RemoveField :exec
DELETE FROM resource_fields WHERE id = $1;

-- name: ListFields :many
SELECT * FROM resource_fields WHERE resource_definition_id = $1
ORDER BY sort_order, key;
