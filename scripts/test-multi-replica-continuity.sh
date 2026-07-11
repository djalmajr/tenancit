#!/bin/sh
set -eu
umask 077

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
run_id="$(date +%s)-$$"
project="${TENANCIT_CONTINUITY_PROJECT:-tenancit-continuity-$run_id}"
started=false

export TENANCIT_E2E_ADMIN_TOKEN="${TENANCIT_E2E_ADMIN_TOKEN:-continuity-admin-$(openssl rand -hex 16)}"
export TENANCIT_E2E_AES_KEY="${TENANCIT_E2E_AES_KEY:-$(openssl rand -base64 32)}"
export TENANCIT_E2E_DB_PASSWORD="${TENANCIT_E2E_DB_PASSWORD:-continuitydb$(openssl rand -hex 16)}"
export TENANCIT_E2E_OPERATIONS_TOKEN="${TENANCIT_E2E_OPERATIONS_TOKEN:-continuity-operations-$(openssl rand -hex 16)}"

compose() {
  docker compose -p "$project" -f "$root_dir/docker-compose.e2e.yml" "$@"
}

cleanup() {
  status=$?
  trap - EXIT INT TERM
  if [ "$started" = true ]; then
    compose down -v --remove-orphans >/dev/null 2>&1 || status=1
  fi
  exit "$status"
}
trap cleanup EXIT INT TERM

wait_ready() {
  url="$1"
  for _ in $(seq 1 120); do
    if curl -fsS "$url/readyz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  compose logs app-e2e postgres-e2e valkey-e2e >&2 || true
  echo "replica did not become ready: $url" >&2
  return 1
}

started=true
compose up -d --build --scale app-e2e=2 app-e2e
port_one="$(compose port --index 1 app-e2e 8080 | sed 's/.*://')"
port_two="$(compose port --index 2 app-e2e 8080 | sed 's/.*://')"
base_one="http://127.0.0.1:$port_one"
base_two="http://127.0.0.1:$port_two"
wait_ready "$base_one"
wait_ready "$base_two"

if date -u -d '+30 days' +%Y-%m-%dT%H:%M:%SZ >/dev/null 2>&1; then
  expires_at="$(date -u -d '+30 days' +%Y-%m-%dT%H:%M:%SZ)"
else
  expires_at="$(date -u -v+30d +%Y-%m-%dT%H:%M:%SZ)"
fi
payload="$(jq -cn --arg name "continuity-$run_id" --arg expires "$expires_at" \
  '{name:$name,scopes:["tenant:identify"],rpm_limit:1,expires_at:$expires}')"
idempotency_key="$(openssl rand -hex 16 | sed -E 's/^(.{8})(.{4})(.{4})(.{4})(.{12})$/\1-\2-\3-\4-\5/')"
created="$(curl -fsS \
  -H "Authorization: Bearer $TENANCIT_E2E_ADMIN_TOKEN" \
  -H "Idempotency-Key: $idempotency_key" \
  -H 'Content-Type: application/json' -d "$payload" \
  "$base_one/v1/admin/api-clients")"
client_id="$(printf '%s' "$created" | jq -er '.client.id')"
token="$(printf '%s' "$created" | jq -er '.token')"

first_status="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $token" "$base_one/v1/identify?hostname=continuity.invalid")"
second_status="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $token" "$base_two/v1/identify?hostname=continuity.invalid")"
[ "$first_status" = 404 ] || { echo "first replica did not consume shared bucket: HTTP $first_status" >&2; exit 1; }
[ "$second_status" = 429 ] || { echo "second replica bypassed shared bucket: HTTP $second_status" >&2; exit 1; }

curl -fsS -o /dev/null \
  -H "Authorization: Bearer $TENANCIT_E2E_ADMIN_TOKEN" -X POST \
  "$base_one/v1/admin/api-clients/$client_id/revoke"
revoked_status="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $token" "$base_two/v1/identify?hostname=continuity.invalid")"
[ "$revoked_status" = 401 ] || { echo "revocation was not immediate on replica two: HTTP $revoked_status" >&2; exit 1; }

replica_one_id="$(compose ps -q app-e2e | sed -n '1p')"
docker stop "$replica_one_id" >/dev/null
curl -fsS -H "Authorization: Bearer $TENANCIT_E2E_ADMIN_TOKEN" "$base_two/v1/admin/overview" >/dev/null
curl -fsS -o /dev/null \
  -H "Authorization: Bearer $TENANCIT_E2E_ADMIN_TOKEN" -X DELETE \
  "$base_two/v1/admin/api-clients/$client_id"

echo "multi-replica continuity passed: shared limiter, immediate revocation, one-replica failover"
