#!/bin/sh
set -eu

: "${TENANCIT_ADMIN_TOKEN:?TENANCIT_ADMIN_TOKEN is required}"
: "${TENANCIT_SMOKE_API_TOKEN:?TENANCIT_SMOKE_API_TOKEN is required}"

command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "jq is required" >&2; exit 1; }

base_url="${TENANCIT_BASE_URL:-http://localhost:8080}"
base_url="${base_url%/}"
run_id="$(date -u +%Y%m%d%H%M%S)-$$"
slug="smoke-${run_id}"
hostname="${slug}.invalid"
tmpdir="$(mktemp -d)"
tenant_id=""

cleanup() {
  if [ -n "$tenant_id" ]; then
    curl -sS -o /dev/null \
      -H "Authorization: Bearer ${TENANCIT_ADMIN_TOKEN}" \
      -X DELETE "${base_url}/v1/admin/tenants/${tenant_id}" || true
  fi
  rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM

expect_status() {
  got="$1"
  want="$2"
  label="$3"
  if [ "$got" != "$want" ]; then
    echo "${label}: got HTTP ${got}, want ${want}" >&2
    exit 1
  fi
}

status="$(curl -sS -o "$tmpdir/body.json" -w '%{http_code}' "${base_url}/healthz")"
expect_status "$status" 200 "healthz"
jq -e '.status == "ok"' "$tmpdir/body.json" >/dev/null

status="$(curl -sS -o /dev/null -w '%{http_code}' "${base_url}/v1/admin/overview")"
expect_status "$status" 401 "admin auth rejection"
status="$(curl -sS -o /dev/null -w '%{http_code}' "${base_url}/v1/identify?hostname=${hostname}")"
expect_status "$status" 401 "consumer auth rejection"

tenant_payload="$(jq -cn --arg slug "$slug" --arg name "Smoke ${run_id}" '{slug:$slug,name:$name}')"
status="$(curl -sS -o "$tmpdir/tenant.json" -w '%{http_code}' \
  -H "Authorization: Bearer ${TENANCIT_ADMIN_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "$tenant_payload" \
  "${base_url}/v1/admin/tenants")"
expect_status "$status" 201 "create tenant"
tenant_id="$(jq -er '.id' "$tmpdir/tenant.json")"

domain_payload="$(jq -cn --arg hostname "$hostname" '{hostname:$hostname}')"
status="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer ${TENANCIT_ADMIN_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "$domain_payload" \
  "${base_url}/v1/admin/tenants/${tenant_id}/domains")"
expect_status "$status" 201 "create tenant domain"

status="$(curl -sS -D "$tmpdir/identify.headers" -o "$tmpdir/identify.json" -w '%{http_code}' \
  -H "Authorization: Bearer ${TENANCIT_SMOKE_API_TOKEN}" \
  "${base_url}/v1/identify?hostname=${hostname}")"
expect_status "$status" 200 "identify tenant"
jq -e --arg slug "$slug" '.tenantSlug == $slug and (has("resources") | not)' "$tmpdir/identify.json" >/dev/null

status="$(curl -sS -D "$tmpdir/resolve.headers" -o "$tmpdir/resolve.json" -w '%{http_code}' \
  -H "Authorization: Bearer ${TENANCIT_SMOKE_API_TOKEN}" \
  "${base_url}/v1/resolve?tenantId=${slug}")"
expect_status "$status" 200 "resolve tenant"
jq -e --arg slug "$slug" '.tenantSlug == $slug' "$tmpdir/resolve.json" >/dev/null
grep -Eiq '^cache-control: private, no-store\r?$' "$tmpdir/resolve.headers"
grep -Eiq '^x-content-type-options: nosniff\r?$' "$tmpdir/resolve.headers"
etag="$(grep -i '^etag:' "$tmpdir/resolve.headers" | head -n 1 | cut -d: -f2- | sed 's/^[[:space:]]*//; s/\r$//')"
[ -n "$etag" ] || { echo "resolve response did not include ETag" >&2; exit 1; }

status="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer ${TENANCIT_SMOKE_API_TOKEN}" \
  -H "If-None-Match: ${etag}" \
  "${base_url}/v1/resolve?tenantId=${slug}")"
expect_status "$status" 304 "conditional resolve"

status="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer ${TENANCIT_ADMIN_TOKEN}" \
  -X DELETE "${base_url}/v1/admin/tenants/${tenant_id}")"
expect_status "$status" 204 "cleanup tenant"
deleted_id="$tenant_id"
tenant_id=""

status="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer ${TENANCIT_ADMIN_TOKEN}" \
  "${base_url}/v1/admin/tenants/${deleted_id}")"
expect_status "$status" 404 "verify cleanup"

echo "smoke ok: health, auth boundaries, create, identify, resolve, ETag, cleanup"
