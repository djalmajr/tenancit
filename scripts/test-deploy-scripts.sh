#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

cat >"$tmpdir/docker" <<'EOF'
#!/bin/sh
exit 0
EOF
cat >"$tmpdir/psql" <<'EOF'
#!/bin/sh
case "$*" in
  *postgres://migration*has_schema_privilege*) printf 't\n' ;;
  *postgres://runtime*has_schema_privilege*) printf 'f\n' ;;
  *postgres://migration*current_user*) printf 'migration\n' ;;
  *postgres://runtime*current_user*) printf 'runtime\n' ;;
  *postgres://jobs*has_function_privilege*) printf 't\n' ;;
  *postgres://jobs*current_user*) printf 'jobs\n' ;;
  *"SELECT EXISTS"*) printf 't\n' ;;
  *) : ;;
esac
EOF
cat >"$tmpdir/curl" <<'EOF'
#!/bin/sh
output=''
while [ "$#" -gt 0 ]; do
  if [ "$1" = '-o' ]; then
    shift
    output="$1"
  fi
  shift
done
if [ -n "$output" ]; then
  printf '{"kind":"%s","source":"%s"}\n' "$TENANCIT_OPERATION_KIND" "$TENANCIT_OPERATION_SOURCE" >"$output"
fi
printf '201'
EOF
chmod +x "$tmpdir/docker" "$tmpdir/psql" "$tmpdir/curl"

export PATH="$tmpdir:$PATH"
export TENANCIT_IMAGE="registry.invalid/tenancit"
export TENANCIT_MIGRATION_DATABASE_URL="postgres://migration@db/tenancit?sslmode=require"
export TENANCIT_RUNTIME_DATABASE_URL="postgres://runtime@db/tenancit?sslmode=require"
export TENANCIT_JOBS_DATABASE_URL="postgres://jobs@db/tenancit?sslmode=require"
export TENANCIT_PUBLIC_BASE_URL="https://tenancit.invalid"
export TENANCIT_ADMIN_ORIGIN="https://admin.tenancit.invalid"
export TENANCIT_OIDC_ISSUER="https://id.tenancit.invalid"
export TENANCIT_VALKEY_URL="rediss://valkey.tenancit.invalid"
export OTEL_EXPORTER_OTLP_ENDPOINT="https://otel.tenancit.invalid"
export TENANCIT_SMOKE_API_TOKEN="smoke-token-long-enough"
export TENANCIT_SMOKE_HOSTNAME="smoke.tenancit.invalid"
export TENANCIT_SMOKE_TENANT_SLUG="smoke"
export TENANCIT_OPERATIONS_REPORT_TOKEN="operations-token-long-enough"
valid_digest="sha256:$(printf '%064d' 0)"

expect_failure() {
  label="$1"
  shift
  if "$@" >"$tmpdir/stdout" 2>"$tmpdir/stderr"; then
    echo "$label unexpectedly succeeded" >&2
    exit 1
  fi
}

TENANCIT_IMAGE_DIGEST="$valid_digest" "$root_dir/scripts/deploy-preflight.sh" >/dev/null
expect_failure "non-hex digest" env TENANCIT_IMAGE_DIGEST="sha256:$(printf '%063d' 0)z" "$root_dir/scripts/deploy-preflight.sh"
expect_failure "short digest" env TENANCIT_IMAGE_DIGEST="sha256:abc" "$root_dir/scripts/deploy-preflight.sh"
expect_failure "mutable tag" env TENANCIT_IMAGE_DIGEST="latest" "$root_dir/scripts/deploy-preflight.sh"
expect_failure "plain HTTP" env TENANCIT_IMAGE_DIGEST="$valid_digest" TENANCIT_PUBLIC_BASE_URL="http://tenancit.invalid" "$root_dir/scripts/deploy-preflight.sh"
expect_failure "shared database role" env TENANCIT_IMAGE_DIGEST="$valid_digest" TENANCIT_RUNTIME_DATABASE_URL="$TENANCIT_MIGRATION_DATABASE_URL" "$root_dir/scripts/deploy-preflight.sh"
expect_failure "shared jobs role" env TENANCIT_IMAGE_DIGEST="$valid_digest" TENANCIT_JOBS_DATABASE_URL="$TENANCIT_RUNTIME_DATABASE_URL" "$root_dir/scripts/deploy-preflight.sh"
expect_failure "insecure database" env TENANCIT_IMAGE_DIGEST="$valid_digest" TENANCIT_RUNTIME_DATABASE_URL="postgres://runtime@db/tenancit?sslmode=disable" "$root_dir/scripts/deploy-preflight.sh"

TENANCIT_ROLLBACK_IMAGE_DIGEST="$valid_digest" "$root_dir/scripts/deploy-rollback.sh" >/dev/null
expect_failure "invalid rollback digest" env TENANCIT_ROLLBACK_IMAGE_DIGEST="sha256:$(printf '%063d' 0)z" "$root_dir/scripts/deploy-rollback.sh"

TENANCIT_OPERATIONS_BASE_URL="https://tenancit.invalid" \
TENANCIT_OPERATIONS_REPORT_TOKEN="operations-token-long-enough" \
TENANCIT_OPERATION_KIND="backup" TENANCIT_OPERATION_SOURCE="contract-test" \
TENANCIT_OPERATION_STATUS="healthy" TENANCIT_OPERATION_IDEMPOTENCY_KEY="backup-contract-1" \
  "$root_dir/scripts/report-operation.sh" >/dev/null
expect_failure "invalid report kind" env \
  TENANCIT_OPERATIONS_BASE_URL="https://tenancit.invalid" \
  TENANCIT_OPERATIONS_REPORT_TOKEN="operations-token-long-enough" \
  TENANCIT_OPERATION_KIND="arbitrary" TENANCIT_OPERATION_SOURCE="contract-test" \
  TENANCIT_OPERATION_STATUS="healthy" TENANCIT_OPERATION_IDEMPOTENCY_KEY="bad-contract-1" \
  "$root_dir/scripts/report-operation.sh"

echo "deploy script contract tests passed"
