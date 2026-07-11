-- +goose Up
CREATE TABLE operational_reports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind text NOT NULL,
    source text NOT NULL,
    status text NOT NULL,
    occurred_at timestamptz NOT NULL,
    fresh_until timestamptz NOT NULL,
    received_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    idempotency_key text NOT NULL,
    payload_hash bytea NOT NULL,
    credential_version text NOT NULL,
    CONSTRAINT operational_report_kind_valid CHECK (kind IN ('backup','restore','rewrap','migration')),
    CONSTRAINT operational_report_status_valid CHECK (status IN ('healthy','degraded','failed')),
    CONSTRAINT operational_report_source_valid CHECK (btrim(source) <> '' AND length(source) <= 128),
    CONSTRAINT operational_report_freshness_valid CHECK (fresh_until > occurred_at),
    CONSTRAINT operational_report_idempotency_valid CHECK (btrim(idempotency_key) <> '' AND length(idempotency_key) <= 128),
    CONSTRAINT operational_report_credential_valid CHECK (btrim(credential_version) <> ''),
    CONSTRAINT operational_report_idempotency_unique UNIQUE (source, idempotency_key)
);

CREATE INDEX operational_reports_latest_idx
    ON operational_reports (kind, source, occurred_at DESC, id DESC);

-- +goose StatementBegin
CREATE FUNCTION reject_operational_report_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'operational reports are append-only';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER operational_reports_append_only
BEFORE UPDATE OR DELETE ON operational_reports
FOR EACH ROW EXECUTE FUNCTION reject_operational_report_mutation();

-- +goose Down
DROP TRIGGER IF EXISTS operational_reports_append_only ON operational_reports;
DROP FUNCTION IF EXISTS reject_operational_report_mutation();
DROP TABLE IF EXISTS operational_reports;
