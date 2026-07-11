#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)"
: "${TENANCIT_BACKUP_FILE:?set the absolute dump path}"
: "${TENANCIT_RESTORE_DATABASE_URL:?set the isolated empty restore DSN}"
: "${TENANCIT_RESTORE_SOURCE:?set a stable restore source identifier}"

command -v pg_restore >/dev/null 2>&1 || { echo "pg_restore is required" >&2; exit 1; }
command -v psql >/dev/null 2>&1 || { echo "psql is required" >&2; exit 1; }
case "$TENANCIT_BACKUP_FILE" in /*) ;; *) echo "TENANCIT_BACKUP_FILE must be absolute" >&2; exit 1 ;; esac
test -f "$TENANCIT_BACKUP_FILE"
dump_dir="$(CDPATH= cd -- "$(dirname -- "$TENANCIT_BACKUP_FILE")" && pwd -P)"
case "$dump_dir/" in "$root_dir/"*) echo "backup file must be outside the checkout" >&2; exit 1 ;; esac

pg_restore --list "$TENANCIT_BACKUP_FILE" >/dev/null
if psql "$TENANCIT_RESTORE_DATABASE_URL" -Atqc "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public')" | grep -qx t; then
  echo "restore target is not empty" >&2
  exit 1
fi
pg_restore --dbname="$TENANCIT_RESTORE_DATABASE_URL" --no-owner --no-privileges "$TENANCIT_BACKUP_FILE"
table_count="$(psql "$TENANCIT_RESTORE_DATABASE_URL" -Atqc "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'")"
tenant_count="$(psql "$TENANCIT_RESTORE_DATABASE_URL" -Atqc "SELECT count(*) FROM tenants")"
[ "$table_count" -gt 0 ] || { echo "restore contains no public tables" >&2; exit 1; }

if [ -n "${TENANCIT_OPERATIONS_BASE_URL:-}" ]; then
  run_id="$(date -u +%Y%m%dT%H%M%SZ)-$$"
  TENANCIT_OPERATION_KIND=restore \
  TENANCIT_OPERATION_SOURCE="$TENANCIT_RESTORE_SOURCE" \
  TENANCIT_OPERATION_STATUS=healthy \
  TENANCIT_OPERATION_IDEMPOTENCY_KEY="restore-$TENANCIT_RESTORE_SOURCE-$run_id" \
  TENANCIT_OPERATION_DETAILS_JSON="$(printf '{\"table_count\":%s,\"tenant_count\":%s}' "$table_count" "$tenant_count")" \
    "$root_dir/scripts/report-operation.sh"
fi

echo "restore drill passed: tables=$table_count tenants=$tenant_count"
