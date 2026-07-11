#!/bin/sh
set -eu
umask 077

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)"
: "${TENANCIT_BACKUP_DATABASE_URL:?set the read-only backup DSN}"
: "${TENANCIT_BACKUP_DIR:?set an absolute off-checkout backup directory}"
: "${TENANCIT_BACKUP_SOURCE:?set a stable backup source identifier}"

command -v pg_dump >/dev/null 2>&1 || { echo "pg_dump is required" >&2; exit 1; }
command -v pg_restore >/dev/null 2>&1 || { echo "pg_restore is required" >&2; exit 1; }
case "$TENANCIT_BACKUP_DIR" in /*) ;; *) echo "TENANCIT_BACKUP_DIR must be absolute" >&2; exit 1 ;; esac
mkdir -p "$TENANCIT_BACKUP_DIR"
backup_dir="$(CDPATH= cd -- "$TENANCIT_BACKUP_DIR" && pwd -P)"
case "$backup_dir/" in "$root_dir/"*) echo "backup directory must be outside the checkout" >&2; exit 1 ;; esac

run_id="$(date -u +%Y%m%dT%H%M%SZ)-$$"
dump="$backup_dir/tenancit-$run_id.dump"
keep_dump=false
cleanup() {
  if [ "$keep_dump" != true ]; then
    rm -f "$dump"
  fi
}
trap cleanup EXIT INT TERM
pg_dump --dbname="$TENANCIT_BACKUP_DATABASE_URL" --format=custom --file="$dump"
pg_restore --list "$dump" >/dev/null
size_bytes="$(wc -c <"$dump" | tr -d ' ')"
[ "$size_bytes" -gt 0 ] || { echo "backup is empty" >&2; exit 1; }
if command -v shasum >/dev/null 2>&1; then
  checksum="$(shasum -a 256 "$dump" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  checksum="$(sha256sum "$dump" | awk '{print $1}')"
else
  echo "shasum or sha256sum is required" >&2
  exit 1
fi
[ -n "$checksum" ] || { echo "backup checksum is empty" >&2; exit 1; }
keep_dump=true

if [ -n "${TENANCIT_OPERATIONS_BASE_URL:-}" ]; then
  TENANCIT_OPERATION_KIND=backup \
  TENANCIT_OPERATION_SOURCE="$TENANCIT_BACKUP_SOURCE" \
  TENANCIT_OPERATION_STATUS=healthy \
  TENANCIT_OPERATION_IDEMPOTENCY_KEY="backup-$TENANCIT_BACKUP_SOURCE-$run_id" \
  TENANCIT_OPERATION_DETAILS_JSON="$(printf '{\"size_bytes\":%s,\"sha256\":\"%s\"}' "$size_bytes" "$checksum")" \
    "$root_dir/scripts/report-operation.sh"
fi

echo "validated backup: $dump"
