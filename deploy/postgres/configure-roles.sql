\set ON_ERROR_STOP on

SELECT format('CREATE ROLE %I NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT', role_name)
FROM (VALUES ('tenancit_runtime'), ('tenancit_jobs'), ('tenancit_backup'), ('tenancit_rewrap')) AS roles(role_name)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = role_name)
\gexec

SELECT format('ALTER DATABASE %I OWNER TO %I', :'database_name', :'migration_login')
\gexec
\connect :"database_name"
SELECT format('ALTER SCHEMA public OWNER TO %I', :'migration_login')
\gexec

-- Adopt objects from installations that previously migrated as postgres or a
-- shared runtime login. Index/constraint ownership follows the parent table.
SELECT format('ALTER TABLE %I.%I OWNER TO %I', n.nspname, c.relname, :'migration_login')
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p')
\gexec
SELECT format('ALTER SEQUENCE %I.%I OWNER TO %I', n.nspname, c.relname, :'migration_login')
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'S'
\gexec
SELECT format('ALTER VIEW %I.%I OWNER TO %I', n.nspname, c.relname, :'migration_login')
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'v'
\gexec
SELECT format('ALTER MATERIALIZED VIEW %I.%I OWNER TO %I', n.nspname, c.relname, :'migration_login')
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'm'
\gexec
SELECT format(
  'ALTER %s %I.%I(%s) OWNER TO %I',
  CASE p.prokind WHEN 'p' THEN 'PROCEDURE' ELSE 'FUNCTION' END,
  n.nspname,
  p.proname,
  pg_get_function_identity_arguments(p.oid),
  :'migration_login'
)
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE n.nspname = 'public' AND p.prokind IN ('f', 'p')
\gexec

SELECT format('GRANT tenancit_runtime TO %I', :'runtime_login')
\gexec
SELECT format('GRANT tenancit_jobs TO %I', :'jobs_login')
\gexec
SELECT format('GRANT tenancit_backup TO %I', :'backup_login')
\gexec
SELECT format('GRANT tenancit_rewrap TO %I', :'rewrap_login')
\gexec
