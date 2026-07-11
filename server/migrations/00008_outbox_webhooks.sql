-- +goose Up
ALTER TABLE api_client_scopes DROP CONSTRAINT api_client_scopes_scope_check;
ALTER TABLE api_client_scopes ADD CONSTRAINT api_client_scopes_scope_check
    CHECK (scope IN ('tenant:identify', 'resource:resolve', 'events:read'));
ALTER TABLE api_client_usage_daily DROP CONSTRAINT api_client_usage_daily_operation_check;
ALTER TABLE api_client_usage_daily ADD CONSTRAINT api_client_usage_daily_operation_check
    CHECK (operation IN ('identify', 'resolve', 'events'));

CREATE TABLE webhook_targets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    format text NOT NULL DEFAULT 'generic',
    status text NOT NULL DEFAULT 'active',
    url_cipher bytea NOT NULL,
    url_nonce bytea NOT NULL,
    url_key_version smallint NOT NULL,
    signing_secret_cipher bytea NOT NULL,
    signing_secret_nonce bytea NOT NULL,
    signing_secret_key_version smallint NOT NULL,
    allow_loopback_http boolean NOT NULL DEFAULT false,
    consecutive_failures integer NOT NULL DEFAULT 0,
    circuit_open_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT webhook_targets_name_nonempty CHECK (btrim(name) <> ''),
    CONSTRAINT webhook_targets_name_unique UNIQUE (name),
    CONSTRAINT webhook_targets_format_valid CHECK (format IN ('generic', 'slack', 'discord', 'teams')),
    CONSTRAINT webhook_targets_status_valid CHECK (status IN ('active', 'disabled')),
    CONSTRAINT webhook_targets_failures_nonnegative CHECK (consecutive_failures >= 0)
);

CREATE TABLE outbox_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type text NOT NULL,
    event_version integer NOT NULL DEFAULT 1,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    request_id text NOT NULL,
    payload jsonb NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT outbox_event_type_nonempty CHECK (btrim(event_type) <> ''),
    CONSTRAINT outbox_event_version_positive CHECK (event_version > 0),
    CONSTRAINT outbox_aggregate_nonempty CHECK (btrim(aggregate_type) <> '' AND btrim(aggregate_id) <> ''),
    CONSTRAINT outbox_request_nonempty CHECK (btrim(request_id) <> ''),
    CONSTRAINT outbox_payload_object CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX outbox_events_feed_idx ON outbox_events (occurred_at DESC, id DESC);
CREATE INDEX outbox_events_aggregate_idx ON outbox_events (aggregate_type, aggregate_id, occurred_at DESC);

-- +goose StatementBegin
CREATE FUNCTION reject_outbox_event_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'outbox events are append-only';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER outbox_events_append_only
BEFORE UPDATE ON outbox_events
FOR EACH ROW EXECUTE FUNCTION reject_outbox_event_mutation();

CREATE TABLE webhook_deliveries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id uuid NOT NULL REFERENCES outbox_events(id) ON DELETE RESTRICT,
    target_id uuid NOT NULL REFERENCES webhook_targets(id) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'pending',
    attempt_count integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    lease_token uuid,
    lease_expires_at timestamptz,
    last_http_status integer,
    last_error_code text,
    delivered_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT webhook_delivery_unique UNIQUE (event_id, target_id),
    CONSTRAINT webhook_delivery_status_valid CHECK (status IN ('pending', 'delivering', 'retry', 'delivered', 'dead_letter')),
    CONSTRAINT webhook_delivery_attempts_nonnegative CHECK (attempt_count >= 0),
    CONSTRAINT webhook_delivery_http_status_valid CHECK (last_http_status IS NULL OR last_http_status BETWEEN 100 AND 599)
);

CREATE INDEX webhook_deliveries_due_idx
    ON webhook_deliveries (next_attempt_at, id)
    WHERE status IN ('pending', 'retry', 'delivering');
CREATE INDEX webhook_deliveries_target_idx
    ON webhook_deliveries (target_id, created_at DESC);

CREATE TABLE webhook_dead_letters (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id uuid NOT NULL UNIQUE REFERENCES webhook_deliveries(id) ON DELETE RESTRICT,
    reason_code text NOT NULL,
    failed_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    replayed_at timestamptz,
    CONSTRAINT webhook_dead_letter_reason_nonempty CHECK (btrim(reason_code) <> '')
);

-- +goose Down
DROP TABLE IF EXISTS webhook_dead_letters;
DROP TABLE IF EXISTS webhook_deliveries;
DROP TRIGGER IF EXISTS outbox_events_append_only ON outbox_events;
DROP FUNCTION IF EXISTS reject_outbox_event_mutation();
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS webhook_targets;
ALTER TABLE api_client_usage_daily DROP CONSTRAINT api_client_usage_daily_operation_check;
ALTER TABLE api_client_usage_daily ADD CONSTRAINT api_client_usage_daily_operation_check CHECK (operation IN ('identify', 'resolve'));
DELETE FROM api_client_scopes WHERE scope = 'events:read';
ALTER TABLE api_client_scopes DROP CONSTRAINT api_client_scopes_scope_check;
ALTER TABLE api_client_scopes ADD CONSTRAINT api_client_scopes_scope_check CHECK (scope IN ('tenant:identify', 'resource:resolve'));
