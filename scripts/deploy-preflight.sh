#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
compose_file="${TENANCIT_DEPLOY_COMPOSE_FILE:-$root_dir/deploy/docker-compose.production.yml}"
: "${TENANCIT_IMAGE:?set immutable image repository}"
: "${TENANCIT_IMAGE_DIGEST:?set immutable image digest}"
: "${TENANCIT_MIGRATION_DATABASE_URL:?set migration DSN}"
: "${TENANCIT_RUNTIME_DATABASE_URL:?set runtime DSN}"
: "${TENANCIT_JOBS_DATABASE_URL:?set dedicated jobs DSN}"
: "${TENANCIT_PUBLIC_BASE_URL:?set public HTTPS base URL}"
: "${TENANCIT_ADMIN_ORIGIN:?set public HTTPS admin origin}"
: "${TENANCIT_OIDC_ISSUER:?set HTTPS OIDC issuer}"
: "${TENANCIT_VALKEY_URL:?set TLS/authenticated Valkey URL}"
: "${OTEL_EXPORTER_OTLP_ENDPOINT:?set HTTPS OTLP endpoint}"
: "${TENANCIT_SMOKE_API_TOKEN:?set dedicated tenant:identify smoke token}"
: "${TENANCIT_SMOKE_HOSTNAME:?set smoke hostname}"
: "${TENANCIT_SMOKE_TENANT_SLUG:?set expected smoke tenant slug}"
: "${TENANCIT_OPERATIONS_REPORT_TOKEN:?set dedicated operations report token}"

digest_hex="${TENANCIT_IMAGE_DIGEST#sha256:}"
case "$TENANCIT_IMAGE_DIGEST" in
  sha256:*) ;;
  *) echo "TENANCIT_IMAGE_DIGEST must be a sha256 digest" >&2; exit 1 ;;
esac
case "$digest_hex" in
  ''|*[!0-9a-f]*) echo "TENANCIT_IMAGE_DIGEST must contain only lowercase hexadecimal characters" >&2; exit 1 ;;
esac
if [ "${#digest_hex}" -ne 64 ]; then
  echo "TENANCIT_IMAGE_DIGEST must contain exactly 64 hexadecimal characters" >&2
  exit 1
fi
case "$TENANCIT_PUBLIC_BASE_URL" in
  https://*) ;;
  *) echo "TENANCIT_PUBLIC_BASE_URL must use HTTPS" >&2; exit 1 ;;
esac
for secure_url in "$TENANCIT_ADMIN_ORIGIN" "$TENANCIT_OIDC_ISSUER" "$OTEL_EXPORTER_OTLP_ENDPOINT"; do
  case "$secure_url" in https://*) ;; *) echo "admin origin, OIDC issuer, and OTLP endpoint must use HTTPS" >&2; exit 1 ;; esac
done
case "$TENANCIT_VALKEY_URL" in rediss://*) ;; *) echo "production Valkey URL must use rediss://" >&2; exit 1 ;; esac
if [ "$TENANCIT_MIGRATION_DATABASE_URL" = "$TENANCIT_RUNTIME_DATABASE_URL" ] ||
   [ "$TENANCIT_MIGRATION_DATABASE_URL" = "$TENANCIT_JOBS_DATABASE_URL" ] ||
   [ "$TENANCIT_RUNTIME_DATABASE_URL" = "$TENANCIT_JOBS_DATABASE_URL" ]; then
  echo "migration, runtime and jobs DSNs must use distinct roles" >&2
  exit 1
fi
for database_url in "$TENANCIT_MIGRATION_DATABASE_URL" "$TENANCIT_RUNTIME_DATABASE_URL" "$TENANCIT_JOBS_DATABASE_URL"; do
  case "$database_url" in
    *sslmode=require*|*sslmode=verify-ca*|*sslmode=verify-full*) ;;
    *) echo "production PostgreSQL DSNs must require TLS" >&2; exit 1 ;;
  esac
done

docker manifest inspect "$TENANCIT_IMAGE@$TENANCIT_IMAGE_DIGEST" >/dev/null
docker compose -f "$compose_file" config >/dev/null

migration_user="$(psql "$TENANCIT_MIGRATION_DATABASE_URL" -Atqc 'SELECT current_user')"
runtime_user="$(psql "$TENANCIT_RUNTIME_DATABASE_URL" -Atqc 'SELECT current_user')"
jobs_user="$(psql "$TENANCIT_JOBS_DATABASE_URL" -Atqc 'SELECT current_user')"
if [ -z "$migration_user" ] || [ -z "$runtime_user" ] || [ -z "$jobs_user" ] ||
   [ "$migration_user" = "$runtime_user" ] || [ "$migration_user" = "$jobs_user" ] || [ "$runtime_user" = "$jobs_user" ]; then
  echo "migration, runtime and jobs DSNs must authenticate as distinct roles" >&2
  exit 1
fi
migration_create="$(psql "$TENANCIT_MIGRATION_DATABASE_URL" -Atqc "SELECT has_schema_privilege(current_user,'public','CREATE')")"
if [ "$migration_create" != "t" ]; then
  echo "migration role cannot create schema objects" >&2
  exit 1
fi
runtime_create="$(psql "$TENANCIT_RUNTIME_DATABASE_URL" -Atqc "SELECT has_schema_privilege(current_user,'public','CREATE')")"
if [ "$runtime_create" != "f" ]; then
  echo "runtime role can create schema objects" >&2
  exit 1
fi
psql "$TENANCIT_RUNTIME_DATABASE_URL" -v ON_ERROR_STOP=1 -Atqc "SELECT 1 FROM tenants LIMIT 1" >/dev/null
jobs_maintenance="$(psql "$TENANCIT_JOBS_DATABASE_URL" -Atqc "SELECT has_function_privilege(current_user,'maintain_admin_audit_partitions(timestamptz,integer,integer)','EXECUTE')")"
if [ "$jobs_maintenance" != "t" ]; then
  echo "jobs role cannot execute bounded audit maintenance" >&2
  exit 1
fi
backup_fresh="$(psql "$TENANCIT_RUNTIME_DATABASE_URL" -Atqc "SELECT EXISTS(SELECT 1 FROM operational_reports WHERE kind='backup' AND status='healthy' AND fresh_until>clock_timestamp())")"
if [ "$backup_fresh" != "t" ]; then
  echo "no fresh healthy backup report" >&2
  exit 1
fi

echo "preflight ok: immutable image, distinct DB roles, runtime no-DDL, jobs maintenance, fresh backup"
