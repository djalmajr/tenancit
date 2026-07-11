#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
: "${TENANCIT_POSTGRES_ADMIN_URL:?set the administrative PostgreSQL URL}"
: "${TENANCIT_DATABASE_NAME:?set the target database name}"
: "${TENANCIT_MIGRATION_LOGIN:?set the pre-provisioned migration login}"
: "${TENANCIT_RUNTIME_LOGIN:?set the pre-provisioned runtime login}"
: "${TENANCIT_JOBS_LOGIN:?set the pre-provisioned jobs login}"
: "${TENANCIT_BACKUP_LOGIN:?set the pre-provisioned backup login}"

for identifier in "$TENANCIT_DATABASE_NAME" "$TENANCIT_MIGRATION_LOGIN" "$TENANCIT_RUNTIME_LOGIN" "$TENANCIT_JOBS_LOGIN" "$TENANCIT_BACKUP_LOGIN"; do
  case "$identifier" in
    ''|*[!A-Za-z0-9_-]*) echo "PostgreSQL identifiers may contain only letters, digits, underscore, and hyphen" >&2; exit 1 ;;
  esac
done

exec psql "$TENANCIT_POSTGRES_ADMIN_URL" \
  -v database_name="$TENANCIT_DATABASE_NAME" \
  -v migration_login="$TENANCIT_MIGRATION_LOGIN" \
  -v runtime_login="$TENANCIT_RUNTIME_LOGIN" \
  -v jobs_login="$TENANCIT_JOBS_LOGIN" \
  -v backup_login="$TENANCIT_BACKUP_LOGIN" \
  -f "$root_dir/deploy/postgres/configure-roles.sql"
