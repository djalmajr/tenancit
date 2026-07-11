-- +goose Up
CREATE TABLE audit_legal_holds (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    from_time timestamptz NOT NULL,
    to_time timestamptz NOT NULL,
    reason text NOT NULL CHECK (btrim(reason) <> '' AND octet_length(reason) <= 500),
    created_by_kind text NOT NULL CHECK (created_by_kind IN ('shared_admin_token', 'break_glass', 'oidc_user')),
    created_by_issuer text,
    created_by_subject text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    released_at timestamptz,
    released_by_subject text,
    CHECK (to_time > from_time),
    CHECK ((created_by_kind = 'oidc_user') = (created_by_issuer IS NOT NULL)),
    CHECK ((released_at IS NULL) = (released_by_subject IS NULL))
);

CREATE INDEX audit_legal_holds_active_window_idx
    ON audit_legal_holds (from_time, to_time) WHERE released_at IS NULL;

CREATE TABLE audit_partition_registry (
    partition_name text PRIMARY KEY CHECK (partition_name ~ '^admin_audit_events_[0-9]{6}$'),
    from_time timestamptz NOT NULL UNIQUE,
    to_time timestamptz NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (to_time > from_time)
);

CREATE TABLE audit_export_jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key uuid NOT NULL,
    request_fingerprint bytea NOT NULL CHECK (octet_length(request_fingerprint) = 32),
    requested_by_kind text NOT NULL CHECK (requested_by_kind IN ('shared_admin_token', 'oidc_user')),
    requested_by_issuer text,
    requested_by_subject text NOT NULL,
    filters jsonb NOT NULL CHECK (jsonb_typeof(filters) = 'object' AND octet_length(filters::text) <= 4096),
    format text NOT NULL CHECK (format IN ('csv', 'jsonl')),
    status text NOT NULL CHECK (status IN ('pending', 'processing', 'ready', 'failed', 'expired')),
    row_count bigint CHECK (row_count IS NULL OR row_count >= 0),
    payload_cipher bytea,
    nonce bytea,
    key_version smallint,
    failure_code text,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    started_at timestamptz,
    completed_at timestamptz,
    expires_at timestamptz NOT NULL,
    downloaded_at timestamptz,
    UNIQUE (requested_by_kind, requested_by_subject, idempotency_key),
    CHECK ((requested_by_kind = 'oidc_user') = (requested_by_issuer IS NOT NULL)),
    CHECK ((status = 'ready') = (payload_cipher IS NOT NULL)),
    CHECK ((payload_cipher IS NULL) = (nonce IS NULL)),
    CHECK ((payload_cipher IS NULL) = (key_version IS NULL)),
    CHECK (expires_at > created_at)
);

CREATE INDEX audit_export_jobs_pending_idx
    ON audit_export_jobs (created_at, id) WHERE status = 'pending';
CREATE INDEX audit_export_jobs_expiry_idx
    ON audit_export_jobs (expires_at) WHERE status IN ('pending', 'processing', 'ready');

-- +goose StatementBegin
CREATE FUNCTION maintain_admin_audit_partitions(
    p_current_time timestamptz,
    p_retention_days integer,
    p_future_months integer
) RETURNS TABLE(partitions_created integer, partitions_dropped integer, partitions_held integer)
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, public AS $$
DECLARE
    month_start timestamptz;
    month_end timestamptz;
    partition_name text;
    candidate record;
    offset_value integer;
BEGIN
    IF p_retention_days < 1 OR p_future_months < 1 OR p_future_months > 24 THEN
        RAISE EXCEPTION 'invalid audit maintenance configuration';
    END IF;
    IF NOT pg_try_advisory_xact_lock(6072342442203466069) THEN
        RAISE EXCEPTION 'audit maintenance already running';
    END IF;
    partitions_created := 0; partitions_dropped := 0; partitions_held := 0;

    IF EXISTS (SELECT 1 FROM public.admin_audit_events_default LIMIT 1) THEN
        LOCK TABLE public.admin_audit_events IN ACCESS EXCLUSIVE MODE;
        ALTER TABLE public.admin_audit_events DETACH PARTITION public.admin_audit_events_default;
        FOR candidate IN
            SELECT DISTINCT date_trunc('month', occurred_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' AS from_time
            FROM public.admin_audit_events_default
            ORDER BY 1
        LOOP
            month_start := candidate.from_time;
            month_end := month_start + interval '1 month';
            partition_name := 'admin_audit_events_' || to_char(month_start, 'YYYYMM');
            EXECUTE format('CREATE TABLE IF NOT EXISTS public.%I PARTITION OF public.admin_audit_events FOR VALUES FROM (%L) TO (%L)', partition_name, month_start, month_end);
            EXECUTE format('INSERT INTO public.admin_audit_events SELECT * FROM public.admin_audit_events_default WHERE occurred_at >= %L AND occurred_at < %L', month_start, month_end);
            INSERT INTO public.audit_partition_registry(partition_name,from_time,to_time)
            VALUES (partition_name,month_start,month_end) ON CONFLICT DO NOTHING;
            IF FOUND THEN partitions_created := partitions_created + 1; END IF;
        END LOOP;
        DROP TABLE public.admin_audit_events_default;
        CREATE TABLE public.admin_audit_events_default PARTITION OF public.admin_audit_events DEFAULT;
    END IF;

    FOR offset_value IN 0..p_future_months LOOP
        month_start := date_trunc('month', p_current_time AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' + make_interval(months => offset_value);
        month_end := month_start + interval '1 month';
        partition_name := 'admin_audit_events_' || to_char(month_start, 'YYYYMM');
        EXECUTE format('CREATE TABLE IF NOT EXISTS public.%I PARTITION OF public.admin_audit_events FOR VALUES FROM (%L) TO (%L)', partition_name, month_start, month_end);
        INSERT INTO public.audit_partition_registry(partition_name,from_time,to_time)
        VALUES (partition_name,month_start,month_end) ON CONFLICT DO NOTHING;
        IF FOUND THEN partitions_created := partitions_created + 1; END IF;
    END LOOP;
    FOR candidate IN
        SELECT p.partition_name, EXISTS(
            SELECT 1 FROM public.audit_legal_holds h
            WHERE h.released_at IS NULL AND h.from_time < p.to_time AND h.to_time > p.from_time
        ) AS held
        FROM public.audit_partition_registry p
        WHERE p.to_time <= p_current_time - make_interval(days => p_retention_days)
        ORDER BY p.from_time
    LOOP
        IF candidate.held THEN
            partitions_held := partitions_held + 1;
        ELSE
            DELETE FROM public.audit_partition_registry p WHERE p.partition_name = candidate.partition_name;
            EXECUTE format('DROP TABLE public.%I', candidate.partition_name);
            partitions_dropped := partitions_dropped + 1;
        END IF;
    END LOOP;
    RETURN NEXT;
END;
$$;
-- +goose StatementEnd

REVOKE ALL ON FUNCTION maintain_admin_audit_partitions(timestamptz,integer,integer) FROM PUBLIC;

-- Existing rows remain in the default partition. New writes are routed to
-- monthly partitions created by the maintenance worker before the month starts.

-- +goose Down
DROP FUNCTION IF EXISTS maintain_admin_audit_partitions(timestamptz,integer,integer);
DROP TABLE IF EXISTS audit_export_jobs;
DROP TABLE IF EXISTS audit_partition_registry;
DROP TABLE IF EXISTS audit_legal_holds;
