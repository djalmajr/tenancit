#!/bin/sh
set -eu
umask 077

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

command -v docker >/dev/null 2>&1 || { echo "docker compose is required" >&2; exit 1; }
cat >"$tmpdir/rewrap.env" <<'EOF'
TENANCIT_REWRAP_DATABASE_URL=postgres://rewrap:secret@db/tenancit?sslmode=require
TENANCIT_AES_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
TENANCIT_AES_KEY_VERSION=2
TENANCIT_AES_KEY_V1=ERERERERERERERERERERERERERERERERERERERERERE=
TENANCIT_OPERATIONS_BASE_URL=https://tenancit.invalid
TENANCIT_OPERATIONS_REPORT_TOKEN=operations-token-long-enough-for-contract
TENANCIT_REWRAP_SOURCE=compose-contract
OTEL_EXPORTER_OTLP_ENDPOINT=https://otel.invalid
EOF

export TENANCIT_IMAGE=registry.invalid/tenancit
export TENANCIT_IMAGE_DIGEST=sha256:0000000000000000000000000000000000000000000000000000000000000000
export TENANCIT_NETWORK=ingress
export TENANCIT_REWRAP_TARGET_VERSION=2
export TENANCIT_REWRAP_JOB_ID=00000000-0000-4000-8000-000000000007
export TENANCIT_REWRAP_ENV_FILE="$tmpdir/rewrap.env"
docker compose -f "$root_dir/deploy/docker-compose.rewrap.yml" config >"$tmpdir/rewrap.yml"
grep -q 'image: registry.invalid/tenancit@sha256:' "$tmpdir/rewrap.yml"
grep -q -- '--dry-run' "$tmpdir/rewrap.yml"
grep -q -- '--confirm-write' "$tmpdir/rewrap.yml"
grep -q '/tenancit-rewrap' "$tmpdir/rewrap.yml"

echo "Compose deployment contracts passed"
