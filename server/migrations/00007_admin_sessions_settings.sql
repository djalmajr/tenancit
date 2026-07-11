-- +goose Up
CREATE TABLE admin_settings (
    key text PRIMARY KEY,
    value text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_by_kind text NOT NULL,
    updated_by_issuer text,
    updated_by_subject text NOT NULL,
    CONSTRAINT admin_settings_key_nonempty CHECK (btrim(key) <> ''),
    CONSTRAINT admin_settings_value_nonempty CHECK (btrim(value) <> '')
);

CREATE TABLE admin_settings_revision (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);

INSERT INTO admin_settings_revision (singleton, version) VALUES (true, 1);

CREATE INDEX admin_sessions_principal_idx
    ON admin_sessions (actor_issuer, actor_subject, created_at DESC);
CREATE INDEX admin_sessions_active_idx
    ON admin_sessions (last_used_at DESC)
    WHERE revoked_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS admin_sessions_active_idx;
DROP INDEX IF EXISTS admin_sessions_principal_idx;
DROP TABLE IF EXISTS admin_settings_revision;
DROP TABLE IF EXISTS admin_settings;
