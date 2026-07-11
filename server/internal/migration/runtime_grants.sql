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
END
$roles$;

REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO tenancit_runtime, tenancit_jobs, tenancit_backup;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO tenancit_runtime;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO tenancit_runtime;
REVOKE TRUNCATE, REFERENCES, TRIGGER ON ALL TABLES IN SCHEMA public FROM tenancit_runtime;
REVOKE UPDATE, DELETE ON admin_audit_events, outbox_events, operational_reports FROM tenancit_runtime;
GRANT SELECT, INSERT ON admin_audit_events, outbox_events, operational_reports TO tenancit_runtime;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO tenancit_backup;
GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO tenancit_backup;

GRANT SELECT, INSERT, UPDATE, DELETE ON
    api_client_usage_daily, webhook_targets, webhook_deliveries,
    webhook_dead_letters, outbox_events, operational_reports
TO tenancit_jobs;
REVOKE UPDATE, DELETE ON outbox_events, operational_reports FROM tenancit_jobs;
GRANT SELECT, INSERT ON outbox_events, operational_reports TO tenancit_jobs;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO tenancit_runtime;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO tenancit_runtime;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT ON TABLES TO tenancit_backup;
