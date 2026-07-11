#!/bin/sh
set -eu
umask 077

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
project="${TENANCIT_OIDC_E2E_PROJECT:-tenancit-oidc-e2e-$$}"
server_pid=""
server_log="$root_dir/output/oidc-e2e-server-$$.log"
output_dir="$root_dir/output/playwright/oidc-$$"

export TENANCIT_OIDC_E2E_DB_PASSWORD="${TENANCIT_OIDC_E2E_DB_PASSWORD:-oidcdb$(openssl rand -hex 16)}"
export TENANCIT_E2E_ADMIN_TOKEN="${TENANCIT_E2E_ADMIN_TOKEN:-$(openssl rand -base64 36 | tr -d '\n')}"
export TENANCIT_E2E_AES_KEY="${TENANCIT_E2E_AES_KEY:-$(openssl rand -base64 32)}"

compose() {
  docker compose -p "$project" -f "$root_dir/docker-compose.oidc-e2e.yml" "$@"
}

cleanup() {
  status=$?
  trap - EXIT INT TERM
  if [ -n "$server_pid" ]; then
    kill "$server_pid" >/dev/null 2>&1 || true
    wait "$server_pid" >/dev/null 2>&1 || true
  fi
  if [ "$status" -ne 0 ]; then
    compose logs dex-oidc-e2e >&2 || true
  fi
  compose down -v --remove-orphans >/dev/null 2>&1 || status=1
  if [ -d "$output_dir" ]; then
    find "$output_dir" -type f -name 'error-context.md' -delete
  fi
  if [ "$status" -ne 0 ] && [ -f "$server_log" ]; then
    sed -E 's/(token|secret|password)=[^ ]+/\1=[REDACTED]/gi' "$server_log" >&2 || true
  fi
  rm -f "$server_log"
  exit "$status"
}
trap cleanup EXIT INT TERM

mkdir -p "$root_dir/output"
compose up -d

ready=false
for _ in $(seq 1 120); do
  if curl -fsS "http://127.0.0.1:15556/dex/.well-known/openid-configuration" >/dev/null 2>&1 && \
     docker compose -p "$project" -f "$root_dir/docker-compose.oidc-e2e.yml" exec -T postgres-oidc-e2e \
       pg_isready -U postgres -d tenancit-oidc-e2e >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 1
done
if [ "$ready" != true ]; then
  compose logs >&2 || true
  echo "OIDC E2E infrastructure did not become ready" >&2
  exit 1
fi

make -C "$root_dir" build
TENANCIT_DATABASE_URL="postgres://postgres:${TENANCIT_OIDC_E2E_DB_PASSWORD}@127.0.0.1:15433/tenancit-oidc-e2e?sslmode=disable" \
  "$root_dir/server/bin/migrate"
TENANCIT_ADDR="127.0.0.1:18081" \
TENANCIT_DATABASE_URL="postgres://postgres:${TENANCIT_OIDC_E2E_DB_PASSWORD}@127.0.0.1:15433/tenancit-oidc-e2e?sslmode=disable" \
TENANCIT_ADMIN_AUTH_MODE="oidc" \
TENANCIT_DEV_MODE="true" \
TENANCIT_ADMIN_ORIGIN="http://127.0.0.1:18081" \
TENANCIT_OIDC_ISSUER="http://127.0.0.1:15556/dex" \
TENANCIT_OIDC_CLIENT_ID="tenancit" \
TENANCIT_OIDC_CLIENT_SECRET="tenancit-oidc-e2e-secret" \
TENANCIT_OIDC_ROLE_CLAIM="groups" \
TENANCIT_OIDC_ROLE_MAPPINGS='{"authors":"security_admin"}' \
TENANCIT_BREAK_GLASS_ENABLED="true" \
TENANCIT_BREAK_GLASS_VERSION="e2e-current" \
TENANCIT_ADMIN_TOKEN="$TENANCIT_E2E_ADMIN_TOKEN" \
TENANCIT_VALKEY_URL="redis://127.0.0.1:16380/0" \
TENANCIT_AES_KEY="$TENANCIT_E2E_AES_KEY" \
TENANCIT_AES_KEY_VERSION="1" \
  "$root_dir/server/bin/server" >"$server_log" 2>&1 &
server_pid=$!

ready=false
for _ in $(seq 1 120); do
  if curl -fsS "http://127.0.0.1:18081/healthz" >/dev/null 2>&1; then
    ready=true
    break
  fi
  if ! kill -0 "$server_pid" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
if [ "$ready" != true ]; then
  sed -E 's/(token|secret|password)=[^ ]+/\1=[REDACTED]/gi' "$server_log" >&2 || true
  echo "OIDC E2E application did not become ready" >&2
  exit 1
fi

cd "$root_dir/web"
TENANCIT_E2E_AUTH_MODE=oidc \
TENANCIT_E2E_BASE_URL="http://127.0.0.1:18081" \
TENANCIT_E2E_RETRIES=0 \
TENANCIT_E2E_OUTPUT_DIR="$output_dir" \
  bunx playwright test --project=chromium --no-deps 'oidc-.*\.e2e\.test\.ts'
