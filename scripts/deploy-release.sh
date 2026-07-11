#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
compose_file="${TENANCIT_DEPLOY_COMPOSE_FILE:-$root_dir/deploy/docker-compose.production.yml}"
"$root_dir/scripts/deploy-preflight.sh"
docker compose -f "$compose_file" run --rm migrate
if ! curl -fsS "$TENANCIT_PUBLIC_BASE_URL/readyz" >/dev/null; then
  echo "previous release is not compatible with the expanded schema; rollout stopped before replacing replicas" >&2
  exit 1
fi
docker compose -f "$compose_file" up -d --no-deps --scale app=2 app

ready=false
for _ in $(seq 1 120); do
  if curl -fsS "$TENANCIT_PUBLIC_BASE_URL/readyz" >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 1
done
if [ "$ready" != true ]; then
  echo "rollout did not become ready; invoke rollback with the previous digest" >&2
  exit 1
fi
TENANCIT_BASE_URL="$TENANCIT_PUBLIC_BASE_URL" "$root_dir/scripts/post-deploy-production-smoke.sh"
TENANCIT_OPERATIONS_BASE_URL="$TENANCIT_PUBLIC_BASE_URL" \
TENANCIT_OPERATION_KIND=migration \
TENANCIT_OPERATION_SOURCE="deploy-release" \
TENANCIT_OPERATION_STATUS=healthy \
TENANCIT_OPERATION_IDEMPOTENCY_KEY="migration-$TENANCIT_IMAGE_DIGEST" \
TENANCIT_OPERATION_DETAILS_JSON="$(printf '{\"image_digest\":\"%s\"}' "$TENANCIT_IMAGE_DIGEST")" \
  "$root_dir/scripts/report-operation.sh"
echo "deploy ok: $TENANCIT_IMAGE@$TENANCIT_IMAGE_DIGEST"
