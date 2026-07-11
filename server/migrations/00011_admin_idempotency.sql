-- +goose Up
CREATE TABLE admin_idempotency_records (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_kind text NOT NULL CHECK (actor_kind IN ('shared_admin_token', 'oidc_user')),
    actor_issuer text NOT NULL DEFAULT '',
    actor_subject text NOT NULL,
    operation text NOT NULL CHECK (btrim(operation) <> '' AND octet_length(operation) <= 256),
    idempotency_key uuid NOT NULL,
    request_fingerprint bytea NOT NULL CHECK (octet_length(request_fingerprint) = 32),
    status text NOT NULL DEFAULT 'processing' CHECK (status IN ('processing', 'completed')),
    http_status smallint CHECK (http_status BETWEEN 200 AND 299),
    content_type text CHECK (content_type IS NULL OR octet_length(content_type) <= 128),
    response_cipher bytea,
    nonce bytea,
    key_version smallint,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    completed_at timestamptz,
    expires_at timestamptz NOT NULL,
    UNIQUE (actor_kind, actor_issuer, actor_subject, operation, idempotency_key),
    CHECK ((actor_kind = 'oidc_user') = (actor_issuer <> '')),
    CHECK ((status = 'completed') = (completed_at IS NOT NULL)),
    CHECK ((status = 'completed') = (response_cipher IS NOT NULL)),
    CHECK ((response_cipher IS NULL) = (nonce IS NULL)),
    CHECK ((response_cipher IS NULL) = (key_version IS NULL)),
    CHECK (expires_at > created_at)
);

CREATE INDEX admin_idempotency_records_expiry_idx
    ON admin_idempotency_records (expires_at);

-- +goose Down
DROP TABLE IF EXISTS admin_idempotency_records;
