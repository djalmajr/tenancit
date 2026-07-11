#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
: "${TENANCIT_ROLE_TEST_ADMIN_URL:?set an isolated PostgreSQL admin DSN}"
command -v psql >/dev/null 2>&1 || { echo "psql is required" >&2; exit 1; }
test -x "$root_dir/server/bin/migrate" || { echo "build server/bin/migrate first" >&2; exit 1; }

suffix="$$"
database="tenancit_roles_$suffix"
migration_login="tenancit_migration_$suffix"
runtime_login="tenancit_runtime_$suffix"
jobs_login="tenancit_jobs_$suffix"
backup_login="tenancit_backup_$suffix"

cleanup() {
  psql "$TENANCIT_ROLE_TEST_ADMIN_URL" -v ON_ERROR_STOP=1 \
    -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='$database' AND pid<>pg_backend_pid()" \
    -c "DROP DATABASE IF EXISTS $database" \
    -c "DROP ROLE IF EXISTS $runtime_login" \
    -c "DROP ROLE IF EXISTS $jobs_login" \
    -c "DROP ROLE IF EXISTS $backup_login" \
    -c "DROP ROLE IF EXISTS $migration_login" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM
cleanup

psql "$TENANCIT_ROLE_TEST_ADMIN_URL" -v ON_ERROR_STOP=1 \
  -c "CREATE ROLE $migration_login LOGIN PASSWORD 'migration-test'" \
  -c "CREATE ROLE $runtime_login LOGIN PASSWORD 'runtime-test'" \
  -c "CREATE ROLE $jobs_login LOGIN PASSWORD 'jobs-test'" \
  -c "CREATE ROLE $backup_login LOGIN PASSWORD 'backup-test'" \
  -c "CREATE DATABASE $database" >/dev/null

# Reproduce an existing installation whose schema is still owned by the old
# shared admin login, then adopt it into the dedicated migration role.
admin_without_query="${TENANCIT_ROLE_TEST_ADMIN_URL%%[?]*}"
case "$admin_without_query" in
  */*) old_owner_url="${admin_without_query%/*}/$database" ;;
  *) echo "admin DSN must include a database path" >&2; exit 1 ;;
esac
if [ "$admin_without_query" != "$TENANCIT_ROLE_TEST_ADMIN_URL" ]; then
  admin_query="${TENANCIT_ROLE_TEST_ADMIN_URL#*[?]}"
  old_owner_url="$old_owner_url?$admin_query"
fi
TENANCIT_DATABASE_URL="$old_owner_url" "$root_dir/server/bin/migrate" >/dev/null
TENANCIT_POSTGRES_ADMIN_URL="$TENANCIT_ROLE_TEST_ADMIN_URL" \
TENANCIT_DATABASE_NAME="$database" TENANCIT_MIGRATION_LOGIN="$migration_login" \
TENANCIT_RUNTIME_LOGIN="$runtime_login" TENANCIT_JOBS_LOGIN="$jobs_login" \
TENANCIT_BACKUP_LOGIN="$backup_login" "$root_dir/deploy/postgres/configure-roles.sh" >/dev/null

base="${TENANCIT_ROLE_TEST_HOSTPORT:-127.0.0.1:5432}/$database?sslmode=disable"
migration_url="postgres://${migration_login}:migration-test@$base"
runtime_url="postgres://${runtime_login}:runtime-test@$base"
jobs_url="postgres://${jobs_login}:jobs-test@$base"
backup_url="postgres://${backup_login}:backup-test@$base"
TENANCIT_DATABASE_URL="$migration_url" "$root_dir/server/bin/migrate" >/dev/null

[ "$(psql "$runtime_url" -Atqc 'SELECT count(*) FROM tenants')" = 0 ]
if psql "$runtime_url" -v ON_ERROR_STOP=1 -c 'CREATE TABLE forbidden(id integer)' >/dev/null 2>&1; then
  echo "runtime role created DDL" >&2; exit 1
fi
[ "$(psql "$backup_url" -Atqc 'SELECT count(*) FROM tenants')" = 0 ]
if psql "$backup_url" -v ON_ERROR_STOP=1 -c "INSERT INTO tenants(slug,name) VALUES('forbidden','forbidden')" >/dev/null 2>&1; then
  echo "backup role wrote domain data" >&2; exit 1
fi
[ "$(psql "$jobs_url" -Atqc 'SELECT count(*) FROM webhook_deliveries')" = 0 ]
if psql "$jobs_url" -v ON_ERROR_STOP=1 -c "INSERT INTO tenants(slug,name) VALUES('forbidden','forbidden')" >/dev/null 2>&1; then
  echo "jobs role wrote domain data" >&2; exit 1
fi
unowned="$(psql "$migration_url" -Atqc "SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind IN ('r','p','S') AND pg_get_userbyid(c.relowner)<>current_user")"
[ "$unowned" = 0 ] || { echo "migration role did not adopt all schema objects" >&2; exit 1; }

echo "PostgreSQL role separation and existing-schema adoption passed"
