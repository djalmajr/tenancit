DO $roles$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tenancit_runtime') THEN
        CREATE ROLE tenancit_runtime NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tenancit_jobs') THEN
        CREATE ROLE tenancit_jobs NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tenancit_backup') THEN
        CREATE ROLE tenancit_backup NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tenancit_rewrap') THEN
        CREATE ROLE tenancit_rewrap NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
    END IF;
END
$roles$;

REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO tenancit_runtime, tenancit_jobs, tenancit_backup, tenancit_rewrap;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO tenancit_runtime;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO tenancit_runtime;
REVOKE TRUNCATE, REFERENCES, TRIGGER ON ALL TABLES IN SCHEMA public FROM tenancit_runtime;
REVOKE UPDATE, DELETE ON admin_audit_events, outbox_events, operational_reports FROM tenancit_runtime;
GRANT SELECT, INSERT ON admin_audit_events, outbox_events, operational_reports TO tenancit_runtime;
REVOKE ALL ON audit_legal_holds, audit_partition_registry, audit_export_jobs FROM tenancit_runtime;
GRANT SELECT, INSERT ON audit_legal_holds, audit_export_jobs TO tenancit_runtime;
GRANT UPDATE (released_at, released_by_subject) ON audit_legal_holds TO tenancit_runtime;
GRANT UPDATE (status, row_count, payload_cipher, nonce, key_version, failure_code, started_at, completed_at, downloaded_at) ON audit_export_jobs TO tenancit_runtime;
GRANT SELECT ON audit_partition_registry TO tenancit_runtime;
REVOKE ALL ON admin_idempotency_records FROM tenancit_runtime;
GRANT SELECT, INSERT ON admin_idempotency_records TO tenancit_runtime;
GRANT UPDATE (status, http_status, content_type, response_cipher, nonce, key_version, completed_at) ON admin_idempotency_records TO tenancit_runtime;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO tenancit_backup;
GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO tenancit_backup;

REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM tenancit_rewrap;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM tenancit_rewrap;
GRANT SELECT (id, value_plain, value_cipher, nonce, key_version) ON tenant_resource_values TO tenancit_rewrap;
GRANT UPDATE (value_cipher, nonce, key_version) ON tenant_resource_values TO tenancit_rewrap;
GRANT SELECT ON operational_reports TO tenancit_rewrap;

GRANT SELECT, INSERT, UPDATE, DELETE ON
    api_client_usage_daily, webhook_targets, webhook_deliveries,
    webhook_dead_letters, outbox_events, operational_reports
TO tenancit_jobs;
REVOKE UPDATE, DELETE ON outbox_events, operational_reports FROM tenancit_jobs;
GRANT SELECT, INSERT ON outbox_events, operational_reports TO tenancit_jobs;
GRANT SELECT, INSERT, UPDATE ON audit_export_jobs TO tenancit_jobs;
GRANT SELECT, DELETE ON admin_idempotency_records TO tenancit_jobs;
GRANT SELECT ON admin_audit_events, audit_legal_holds, audit_partition_registry, admin_settings, admin_settings_revision TO tenancit_jobs;
GRANT EXECUTE ON FUNCTION maintain_admin_audit_partitions(timestamptz,integer,integer) TO tenancit_jobs;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO tenancit_runtime;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO tenancit_runtime;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT ON TABLES TO tenancit_backup;
