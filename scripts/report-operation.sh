#!/bin/sh
set -eu

: "${TENANCIT_OPERATIONS_BASE_URL:?set the Tenancit operations base URL}"
: "${TENANCIT_OPERATIONS_REPORT_TOKEN:?set the dedicated operations report token}"
: "${TENANCIT_OPERATION_KIND:?set backup, restore, migration, or rewrap}"
: "${TENANCIT_OPERATION_SOURCE:?set a stable non-secret source identifier}"
: "${TENANCIT_OPERATION_STATUS:?set healthy or failed}"
: "${TENANCIT_OPERATION_IDEMPOTENCY_KEY:?set a stable idempotency key}"

command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "jq is required" >&2; exit 1; }

case "$TENANCIT_OPERATION_KIND" in backup|restore|migration|rewrap) ;; *) echo "invalid operation kind" >&2; exit 1 ;; esac
case "$TENANCIT_OPERATION_STATUS" in healthy|failed) ;; *) echo "invalid operation status" >&2; exit 1 ;; esac
case "${TENANCIT_OPERATION_FRESH_FOR_SECONDS:-3600}" in ''|*[!0-9]*) echo "freshness must be an integer" >&2; exit 1 ;; esac

occurred_at="${TENANCIT_OPERATION_OCCURRED_AT:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
details="${TENANCIT_OPERATION_DETAILS_JSON:-}"
if [ -z "$details" ]; then
  details='{}'
fi
jq -e 'type == "object"' >/dev/null <<EOF
$details
EOF

payload="$(jq -cn \
  --arg kind "$TENANCIT_OPERATION_KIND" \
  --arg source "$TENANCIT_OPERATION_SOURCE" \
  --arg status "$TENANCIT_OPERATION_STATUS" \
  --arg occurred_at "$occurred_at" \
  --argjson fresh_for_seconds "${TENANCIT_OPERATION_FRESH_FOR_SECONDS:-3600}" \
  --argjson details "$details" \
  '{kind:$kind,source:$source,status:$status,occurred_at:$occurred_at,fresh_for_seconds:$fresh_for_seconds,details:$details}')"

tmp="$(mktemp)"
headers="$(mktemp)"
chmod 600 "$tmp" "$headers"
printf 'Authorization: Bearer %s\n' "$TENANCIT_OPERATIONS_REPORT_TOKEN" >"$headers"
trap 'rm -f "$tmp" "$headers"' EXIT INT TERM
status="$(curl -sS -o "$tmp" -w '%{http_code}' \
  -H "@$headers" \
  -H "Idempotency-Key: ${TENANCIT_OPERATION_IDEMPOTENCY_KEY}" \
  -H 'Content-Type: application/json' \
  -d "$payload" \
  "${TENANCIT_OPERATIONS_BASE_URL%/}/v1/operations/reports")"
case "$status" in
  200|201) ;;
  *) echo "operation report rejected with HTTP $status" >&2; exit 1 ;;
esac
jq -e --arg kind "$TENANCIT_OPERATION_KIND" --arg source "$TENANCIT_OPERATION_SOURCE" \
  '.kind == $kind and .source == $source' "$tmp" >/dev/null

echo "operation report accepted: $TENANCIT_OPERATION_KIND/$TENANCIT_OPERATION_STATUS"
