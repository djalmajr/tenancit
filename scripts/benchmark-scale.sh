#!/bin/sh
set -eu
umask 077

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
run_id="$(date -u +%Y%m%dt%H%M%Sz)-$$"
project="tenancit-scale-$run_id"
output_dir="${TENANCIT_SCALE_OUTPUT_DIR:-$root_dir/benchmarks/scale/results/$run_id}"
lock_dir="${TMPDIR:-/tmp}/tenancit-scale-benchmark.lock"
started_stack=false

compose() {
  docker compose -p "$project" -f "$root_dir/docker-compose.e2e.yml" "$@"
}

cleanup() {
  status=$?
  trap - EXIT INT TERM
  if [ "$started_stack" = true ]; then
    compose down -v --remove-orphans >/dev/null 2>&1 || status=1
    if [ -n "$(docker ps -aq --filter "label=com.docker.compose.project=$project")" ]; then
      echo "scale cleanup left containers for project $project" >&2
      status=1
    fi
  fi
  rmdir "$lock_dir" >/dev/null 2>&1 || status=1
  exit "$status"
}
trap cleanup EXIT INT TERM

if ! mkdir "$lock_dir" 2>/dev/null; then
  trap - EXIT INT TERM
  echo "another scale benchmark is already running ($lock_dir)" >&2
  exit 1
fi

export TENANCIT_E2E_ADMIN_TOKEN="${TENANCIT_E2E_ADMIN_TOKEN:-scale-admin-$(openssl rand -hex 16)}"
export TENANCIT_E2E_AES_KEY="${TENANCIT_E2E_AES_KEY:-$(openssl rand -base64 32)}"
export TENANCIT_E2E_DB_PASSWORD="${TENANCIT_E2E_DB_PASSWORD:-scaledb$(openssl rand -hex 16)}"

mkdir -p "$output_dir"
chmod 700 "$output_dir"
started_stack=true
compose up -d --build postgres-e2e app-e2e
port="$(compose port app-e2e 8080 | sed 's/.*://')"
base_url="http://127.0.0.1:$port"

ready=false
for _ in $(seq 1 120); do
  if curl -fsS "$base_url/healthz" >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 1
done
if [ "$ready" != true ]; then
  compose logs >&2 || true
  echo "scale stack did not become ready" >&2
  exit 1
fi

commit="$(git -C "$root_dir" rev-parse HEAD)"
dirty=0
test -z "$(git -C "$root_dir" status --porcelain)" || dirty=1
postgres_version="$(compose exec -T postgres-e2e psql -U postgres -d tenancit-e2e -Atc 'show server_version')"
chromium_version="$(/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --version 2>/dev/null || echo unknown)"

for size in 100 500 1000 5000; do
  compose exec -T postgres-e2e psql -U postgres -d tenancit-e2e -v size="$size" \
    < "$root_dir/benchmarks/scale/seed.sql" >/dev/null
  plans_dir="$output_dir/query-plans/$size"
  mkdir -p "$plans_dir"
  compose exec -T postgres-e2e psql -U postgres -d tenancit-e2e -Atc \
    'EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) SELECT * FROM tenants ORDER BY name' \
    > "$plans_dir/tenants.json"
  compose exec -T postgres-e2e psql -U postgres -d tenancit-e2e -Atc \
    'EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) SELECT * FROM api_clients ORDER BY created_at DESC' \
    > "$plans_dir/api-clients.json"
  compose exec -T postgres-e2e psql -U postgres -d tenancit-e2e -Atc \
    'EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) SELECT rd.id, count(rf.id) FROM resource_definitions rd LEFT JOIN resource_fields rf ON rf.resource_definition_id = rd.id GROUP BY rd.id ORDER BY rd.name' \
    > "$plans_dir/definitions.json"
  for run in 1 2; do
    TENANCIT_SCALE_BASE_URL="$base_url" \
    TENANCIT_SCALE_CARDINALITY="$size" \
    TENANCIT_SCALE_RUN="$run" \
    TENANCIT_SCALE_COMMIT="$commit" \
    TENANCIT_SCALE_DIRTY="$dirty" \
    TENANCIT_SCALE_POSTGRES_VERSION="$postgres_version" \
    TENANCIT_SCALE_CHROMIUM_VERSION="$chromium_version" \
      bun "$root_dir/web/scripts/benchmark-scale.mjs" "$output_dir/raw-$size-$run.json"
  done
done

TENANCIT_SCALE_OBSERVED_VOLUME="${TENANCIT_SCALE_OBSERVED_VOLUME:-0}" \
  bun "$root_dir/web/scripts/summarize-scale.mjs" "$output_dir"

echo "$output_dir"
