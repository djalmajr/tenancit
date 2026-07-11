#!/bin/sh
set -eu
umask 077

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
run_id="$(date +%s)-$$"
project="${TENANCIT_E2E_PROJECT:-tenancit-e2e-${run_id}}"
product_base_url="${TENANCIT_E2E_BASE_URL:-}"
vite_base_url="${TENANCIT_E2E_VITE_BASE_URL:-}"
started_stack=false
ephemeral_stack=false
output_dir="$root_dir/output/playwright/$run_id"

export TENANCIT_E2E_ADMIN_TOKEN="${TENANCIT_E2E_ADMIN_TOKEN:-e2e-admin-$(openssl rand -hex 16)}"
export TENANCIT_E2E_AES_KEY="${TENANCIT_E2E_AES_KEY:-$(openssl rand -base64 32)}"
export TENANCIT_E2E_DB_PASSWORD="${TENANCIT_E2E_DB_PASSWORD:-e2edb$(openssl rand -hex 16)}"

compose() {
  docker compose -p "$project" -f "$root_dir/docker-compose.e2e.yml" "$@"
}

cleanup() {
  status=$?
  trap - EXIT INT TERM
  # Playwright matcher failures can retain the full page snapshot in this
  # attachment even when trace and screenshot capture are disabled.
  if [ -d "$output_dir" ]; then
    find "$output_dir" -type f -name 'error-context.md' -delete
  fi
  if [ "$started_stack" = true ]; then
    compose down -v --remove-orphans >/dev/null 2>&1 || status=1
    if [ -n "$(docker ps -aq --filter "label=com.docker.compose.project=$project")" ]; then
      echo "E2E cleanup left containers for project $project" >&2
      status=1
    fi
    if [ -n "$(docker volume ls -q --filter "label=com.docker.compose.project=$project")" ]; then
      echo "E2E cleanup left volumes for project $project" >&2
      status=1
    fi
    if [ -n "$(docker network ls -q --filter "label=com.docker.compose.project=$project")" ]; then
      echo "E2E cleanup left networks for project $project" >&2
      status=1
    fi
  fi
  exit "$status"
}
trap cleanup EXIT INT TERM

wait_until_ready() {
  url="$1"
  label="$2"
  ready=false
  for _ in $(seq 1 120); do
    if curl -fsS "$url/healthz" >/dev/null 2>&1; then
      ready=true
      break
    fi
    sleep 1
  done
  if [ "$ready" != true ]; then
    compose logs app-e2e web-e2e postgres-e2e >&2 || true
    echo "$label did not become ready at $url" >&2
    exit 1
  fi
}

if [ "${TENANCIT_E2E_EXTERNAL:-0}" != "1" ]; then
  started_stack=true
  ephemeral_stack=true
  compose up -d --build
  product_port="$(compose port app-e2e 8080 | sed 's/.*://')"
  vite_port="$(compose port web-e2e 5180 | sed 's/.*://')"
  product_base_url="http://127.0.0.1:${product_port}"
  vite_base_url="http://127.0.0.1:${vite_port}"
  wait_until_ready "$product_base_url" "packaged product"
  wait_until_ready "$vite_base_url" "Vite proxy"
else
  if [ "${TENANCIT_E2E_EXTERNAL_MUTATIONS_ACK:-0}" != "1" ]; then
    echo "External E2E requires TENANCIT_E2E_EXTERNAL_MUTATIONS_ACK=1 and a dedicated empty disposable database." >&2
    exit 1
  fi
  if [ -z "$product_base_url" ]; then
    echo "TENANCIT_E2E_BASE_URL is required with TENANCIT_E2E_EXTERNAL=1" >&2
    exit 1
  fi
  if [ "${TENANCIT_E2E_SKIP_VITE:-0}" != "1" ] && [ -z "$vite_base_url" ]; then
    echo "TENANCIT_E2E_VITE_BASE_URL is required unless TENANCIT_E2E_SKIP_VITE=1" >&2
    exit 1
  fi
fi

cd "$root_dir/web"
rm -rf "$root_dir/output/playwright/test-results" "$output_dir"
mkdir -p "$output_dir"
chmod 700 "$root_dir/output/playwright" "$output_dir"
export PLAYWRIGHT_NO_COPY_PROMPT=1
export TENANCIT_E2E_EPHEMERAL="$ephemeral_stack"
export TENANCIT_E2E_OUTPUT_DIR="$output_dir"
TENANCIT_E2E_BASE_URL="$product_base_url" bunx playwright test "$@"

if [ "${TENANCIT_E2E_SKIP_VITE:-0}" != "1" ]; then
  TENANCIT_E2E_BASE_URL="$vite_base_url" bunx playwright test \
    --project=chromium --no-deps route-smoke.e2e.test.ts
fi
