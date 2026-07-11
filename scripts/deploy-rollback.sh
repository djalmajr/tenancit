#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
compose_file="${TENANCIT_DEPLOY_COMPOSE_FILE:-$root_dir/deploy/docker-compose.production.yml}"
: "${TENANCIT_ROLLBACK_IMAGE_DIGEST:?set the previously validated digest}"
rollback_hex="${TENANCIT_ROLLBACK_IMAGE_DIGEST#sha256:}"
case "$TENANCIT_ROLLBACK_IMAGE_DIGEST" in
  sha256:*) ;;
  *) echo "rollback digest must be sha256" >&2; exit 1 ;;
esac
case "$rollback_hex" in
  ''|*[!0-9a-f]*) echo "rollback digest must contain only lowercase hexadecimal characters" >&2; exit 1 ;;
esac
if [ "${#rollback_hex}" -ne 64 ]; then
  echo "rollback digest must contain exactly 64 hexadecimal characters" >&2
  exit 1
fi
TENANCIT_IMAGE_DIGEST="$TENANCIT_ROLLBACK_IMAGE_DIGEST" \
  docker compose -f "$compose_file" up -d --no-deps --scale app=2 app
echo "rollback requested with DSN/schema preserved: $TENANCIT_ROLLBACK_IMAGE_DIGEST"
