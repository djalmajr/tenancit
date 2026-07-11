-- +goose Up
CREATE TABLE admin_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash text NOT NULL UNIQUE CHECK (token_hash <> ''),
    csrf_token_hash text NOT NULL CHECK (csrf_token_hash <> ''),
    csrf_token_cipher bytea NOT NULL CHECK (octet_length(csrf_token_cipher) > 0),
    csrf_nonce bytea NOT NULL CHECK (octet_length(csrf_nonce) > 0),
    csrf_key_version smallint NOT NULL CHECK (csrf_key_version > 0),
    actor_issuer text NOT NULL CHECK (actor_issuer <> ''),
    actor_subject text NOT NULL CHECK (actor_subject <> ''),
    actor_label text,
    roles text[] NOT NULL,
    permissions text[] NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    last_used_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    expires_at timestamptz NOT NULL,
    idle_expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CHECK (cardinality(roles) > 0),
    CHECK (cardinality(permissions) > 0),
    CHECK (expires_at > created_at),
    CHECK (idle_expires_at > created_at),
    CHECK (idle_expires_at <= expires_at),
    CHECK (last_used_at >= created_at),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE INDEX admin_sessions_identity_idx
    ON admin_sessions (actor_issuer, actor_subject, created_at DESC);
CREATE INDEX admin_sessions_active_expiry_idx
    ON admin_sessions (expires_at, idle_expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE oidc_login_attempts (
    state_hash text PRIMARY KEY CHECK (state_hash <> ''),
    nonce_hash text NOT NULL CHECK (nonce_hash <> ''),
    pkce_verifier_cipher bytea NOT NULL CHECK (octet_length(pkce_verifier_cipher) > 0),
    cipher_nonce bytea NOT NULL CHECK (octet_length(cipher_nonce) > 0),
    key_version smallint NOT NULL CHECK (key_version > 0),
    redirect_after text NOT NULL DEFAULT '/' CHECK (redirect_after LIKE '/%'),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    CHECK (expires_at > created_at),
    CHECK (consumed_at IS NULL OR consumed_at >= created_at)
);

CREATE INDEX oidc_login_attempts_expiry_idx ON oidc_login_attempts (expires_at);

-- +goose Down
DROP TABLE IF EXISTS oidc_login_attempts;
DROP TABLE IF EXISTS admin_sessions;
