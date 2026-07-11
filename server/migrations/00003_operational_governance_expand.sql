-- +goose Up
ALTER TABLE api_clients
    ADD COLUMN token_preview text,
    ADD COLUMN rpm_limit integer,
    ADD COLUMN expires_at timestamptz,
    ADD COLUMN last_used_at timestamptz,
    ADD COLUMN revoked_at timestamptz,
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE api_clients
    ADD CONSTRAINT api_clients_rpm_limit_positive
        CHECK (rpm_limit IS NULL OR rpm_limit > 0) NOT VALID,
    ADD CONSTRAINT api_clients_expiration_after_creation
        CHECK (expires_at IS NULL OR expires_at > created_at) NOT VALID;

CREATE TABLE api_client_scopes (
    api_client_id uuid NOT NULL REFERENCES api_clients(id) ON DELETE CASCADE,
    scope text NOT NULL CHECK (scope IN ('tenant:identify', 'resource:resolve')),
    PRIMARY KEY (api_client_id, scope)
);

INSERT INTO api_client_scopes (api_client_id, scope)
SELECT id, scope
FROM api_clients
CROSS JOIN (VALUES ('tenant:identify'), ('resource:resolve')) AS legacy(scope);

-- +goose StatementBegin
CREATE FUNCTION grant_legacy_api_client_scopes() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO api_client_scopes (api_client_id, scope)
    VALUES (NEW.id, 'tenant:identify'), (NEW.id, 'resource:resolve');
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER api_clients_grant_legacy_scopes
AFTER INSERT ON api_clients
FOR EACH ROW EXECUTE FUNCTION grant_legacy_api_client_scopes();

CREATE TABLE api_client_usage_daily (
    day date NOT NULL,
    api_client_id uuid NOT NULL,
    operation text NOT NULL CHECK (operation IN ('identify', 'resolve')),
    status_class smallint NOT NULL CHECK (status_class BETWEEN 2 AND 5),
    request_count bigint NOT NULL DEFAULT 0 CHECK (request_count >= 0),
    rate_limited_count bigint NOT NULL DEFAULT 0 CHECK (rate_limited_count >= 0),
    PRIMARY KEY (day, api_client_id, operation, status_class)
);

CREATE TABLE admin_audit_events (
    occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    schema_version smallint NOT NULL DEFAULT 1 CHECK (schema_version > 0),
    request_id text NOT NULL CHECK (request_id <> ''),
    actor_kind text NOT NULL CHECK (actor_kind IN ('shared_admin_token', 'break_glass', 'oidc_user', 'unauthenticated')),
    actor_issuer text,
    actor_subject text NOT NULL,
    actor_label text,
    action text NOT NULL CHECK (action <> ''),
    target_type text NOT NULL CHECK (target_type <> ''),
    target_id text NOT NULL,
    result text NOT NULL CHECK (result IN ('success', 'denied', 'error')),
    http_method text NOT NULL,
    route_template text NOT NULL,
    http_status smallint NOT NULL CHECK (http_status BETWEEN 100 AND 599),
    error_code text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (occurred_at, id),
    CHECK (octet_length(metadata::text) <= 8192),
    CHECK ((actor_kind = 'oidc_user') = (actor_issuer IS NOT NULL))
) PARTITION BY RANGE (occurred_at);

CREATE TABLE admin_audit_events_default
    PARTITION OF admin_audit_events DEFAULT;

CREATE INDEX admin_audit_events_order_idx
    ON admin_audit_events (occurred_at DESC, id DESC);
CREATE INDEX admin_audit_events_request_idx
    ON admin_audit_events (request_id, occurred_at DESC);
CREATE INDEX admin_audit_events_target_idx
    ON admin_audit_events (target_type, target_id, occurred_at DESC);
CREATE INDEX admin_audit_events_actor_idx
    ON admin_audit_events (actor_kind, actor_issuer, actor_subject, occurred_at DESC);
CREATE INDEX admin_audit_events_action_idx
    ON admin_audit_events (action, result, occurred_at DESC);

-- +goose StatementBegin
CREATE FUNCTION reject_admin_audit_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'admin audit events are append-only';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER admin_audit_events_append_only
BEFORE UPDATE OR DELETE ON admin_audit_events
FOR EACH ROW EXECUTE FUNCTION reject_admin_audit_mutation();

-- +goose Down
DROP TABLE IF EXISTS admin_audit_events CASCADE;
DROP FUNCTION IF EXISTS reject_admin_audit_mutation();
DROP TABLE IF EXISTS api_client_usage_daily;
DROP TRIGGER IF EXISTS api_clients_grant_legacy_scopes ON api_clients;
DROP FUNCTION IF EXISTS grant_legacy_api_client_scopes();
DROP TABLE IF EXISTS api_client_scopes;
ALTER TABLE api_clients
    DROP CONSTRAINT IF EXISTS api_clients_expiration_after_creation,
    DROP CONSTRAINT IF EXISTS api_clients_rpm_limit_positive,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS revoked_at,
    DROP COLUMN IF EXISTS last_used_at,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS rpm_limit,
    DROP COLUMN IF EXISTS token_preview;
