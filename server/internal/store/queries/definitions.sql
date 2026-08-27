-- name: CreateDefinition :one
INSERT INTO resource_definitions (key, name, description, icon)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetDefinition :one
SELECT * FROM resource_definitions WHERE id = $1;

-- name: GetResourceField :one
SELECT * FROM resource_fields
WHERE id = sqlc.arg(field_id) AND resource_definition_id = sqlc.arg(resource_definition_id)
FOR UPDATE;

-- name: GetDefinitionByKey :one
SELECT * FROM resource_definitions WHERE key = $1;

-- name: ListDefinitions :many
SELECT * FROM resource_definitions ORDER BY name;

-- name: ListDefinitionsWithCounts :many
SELECT rd.*,
       count(rf.id)::int AS field_count,
       count(rf.id) FILTER (WHERE rf.is_secret)::int AS secret_count
FROM resource_definitions rd
LEFT JOIN resource_fields rf ON rf.resource_definition_id = rd.id
GROUP BY rd.id
ORDER BY rd.name;

-- name: SetDefinitionStatus :one
UPDATE resource_definitions SET status = $2, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: UpdateDefinition :one
UPDATE resource_definitions
SET name = sqlc.arg(name),
    description = sqlc.arg(description),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteDefinition :execrows
DELETE FROM resource_definitions WHERE id = $1;

-- name: AddField :one
WITH inserted AS (
    INSERT INTO resource_fields
      (resource_definition_id, key, label, hint, data_type, required, is_secret, sort_order)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING *
), touched AS (
    UPDATE resource_definitions rd
    SET updated_at = clock_timestamp()
    FROM inserted i
    WHERE rd.id = i.resource_definition_id
    RETURNING rd.id
)
SELECT i.*
FROM inserted i
JOIN touched t ON t.id = i.resource_definition_id;

-- name: UpdateField :one
WITH updated AS (
    UPDATE resource_fields rf
    SET label = sqlc.arg(label),
        required = sqlc.arg(required)
    WHERE rf.id = sqlc.arg(field_id)
      AND rf.resource_definition_id = sqlc.arg(resource_definition_id)
    RETURNING rf.*
), touched AS (
    UPDATE resource_definitions rd
    SET updated_at = clock_timestamp()
    FROM updated u
    WHERE rd.id = u.resource_definition_id
    RETURNING rd.id
)
SELECT u.*
FROM updated u
JOIN touched t ON t.id = u.resource_definition_id;

-- name: RemoveField :one
WITH deleted AS (
    DELETE FROM resource_fields rf
    WHERE rf.id = sqlc.arg(field_id)
      AND rf.resource_definition_id = sqlc.arg(resource_definition_id)
    RETURNING rf.resource_definition_id
), touched AS (
    UPDATE resource_definitions rd
    SET updated_at = clock_timestamp()
    FROM deleted d
    WHERE rd.id = d.resource_definition_id
    RETURNING rd.id
)
SELECT count(*)::bigint
FROM deleted d
LEFT JOIN touched t ON t.id = d.resource_definition_id;

-- name: ListFields :many
SELECT * FROM resource_fields WHERE resource_definition_id = $1
ORDER BY sort_order, key;
