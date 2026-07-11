#!/bin/sh
set -eu
umask 077

: "${TENANCIT_BASE_URL:?set the deployed HTTPS base URL}"
: "${TENANCIT_SMOKE_API_TOKEN:?set a dedicated tenant:identify smoke token}"
: "${TENANCIT_SMOKE_HOSTNAME:?set a pre-provisioned smoke hostname}"
: "${TENANCIT_SMOKE_TENANT_SLUG:?set the expected smoke tenant slug}"
command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "jq is required" >&2; exit 1; }

base_url="${TENANCIT_BASE_URL%/}"
tmpdir="$(mktemp -d)"
headers="$tmpdir/consumer.headers"
chmod 700 "$tmpdir"
printf 'Authorization: Bearer %s\n' "$TENANCIT_SMOKE_API_TOKEN" >"$headers"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

expect_status() {
  got="$1"; want="$2"; label="$3"
  [ "$got" = "$want" ] || { echo "$label: got HTTP $got, want $want" >&2; exit 1; }
}

status="$(curl -sS -o "$tmpdir/health.json" -w '%{http_code}' "$base_url/healthz")"
expect_status "$status" 200 healthz
jq -e '.status == "ok"' "$tmpdir/health.json" >/dev/null
status="$(curl -sS -o "$tmpdir/ready.json" -w '%{http_code}' "$base_url/readyz")"
expect_status "$status" 200 readyz
jq -e '.status == "healthy"' "$tmpdir/ready.json" >/dev/null

status="$(curl -sS -o "$tmpdir/auth.json" -w '%{http_code}' "$base_url/v1/auth/config")"
expect_status "$status" 200 'OIDC config'
jq -e '.mode == "oidc" and .login_url == "/v1/auth/login"' "$tmpdir/auth.json" >/dev/null
status="$(curl -sS -D "$tmpdir/login.headers" -o /dev/null -w '%{http_code}' "$base_url/v1/auth/login")"
expect_status "$status" 302 'OIDC login start'
grep -Eiq '^location: https://[^[:space:]]+' "$tmpdir/login.headers"

status="$(curl -sS -o /dev/null -w '%{http_code}' "$base_url/v1/admin/overview")"
expect_status "$status" 401 'admin auth rejection'
status="$(curl -sS -o /dev/null -w '%{http_code}' --get --data-urlencode "hostname=$TENANCIT_SMOKE_HOSTNAME" "$base_url/v1/identify")"
expect_status "$status" 401 'consumer auth rejection'
status="$(curl -sS -o "$tmpdir/identify.json" -w '%{http_code}' -H "@$headers" \
  --get --data-urlencode "hostname=$TENANCIT_SMOKE_HOSTNAME" "$base_url/v1/identify")"
expect_status "$status" 200 'smoke tenant identify'
jq -e --arg slug "$TENANCIT_SMOKE_TENANT_SLUG" '.tenantSlug == $slug and (has("resources") | not)' "$tmpdir/identify.json" >/dev/null

echo "production smoke ok: liveness, readiness, OIDC, auth boundaries, least-privilege identify"
