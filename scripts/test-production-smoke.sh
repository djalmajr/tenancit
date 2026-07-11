#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

cat >"$tmpdir/curl" <<'EOF'
#!/bin/sh
output=''
response_headers=''
authenticated=false
url=''
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) shift; output="$1" ;;
    -D) shift; response_headers="$1" ;;
    -H) shift; case "$1" in @*) authenticated=true ;; esac ;;
    http*) url="$1" ;;
  esac
  shift
done
status=200
body='{}'
case "$url" in
  */healthz) body='{"status":"ok"}' ;;
  */readyz) body='{"status":"healthy"}' ;;
  */v1/auth/config) body='{"mode":"oidc","login_url":"/v1/auth/login"}' ;;
  */v1/auth/login) status=302; printf 'Location: https://id.invalid/authorize\r\n' >"$response_headers" ;;
  */v1/admin/overview) status=401 ;;
  */v1/identify)
    if [ "$authenticated" = true ]; then body='{"tenantSlug":"smoke"}'; else status=401; fi
    ;;
esac
if [ -n "$output" ] && [ "$output" != '/dev/null' ]; then printf '%s\n' "$body" >"$output"; fi
printf '%s' "$status"
EOF
chmod +x "$tmpdir/curl"

PATH="$tmpdir:$PATH" \
TENANCIT_BASE_URL='https://tenancit.invalid' \
TENANCIT_SMOKE_API_TOKEN='tnc_smoke_token' \
TENANCIT_SMOKE_HOSTNAME='smoke.invalid' \
TENANCIT_SMOKE_TENANT_SLUG='smoke' \
  "$root_dir/scripts/post-deploy-production-smoke.sh" >/dev/null

echo "production smoke contract tests passed"
