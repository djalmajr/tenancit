-- +goose Up
CREATE UNIQUE INDEX api_clients_name_normalized_unique
    ON api_clients (lower(btrim(name)));

CREATE TABLE api_client_previous_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    api_client_id uuid NOT NULL REFERENCES api_clients(id) ON DELETE CASCADE,
    key_hash text NOT NULL UNIQUE,
    valid_until timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX api_client_previous_tokens_client_idx
    ON api_client_previous_tokens (api_client_id, valid_until DESC);

-- +goose Down
DROP TABLE IF EXISTS api_client_previous_tokens;
DROP INDEX IF EXISTS api_clients_name_normalized_unique;
